package tui

import (
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// Semantic palette colors per DESIGN.md
var (
	ColorCyan   = lipgloss.Color("#00FFFF")
	ColorGreen  = lipgloss.Color("#04B575")
	ColorYellow = lipgloss.Color("#FFA500")
	ColorRed    = lipgloss.Color("#FF4444")
	ColorMuted  = lipgloss.Color("#888888")

	StyleFocus     = lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	StyleSuccess   = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	StyleWarning   = lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
	StyleError     = lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
	StyleMuted     = lipgloss.NewStyle().Foreground(ColorMuted)
	StyleHighlight = lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
)

// AppTheme returns the standard theme for WPTUI with high-visibility cyan active border.
func AppTheme() *huh.Styles {
	theme := huh.ThemeBase(true)

	// Apply high-visibility bright cyan (#00FFFF) to focused input border and title
	theme.Focused.Base = theme.Focused.Base.
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderLeftForeground(ColorCyan)
	theme.Focused.Title = theme.Focused.Title.Foreground(ColorCyan).Bold(true)

	// Highlight active cursor option in bold bright cyan
	theme.Focused.SelectSelector = theme.Focused.SelectSelector.Foreground(ColorCyan).Bold(true)
	theme.Focused.MultiSelectSelector = theme.Focused.MultiSelectSelector.Foreground(ColorCyan).Bold(true)
	theme.Focused.Option = theme.Focused.Option.Foreground(ColorCyan).Bold(true)

	// Standardize checked indicator prefix to [x] and unchecked to [ ]
	theme.Focused.SelectedPrefix = lipgloss.NewStyle().SetString("[x] ").Foreground(ColorGreen).Bold(true)
	theme.Focused.UnselectedPrefix = lipgloss.NewStyle().SetString("[ ] ")
	theme.Blurred.SelectedPrefix = lipgloss.NewStyle().SetString("[x] ").Foreground(ColorGreen).Bold(true)
	theme.Blurred.UnselectedPrefix = lipgloss.NewStyle().SetString("[ ] ")

	// Highlight checked items in bold vibrant green
	theme.Focused.SelectedOption = theme.Focused.SelectedOption.Foreground(ColorGreen).Bold(true)
	theme.Blurred.SelectedOption = theme.Blurred.SelectedOption.Foreground(ColorGreen)

	return theme
}
// CustomTheme returns the huh.Theme value for WPTUI forms.
func CustomTheme() huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		return AppTheme()
	})
}
