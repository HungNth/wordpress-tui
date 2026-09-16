package tui

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/huh/v2"
	"wptui/internal/config"
	"wptui/internal/create"
)

type CreateInputs struct {
	WebsiteName   string
	WebsiteSlug   string
	AdminUsername string
	AdminPassword string
	AdminEmail    string
	ApplyTweaks   bool
}

// SlugAvailabilityChecker checks whether a website slug collides with an existing directory or database.
type SlugAvailabilityChecker func(slug string) error

// ValidateSlugWithChecker validates slug syntax first; only syntactically valid slugs invoke the availability checker.
func ValidateSlugWithChecker(slug string, checker SlugAvailabilityChecker) error {
	if err := create.ValidateSlug(slug); err != nil {
		return err
	}
	if checker != nil {
		return checker(slug)
	}
	return nil
}

// BuildCreateForm creates the Huh form for collecting website creation parameters.
func BuildCreateForm(inputs *CreateInputs, cfg *config.Config, checker ...SlugAvailabilityChecker) *huh.Form {
	if inputs.AdminUsername == "" {
		inputs.AdminUsername = cfg.DefaultAdminUsername
	}
	if inputs.AdminPassword == "" {
		inputs.AdminPassword = cfg.DefaultAdminPassword
	}
	if inputs.AdminEmail == "" {
		inputs.AdminEmail = cfg.DefaultAdminEmail
	}

	var check SlugAvailabilityChecker
	if len(checker) > 0 && checker[0] != nil {
		check = checker[0]
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("Website Slug for %q", inputs.WebsiteName)).
				Description("Identifier for folder, database, and .test domain (1-63 chars)").
				Value(&inputs.WebsiteSlug).
				Validate(func(s string) error {
					return ValidateSlugWithChecker(s, check)
				}),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Admin Username").
				Description(fmt.Sprintf("WordPress administrator username (default: %s)", cfg.DefaultAdminUsername)).
				Value(&inputs.AdminUsername),

			huh.NewInput().
				Title("Admin Password").
				Description("WordPress administrator password").
				EchoMode(huh.EchoModePassword).
				Value(&inputs.AdminPassword),

			huh.NewInput().
				Title("Admin Email").
				Description(fmt.Sprintf("WordPress administrator email (default: %s)", cfg.DefaultAdminEmail)).
				Value(&inputs.AdminEmail).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return nil
					}
					return config.ValidateEmail(s)
				}),

			huh.NewConfirm().
				Title("Apply WordPress Tweaks?").
				Description("Apply debug settings, custom permalinks, VN timezone, and locale").
				Value(&inputs.ApplyTweaks),
		),
	)
}

// PromptCreateInputs prompts the user for create options, pre-filling slug from name.
func PromptCreateInputs(cfg *config.Config, checker ...SlugAvailabilityChecker) (*CreateInputs, error) {
	inputs := &CreateInputs{}

	// First ask for Website Name
	nameForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Website Name").
				Description("Human-readable title for your WordPress website").
				Value(&inputs.WebsiteName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("website name cannot be empty")
					}
					return nil
				}),
		),
	)

	if err := nameForm.Run(); err != nil {
		return nil, err
	}

	// Suggest slug from name
	inputs.WebsiteSlug = create.Slugify(inputs.WebsiteName)

	mainForm := BuildCreateForm(inputs, cfg, checker...)
	if err := mainForm.Run(); err != nil {
		return nil, err
	}

	return inputs, nil
}
