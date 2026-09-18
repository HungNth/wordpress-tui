package tui_test

import (
	"os"
	"path/filepath"
	"testing"

	"wptui/internal/restore"
	"wptui/internal/tui"
)

func TestScanBackupArchives_OrderingAndExtensions(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tempDir, "archive1.zip"), []byte("data"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "archive2.wpress"), []byte("data"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "notes.txt"), []byte("data"), 0644)

	// Test StrategyFullZIP scans only .zip
	zipFiles, err := tui.ScanBackupArchives(tempDir, restore.StrategyFullZIP)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(zipFiles) != 1 || filepath.Base(zipFiles[0]) != "archive1.zip" {
		t.Errorf("expected only archive1.zip, got: %v", zipFiles)
	}

	// Test StrategyAI1WM scans only .wpress
	wpressFiles, err := tui.ScanBackupArchives(tempDir, restore.StrategyAI1WM)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wpressFiles) != 1 || filepath.Base(wpressFiles[0]) != "archive2.wpress" {
		t.Errorf("expected only archive2.wpress, got: %v", wpressFiles)
	}
}

func TestValidateArchivePath(t *testing.T) {
	tempDir := t.TempDir()
	validZip := filepath.Join(tempDir, "valid.zip")
	_ = os.WriteFile(validZip, []byte("data"), 0644)

	// Valid zip
	if err := tui.ValidateArchivePath(validZip, restore.StrategyFullZIP); err != nil {
		t.Errorf("expected valid zip to pass validation, got: %v", err)
	}

	// Non-existent file
	if err := tui.ValidateArchivePath(filepath.Join(tempDir, "missing.zip"), restore.StrategyFullZIP); err == nil {
		t.Error("expected error for non-existent file")
	}

	// Directory instead of file
	if err := tui.ValidateArchivePath(tempDir, restore.StrategyFullZIP); err == nil {
		t.Error("expected error for directory path")
	}

	// Wrong extension
	invalidExt := filepath.Join(tempDir, "archive.tar")
	_ = os.WriteFile(invalidExt, []byte("data"), 0644)
	if err := tui.ValidateArchivePath(invalidExt, restore.StrategyFullZIP); err == nil {
		t.Error("expected error for non-zip extension")
	}
}
