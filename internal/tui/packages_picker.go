package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/huh/v2"
	"wptui/internal/config"
	"wptui/internal/packages"
)

// SelectPackagesFlow runs the interactive prompt and search loop for selecting plugins or themes.
func SelectPackagesFlow(
	ctx context.Context,
	itemType string, // "plugin" or "theme"
	defaultOptions []huh.Option[string],
	catalog []packages.CatalogItem,
) ([]string, error) {
	var selected []string

	// Initial default selection
	if len(defaultOptions) > 0 {
		var initialChoices []string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title(fmt.Sprintf("Select %ss from defaults (or continue to search)", itemType)).
					Description("Space to select, Enter to confirm").
					Options(defaultOptions...).
					Value(&initialChoices),
			),
		)

		if err := form.Run(); err != nil {
			return nil, err
		}
		selected = append(selected, initialChoices...)
	}

	// Search loop
	if len(catalog) == 0 {
		return deduplicateStrings(selected), nil
	}

	for {
		var wantSearch bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Search %s catalog?", itemType)).
					Description(fmt.Sprintf("Currently selected: %d %ss", len(selected), itemType)).
					Value(&wantSearch),
			),
		)

		if err := confirmForm.Run(); err != nil {
			return nil, err
		}

		if !wantSearch {
			break
		}

		var query string
		queryForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fmt.Sprintf("Search %s name or slug", itemType)).
					Value(&query),
			),
		)

		if err := queryForm.Run(); err != nil {
			return nil, err
		}

		matches := packages.FilterCatalog(catalog, itemType, query)
		if len(matches) == 0 {
			fmt.Printf("No matching %ss found for %q.\n", itemType, query)
			continue
		}

		opts := make([]huh.Option[string], 0, len(matches))
		for _, m := range matches {
			opts = append(opts, huh.NewOption(fmt.Sprintf("%s (%s v%s)", m.Name, m.Slug, m.Version), m.Slug))
		}

		var searchChoices []string
		resultsForm := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title(fmt.Sprintf("Matching %ss for %q", itemType, query)).
					Options(opts...).
					Value(&searchChoices),
			),
		)

		if err := resultsForm.Run(); err != nil {
			return nil, err
		}

		selected = append(selected, searchChoices...)
	}

	return deduplicateStrings(selected), nil
}

func deduplicateStrings(in []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// PromptPackageSelections collects plugin and theme choices for website creation using a pre-fetched catalog.
func PromptPackageSelections(ctx context.Context, cfg *config.Config, catalog []packages.CatalogItem) ([]string, []string, error) {
	if strings.TrimSpace(cfg.PackagesAPIURL) == "" {
		return nil, nil, nil
	}
	var wantPlugins bool
	var wantThemes bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Install plugins?").
				Description("Choose plugins from defaults or search package catalog").
				Value(&wantPlugins),

			huh.NewConfirm().
				Title("Install additional themes?").
				Description(fmt.Sprintf("Default theme is %q. Install extra themes?", cfg.DefaultThemeSlug)).
				Value(&wantThemes),
		),
	)

	if err := form.Run(); err != nil {
		return nil, nil, err
	}

	var plugins []string
	if wantPlugins {
		var pluginOpts []huh.Option[string]
		for _, p := range cfg.Plugins {
			pluginOpts = append(pluginOpts, huh.NewOption(p.Name, p.Slug))
		}
		pList, err := SelectPackagesFlow(ctx, "plugin", pluginOpts, catalog)
		if err != nil {
			return nil, nil, err
		}
		plugins = pList
	}

	var themes []string
	if wantThemes {
		var themeOpts []huh.Option[string]
		for _, th := range cfg.Themes {
			if th.Slug != cfg.DefaultThemeSlug {
				themeOpts = append(themeOpts, huh.NewOption(th.Name, th.Slug))
			}
		}
		thList, err := SelectPackagesFlow(ctx, "theme", themeOpts, catalog)
		if err != nil {
			return nil, nil, err
		}
		themes = thList
	}

	return plugins, themes, nil
}
