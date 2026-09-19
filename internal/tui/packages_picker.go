package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"wptui/internal/config"
	"wptui/internal/packages"
)

const SearchOptionKey = "__search__"

// BuildPackageOptions builds the initial option list, placing inline search at the very top (index 0) when catalog is available.
func BuildPackageOptions(defaultOptions []huh.Option[string], hasCatalog bool) []huh.Option[string] {
	if !hasCatalog {
		opts := make([]huh.Option[string], len(defaultOptions))
		copy(opts, defaultOptions)
		return opts
	}

	opts := make([]huh.Option[string], 0, len(defaultOptions)+1)
	opts = append(opts, huh.NewOption("Search catalog...", SearchOptionKey))
	opts = append(opts, defaultOptions...)
	return opts
}

func BuildPluginOptions(items []config.PluginItem) []huh.Option[string] {
	opts := make([]huh.Option[string], 0, len(items))
	for _, it := range items {
		opts = append(opts, huh.NewOption(fmt.Sprintf("%s (%s)", it.Name, it.Slug), it.Slug))
	}
	return opts
}

func BuildThemeOptions(items []config.ThemeItem) []huh.Option[string] {
	opts := make([]huh.Option[string], 0, len(items))
	for _, it := range items {
		opts = append(opts, huh.NewOption(fmt.Sprintf("%s (%s)", it.Name, it.Slug), it.Slug))
	}
	return opts
}
// ExtractSelectedPackages separates normal package slugs from the inline search trigger.
func ExtractSelectedPackages(choices []string) (selected []string, wantsSearch bool) {
	for _, ch := range choices {
		if ch == SearchOptionKey {
			wantsSearch = true
		} else {
			selected = append(selected, ch)
		}
	}
	return selected, wantsSearch
}

func SelectPackagesFlow(
	ctx context.Context,
	itemType packages.PackageType,
	defaultOptions []huh.Option[string],
	catalog []packages.CatalogItem,
) ([]string, error) {
	var selected []string

	hasCatalog := len(catalog) > 0
	opts := BuildPackageOptions(defaultOptions, hasCatalog)

	var initialChoices []string
	if len(opts) > 0 {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title(fmt.Sprintf("Select %ss from defaults (or search)", itemType)).
					Description("Space to select, Enter to confirm").
					Options(opts...).
					Value(&initialChoices),
			),
		).WithTheme(CustomTheme())

		if err := form.Run(); err != nil {
			return nil, err
		}
	}

	normSelected, wantsSearch := ExtractSelectedPackages(initialChoices)
	selected = append(selected, normSelected...)

	// If user did not select search, finish immediately
	if !wantsSearch || !hasCatalog {
		return deduplicateStrings(selected), nil
	}

	// Launch unified live search model
	model := NewLiveSearchModel(itemType, catalog, selected)
	prog := tea.NewProgram(model)
	finalModel, err := prog.Run()
	if err != nil {
		return nil, err
	}

	liveM, ok := finalModel.(*LiveSearchModel)
	if !ok {
		return deduplicateStrings(selected), nil
	}
	if liveM.IsCancelled() {
		return nil, huh.ErrUserAborted
	}
	if liveM.IsAborted() {
		// User closed search via Esc: retain choices made before search
		return deduplicateStrings(selected), nil
	}
	return liveM.FinalSelected(), nil
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
	).WithTheme(CustomTheme())

	if err := form.Run(); err != nil {
		return nil, nil, err
	}

	var plugins []string
	if wantPlugins {
		var pluginOpts []huh.Option[string]
		for _, p := range cfg.Plugins {
			pluginOpts = append(pluginOpts, huh.NewOption(p.Name, p.Slug))
		}
		pList, err := SelectPackagesFlow(ctx, packages.PackageTypePlugin, pluginOpts, catalog)
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
		thList, err := SelectPackagesFlow(ctx, packages.PackageTypeTheme, themeOpts, catalog)
		if err != nil {
			return nil, nil, err
		}
		themes = thList
	}

	return plugins, themes, nil
}
