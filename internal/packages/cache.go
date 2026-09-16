package packages

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

type Cache struct {
	rootDir  string
	lockPath string
	fileLock *flock.Flock
}

func DefaultCacheDir() (string, error) {
	userCache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user cache dir: %w", err)
	}
	return filepath.Join(userCache, "wptui", "packages"), nil
}

func NewCache(customDir ...string) (*Cache, error) {
	return NewCacheWithContext(context.Background(), customDir...)
}

func NewCacheWithContext(ctx context.Context, customDir ...string) (*Cache, error) {
	var dir string
	if len(customDir) > 0 && customDir[0] != "" {
		dir = customDir[0]
	} else {
		var err error
		dir, err = DefaultCacheDir()
		if err != nil {
			return nil, err
		}
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache root directory: %w", err)
	}

	lockPath := filepath.Join(dir, "data.lock")
	fileLock := flock.New(lockPath)

	c := &Cache{
		rootDir:  dir,
		lockPath: lockPath,
		fileLock: fileLock,
	}

	err := c.withLock(ctx, func() error {
		_, err := c.loadOrQuarantineManifest()
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache lock or manifest: %w", err)
	}

	return c, nil
}

func (c *Cache) RootDir() string {
	return c.rootDir
}

func (c *Cache) withLock(ctx context.Context, fn func() error) error {
	locked, err := c.fileLock.TryLockContext(ctx, 25*time.Millisecond)
	if err != nil {
		return fmt.Errorf("file lock error on %s: %w", c.lockPath, err)
	}
	if !locked {
		return fmt.Errorf("could not acquire lock on %s: %w", c.lockPath, ctx.Err())
	}
	defer func() {
		_ = c.fileLock.Unlock()
	}()
	return fn()
}

func (c *Cache) manifestPath() string {
	return filepath.Join(c.rootDir, "data.json")
}

func (c *Cache) loadOrQuarantineManifest() (*Manifest, error) {
	p := c.manifestPath()
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		m := &Manifest{SchemaVersion: 1, Packages: []CacheEntry{}}
		return m, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read cache manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		corruptName := fmt.Sprintf("data.json.corrupt-%d", time.Now().UnixNano())
		corruptPath := filepath.Join(c.rootDir, corruptName)
		_ = os.Rename(p, corruptPath)

		fresh := &Manifest{SchemaVersion: 1, Packages: []CacheEntry{}}
		return fresh, nil
	}

	return &m, nil
}

func (c *Cache) saveManifest(m *Manifest) error {
	tmpPath := filepath.Join(c.rootDir, fmt.Sprintf("data.json.tmp-%d", time.Now().UnixNano()))
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp manifest: %w", err)
	}

	if err := os.Rename(tmpPath, c.manifestPath()); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to commit manifest: %w", err)
	}

	return nil
}

func (c *Cache) safeRelativePath(relPath string) (string, error) {
	cleanRel := filepath.Clean(relPath)
	if filepath.IsAbs(cleanRel) || strings.HasPrefix(cleanRel, "..") {
		return "", errors.New("cache file path escapes cache root")
	}

	absPath := filepath.Join(c.rootDir, cleanRel)
	cleanAbs := filepath.Clean(absPath)
	cleanRoot := filepath.Clean(c.rootDir)
	if !strings.HasPrefix(cleanAbs, cleanRoot) {
		return "", errors.New("cache file path escapes cache root")
	}

	// Check every component from cleanAbs up to cleanRoot to prevent symlink directory traversal
	curr := cleanAbs
	for curr != cleanRoot && curr != filepath.Dir(curr) {
		fi, err := os.Lstat(curr)
		if err == nil && fi.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlink detected in path component %s", curr)
		}
		curr = filepath.Dir(curr)
	}

	return cleanAbs, nil
}

