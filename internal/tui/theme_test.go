package tui_test

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
	"wptui/internal/tui"
)

func TestAppTheme_FocusedBorderCyan(t *testing.T) {
	theme := tui.AppTheme()
	if theme == nil {
		t.Fatal("expected non-nil theme from AppTheme()")
	}

	fg := theme.Focused.Base.GetBorderLeftForeground()
	if fg == nil {
		t.Fatal("expected border left foreground to be configured on Focused.Base")
	}

	expectedColor := lipgloss.Color("#00FFFF")
	r1, g1, b1, a1 := fg.RGBA()
	r2, g2, b2, a2 := expectedColor.RGBA()
	if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
		t.Errorf("expected border left foreground color %v, got %v", color.Color(expectedColor), fg)
	}

	if !theme.Focused.Base.GetBorderLeft() {
		t.Errorf("expected Focused.Base to have left border enabled")
	}
}
