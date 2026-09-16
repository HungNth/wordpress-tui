package packages

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"wptui/internal/config"
)

type Resolver struct {
	cfg   *config.Config
	cache *Cache
	opts  DownloadOptions
}

func NewResolver(cfg *config.Config, cache *Cache, opts ...DownloadOptions) *Resolver {
	var opt DownloadOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	return &Resolver{
		cfg:   cfg,
		cache: cache,
		opts:  opt,
	}
}

// ValidatePackageRef ensures type is plugin or theme and slug contains no path traversal or control codes.
func ValidatePackageRef(ref PackageRef) error {
	if ref.Type != "plugin" && ref.Type != "theme" {
		return fmt.Errorf("invalid package type %q (must be plugin or theme)", ref.Type)
	}
	if strings.TrimSpace(ref.Slug) == "" {
		return errors.New("package slug cannot be empty")
	}
	for _, r := range ref.Slug {
		if r < 32 || r == 127 || !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("package slug %q contains invalid characters, control codes, or path traversal", ref.Slug)
		}
	}
	return nil
}

// ResolvePackage checks the cache for an exact-version hit, downloads if missing or newer, and stores to cache.
func (r *Resolver) ResolvePackage(ctx context.Context, ref PackageRef, stageDir string) (*Artifact, error) {
	if err := ValidatePackageRef(ref); err != nil {
		return nil, err
	}

	if strings.TrimSpace(r.cfg.PackagesAPIURL) == "" {
		return nil, errors.New("packages API is disabled: packages_api_url is empty")
	}

	destPath := filepath.Join(stageDir, ref.Type, ref.Slug, "package.zip")
	destPath = filepath.Clean(destPath)
	rel, err := filepath.Rel(stageDir, destPath)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return nil, errors.New("destination path escapes staging directory")
	}

	var cachedEntry *CacheEntry
	var hasCache bool
	if r.cache != nil {
		if entry, found, err := r.cache.Get(ctx, ref); err == nil && found {
			cachedEntry = entry
			hasCache = true
		}
	}

	meta, err := FetchMetadata(ctx, nil, r.cfg.PackagesAPIURL, ref.Type, ref.Slug, r.cfg.PackagesAPIKey)
	if err != nil {
		if hasCache && IsTransientError(err) {
			stagedEntry, staged, stageErr := r.cache.StageTo(ctx, ref, destPath)
			if stageErr == nil && staged {
				return &Artifact{
					Ref:         ref,
					Version:     stagedEntry.Version,
					Path:        destPath,
					Size:        stagedEntry.Size,
					SHA256:      stagedEntry.SHA256,
					IsStale:     true,
					StaleReason: fmt.Sprintf("metadata fetch failed: %v", err),
				}, nil
			}
		}
		return nil, fmt.Errorf("failed to fetch metadata for %s %q: %w", ref.Type, ref.Slug, err)
	}

	size, err := strconv.ParseInt(strings.TrimSpace(meta.Size), 10, 64)
	if err != nil || size <= 0 {
		return nil, fmt.Errorf("invalid metadata size: %q", meta.Size)
	}

	// Exact-version cache hit: version and metadata size must both match
	if hasCache && cachedEntry.Version == meta.Version && cachedEntry.Size == size {
		stagedEntry, staged, stageErr := r.cache.StageTo(ctx, ref, destPath)
		if stageErr == nil && staged {
			return &Artifact{
				Ref:      ref,
				Version:  stagedEntry.Version,
				Path:     destPath,
				Size:     stagedEntry.Size,
				SHA256:   stagedEntry.SHA256,
				IsCached: true,
				IsStale:  false,
			}, nil
		}
	}

	if err := SafeDownload(ctx, meta.DownloadURL, size, destPath, r.opts); err != nil {
		if hasCache && IsTransientError(err) {
			// Reject metadata-size-mismatched artifacts on stale fallback when version matches
			if cachedEntry.Version == meta.Version && cachedEntry.Size != size {
				return nil, fmt.Errorf("cached artifact size %d does not match metadata size %d for version %s", cachedEntry.Size, size, meta.Version)
			}
			stagedEntry, staged, stageErr := r.cache.StageTo(ctx, ref, destPath)
			if stageErr == nil && staged {
				return &Artifact{
					Ref:         ref,
					Version:     stagedEntry.Version,
					Path:        destPath,
					Size:        stagedEntry.Size,
					SHA256:      stagedEntry.SHA256,
					IsStale:     true,
					StaleReason: fmt.Sprintf("download failed: %v", err),
				}, nil
			}
		}
		return nil, fmt.Errorf("failed to download package %q: %w", ref.Slug, err)
	}

	if r.cache != nil {
		stored, err := r.cache.Put(ctx, ref, meta.Version, destPath)
		if err == nil {
			return &Artifact{
				Ref:     ref,
				Version: stored.Version,
				Path:    destPath,
				Size:    stored.Size,
				SHA256:  stored.SHA256,
				IsStale: false,
			}, nil
		}
	}

	// Compute SHA-256 for non-cached artifact
	f, err := os.Open(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read downloaded archive: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return nil, fmt.Errorf("failed to compute hash: %w", err)
	}
	shaHex := hex.EncodeToString(hasher.Sum(nil))

	return &Artifact{
		Ref:     ref,
		Version: meta.Version,
		Path:    destPath,
		Size:    size,
		SHA256:  shaHex,
		IsStale: false,
	}, nil
}

// ResolveAll resolves a batch of packages, deduplicating references first.
func (r *Resolver) ResolveAll(ctx context.Context, refs []PackageRef, stageDir string) ([]Artifact, error) {
	uniqueRefs := DeduplicatePackages(refs)
	artifacts := make([]Artifact, 0, len(uniqueRefs))
	for _, ref := range uniqueRefs {
		art, err := r.ResolvePackage(ctx, ref, stageDir)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, *art)
	}
	return artifacts, nil
}
