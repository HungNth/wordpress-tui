package tui

import (
	"fmt"

	"charm.land/huh/v2"
)

type SettingsAction string

const (
	ActionOpenVSCode   SettingsAction = "vscode"
	ActionOpenCache    SettingsAction = "cache"
	ActionSettingsBack SettingsAction = "back"
)

type SettingsOption struct {
	Label  string
	Action SettingsAction
}

func GetSettingsOptions() []SettingsOption {
	return []SettingsOption{
		{Label: "Open config.json in VS Code", Action: ActionOpenVSCode},
		{Label: "Open Cache Directory", Action: ActionOpenCache},
		{Label: "Back to Main Menu", Action: ActionSettingsBack},
	}
}

// PromptSettingsAction displays an interactive sub-menu for application settings.
func PromptSettingsAction() (SettingsAction, error) {
	items := GetSettingsOptions()
	options := make([]huh.Option[SettingsAction], len(items))
	for i, item := range items {
		options[i] = huh.NewOption(item.Label, item.Action)
	}

	var choice SettingsAction
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[SettingsAction]().
				Title("WPTUI / Settings").
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
	fmt.Printf("\n  %s %s\n\n", StyleSuccess.Render("[✓]"), msg)
}

// PrintSettingsError prints an error message in red.
func PrintSettingsError(msg string) {
	fmt.Printf("\n  %s %s\n\n", StyleError.Render("[✗]"), msg)
}