func (c *Cache) validateFileIntegrity(relPath string, expectedSize int64, expectedSHA string) error {
	absPath, err := c.safeRelativePath(relPath)
	if err != nil {
		return err
	}

	fi, err := os.Lstat(absPath)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
		return errors.New("cached artifact must be a regular file and cannot be a symlink")
	}
	if fi.Size() != expectedSize {
		return fmt.Errorf("cached file size mismatch: %d != %d", fi.Size(), expectedSize)
	}

	f, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return err
	}
	actualSHA := hex.EncodeToString(hasher.Sum(nil))
	if actualSHA != expectedSHA {
		return fmt.Errorf("cached artifact SHA256 mismatch: %s != %s", actualSHA, expectedSHA)
	}

	zr, err := zip.OpenReader(absPath)
	if err != nil {
		return fmt.Errorf("cached artifact is not a readable ZIP: %w", err)
	}
	_ = zr.Close()

	return nil
}

// Get finds the matching cache entry, validates its physical integrity, and returns it.
func (c *Cache) Get(ctx context.Context, ref PackageRef) (*CacheEntry, bool, error) {
	if err := ValidatePackageRef(ref); err != nil {
		return nil, false, err
	}

	var foundEntry *CacheEntry
	var found bool

	err := c.withLock(ctx, func() error {
		m, err := c.loadOrQuarantineManifest()
		if err != nil {
			return err
		}

		entryIdx := -1
		for i, p := range m.Packages {
			if p.Type == ref.Type && p.Slug == ref.Slug {
				entryIdx = i
				break
			}
		}

		if entryIdx < 0 {
			return nil
		}

		candidate := m.Packages[entryIdx]
		safePath, pathErr := c.safeRelativePath(candidate.FilePath)
		if pathErr != nil {
			m.Packages = append(m.Packages[:entryIdx], m.Packages[entryIdx+1:]...)
			_ = c.saveManifest(m)
			return nil
		}

		if err := c.validateFileIntegrity(candidate.FilePath, candidate.Size, candidate.SHA256); err != nil {
			_ = os.Remove(safePath)
			m.Packages = append(m.Packages[:entryIdx], m.Packages[entryIdx+1:]...)
			_ = c.saveManifest(m)
			return nil
		}

		foundEntry = &candidate
		found = true
		return nil
	})

	if err != nil {
		return nil, false, err
	}
	return foundEntry, found, nil
}

