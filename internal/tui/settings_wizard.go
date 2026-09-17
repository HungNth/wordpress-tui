package tui

import (
	"fmt"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

type SettingsAction string

const (
	ActionOpenVSCode   SettingsAction = "vscode"
	ActionOpenCache    SettingsAction = "cache"
	ActionSettingsBack SettingsAction = "back"
)

// PromptSettingsAction displays an interactive sub-menu for application settings.
func PromptSettingsAction() (SettingsAction, error) {
	options := []huh.Option[SettingsAction]{
		huh.NewOption("1. Open config.json in VS Code", ActionOpenVSCode),
		huh.NewOption("2. Open Cache Directory", ActionOpenCache),
		huh.NewOption("← Back to Main Menu", ActionSettingsBack),
	}

	var choice SettingsAction
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[SettingsAction]().
				Title("Application Settings").
				Description("Manage configuration and inspect local cache assets").
				Options(options...).
				Value(&choice),
		),
	).WithTheme(CustomTheme())

	if err := form.Run(); err != nil {
		return "", err
	}

	return choice, nil
}

// PrintSettingsSuccess prints a success message in green.
func PrintSettingsSuccess(msg string) {
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	fmt.Printf("\n  %s %s\n\n", green.Render("[✓]"), msg)
}

// PrintSettingsError prints an error message in red.
func PrintSettingsError(msg string) {
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF3333")).Bold(true)
	fmt.Printf("\n  %s %s\n\n", red.Render("[✗]"), msg)
}
