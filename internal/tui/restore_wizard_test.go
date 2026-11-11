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

func TestScanBackupArchives_MissingOrEmptyDirectoryReturnsError(t *testing.T) {
	emptyExisting := t.TempDir()
	files, err := tui.ScanBackupArchives(emptyExisting, restore.StrategyFullZIP)
	if err != nil {
		t.Fatalf("expected empty existing directory to succeed, got error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 archives in empty directory, got %v", files)
	}

	missingDir := filepath.Join(t.TempDir(), "nonexistent-backups")
	if _, err := tui.ScanBackupArchives(missingDir, restore.StrategyFullZIP); err == nil {
		t.Errorf("expected error when backup directory does not exist, got nil")
	}

	if _, err := tui.ScanBackupArchives("   ", restore.StrategyFullZIP); err == nil {
		t.Errorf("expected error when backup directory path is blank, got nil")
	}
}

func TestNormalizeAndValidateArchivePath(t *testing.T) {
	externalDir := t.TempDir()
	validZip := filepath.Join(externalDir, "valid.zip")
	if err := os.WriteFile(validZip, []byte("PK\x03\x04"), 0644); err != nil {
		t.Fatal(err)
	}
	upperZip := filepath.Join(externalDir, "ThirdParty Export (2026)+1.ZIP")
	if err := os.WriteFile(upperZip, []byte("PK\x03\x04"), 0644); err != nil {
		t.Fatal(err)
	}
	upperWpress := filepath.Join(externalDir, "Client Backup.WPRESS")
	if err := os.WriteFile(upperWpress, []byte("wpress-data"), 0644); err != nil {
		t.Fatal(err)
	}
	uncleanZip := filepath.Join(externalDir, "nested", "..", "valid.zip")

	cases := []struct {
		name       string
		input      string
		strategy   restore.Strategy
		wantPath   string
		wantErrSub string
	}{
		{
			name:     "plain absolute zip path",
			input:    validZip,
			strategy: restore.StrategyFullZIP,
			wantPath: validZip,
		},
		{
			name:     "quoted path with surrounding whitespace and uppercase ZIP",
			input:    "  \"" + upperZip + "\"\t\n",
			strategy: restore.StrategyFullZIP,
			wantPath: upperZip,
		},
		{
			name:     "quoted uppercase WPRESS path",
			input:    "\"" + upperWpress + "\"",
			strategy: restore.StrategyAI1WM,
			wantPath: upperWpress,
		},
		{
			name:     "unclean dot-segment path normalized cleanly",
			input:    "\"" + uncleanZip + "\"",
			strategy: restore.StrategyFullZIP,
			wantPath: validZip,
		},
		{
			name:       "empty input",
			input:      "   ",
			strategy:   restore.StrategyFullZIP,
			wantErrSub: "empty",
		},
		{
			name:       "empty quoted input",
			input:      `""`,
			strategy:   restore.StrategyFullZIP,
			wantErrSub: "empty",
		},
		{
			name:       "unmatched leading quote",
			input:      `"` + validZip,
			strategy:   restore.StrategyFullZIP,
			wantErrSub: "quote",
		},
		{
			name:       "unmatched trailing quote",
			input:      validZip + `"`,
			strategy:   restore.StrategyFullZIP,
			wantErrSub: "quote",
		},
		{
			name:       "lone double quote",
			input:      `"`,
			strategy:   restore.StrategyFullZIP,
			wantErrSub: "quote",
		},
		{
			name:       "triple double quotes rejected as unmatched",
			input:      `"""`,
			strategy:   restore.StrategyFullZIP,
			wantErrSub: "quote",
		},
		{
			name:       "double-wrapped outer quotes rejected",
			input:      `""` + validZip + `""`,
			strategy:   restore.StrategyFullZIP,
			wantErrSub: "absolute",
		},
		{
			name:       "relative filename rejected",
			input:      "valid.zip",
			strategy:   restore.StrategyFullZIP,
			wantErrSub: "absolute",
		},
		{
			name:       "relative dot path rejected",
			input:      filepath.Join(".", "valid.zip"),
			strategy:   restore.StrategyFullZIP,
			wantErrSub: "absolute",
		},
		{
			name:       "nonexistent file rejected",
			input:      filepath.Join(externalDir, "missing.zip"),
			strategy:   restore.StrategyFullZIP,
			wantErrSub: "does not exist",
		},
		{
			name:       "directory rejected as non-regular file",
			input:      externalDir,
			strategy:   restore.StrategyFullZIP,
			wantErrSub: "regular file",
		},
		{
			name:       "wrong extension for Full ZIP names .zip",
			input:      upperWpress,
			strategy:   restore.StrategyFullZIP,
			wantErrSub: ".zip",
		},
		{
			name:       "wrong extension for AI1WM names .wpress",
			input:      validZip,
			strategy:   restore.StrategyAI1WM,
			wantErrSub: ".wpress",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tui.NormalizeAndValidateArchivePath(tc.input, tc.strategy)
			if tc.wantErrSub != "" {
				if err == nil {
					t.Fatalf("expected error containing %q for input %q, got path %q", tc.wantErrSub, tc.input, got)
				}
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.wantErrSub)) {
					t.Errorf("expected error containing %q, got %q", tc.wantErrSub, err.Error())
				}
				if vErr := tui.ValidateArchivePath(tc.input, tc.strategy); vErr == nil {
					t.Errorf("expected ValidateArchivePath(%q) to also fail", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.input, err)
			}
			if got != tc.wantPath {
				t.Errorf("expected normalized path %q, got %q", tc.wantPath, got)
			}
			if vErr := tui.ValidateArchivePath(tc.input, tc.strategy); vErr != nil {
				t.Errorf("expected ValidateArchivePath(%q) to succeed, got %v", tc.input, vErr)
			}
		})
	}

	t.Run("symlink rejected as non-regular file", func(t *testing.T) {
		linkPath := filepath.Join(externalDir, "linked.zip")
		if err := os.Symlink(validZip, linkPath); err != nil {
			t.Skipf("symlink creation unavailable on host: %v", err)
		}
		if _, err := tui.NormalizeAndValidateArchivePath(linkPath, restore.StrategyFullZIP); err == nil {
			t.Fatalf("expected symlink %q to be rejected", linkPath)
		}
	})
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
