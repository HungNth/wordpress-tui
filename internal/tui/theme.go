package tui

import (
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// AppTheme returns the standard theme for WPTUI with high-visibility cyan active border.
func AppTheme() *huh.Styles {
	theme := huh.ThemeBase(true)
	cyan := lipgloss.Color("#00FFFF")
	green := lipgloss.Color("#04B575")

	// Apply high-visibility bright cyan (#00FFFF) to focused input border and title
	theme.Focused.Base = theme.Focused.Base.
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderLeftForeground(cyan)
	theme.Focused.Title = theme.Focused.Title.Foreground(cyan).Bold(true)

	// Highlight active cursor option in bold bright cyan
	theme.Focused.SelectSelector = theme.Focused.SelectSelector.Foreground(cyan).Bold(true)
	theme.Focused.MultiSelectSelector = theme.Focused.MultiSelectSelector.Foreground(cyan).Bold(true)
	theme.Focused.Option = theme.Focused.Option.Foreground(cyan).Bold(true)

	// Standardize checked indicator prefix to [x] and unchecked to [ ]
	theme.Focused.SelectedPrefix = lipgloss.NewStyle().SetString("[x] ").Foreground(green).Bold(true)
	theme.Focused.UnselectedPrefix = lipgloss.NewStyle().SetString("[ ] ")
	theme.Blurred.SelectedPrefix = lipgloss.NewStyle().SetString("[x] ").Foreground(green).Bold(true)
	theme.Blurred.UnselectedPrefix = lipgloss.NewStyle().SetString("[ ] ")

	// Highlight checked items in bold vibrant green
	theme.Focused.SelectedOption = theme.Focused.SelectedOption.Foreground(green).Bold(true)
	theme.Blurred.SelectedOption = theme.Blurred.SelectedOption.Foreground(green)

	return theme
}
// CustomTheme returns the huh.Theme value for WPTUI forms.
func CustomTheme() huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		return AppTheme()
	})
}
