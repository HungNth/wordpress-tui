package tui

import (
	"errors"
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

// ResolveAndValidateSlug normalizes slug input: if empty, it derives the slug from websiteName;
// it then validates syntax and executes the availability checker.
func ResolveAndValidateSlug(websiteName, websiteSlug string, checker SlugAvailabilityChecker) (string, error) {
	slug := strings.TrimSpace(websiteSlug)
	if slug == "" {
		slug = create.Slugify(websiteName)
		if slug == "" {
			return "", errors.New("website slug cannot be generated from empty website name")
		}
	}

	if err := ValidateSlugWithChecker(slug, checker); err != nil {
		return "", err
	}
	return slug, nil
}

// BuildWebsiteInputsForm creates the unified Huh form collecting website parameters with optional tweaks.
func BuildWebsiteInputsForm(inputs *CreateInputs, cfg *config.Config, includeTweaks bool, checker ...SlugAvailabilityChecker) *huh.Form {
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

	secondGroupFields := []huh.Field{
		huh.NewInput().
			Title("Admin Username").
			Description("WordPress administrator username (default: " + cfg.DefaultAdminUsername + ")").
			Value(&inputs.AdminUsername),

		huh.NewInput().
			Title("Admin Password").
			Description("WordPress administrator password").
			EchoMode(huh.EchoModePassword).
			Value(&inputs.AdminPassword),

		huh.NewInput().
			Title("Admin Email").
			Description("WordPress administrator email (default: " + cfg.DefaultAdminEmail + ")").
			Value(&inputs.AdminEmail).
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return nil
				}
				return config.ValidateEmail(s)
			}),
	}

	if includeTweaks {
		secondGroupFields = append(secondGroupFields,
			huh.NewConfirm().
				Title("Apply WordPress Tweaks?").
				Description("Apply debug settings, custom permalinks, VN timezone, and locale").
				Value(&inputs.ApplyTweaks),
		)
	}

	return huh.NewForm(
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

			huh.NewInput().
				Title("Website Slug").
				Description("Folder, database, and .test domain (1-63 chars; leave blank to auto-generate from name)").
				Value(&inputs.WebsiteSlug).
				Validate(func(s string) error {
					resolved, err := ResolveAndValidateSlug(inputs.WebsiteName, s, check)
					if err != nil {
						return err
					}
					inputs.WebsiteSlug = resolved
					return nil
				}),
		),
		huh.NewGroup(secondGroupFields...),
	).WithTheme(CustomTheme())
}

// BuildCreateForm creates the unified Huh form collecting website parameters with tweaks.
func BuildCreateForm(inputs *CreateInputs, cfg *config.Config, checker ...SlugAvailabilityChecker) *huh.Form {
	return BuildWebsiteInputsForm(inputs, cfg, true, checker...)
}

// BuildRestoreForm creates the unified Huh form collecting website parameters without tweaks.
func BuildRestoreForm(inputs *CreateInputs, cfg *config.Config, checker ...SlugAvailabilityChecker) *huh.Form {
	return BuildWebsiteInputsForm(inputs, cfg, false, checker...)
}

// PromptCreateInputs prompts the user with the unified create form.
func PromptCreateInputs(cfg *config.Config, checker ...SlugAvailabilityChecker) (*CreateInputs, error) {
	inputs := &CreateInputs{}

	form := BuildCreateForm(inputs, cfg, checker...)
	if err := form.Run(); err != nil {
		return nil, err
	}

	// If slug was left blank, derive automatically from confirmed Website Name
	resolvedSlug, err := ResolveAndValidateSlug(inputs.WebsiteName, inputs.WebsiteSlug, nil)
	if err != nil {
		return nil, err
	}
	inputs.WebsiteSlug = resolvedSlug

	return inputs, nil
}
