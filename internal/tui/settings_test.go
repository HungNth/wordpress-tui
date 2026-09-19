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

func TestPromptSettingsAction_LabelsMatchDesignContract(t *testing.T) {
	options := tui.GetSettingsOptions()
	if len(options) == 0 {
		t.Fatal("expected non-empty settings options")
	}
	for _, opt := range options {
		// Must not have numeric prefixes like "1. " or "2. "
		if strings.HasPrefix(opt.Label, "1. ") || strings.HasPrefix(opt.Label, "2. ") {
			t.Errorf("expected plain label without numeric accelerator prefix, got %q", opt.Label)
		}
		// Must not have emoji or arrow symbols
		if strings.Contains(opt.Label, "←") {
			t.Errorf("expected plain label without arrow symbols, got %q", opt.Label)
		}
	}
}
