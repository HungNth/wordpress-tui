package core

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultVersionCheckURL = "https://api.wordpress.org/core/version-check/1.7/"
const DefaultCheckTimeout = 3 * time.Second

type CorePackages struct {
	Full      string `json:"full"`
	NoContent string `json:"no_content"`
}

type CoreOffer struct {
	Response string       `json:"response"`
	Current  string       `json:"current"`
	Version  string       `json:"version"`
	Packages CorePackages `json:"packages"`
}

type CoreVersionCheckResponse struct {
	Offers []CoreOffer `json:"offers"`
}

type ResolverOption func(*Resolver)

func WithVersionCheckURL(u string) ResolverOption {
	return func(r *Resolver) {
		r.versionCheckURL = u
	}
}

func WithHTTPClient(client *http.Client) ResolverOption {
	return func(r *Resolver) {
		r.httpClient = client
	}
}

func WithCheckTimeout(d time.Duration) ResolverOption {
	return func(r *Resolver) {
		r.checkTimeout = d
	}
}

type Resolver struct {
	cache           *Cache
	httpClient      *http.Client
	versionCheckURL string
	checkTimeout    time.Duration
}

func NewResolver(cache *Cache, opts ...ResolverOption) *Resolver {
	r := &Resolver{
		cache:           cache,
		httpClient:      &http.Client{Timeout: 5 * time.Minute},
		versionCheckURL: DefaultVersionCheckURL,
		checkTimeout:    DefaultCheckTimeout,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *Resolver) checkVersion(ctx context.Context) (*CoreOffer, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.versionCheckURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create version check request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("version check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("version check returned HTTP status %d", resp.StatusCode)
	}

	var data CoreVersionCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode version check response: %w", err)
	}

	if len(data.Offers) == 0 {
		return nil, errors.New("no core offers returned by version check")
	}

	offer := data.Offers[0]
	if offer.Packages.NoContent == "" {
		return nil, errors.New("version check offer did not contain a no_content download package")
	}

	return &offer, nil
}

func (r *Resolver) downloadAndCommit(ctx context.Context, version, downloadURL string) (string, error) {
	partPath := filepath.Join(r.cache.RootDir(), fmt.Sprintf("core-download-%d.part", time.Now().UnixNano()))
	defer func() {
		_ = os.Remove(partPath)
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("core download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("core download returned HTTP status %d", resp.StatusCode)
	}

	partFile, err := os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to create temp download file: %w", err)
	}

	if _, err := io.Copy(partFile, resp.Body); err != nil {
		_ = partFile.Close()
		return "", fmt.Errorf("failed to stream core download: %w", err)
	}
	if err := partFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp download file: %w", err)
	}

	// Validate ZIP archive structure
	zr, err := zip.OpenReader(partPath)
	if err != nil {
		return "", fmt.Errorf("downloaded core archive is corrupt or invalid zip: %w", err)
	}
	_ = zr.Close()

	manifest, err := r.cache.Put(ctx, version, partPath)
	if err != nil {
		return "", fmt.Errorf("failed to commit core archive to cache: %w", err)
	}

	safePath, err := r.cache.safeRelativePath(manifest.FilePath)
	if err != nil {
		return "", err
	}

	return safePath, nil
}

func (r *Resolver) Resolve(ctx context.Context) (string, string, error) {
	checkCtx, cancel := context.WithTimeout(ctx, r.checkTimeout)
	defer cancel()

	offer, checkErr := r.checkVersion(checkCtx)
	var downloadErr error

	if checkErr == nil && offer != nil {
		targetVersion := offer.Version
		if strings.TrimSpace(targetVersion) == "" {
			targetVersion = offer.Current
		}

		if m, absPath, found, err := r.cache.Get(ctx); err == nil && found && m.Version == targetVersion {
			// Cache hit: latest version is already cached
			return absPath, m.Version, nil
		}

		// Download new/updated version
		absPath, err := r.downloadAndCommit(ctx, targetVersion, offer.Packages.NoContent)
		if err == nil {
			return absPath, targetVersion, nil
		}
		downloadErr = err
	}

	// Fallback to existing cache if network failed or download failed
	if m, absPath, found, err := r.cache.Get(ctx); err == nil && found {
		return absPath, m.Version, nil
	}

	return "", "", fmt.Errorf("initial WordPress core download failed and no cached core archive exists: check error: %v, download error: %v (network connection required to download WordPress core)", checkErr, downloadErr)
}

// ResolveCoreArchive is a convenience function that initializes the default Core Cache
// and resolves the latest verified Core Archive.
func ResolveCoreArchive(ctx context.Context, customDir ...string) (string, string, error) {
	cache, err := NewCache(customDir...)
	if err != nil {
		return "", "", fmt.Errorf("failed to initialize core cache: %w", err)
	}
	return NewResolver(cache).Resolve(ctx)
}

