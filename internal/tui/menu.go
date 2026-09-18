package tui

import (
	"errors"
	"fmt"

	"charm.land/huh/v2"
)

type MenuItem struct {
	Key         string
	Title       string
	Description string
	Disabled    bool
}

func GetMenuItems() []MenuItem {
	return []MenuItem{
		{Key: "create", Title: "Create", Description: "Provision a new local WordPress website", Disabled: false},
		{Key: "config", Title: "Config", Description: "Configure an existing website (tweaks, admin, packages)", Disabled: false},
		{Key: "delete", Title: "Delete", Description: "De-provision and delete local WordPress websites", Disabled: false},
		{Key: "backup", Title: "Backup", Description: "Backup an existing WordPress website (Full or AI1WM)", Disabled: false},
		{Key: "restore", Title: "Restore", Description: "Restore website from backup (Full ZIP or AI1WM)", Disabled: false},
		{Key: "settings", Title: "Settings", Description: "Application settings (edit config, open cache)", Disabled: false},
		{Key: "exit", Title: "Exit", Description: "Exit WPTUI", Disabled: false},
	}
}

// BuildMainMenuForm builds a Huh Form for the main menu.
func BuildMainMenuForm(choice *string) *huh.Form {
	items := GetMenuItems()
	opts := make([]huh.Option[string], 0)

	for _, item := range items {
		if !item.Disabled {
			opts = append(opts, huh.NewOption(fmt.Sprintf("%s — %s", item.Title, item.Description), item.Key))
		}
	}

	description := "Select an available action below."
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("WPTUI — WordPress Local Manager").
				Description(description).
				Options(opts...).
				Value(choice),
		),
	).WithTheme(CustomTheme())
}

// RunMainMenu displays the interactive main menu and returns the selected action key.
func RunMainMenu() (string, error) {
	var choice string
	form := BuildMainMenuForm(&choice)
	err := form.Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "exit", nil
		}
		return "", err
	}
	return choice, nil
}