// StageTo validates the cached artifact under the cache lock and copies its verified bytes into destPath.
func (c *Cache) StageTo(ctx context.Context, ref PackageRef, destPath string) (*CacheEntry, bool, error) {
	if err := ValidatePackageRef(ref); err != nil {
		return nil, false, err
	}

	var foundEntry *CacheEntry
	var found bool

	err := c.withLock(ctx, func() error {
		m, err := c.loadOrQuarantineManifest()
		if err != nil {
			return err
		}

		entryIdx := -1
		for i, p := range m.Packages {
			if p.Type == ref.Type && p.Slug == ref.Slug {
				entryIdx = i
				break
			}
		}

		if entryIdx < 0 {
			return nil
		}

		candidate := m.Packages[entryIdx]
		safePath, pathErr := c.safeRelativePath(candidate.FilePath)
		if pathErr != nil {
			m.Packages = append(m.Packages[:entryIdx], m.Packages[entryIdx+1:]...)
			_ = c.saveManifest(m)
			return nil
		}

		if err := c.validateFileIntegrity(candidate.FilePath, candidate.Size, candidate.SHA256); err != nil {
			_ = os.Remove(safePath)
			m.Packages = append(m.Packages[:entryIdx], m.Packages[entryIdx+1:]...)
			_ = c.saveManifest(m)
			return nil
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		sf, err := os.Open(safePath)
		if err != nil {
			return err
		}
		defer sf.Close()

		df, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		defer df.Close()

		if _, err := io.Copy(df, sf); err != nil {
			return err
		}

		foundEntry = &candidate
		found = true
		return nil
	})

	if err != nil {
		return nil, false, err
	}
	return foundEntry, found, nil
}

// Put adds or atomically replaces a package entry in the cache, locking publication and manifest together.
func (c *Cache) Put(ctx context.Context, ref PackageRef, version, srcFile string) (*CacheEntry, error) {
	if err := ValidatePackageRef(ref); err != nil {
		return nil, err
	}
	if strings.TrimSpace(version) == "" {
		return nil, errors.New("version cannot be empty")
	}

	fi, err := os.Lstat(srcFile)
	if err != nil {
		return nil, fmt.Errorf("source artifact not found: %w", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
		return nil, errors.New("source artifact must be a regular non-symlink file")
	}

	// Compute hash of source file
	sf, err := os.Open(srcFile)
	if err != nil {
		return nil, err
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, sf); err != nil {
		_ = sf.Close()
		return nil, err
	}
	_ = sf.Close()
	shaHex := hex.EncodeToString(hasher.Sum(nil))

	// Verify it's a valid ZIP
	zr, err := zip.OpenReader(srcFile)
	if err != nil {
		return nil, fmt.Errorf("source file is not a valid ZIP: %w", err)
	}
	_ = zr.Close()

	safeVersion := url.PathEscape(version)
	safeVersion = strings.ReplaceAll(safeVersion, "/", "_")
	safeVersion = strings.ReplaceAll(safeVersion, `\`, "_")

	relPath := filepath.Join("files", string(ref.Type), ref.Slug, fmt.Sprintf("%s-%s.zip", safeVersion, shaHex[:12]))
	destPath, err := c.safeRelativePath(relPath)
	if err != nil {
		return nil, fmt.Errorf("invalid destination path: %w", err)
	}

	var newEntry CacheEntry
	err = c.withLock(ctx, func() error {
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create package cache folder: %w", err)
		}

		// Copy into destination path under lock
		destTmp := fmt.Sprintf("%s.tmp-%d", destPath, time.Now().UnixNano())
		df, err := os.OpenFile(destTmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		sf, err = os.Open(srcFile)
		if err != nil {
			_ = df.Close()
			_ = os.Remove(destTmp)
			return err
		}
		if _, err := io.Copy(df, sf); err != nil {
			_ = sf.Close()
			_ = df.Close()
			_ = os.Remove(destTmp)
			return err
		}
		_ = sf.Close()
		_ = df.Close()
		m, err := c.loadOrQuarantineManifest()
		if err != nil {
			_ = os.Remove(destTmp)
			return err
		}

		var oldFilePath string
		entryIdx := -1
		for i, p := range m.Packages {
			if p.Type == ref.Type && p.Slug == ref.Slug {
				entryIdx = i
				oldFilePath = p.FilePath
				break
			}
		}

		// If exact path is already committed, retain existing verified archive
		if oldFilePath != "" && oldFilePath == relPath {
			_ = os.Remove(destTmp)
			newEntry = CacheEntry{
				Type:         ref.Type,
				Slug:         ref.Slug,
				Version:      version,
				FilePath:     relPath,
				Size:         fi.Size(),
				SHA256:       shaHex,
				DownloadedAt: time.Now().UTC().Format(time.RFC3339),
			}
			m.Packages[entryIdx] = newEntry
			return c.saveManifest(m)
		}

		if err := os.Rename(destTmp, destPath); err != nil {
			_ = os.Remove(destTmp)
			return fmt.Errorf("failed to commit cache file: %w", err)
		}

		newEntry = CacheEntry{
			Type:         ref.Type,
			Slug:         ref.Slug,
			Version:      version,
			FilePath:     relPath,
			Size:         fi.Size(),
			SHA256:       shaHex,
			DownloadedAt: time.Now().UTC().Format(time.RFC3339),
		}

		if entryIdx >= 0 {
			m.Packages[entryIdx] = newEntry
		} else {
			m.Packages = append(m.Packages, newEntry)
		}

		if err := c.saveManifest(m); err != nil {
			_ = os.Remove(destPath)
			return err
		}

		// Delete old version file only after manifest is committed successfully
		if oldFilePath != "" && oldFilePath != relPath {
			if oldSafePath, err := c.safeRelativePath(oldFilePath); err == nil {
				_ = os.Remove(oldSafePath)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &newEntry, nil
}
