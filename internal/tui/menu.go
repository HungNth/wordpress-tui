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
		{Key: "config", Title: "Config", Description: "Configuration manager (Coming soon)", Disabled: true},
		{Key: "delete", Title: "Delete", Description: "Delete website and database (Coming soon)", Disabled: true},
		{Key: "backup", Title: "Backup", Description: "Backup website (Coming soon)", Disabled: true},
		{Key: "restore", Title: "Restore", Description: "Restore website (Coming soon)", Disabled: true},
		{Key: "settings", Title: "Settings", Description: "Application settings (Coming soon)", Disabled: true},
		{Key: "exit", Title: "Exit", Description: "Exit WPTUI", Disabled: false},
	}
}

// BuildMainMenuForm builds a Huh Form for the main menu.
func BuildMainMenuForm(choice *string) *huh.Form {
	items := GetMenuItems()
	opts := make([]huh.Option[string], 0, len(items))

	for _, item := range items {
		label := item.Title
		if item.Disabled {
			label = fmt.Sprintf("%-10s [Coming soon - %s]", item.Title, item.Description)
		} else {
			label = fmt.Sprintf("%-10s [%s]", item.Title, item.Description)
		}
		opts = append(opts, huh.NewOption(label, item.Key))
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("WPTUI — WordPress Local Manager").
				Description("Choose an operation to perform").
				Options(opts...).
				Validate(func(v string) error {
					for _, item := range items {
						if item.Key == v && item.Disabled {
							return fmt.Errorf("%s is not available yet (Coming soon)", item.Title)
						}
					}
					return nil
				}).
				Value(choice),
		),
	)
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
