package tui

import (
	"context"
	"strings"

	"wptui/internal/config"
	"wptui/internal/packages"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

func SelectPackagesFlow(
	ctx context.Context,
	itemType packages.PackageType,
	catalog []packages.CatalogItem,
) ([]string, error) {
	if len(catalog) == 0 {
		return nil, nil
	}

	var filtered []packages.CatalogItem
	for _, it := range catalog {
		if it.Type == string(itemType) {
			filtered = append(filtered, it)
		}
	}
	if len(filtered) == 0 {
		return nil, nil
	}

	model := NewLiveSearchModel(itemType, catalog, nil)
	prog := tea.NewProgram(model)
	finalModel, err := prog.Run()
	if err != nil {
		return nil, err
	}

	liveM, ok := finalModel.(*LiveSearchModel)
	if !ok {
		return nil, nil
	}
	if liveM.IsCancelled() {
		return nil, huh.ErrUserAborted
	}
	if liveM.IsAborted() {
		return nil, nil
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
	if strings.TrimSpace(cfg.PackagesAPIURL) == "" || len(catalog) == 0 {
		return nil, nil, nil
	}

	hasPlugins := false
	hasThemes := false
	for _, it := range catalog {
		if it.Type == "plugin" {
			hasPlugins = true
		} else if it.Type == "theme" {
			hasThemes = true
		}
	}
	if !hasPlugins && !hasThemes {
		return nil, nil, nil
	}

	var plugins []string
	if hasPlugins {
		var wantPlugins bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Install plugins?").
					Description("Choose plugins from package catalog").
					Value(&wantPlugins),
			),
		).WithTheme(CustomTheme())

		if err := form.Run(); err != nil {
			return nil, nil, err
		}

		if wantPlugins {
			pList, err := SelectPackagesFlow(ctx, packages.PackageTypePlugin, catalog)
			if err != nil {
				return nil, nil, err
			}
			plugins = pList
		}
	}

	var themes []string
	if hasThemes {
		var wantThemes bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Install themes?").
					Description("Choose themes from package catalog").
					Value(&wantThemes),
			),
		).WithTheme(CustomTheme())

		if err := form.Run(); err != nil {
			return nil, nil, err
		}

		if wantThemes {
			thList, err := SelectPackagesFlow(ctx, packages.PackageTypeTheme, catalog)
			if err != nil {
				return nil, nil, err
			}
			themes = thList
		}
	}

	return plugins, themes, nil
}
