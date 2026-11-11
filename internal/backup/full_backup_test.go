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
	"time"

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

func TestCreateFullZipArchive_CanceledContextReturnsCanceledAndClosesArchive(t *testing.T) {
	siteDir := t.TempDir()
	destZip := filepath.Join(t.TempDir(), "canceled.zip")
	if err := os.WriteFile(filepath.Join(siteDir, "index.php"), []byte("<?php"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := backup.CreateFullZipArchive(ctx, siteDir, destZip, "", nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("CreateFullZipArchive error = %v, want context.Canceled", err)
	}

	// Cancellation must still finalize and close the archive before returning.
	reader, openErr := zip.OpenReader(destZip)
	if openErr != nil {
		t.Fatalf("canceled archive was not finalized/closed: %v", openErr)
	}
	_ = reader.Close()
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

func TestRelocateFile_CollisionPreservesExistingDestinationAndSource(t *testing.T) {
	tempDir := t.TempDir()
	src := filepath.Join(tempDir, "new-archive.zip")
	dest := filepath.Join(tempDir, "existing-archive.zip")

	existingContent := []byte("original-protected-archive")
	newContent := []byte("new-attempt-archive")
	if err := os.WriteFile(dest, existingContent, 0644); err != nil {
		t.Fatal(err)
	}
	fixedTime := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(dest, fixedTime, fixedTime); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, newContent, 0644); err != nil {
		t.Fatal(err)
	}

	err := backup.RelocateFile(src, dest)
	if err == nil {
		t.Fatal("expected collision error when destination archive already exists")
	}
	if !errors.Is(err, os.ErrExist) {
		t.Errorf("expected errors.Is(err, os.ErrExist), got: %v", err)
	}
	if !strings.Contains(err.Error(), dest) {
		t.Errorf("expected collision error to identify destination %q, got: %v", dest, err)
	}

	gotDest, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed reading existing destination: %v", err)
	}
	if string(gotDest) != string(existingContent) {
		t.Errorf("existing destination content = %q, want %q", gotDest, existingContent)
	}
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("failed stating existing destination: %v", err)
	}
	if !info.ModTime().Equal(fixedTime) {
		t.Errorf("existing destination mtime changed: got %v, want %v", info.ModTime(), fixedTime)
	}

	gotSrc, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("expected source archive to be preserved: %v", err)
	}
	if string(gotSrc) != string(newContent) {
		t.Errorf("source content = %q, want %q", gotSrc, newContent)
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

func TestRelocateFile_CrossVolumeSuccessAndCollision(t *testing.T) {
	cwdVol := filepath.VolumeName(mustAbs(t, "."))
	srcBase := t.TempDir()
	srcVol := filepath.VolumeName(srcBase)

	var destBase string
	if !strings.EqualFold(srcVol, cwdVol) {
		var err error
		destBase, err = os.MkdirTemp(".", ".wptui-crossvol-test-*")
		if err != nil {
			t.Skipf("cross-volume temp dir unavailable in workspace: %v", err)
		}
		destBase = mustAbs(t, destBase)
		t.Cleanup(func() { _ = os.RemoveAll(destBase) })
	} else {
		t.Skipf("t.TempDir() (%s) and workspace (%s) share volume; cross-volume surface unavailable", srcVol, cwdVol)
	}

	// 1. Cross-volume success
	src1 := filepath.Join(srcBase, "cross-source.zip")
	dest1 := filepath.Join(destBase, "cross-dest.zip")
	want := []byte("cross-volume-archive-payload")
	if err := os.WriteFile(src1, want, 0644); err != nil {
		t.Fatal(err)
	}
	if err := backup.RelocateFile(src1, dest1); err != nil {
		t.Fatalf("cross-volume RelocateFile failed: %v", err)
	}
	if _, err := os.Stat(src1); !os.IsNotExist(err) {
		t.Errorf("expected cross-volume source to be removed, stat err: %v", err)
	}
	got, err := os.ReadFile(dest1)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("cross-volume dest = %q, want %q", got, want)
	}

	// 2. Cross-volume collision preserves existing dest and source
	src2 := filepath.Join(srcBase, "cross-new.zip")
	if err := os.WriteFile(src2, []byte("new-payload"), 0644); err != nil {
		t.Fatal(err)
	}
	err = backup.RelocateFile(src2, dest1)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected cross-volume collision os.ErrExist, got: %v", err)
	}
	gotAfter, _ := os.ReadFile(dest1)
	if string(gotAfter) != string(want) {
		t.Errorf("cross-volume existing dest overwritten: got %q, want %q", gotAfter, want)
	}
	if _, err := os.Stat(src2); err != nil {
		t.Errorf("cross-volume colliding source should remain: %v", err)
	}
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestRunFullBackup_CollisionAndCancellationStructuredArtifactMetadata(t *testing.T) {
	siteDir := t.TempDir()
	slug := "structured-full"
	backupDir := filepath.Join(t.TempDir(), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(siteDir, "index.php"), []byte("<?php"), 0644); err != nil {
		t.Fatal(err)
	}

	fixedNow := time.Date(2026, 11, 11, 10, 20, 30, 0, time.Local)
	restoreNow := backup.SetNowForTesting(func() time.Time { return fixedNow })
	defer restoreNow()

	collidingDest := filepath.Join(backupDir, fmt.Sprintf("full_%s_%s.zip", slug, fixedNow.Format("2006-01-02_15-04-05")))
	originalBytes := []byte("existing-full-zip")
	if err := os.WriteFile(collidingDest, originalBytes, 0644); err != nil {
		t.Fatal(err)
	}

	mock := &mockWPClient{
		runFn: func(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
			if len(args) >= 3 && args[0] == "db" && args[1] == "export" {
				_ = os.WriteFile(filepath.Join(dir, args[2]), []byte("-- sql dump"), 0644)
			}
			return "", "", nil
		},
	}

	_, err := backup.RunFullBackup(t.Context(), siteDir, slug, backupDir, nil, mock, nil)
	if err == nil {
		t.Fatal("expected collision error from RunFullBackup")
	}

	artErr, ok := errors.AsType[*backup.ArtifactError](err)
	if !ok {
		t.Fatalf("expected *backup.ArtifactError, got %T (%v)", err, err)
	}
	expectedSQL := filepath.Join(siteDir, slug+".sql")
	if artErr.Artifact.Kind != backup.ArtifactSQLDump || artErr.Artifact.Path != expectedSQL {
		t.Errorf("unexpected retained artifact metadata: %+v", artErr.Artifact)
	}
	if artErr.ConflictPath != collidingDest {
		t.Errorf("ConflictPath = %q, want %q", artErr.ConflictPath, collidingDest)
	}
	if artErr.Artifact.CleanupOutcome == "" {
		t.Errorf("expected non-empty CleanupOutcome")
	}
	gotExisting, _ := os.ReadFile(collidingDest)
	if string(gotExisting) != string(originalBytes) {
		t.Errorf("colliding archive altered: got %q, want %q", gotExisting, originalBytes)
	}
	if _, statErr := os.Stat(expectedSQL); statErr != nil {
		t.Errorf("expected SQL dump retained at %s: %v", expectedSQL, statErr)
	}

	// Cancellation after DB export must also return structured ArtifactError and retain dump.
	_ = os.Remove(expectedSQL)
	cancelCtx, cancel := context.WithCancel(t.Context())
	cancelMock := &mockWPClient{
		runFn: func(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
			if len(args) >= 3 && args[0] == "db" && args[1] == "export" {
				_ = os.WriteFile(filepath.Join(dir, args[2]), []byte("-- sql dump"), 0644)
				cancel()
			}
			return "", "", nil
		},
	}
	_, cancelErr := backup.RunFullBackup(cancelCtx, siteDir, slug, backupDir, nil, cancelMock, nil)
	if !errors.Is(cancelErr, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", cancelErr)
	}
	cancelArtErr, ok := errors.AsType[*backup.ArtifactError](cancelErr)
	if !ok {
		t.Fatalf("expected *backup.ArtifactError on cancellation after dump, got %T (%v)", cancelErr, cancelErr)
	}
	if cancelArtErr.Artifact.Path != expectedSQL {
		t.Errorf("cancel retained artifact path = %q, want %q", cancelArtErr.Artifact.Path, expectedSQL)
	}
}

