package tui_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"wptui/internal/backup"
	"wptui/internal/tui"
)

func TestPrintBackupSummary(t *testing.T) {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	res := &backup.BackupResult{
		Strategy: backup.StrategyFull,
		FilePath: "/path/to/backups/full_my-site_2026-09-18_10-00-00.zip",
		FileSize: 10485760, // 10MB
		Duration: 2 * time.Second,
	}

	tui.PrintBackupSummary(res)

	_ = w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()

	if !strings.Contains(out, "Backup Complete: Full Source & Database") {
		t.Errorf("expected summary title, got: %q", out)
	}
	if !strings.Contains(out, "full_my-site_2026-09-18_10-00-00.zip") {
		t.Errorf("expected file path in output, got: %q", out)
	}
	if !strings.Contains(out, "10.00 MB") {
		t.Errorf("expected formatted file size '10.00 MB', got: %q", out)
	}
}
