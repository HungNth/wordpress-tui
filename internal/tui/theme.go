package tui

import (
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// AppTheme returns the standard theme for WPTUI with high-visibility cyan active border.
func AppTheme() *huh.Styles {
	theme := huh.ThemeBase(true)
	// Apply high-visibility bright cyan (#00FFFF) to focused input border
	cyan := lipgloss.Color("#00FFFF")
	theme.Focused.Base = theme.Focused.Base.
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderLeftForeground(cyan)
	theme.Focused.Title = theme.Focused.Title.Foreground(cyan).Bold(true)
	return theme
}

// CustomTheme returns the huh.Theme value for WPTUI forms.
func CustomTheme() huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		return AppTheme()
	})
}