func TestExportDatabase_CancelOrFailureReportsRetainedDumpOnlyWhenFileExists(t *testing.T) {
	t.Run("partial dump retained on cancel returns ArtifactError", func(t *testing.T) {
		siteDir := t.TempDir()
		slug := "bravo-slow"
		expectedSQL := filepath.Join(siteDir, slug+".sql")

		ctx, cancel := context.WithCancel(t.Context())
		mock := &mockWPClient{
			runFn: func(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
				if err := os.WriteFile(expectedSQL, []byte("-- partial dump\n"), 0644); err != nil {
					t.Fatal(err)
				}
				cancel()
				return "", "killed", ctx.Err()
			},
		}

		_, err := backup.RunFullBackup(ctx, siteDir, slug, filepath.Join(t.TempDir(), "backups"), nil, mock, nil)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got: %v", err)
		}
		artErr, ok := errors.AsType[*backup.ArtifactError](err)
		if !ok {
			t.Fatalf("expected *backup.ArtifactError when partial SQL dump exists on disk, got %T (%v)", err, err)
		}
		if artErr.Artifact.Kind != backup.ArtifactSQLDump || artErr.Artifact.Path != expectedSQL {
			t.Errorf("unexpected retained artifact: %+v", artErr.Artifact)
		}
		if artErr.Artifact.CleanupOutcome == "" {
			t.Errorf("expected non-empty CleanupOutcome")
		}
	})

	t.Run("failure without dump file returns plain error", func(t *testing.T) {
		siteDir := t.TempDir()
		mock := &mockWPClient{
			runFn: func(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
				return "", "connection refused", errors.New("exit status 1")
			},
		}
		_, err := backup.RunFullBackup(t.Context(), siteDir, "no-dump", filepath.Join(t.TempDir(), "backups"), nil, mock, nil)
		if err == nil {
			t.Fatal("expected error")
		}
		if _, ok := errors.AsType[*backup.ArtifactError](err); ok {
			t.Fatalf("did not expect *backup.ArtifactError when no dump file exists, got %v", err)
		}
	})
}
