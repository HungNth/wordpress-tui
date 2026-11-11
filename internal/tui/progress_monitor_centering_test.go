package tui_test

import (
	"regexp"
	"strings"
	"testing"
	"wptui/internal/tui"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func TestProgressMonitor_LogsBoxIsCentered(t *testing.T) {
	steps := []tui.ProgressStep{
		{ID: "step1", Title: "Step One", Status: tui.StepStatusSuccess},
	}
	monitor := tui.NewProgressMonitorModel("Operation", steps)

	widths := []int{70, 80, 100, 115}
	for _, w := range widths {
		monitor.SetSize(w, 24)
		view := monitor.Render(w, 24)

		lines := strings.Split(view, "\n")
		var topBorderLine, logContentLine string
		for i, l := range lines {
			if strings.Contains(l, "Logs") {
				if i > 0 {
					topBorderLine = lines[i-1]
				}
				logContentLine = l
				break
			}
		}

		if topBorderLine == "" || logContentLine == "" {
			t.Fatalf("width %d: expected to find Logs box in rendered view", w)
		}

		// Both the top border and the content line should be padded on the left by at least 2 spaces
		topPad := len(topBorderLine) - len(strings.TrimLeft(topBorderLine, " "))
		contentPad := len(logContentLine) - len(strings.TrimLeft(logContentLine, " "))

		if topPad < 2 {
			t.Errorf("width %d: expected top border left padding >= 2, got %d (line: %q)", w, topPad, topBorderLine)
		}
		if contentPad < 2 {
			t.Errorf("width %d: expected content line left padding >= 2, got %d (line: %q)", w, contentPad, logContentLine)
		}
	}
}

func TestProgressMonitor_LogTextHasInnerPadding(t *testing.T) {
	steps := []tui.ProgressStep{
		{ID: "step1", Title: "Step One", Status: tui.StepStatusRunning},
	}
	monitor := tui.NewProgressMonitorModel("Operation", steps)
	monitor.SetSize(80, 24)
	monitor.Update(tui.LogLineMsg("Applying tweak: [config_set] WP_DEBUG=true"))

	view := monitor.Render(80, 24)
	lines := strings.Split(view, "\n")

	var tweakLine string
	for _, l := range lines {
		if strings.Contains(l, "Applying tweak") {
			tweakLine = l
			break
		}
	}

	if tweakLine == "" {
		t.Fatalf("expected to find log text line in rendered view")
	}

	plain := stripANSI(tweakLine)
	// Must contain border followed by space and text: e.g. "│ Applying tweak"
	// and NOT touch the border directly: "│Applying tweak"
	if !strings.Contains(plain, "│ Applying tweak") && !strings.Contains(plain, "| Applying tweak") {
		t.Errorf("expected inner padding space between border and log text, got plain: %q", plain)
	}
}
