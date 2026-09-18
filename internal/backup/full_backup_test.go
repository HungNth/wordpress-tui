package backup_test

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wptui/internal/backup"
)

type mockWPClient struct {
	runFn func(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error)
	calls []string
}

func (m *mockWPClient) Run(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
	call := fmt.Sprintf("%s %s in %s", name, strings.Join(args, " "), dir)
	m.calls = append(m.calls, call)
	if m.runFn != nil {
		return m.runFn(ctx, dir, name, args, stdin)
	}
	return "", "", nil
}

func TestRunFullBackup_SuccessAndSQLCleanup(t *testing.T) {
	siteDir := t.TempDir()
	slug := "test-site"
	backupDir := filepath.Join(t.TempDir(), "backups")

	// Create sample website files
	_ = os.WriteFile(filepath.Join(siteDir, "index.php"), []byte("<?php // test"), 0644)
	_ = os.WriteFile(filepath.Join(siteDir, "wp-config.php"), []byte("<?php // config"), 0644)
	_ = os.MkdirAll(filepath.Join(siteDir, "node_modules", "nested"), 0755)
	_ = os.WriteFile(filepath.Join(siteDir, "node_modules", "nested", "pkg.json"), []byte("{}"), 0644)

	mock := &mockWPClient{
		runFn: func(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
			if len(args) >= 3 && args[0] == "db" && args[1] == "export" {
				sqlFile := filepath.Join(dir, args[2])
				_ = os.WriteFile(sqlFile, []byte("-- MySQL dump"), 0644)
				return "Success: Exported to " + args[2], "", nil
			}
			return "", "", nil
		},
	}

	excludes := []string{"node_modules", ".git"}
	res, err := backup.RunFullBackup(context.Background(), siteDir, slug, backupDir, excludes, mock, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res == nil || res.Strategy != backup.StrategyFull {
		t.Fatalf("expected StrategyFull, got: %+v", res)
	}

	// 1. Verify SQL dump is removed from siteDir
	expectedSQL := filepath.Join(siteDir, slug+".sql")
	if _, err := os.Stat(expectedSQL); !os.IsNotExist(err) {
		t.Errorf("temporary SQL dump %s should be deleted after successful backup", expectedSQL)
	}

	// 2. Verify backup zip file exists in backupDir
	if _, err := os.Stat(res.FilePath); err != nil {
		t.Errorf("backup zip file %s not found: %v", res.FilePath, err)
	}

	// 3. Inspect zip contents: ensure forward slashes, includes SQL, excludes node_modules
	r, err := zip.OpenReader(res.FilePath)
	if err != nil {
		t.Fatalf("failed to open generated zip: %v", err)
	}
	defer r.Close()

	foundSQLInZip := false
	foundNodeModulesInZip := false

	for _, f := range r.File {
		if strings.Contains(f.Name, "\\") {
			t.Errorf("zip entry %q must not contain backslashes", f.Name)
		}
		if f.Name == slug+".sql" {
			foundSQLInZip = true
		}
		if strings.Contains(f.Name, "node_modules") {
			foundNodeModulesInZip = true
		}
	}

	if !foundSQLInZip {
		t.Errorf("expected %s to be packaged in zip", slug+".sql")
	}
	if foundNodeModulesInZip {
		t.Errorf("node_modules should have been excluded from zip")
	}
}

func TestRunFullBackup_NestedBackupDirExcluded(t *testing.T) {
	siteDir := t.TempDir()
	slug := "nested-backup-site"
	nestedBackupDir := filepath.Join(siteDir, "backups")

	_ = os.WriteFile(filepath.Join(siteDir, "index.php"), []byte("<?php"), 0644)
	_ = os.MkdirAll(nestedBackupDir, 0755)
	_ = os.WriteFile(filepath.Join(nestedBackupDir, "old-backup.tar"), []byte("tar"), 0644)

	mock := &mockWPClient{
		runFn: func(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
			if len(args) >= 3 && args[0] == "db" && args[1] == "export" {
				sqlFile := filepath.Join(dir, args[2])
				_ = os.WriteFile(sqlFile, []byte("-- dump"), 0644)
				return "", "", nil
			}
			return "", "", nil
		},
	}

	res, err := backup.RunFullBackup(context.Background(), siteDir, slug, nestedBackupDir, nil, mock, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r, err := zip.OpenReader(res.FilePath)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "backups/") {
			t.Errorf("nested backups directory should have been skipped from zip, found entry: %s", f.Name)
		}
	}
}

