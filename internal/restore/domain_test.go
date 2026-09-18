package restore_test

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"wptui/internal/restore"
)

func TestFindWordPressRoot_Success(t *testing.T) {
	tempDir := t.TempDir()
	wpRoot := filepath.Join(tempDir, "subfolder", "wordpress")
	if err := os.MkdirAll(filepath.Join(wpRoot, "wp-content"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(wpRoot, "wp-includes"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wpRoot, "wp-settings.php"), []byte("<?php"), 0644); err != nil {
		t.Fatal(err)
	}

	found, err := restore.FindWordPressRoot(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != wpRoot {
		t.Errorf("got %q, want %q", found, wpRoot)
	}
}

func TestFindWordPressRoot_NoRoot(t *testing.T) {
	tempDir := t.TempDir()
	_, err := restore.FindWordPressRoot(tempDir)
	if err == nil || !errors.Is(err, restore.ErrNoWordPressRoot) {
		t.Fatalf("expected ErrNoWordPressRoot, got: %v", err)
	}
}

func TestFindWordPressRoot_AmbiguousRoot(t *testing.T) {
	tempDir := t.TempDir()
	for _, sub := range []string{"site1", "site2"} {
		root := filepath.Join(tempDir, sub)
		_ = os.MkdirAll(filepath.Join(root, "wp-content"), 0755)
		_ = os.MkdirAll(filepath.Join(root, "wp-includes"), 0755)
		_ = os.WriteFile(filepath.Join(root, "wp-settings.php"), []byte("<?php"), 0644)
	}

	_, err := restore.FindWordPressRoot(tempDir)
	if err == nil || !errors.Is(err, restore.ErrAmbiguousWordPressRoot) {
		t.Fatalf("expected ErrAmbiguousWordPressRoot, got: %v", err)
	}
}

func TestResolveSQLDump_StandardWPTUIArchive(t *testing.T) {
	wpRoot := t.TempDir()
	dumpPath := filepath.Join(wpRoot, "my-site.sql")
	if err := os.WriteFile(dumpPath, []byte("-- dump"), 0644); err != nil {
		t.Fatal(err)
	}
	// Also create an extra .sql file to ensure standard name takes precedence
	_ = os.WriteFile(filepath.Join(wpRoot, "extra.sql"), []byte("-- extra"), 0644)

	archivePath := "/path/to/full_my-site_2026-09-18_12-00-00.zip"
	chosen, err := restore.ResolveSQLDump(archivePath, wpRoot, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chosen != dumpPath {
		t.Errorf("got %q, want %q", chosen, dumpPath)
	}
}

func TestResolveSQLDump_StandardWPTUIArchive_MissingExpectedDumpFails(t *testing.T) {
	wpRoot := t.TempDir()
	// Only create extra.sql, omitting my-site.sql
	_ = os.WriteFile(filepath.Join(wpRoot, "extra.sql"), []byte("-- extra"), 0644)

	archivePath := "/path/to/full_my-site_2026-09-18_12-00-00.zip"
	_, err := restore.ResolveSQLDump(archivePath, wpRoot, nil)
	if err == nil {
		t.Fatal("expected error when standard archive is missing expected <slug>.sql, got nil")
	}
}

func TestResolveSQLDump_SingleFile(t *testing.T) {
	wpRoot := t.TempDir()
	dumpPath := filepath.Join(wpRoot, "database.sql")
	_ = os.WriteFile(dumpPath, []byte("-- dump"), 0644)

	chosen, err := restore.ResolveSQLDump("random-archive.zip", wpRoot, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chosen != dumpPath {
		t.Errorf("got %q, want %q", chosen, dumpPath)
	}
}

func TestResolveSQLDump_MultipleFilesInteractivePrompt(t *testing.T) {
	wpRoot := t.TempDir()
	dump1 := filepath.Join(wpRoot, "backup1.sql")
	dump2 := filepath.Join(wpRoot, "backup2.sql")
	_ = os.WriteFile(dump1, []byte("-- dump1"), 0644)
	_ = os.WriteFile(dump2, []byte("-- dump2"), 0644)

	promptCalled := false
	promptFn := func(candidates []string) (string, error) {
		promptCalled = true
		return dump2, nil
	}

	chosen, err := restore.ResolveSQLDump("random-archive.zip", wpRoot, promptFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !promptCalled {
		t.Error("expected prompt to be called")
	}
	if chosen != dump2 {
		t.Errorf("got %q, want %q", chosen, dump2)
	}
}

func TestResolveSQLDump_ZeroFiles(t *testing.T) {
	wpRoot := t.TempDir()
	_, err := restore.ResolveSQLDump("random-archive.zip", wpRoot, nil)
	if err == nil || !errors.Is(err, restore.ErrMissingSQLDump) {
		t.Fatalf("expected ErrMissingSQLDump, got: %v", err)
	}
}

func TestExtractTablePrefix_FromWPConfig(t *testing.T) {
	tempDir := t.TempDir()
	wpConfig := filepath.Join(tempDir, "wp-config.php")
	content := `<?php
define('DB_NAME', 'test');
$table_prefix = 'custom_wp_';
`
	_ = os.WriteFile(wpConfig, []byte(content), 0644)

	prefix, err := restore.ExtractTablePrefix(tempDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prefix != "custom_wp_" {
		t.Errorf("got %q, want 'custom_wp_'", prefix)
	}
}

func TestExtractTablePrefix_FromSQLDumpFallback(t *testing.T) {
	tempDir := t.TempDir()
	sqlDump := filepath.Join(tempDir, "dump.sql")
	content := `
-- WordPress Database Dump
CREATE TABLE IF NOT EXISTS ` + "`wptest_users`" + ` (
  ID bigint(20) unsigned NOT NULL auto_increment
);
`
	_ = os.WriteFile(sqlDump, []byte(content), 0644)

	prefix, err := restore.ExtractTablePrefix(tempDir, sqlDump)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prefix != "wptest_" {
		t.Errorf("got %q, want 'wptest_'", prefix)
	}
}

func TestExtractTablePrefix_UndeterminedFailsFast(t *testing.T) {
	tempDir := t.TempDir()
	_, err := restore.ExtractTablePrefix(tempDir, "")
	if err == nil || !errors.Is(err, restore.ErrUndeterminedTablePrefix) {
		t.Fatalf("expected ErrUndeterminedTablePrefix, got: %v", err)
	}
}

func TestExtractTablePrefix_FromSQLDumpFallback_AlternateTable(t *testing.T) {
	tempDir := t.TempDir()
	sqlDump := filepath.Join(tempDir, "dump.sql")
	content := `
-- WordPress Database Dump
CREATE TABLE IF NOT EXISTS ` + "`custompref_options`" + ` (
  option_id bigint(20) unsigned NOT NULL auto_increment
);
`
	_ = os.WriteFile(sqlDump, []byte(content), 0644)

	prefix, err := restore.ExtractTablePrefix(tempDir, sqlDump)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prefix != "custompref_" {
		t.Errorf("got %q, want 'custompref_'", prefix)
	}
}

func TestExtractTablePrefix_FromSQLDumpFallback_Multiline(t *testing.T) {
	tempDir := t.TempDir()
	sqlDump := filepath.Join(tempDir, "dump.sql")
	content := `
-- WordPress Database Dump
CREATE TABLE IF NOT EXISTS
  ` + "`multiline_posts`" + ` (
  ID bigint(20) unsigned NOT NULL auto_increment
);
`
	_ = os.WriteFile(sqlDump, []byte(content), 0644)

	prefix, err := restore.ExtractTablePrefix(tempDir, sqlDump)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prefix != "multiline_" {
		t.Errorf("got %q, want 'multiline_'", prefix)
	}
}

func TestExtractTablePrefix_FromSQLDumpFallback_PluginTablePrecedingCore(t *testing.T) {
	tempDir := t.TempDir()
	sqlDump := filepath.Join(tempDir, "dump.sql")
	content := `
CREATE TABLE ` + "`wp_plugin_users`" + ` (id int);
CREATE TABLE ` + "`wp_users`" + ` (id int);
CREATE TABLE ` + "`wp_posts`" + ` (id int);
CREATE TABLE ` + "`wp_options`" + ` (id int);
`
	_ = os.WriteFile(sqlDump, []byte(content), 0644)

	prefix, err := restore.ExtractTablePrefix(tempDir, sqlDump)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prefix != "wp_" {
		t.Errorf("got %q, want 'wp_'", prefix)
	}
}

func TestExtractTablePrefix_FromSQLDumpFallback_TieFailsFast(t *testing.T) {
	tempDir := t.TempDir()
	sqlDump := filepath.Join(tempDir, "dump.sql")
	content := `
CREATE TABLE ` + "`prefixa_users`" + ` (id int);
CREATE TABLE ` + "`prefixb_posts`" + ` (id int);
`
	_ = os.WriteFile(sqlDump, []byte(content), 0644)

	_, err := restore.ExtractTablePrefix(tempDir, sqlDump)
	if err == nil || !errors.Is(err, restore.ErrUndeterminedTablePrefix) {
		t.Fatalf("expected ErrUndeterminedTablePrefix on tie, got: %v", err)
	}
}

func TestExtractZipArchive(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	zf, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(zf)
	w, err := zw.Create("sub/test.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("hello restore"))
	_ = zw.Close()
	_ = zf.Close()

	if err := restore.ExtractZipArchive(zipPath, destDir); err != nil {
		t.Fatalf("ExtractZipArchive failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(destDir, "sub", "test.txt"))
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(data) != "hello restore" {
		t.Errorf("got %q, want 'hello restore'", string(data))
	}
}
