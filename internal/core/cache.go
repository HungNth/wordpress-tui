package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

type Manifest struct {
	SchemaVersion int    `json:"schema_version"`
	Version       string `json:"version"`
	FilePath      string `json:"file_path"`
	Size          int64  `json:"size"`
	SHA256        string `json:"sha256"`
	DownloadedAt  string `json:"downloaded_at"`
}

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
	return filepath.Join(userCache, "wptui", "core"), nil
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
		return nil, fmt.Errorf("failed to create core cache directory: %w", err)
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
		return nil, fmt.Errorf("failed to initialize core cache lock or manifest: %w", err)
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
		return &Manifest{SchemaVersion: 1}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read core cache manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		corruptName := fmt.Sprintf("data.json.corrupt-%d", time.Now().UnixNano())
		corruptPath := filepath.Join(c.rootDir, corruptName)
		_ = os.Rename(p, corruptPath)
		return &Manifest{SchemaVersion: 1}, nil
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
	if cleanAbs != cleanRoot && !strings.HasPrefix(cleanAbs, cleanRoot+string(filepath.Separator)) {
		return "", errors.New("cache file path escapes cache root")
	}

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
	calculatedSHA := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(calculatedSHA, expectedSHA) {
		return fmt.Errorf("cached file SHA256 mismatch: %s != %s", calculatedSHA, expectedSHA)
	}

	return nil
}

func (c *Cache) Get(ctx context.Context) (*Manifest, string, bool, error) {
	var foundManifest *Manifest
	var absPath string
	var found bool

	err := c.withLock(ctx, func() error {
		m, err := c.loadOrQuarantineManifest()
		if err != nil {
			return err
		}

		if m.Version == "" || m.FilePath == "" {
			return nil
		}

		if err := c.validateFileIntegrity(m.FilePath, m.Size, m.SHA256); err != nil {
			// Cached archive is invalid or missing
			return nil
		}

		fullPath, err := c.safeRelativePath(m.FilePath)
		if err != nil {
			return nil
		}

		foundManifest = m
		absPath = fullPath
		found = true
		return nil
	})

	if err != nil {
		return nil, "", false, err
	}

	return foundManifest, absPath, found, nil
}

func (c *Cache) Put(ctx context.Context, version, srcFile string) (*Manifest, error) {
	if strings.TrimSpace(version) == "" {
		return nil, errors.New("core version cannot be empty")
	}

	fi, err := os.Stat(srcFile)
	if err != nil {
		return nil, fmt.Errorf("failed to stat source core archive: %w", err)
	}
	if !fi.Mode().IsRegular() {
		return nil, errors.New("source core archive must be a regular file")
	}

	hasher := sha256.New()
	f, err := os.Open(srcFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open source archive for hashing: %w", err)
	}
	if _, err := io.Copy(hasher, f); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failed to hash source archive: %w", err)
	}
	_ = f.Close()
	shaHex := hex.EncodeToString(hasher.Sum(nil))

	relPath := fmt.Sprintf("wordpress-%s-no-content.zip", version)
	destPath := filepath.Join(c.rootDir, relPath)
	destTmp := filepath.Join(c.rootDir, fmt.Sprintf("%s.tmp-%d", relPath, time.Now().UnixNano()))

	var committedManifest *Manifest

	err = c.withLock(ctx, func() error {
		sf, err := os.Open(srcFile)
		if err != nil {
			return err
		}
		defer sf.Close()

		df, err := os.OpenFile(destTmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}

		if _, err := io.Copy(df, sf); err != nil {
			_ = df.Close()
			_ = os.Remove(destTmp)
			return err
		}
		if err := df.Close(); err != nil {
			_ = os.Remove(destTmp)
			return err
		}

		m, err := c.loadOrQuarantineManifest()
		if err != nil {
			_ = os.Remove(destTmp)
			return err
		}

		oldFilePath := m.FilePath

		if err := os.Rename(destTmp, destPath); err != nil {
			_ = os.Remove(destTmp)
			return fmt.Errorf("failed to commit core archive: %w", err)
		}

		newManifest := &Manifest{
			SchemaVersion: 1,
			Version:       version,
			FilePath:      relPath,
			Size:          fi.Size(),
			SHA256:        shaHex,
			DownloadedAt:  time.Now().UTC().Format(time.RFC3339),
		}

		if err := c.saveManifest(newManifest); err != nil {
			_ = os.Remove(destPath)
			return err
		}

		// Prune previous version archive if different
		if oldFilePath != "" && oldFilePath != relPath {
			if oldSafePath, err := c.safeRelativePath(oldFilePath); err == nil {
				_ = os.Remove(oldSafePath)
			}
		}

		committedManifest = newManifest
		return nil
	})

	if err != nil {
		return nil, err
	}

	return committedManifest, nil
}