func TestRunFullBackup_CleanupFailureReturnsError(t *testing.T) {
	siteDir := t.TempDir()
	slug := "cleanup-fail-site"
	backupDir := filepath.Join(t.TempDir(), "backups")

	_ = os.WriteFile(filepath.Join(siteDir, "index.php"), []byte("<?php"), 0644)

	mock := &mockWPClient{
		runFn: func(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
			if len(args) >= 3 && args[0] == "db" && args[1] == "export" {
				// Create a non-empty directory in place of the dump file so os.Remove fails
				// deterministically on both Windows and POSIX after the archive succeeds.
				dumpDir := filepath.Join(dir, args[2])
				_ = os.MkdirAll(dumpDir, 0755)
				_ = os.WriteFile(filepath.Join(dumpDir, "locked"), []byte("x"), 0644)
				return "", "", nil
			}
			return "", "", nil
		},
	}

	_, err := backup.RunFullBackup(context.Background(), siteDir, slug, backupDir, nil, mock, nil)
	if err == nil {
		t.Fatal("expected error when temporary dump cannot be removed, got nil")
	}
	if !strings.Contains(err.Error(), "failed to remove temporary database dump") {
		t.Errorf("expected cleanup failure error, got: %v", err)
	}

	// The archive itself must still have been produced before cleanup was attempted
	entries, _ := os.ReadDir(backupDir)
	foundZip := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "full_"+slug+"_") && strings.HasSuffix(e.Name(), ".zip") {
			foundZip = true
		}
	}
	if !foundZip {
		t.Errorf("expected archive to be created before cleanup failure, found: %v", entries)
	}
}

func TestRunFullBackup_RelocationFailureRetainsSQLDump(t *testing.T) {
	siteDir := t.TempDir()
	slug := "test-site-reloc-fail"
	backupDir := filepath.Join(t.TempDir(), "backups")

	_ = os.WriteFile(filepath.Join(siteDir, "index.php"), []byte("<?php"), 0644)

	mock := &mockWPClient{
		runFn: func(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
			if len(args) >= 3 && args[0] == "db" && args[1] == "export" {
				sqlFile := filepath.Join(dir, args[2])
				_ = os.WriteFile(sqlFile, []byte("-- dump"), 0644)
				return "", "", nil
			}
			return "", "", nil
		},
	}

	// Inject a failing relocator to simulate move/permission/disk-full failure AFTER zip creation
	restore := backup.SetRelocatorForTesting(func(src, dest string) error {
		return errors.New("simulated disk full / permission denied on move")
	})
	defer restore()

	_, err := backup.RunFullBackup(context.Background(), siteDir, slug, backupDir, nil, mock, nil)
	if err == nil {
		t.Fatal("expected error on simulated relocation failure, got nil")
	}
	if !strings.Contains(err.Error(), "simulated disk full") {
		t.Errorf("expected relocation error message, got: %v", err)
	}

	// SQL dump must be retained on disk after relocation failure
	expectedSQL := filepath.Join(siteDir, slug+".sql")
	if _, err := os.Stat(expectedSQL); os.IsNotExist(err) {
		t.Errorf("SQL dump %s must be retained when archive relocation fails", expectedSQL)
	}
}

func TestRelocateFile_SuccessMovesContentAndRemovesSource(t *testing.T) {
	tempDir := t.TempDir()
	src := filepath.Join(tempDir, "source.zip")
	dest := filepath.Join(tempDir, "destination.zip")
	want := []byte("archive-content")
	if err := os.WriteFile(src, want, 0644); err != nil {
		t.Fatal(err)
	}

	if err := backup.RelocateFile(src, dest); err != nil {
		t.Fatalf("unexpected relocation error: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("expected source to be removed, stat err: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("destination content = %q, want %q", got, want)
	}
}

func TestRelocateFile_FailureRetainsSourceAndCleansTempDestination(t *testing.T) {
	tempDir := t.TempDir()
	src := filepath.Join(tempDir, "source.zip")
	destDir := filepath.Join(tempDir, "existing-directory")
	if err := os.WriteFile(src, []byte("archive-content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := backup.RelocateFile(src, destDir); err == nil {
		t.Fatal("expected relocation error when destination is a directory")
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("source must remain after relocation failure: %v", err)
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".wptui-relocate-") {
			t.Errorf("temporary destination file was not cleaned up: %s", entry.Name())
		}
	}
}
