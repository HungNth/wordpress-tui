package tui_test

import (
	"fmt"
	"strings"
	"testing"
	"wptui/internal/config"
	"wptui/internal/tui"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestAppModel_EntireInterfaceIsCentered(t *testing.T) {
	cfg := config.DefaultConfig("/tmp")
	cfg.AppName = "flock"
	cfg.AppCommand = "flock"

	widths := []int{80, 100, 115, 120, 140, 164}

	for _, w := range widths {
		t.Run(fmt.Sprintf("Width_%d", w), func(t *testing.T) {
			app := tui.NewAppModel(cfg)
			app.Update(tea.WindowSizeMsg{Width: w, Height: 30})

			view := app.ViewString()
			lines := strings.Split(view, "\n")

			var navLine string
			for _, l := range lines {
				if strings.Contains(l, "Navigation") {
					navLine = l
					break
				}
			}

			if navLine == "" {
				t.Fatalf("width %d: Navigation box not found in rendered view", w)
			}

			// Extract plain text to measure spacing
			leftPad := len(navLine) - len(strings.TrimLeft(navLine, " "))
			totalLineWidth := lipgloss.Width(navLine)
			rightPad := w - totalLineWidth

			// Left padding must equal right padding (within 1 space for odd difference)
			diff := leftPad - rightPad
			if diff < 0 {
				diff = -diff
			}
			if diff > 1 {
				t.Errorf("width %d: interface is not centered! LeftPad=%d, RightPad=%d (diff=%d > 1)",
					w, leftPad, rightPad, diff)
			}

			// When terminal is wider than 120 columns (e.g. 140, 164), leftPad must be substantial (> 5)
			if w >= 140 && leftPad < 5 {
				t.Errorf("width %d: expected centered layout with leftPad >= 5 on wide terminal, got %d", w, leftPad)
			}
		})
	}
}
