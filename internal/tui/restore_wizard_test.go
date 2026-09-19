package tui_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func TestPrintRestoreSummary_MetadataFormatting(t *testing.T) {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	res := &restore.Result{
		WebsiteName: "My Site",
		SitePath:    "/sites/my-site",
		SiteURL:     "https://my-site.test",
		Database:    "my_site_db",
		AdminUser:   "admin",
		TLSError:    "failed to register cert",
	}

	tui.PrintRestoreSummary(res)

	_ = w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()

	if strings.Contains(out, "[✓]") {
		t.Errorf("expected clean metadata without decorative [✓] checkmark, got: %q", out)
	}
	if !strings.Contains(out, "Website Name:      My Site") {
		t.Errorf("expected Website Name in summary, got: %q", out)
	}
	if !strings.Contains(out, "https://my-site.test") {
		t.Errorf("expected URL in summary, got: %q", out)
	}
	if !strings.Contains(out, "[!]") || !strings.Contains(out, "failed to register cert") {
		t.Errorf("expected [!] warning in summary, got: %q", out)
	}
}
