package tui_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"wptui/internal/tui"
)

func TestPrintSettingsSuccess(t *testing.T) {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tui.PrintSettingsSuccess("Config reloaded successfully")

	_ = w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()

	if !strings.Contains(out, "[✓]") || !strings.Contains(out, "Config reloaded successfully") {
		t.Errorf("expected success output, got %q", out)
	}
}

func TestPrintSettingsError(t *testing.T) {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tui.PrintSettingsError("Failed to open VS Code")

	_ = w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()

	if !strings.Contains(out, "[✗]") || !strings.Contains(out, "Failed to open VS Code") {
		t.Errorf("expected error output, got %q", out)
	}
}
