package core_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"


	"wptui/internal/core"
)

func createSampleZip(t *testing.T, dir, filename string) (string, string, int64) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("wordpress/index.php")
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("<?php // test core")
	if _, err := w.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	data := buf.Bytes()
	sum := sha256.Sum256(data)
	sumHex := hex.EncodeToString(sum[:])

	fullPath := filepath.Join(dir, filename)
	if err := os.WriteFile(fullPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	return fullPath, sumHex, int64(len(data))
}

func TestDefaultCacheDir(t *testing.T) {
	dir, err := core.DefaultCacheDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(filepath.ToSlash(dir), "wptui/core") {
		t.Errorf("expected path ending with wptui/core, got %s", dir)
	}
}

func TestCache_Empty(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := core.NewCache(tempDir)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}

	manifest, archivePath, found, err := cache.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Errorf("expected empty cache, found %+v", manifest)
	}
	if archivePath != "" {
		t.Errorf("expected empty archive path, got %s", archivePath)
	}
}

func TestCache_PutAndGet(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := core.NewCache(tempDir)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}

	srcDir := t.TempDir()
	srcFile, expectedSHA, expectedSize := createSampleZip(t, srcDir, "sample-7.1.zip")

	ctx := context.Background()
	manifest, err := cache.Put(ctx, "7.1", srcFile)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if manifest.Version != "7.1" {
		t.Errorf("expected version 7.1, got %s", manifest.Version)
	}
	if manifest.SHA256 != expectedSHA {
		t.Errorf("expected sha256 %s, got %s", expectedSHA, manifest.SHA256)
	}
	if manifest.Size != expectedSize {
		t.Errorf("expected size %d, got %d", expectedSize, manifest.Size)
	}

	// Verify Get retrieves the committed archive
	gotManifest, archivePath, found, err := cache.Get(ctx)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("expected cached archive to be found")
	}
	if gotManifest.Version != "7.1" {
		t.Errorf("expected version 7.1, got %s", gotManifest.Version)
	}
	if !strings.HasSuffix(archivePath, "wordpress-7.1-no-content.zip") {
		t.Errorf("expected archive name wordpress-7.1-no-content.zip, got %s", archivePath)
	}

	// File on disk must exist and match
	fi, err := os.Stat(archivePath)
	if err != nil {
		t.Fatalf("failed to stat cached file: %v", err)
	}
	if fi.Size() != expectedSize {
		t.Errorf("file size on disk mismatch: %d != %d", fi.Size(), expectedSize)
	}
}

func TestCache_PurgeOldVersion(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := core.NewCache(tempDir)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}

	srcDir := t.TempDir()
	fileV1, _, _ := createSampleZip(t, srcDir, "sample-7.0.zip")
	fileV2, _, _ := createSampleZip(t, srcDir, "sample-7.1.zip")

	ctx := context.Background()
	_, err = cache.Put(ctx, "7.0", fileV1)
	if err != nil {
		t.Fatalf("Put v1 failed: %v", err)
	}

	v1Path := filepath.Join(cache.RootDir(), "wordpress-7.0-no-content.zip")
	if _, err := os.Stat(v1Path); err != nil {
		t.Fatalf("expected v1 archive to exist before upgrade: %v", err)
	}

	// Put v2
	_, err = cache.Put(ctx, "7.1", fileV2)
	if err != nil {
		t.Fatalf("Put v2 failed: %v", err)
	}

	// V1 archive should be pruned
	if _, err := os.Stat(v1Path); !os.IsNotExist(err) {
		t.Errorf("expected v1 archive to be removed after upgrade, stat err: %v", err)
	}

	// V2 archive must exist
	v2Path := filepath.Join(cache.RootDir(), "wordpress-7.1-no-content.zip")
	if _, err := os.Stat(v2Path); err != nil {
		t.Errorf("expected v2 archive to exist, stat err: %v", err)
	}
}

func TestCache_CorruptManifestQuarantine(t *testing.T) {
	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, "data.json")
	if err := os.WriteFile(manifestFile, []byte("{invalid json corrupt"), 0600); err != nil {
		t.Fatal(err)
	}

	cache, err := core.NewCache(tempDir)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	// Loading should succeed with fresh manifest while corrupt file is renamed
	manifest, _, found, err := cache.Get(context.Background())
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Errorf("expected no found entry from quarantined manifest, got %+v", manifest)
	}

	entries, _ := os.ReadDir(tempDir)
	var foundCorrupt bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "data.json.corrupt-") {
			foundCorrupt = true
			break
		}
	}
	if !foundCorrupt {
		t.Errorf("expected quarantined file data.json.corrupt-* to exist in %s", tempDir)
	}
}
