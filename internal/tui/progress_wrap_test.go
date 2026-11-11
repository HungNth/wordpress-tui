package tui_test

import (
	"strings"
	"testing"
	"wptui/internal/config"
	"wptui/internal/tui"

	tea "charm.land/bubbletea/v2"
)

func extractContentPaneLine(plainLine string) string {
	parts := strings.Split(plainLine, "│")
	if len(parts) >= 4 {
		return parts[3]
	}
	parts = strings.Split(plainLine, "|")
	if len(parts) >= 4 {
		return parts[3]
	}
	return plainLine
}

func TestProgressMonitor_StepperHangingIndentOnWrap(t *testing.T) {
	cfg := config.DefaultConfig("/tmp")
	cfg.AppName = "flock"
	cfg.AppCommand = "flock"
	app := tui.NewAppModel(cfg)
	// Sized so that contentWidth is around 68
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	steps := []tui.ProgressStep{
		{
			ID:     "tweak",
			Title:  "Applying tweak 8 of 29: Applying tweak: [rewrite_structure] =/%category%/%postname%/",
			Status: tui.StepStatusRunning,
		},
	}
	app.StartProgress("Operation", steps)

	view := app.ViewString()
	lines := strings.Split(view, "\n")

	var wrappedLine string
	for _, l := range lines {
		if strings.Contains(l, "%postname%") {
			wrappedLine = l
			break
		}
	}

	if wrappedLine == "" {
		t.Fatalf("expected to find wrapped line containing %%postname%% in view:\n%s", view)
	}

	plain := stripANSI(wrappedLine)
	inner := extractContentPaneLine(plain)

	trimmed := strings.TrimLeft(inner, " ")
	indent := len(inner) - len(trimmed)

	// In the user's bug, "=/%category%..." started immediately at column 0 after border (indent = 0)
	// It must have a hanging indent matching "  ⠋ " (at least 4 spaces)
	if indent < 4 {
		t.Errorf("expected wrapped stepper line to have hanging indent >= 4 spaces, got %d (inner: %q, full line: %q)", indent, inner, wrappedLine)
	}
}

func TestProgressMonitor_SummaryHangingIndentOnWrap(t *testing.T) {
	cfg := config.DefaultConfig("/tmp")
	cfg.AppName = "flock"
	cfg.AppCommand = "flock"
	app := tui.NewAppModel(cfg)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	steps := []tui.ProgressStep{
		{ID: "install", Title: "Install", Status: tui.StepStatusSuccess},
	}
	app.StartProgress("Operation", steps)
	app.Update(tui.OperationCompleteMsg{
		Title:   "Website Provisioned",
		Success: true,
		Summary: "Plugins:      admin-site-enhancements-pro, advanced-custom-fields-pro, seo-by-rank-math-pro, all-in-one-wp-migration-unlimited",
	})

	view := app.ViewString()
	lines := strings.Split(view, "\n")

	var wrappedLine string
	for _, l := range lines {
		if strings.Contains(l, "all-in-one") {
			wrappedLine = l
			break
		}
	}

	if wrappedLine == "" {
		t.Fatalf("expected to find wrapped summary line containing 'all-in-one' in view:\n%s", view)
	}

	plain := stripANSI(wrappedLine)
	inner := extractContentPaneLine(plain)

	trimmed := strings.TrimLeft(inner, " ")
	indent := len(inner) - len(trimmed)

	// The key column is "    Plugins:      " which is 4 + 14 = 18 spaces.
	// Wrapped lines must have hanging indent >= 18 spaces to align with the first value line.
	if indent < 18 {
		t.Errorf("expected wrapped summary line to have hanging indent >= 18 spaces, got %d (inner: %q, full line: %q)", indent, inner, wrappedLine)
	}
}
