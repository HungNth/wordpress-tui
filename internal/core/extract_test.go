package core_test

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wptui/internal/core"
)

func createMultiEntryZip(t *testing.T, entries map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for path, content := range entries {
		w, err := zw.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")
	if err := os.WriteFile(zipPath, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return zipPath
}

func TestExtractCoreArchive_Success(t *testing.T) {
	zipPath := createMultiEntryZip(t, map[string]string{
		"wordpress/":                         "",
		"wordpress/index.php":                "<?php // index",
		"wordpress/wp-includes/":             "",
		"wordpress/wp-includes/version.php":  "<?php $wp_version = '7.1';",
		"wordpress/wp-admin/admin.php":       "<?php // admin",
	})

	destDir := t.TempDir()
	siteDir := filepath.Join(destDir, "mysite")

	err := core.ExtractCoreArchive(zipPath, siteDir)
	if err != nil {
		t.Fatalf("ExtractCoreArchive failed: %v", err)
	}

	// Verify root-stripped files
	expectedFiles := []string{
		filepath.Join(siteDir, "index.php"),
		filepath.Join(siteDir, "wp-includes", "version.php"),
		filepath.Join(siteDir, "wp-admin", "admin.php"),
	}
	for _, f := range expectedFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Errorf("expected file %s to exist: %v", f, err)
		}
		if len(data) == 0 {
			t.Errorf("expected non-empty file %s", f)
		}
	}

	// Verify standard WordPress directories scaffolded
	expectedDirs := []string{
		filepath.Join(siteDir, "wp-content"),
		filepath.Join(siteDir, "wp-content", "themes"),
		filepath.Join(siteDir, "wp-content", "plugins"),
	}
	for _, d := range expectedDirs {
		fi, err := os.Stat(d)
		if err != nil {
			t.Errorf("expected directory %s to exist: %v", d, err)
		} else if !fi.IsDir() {
			t.Errorf("expected %s to be a directory", d)
		}
	}
}

func TestExtractCoreArchive_ZipSlipProtection(t *testing.T) {
	zipPath := createMultiEntryZip(t, map[string]string{
		"wordpress/index.php":              "<?php // safe",
		"wordpress/../../outside.txt":      "pwned",
	})

	destDir := t.TempDir()
	siteDir := filepath.Join(destDir, "mysite")

	err := core.ExtractCoreArchive(zipPath, siteDir)
	if err == nil {
		t.Fatal("expected Zip Slip archive to be rejected with error")
	}
	if !strings.Contains(err.Error(), "illegal") && !strings.Contains(err.Error(), "escapes") {
		t.Errorf("expected error to mention illegal path escaping, got: %v", err)
	}

	outsideFile := filepath.Join(destDir, "outside.txt")
	if _, err := os.Stat(outsideFile); !os.IsNotExist(err) {
		t.Errorf("expected outside file not to be created, stat err: %v", err)
	}
}

func TestExtractCoreArchive_InvalidFile(t *testing.T) {
	tempDir := t.TempDir()
	badFile := filepath.Join(tempDir, "not-a-zip.txt")
	_ = os.WriteFile(badFile, []byte("plain text"), 0600)

	siteDir := filepath.Join(tempDir, "site")
	err := core.ExtractCoreArchive(badFile, siteDir)
	if err == nil {
		t.Fatal("expected error extracting non-zip file")
	}
}
