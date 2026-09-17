package tui_test

import (
	"image/color"
	"strings"
	"testing"

	"charm.land/huh/v2"
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

func TestAppTheme_OptionAndSelectionColors(t *testing.T) {
	theme := tui.AppTheme()
	cyan := lipgloss.Color("#00FFFF")
	green := lipgloss.Color("#04B575")

	// 1. Option foreground matches cyan
	optFg := theme.Focused.Option.GetForeground()
	if optFg == nil {
		t.Fatal("expected Focused.Option foreground to be set")
	}
	r1, g1, b1, a1 := optFg.RGBA()
	r2, g2, b2, a2 := cyan.RGBA()
	if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
		t.Errorf("expected Focused.Option foreground %v, got %v", color.Color(cyan), optFg)
	}
	if !theme.Focused.Option.GetBold() {
		t.Errorf("expected Focused.Option to be bold")
	}

	// 2. SelectedPrefix matches green
	selPrefixFg := theme.Focused.SelectedPrefix.GetForeground()
	if selPrefixFg == nil {
		t.Fatal("expected Focused.SelectedPrefix foreground to be set")
	}
	r1, g1, b1, a1 = selPrefixFg.RGBA()
	r2, g2, b2, a2 = green.RGBA()
	if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
		t.Errorf("expected Focused.SelectedPrefix foreground %v, got %v", color.Color(green), selPrefixFg)
	}
	if !theme.Focused.SelectedPrefix.GetBold() {
		t.Errorf("expected Focused.SelectedPrefix to be bold")
	}
	if theme.Focused.SelectedPrefix.String() != "[x] " && !strings.Contains(theme.Focused.SelectedPrefix.String(), "[x]") {
		t.Errorf("expected Focused.SelectedPrefix to contain [x], got %q", theme.Focused.SelectedPrefix.String())
	}
	// 3. SelectedOption matches green
	selOptFg := theme.Focused.SelectedOption.GetForeground()
	if selOptFg == nil {
		t.Fatal("expected Focused.SelectedOption foreground to be set")
	}
	r1, g1, b1, a1 = selOptFg.RGBA()
	r2, g2, b2, a2 = green.RGBA()
	if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
		t.Errorf("expected Focused.SelectedOption foreground %v, got %v", color.Color(green), selOptFg)
	}
	if !theme.Focused.SelectedOption.GetBold() {
		t.Errorf("expected Focused.SelectedOption to be bold")
	}

	// 4. SelectSelector matches cyan and bold
	selSelectorFg := theme.Focused.SelectSelector.GetForeground()
	if selSelectorFg == nil {
		t.Fatal("expected Focused.SelectSelector foreground to be set")
	}
	r1, g1, b1, a1 = selSelectorFg.RGBA()
	r2, g2, b2, a2 = cyan.RGBA()
	if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
		t.Errorf("expected Focused.SelectSelector foreground %v, got %v", color.Color(cyan), selSelectorFg)
	}
	if !theme.Focused.SelectSelector.GetBold() {
		t.Errorf("expected Focused.SelectSelector to be bold")
	}

	// 5. MultiSelectSelector matches cyan and bold
	multiSelectorFg := theme.Focused.MultiSelectSelector.GetForeground()
	if multiSelectorFg == nil {
		t.Fatal("expected Focused.MultiSelectSelector foreground to be set")
	}
	r1, g1, b1, a1 = multiSelectorFg.RGBA()
	r2, g2, b2, a2 = cyan.RGBA()
	if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
		t.Errorf("expected Focused.MultiSelectSelector foreground %v, got %v", color.Color(cyan), multiSelectorFg)
	}
	if !theme.Focused.MultiSelectSelector.GetBold() {
		t.Errorf("expected Focused.MultiSelectSelector to be bold")
	}
}
func TestRenderMultiSelectView(t *testing.T) {
	val := []string{"1"}
	m := huh.NewMultiSelect[string]().
		Options(huh.NewOption("Option 1", "1"), huh.NewOption("Option 2", "2")).
		Value(&val).
		WithTheme(tui.CustomTheme())
	m.Focus()
	view := m.View()

	// 1. Assert standardized [x] indicator is rendered for selected item
	if !strings.Contains(view, "[x]") {
		t.Errorf("expected view to render [x] checkmark for selected option, got:\n%s", view)
	}

	// 2. Assert [ ] is rendered for unselected item
	if !strings.Contains(view, "[ ]") {
		t.Errorf("expected view to render [ ] checkmark for unselected option, got:\n%s", view)
	}

	// 3. Assert active cursor indicator > is rendered
	if !strings.Contains(view, ">") {
		t.Errorf("expected view to render active cursor >, got:\n%s", view)
	}
}
