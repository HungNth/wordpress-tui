package tui_test

import (
	"context"
	"testing"

	"wptui/internal/config"
	"wptui/internal/packages"
	"wptui/internal/tui"
)

func TestPromptPackageSelections_APIEmptyBypasses(t *testing.T) {
	tempHome := t.TempDir()
	cfg := config.DefaultConfig(tempHome)
	cfg.PackagesAPIURL = "" // Disabled

	plugins, themes, err := tui.PromptPackageSelections(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("expected nil error when API is empty, got %v", err)
	}
	if len(plugins) != 0 || len(themes) != 0 {
		t.Errorf("expected empty plugins and themes when API is disabled, got %v, %v", plugins, themes)
	}
}

func TestCatalogFilteringLogic(t *testing.T) {
	catalog := []packages.CatalogItem{
		{Name: "Advanced Custom Fields PRO", Slug: "advanced-custom-fields-pro", Type: "plugin"},
		{Name: "Admin and Site Enhancements (ASE) Pro", Slug: "admin-site-enhancements-pro", Type: "plugin"},
		{Name: "Xstore", Slug: "xstore", Type: "theme"},
		{Name: "Generic Tool", Slug: "generic-tool", Type: "generic"},
	}

	plugins := packages.FilterCatalog(catalog, packages.PackageTypePlugin, "admin")
	if len(plugins) != 1 || plugins[0].Slug != "admin-site-enhancements-pro" {
		t.Errorf("expected admin-site-enhancements-pro, got %v", plugins)
	}

	themes := packages.FilterCatalog(catalog, packages.PackageTypeTheme, "")
	if len(themes) != 1 || themes[0].Slug != "xstore" {
		t.Errorf("expected 1 theme, got %v", themes)
	}
}

func TestInlineSearchOptionAndExtraction(t *testing.T) {
	// 1. When catalog is absent, no search option is appended
	optsWithoutCat := tui.BuildPackageOptions(nil, false)
	if len(optsWithoutCat) != 0 {
		t.Errorf("expected 0 options when catalog absent, got %d", len(optsWithoutCat))
	}

	// 2. When catalog is present, search option is appended
	optsWithCat := tui.BuildPackageOptions(nil, true)
	if len(optsWithCat) != 1 || optsWithCat[0].Value != tui.SearchOptionKey {
		t.Errorf("expected search option value %q, got %+v", tui.SearchOptionKey, optsWithCat)
	}

	// 3. ExtractSelectedPackages filters search trigger and reports wantsSearch
	choicesWithSearch := []string{"plugin-one", tui.SearchOptionKey, "plugin-two"}
	selected, wantsSearch := tui.ExtractSelectedPackages(choicesWithSearch)
	if !wantsSearch {
		t.Errorf("expected wantsSearch to be true")
	}
	if len(selected) != 2 || selected[0] != "plugin-one" || selected[1] != "plugin-two" {
		t.Errorf("unexpected selected packages: %v", selected)
	}

	// 4. ExtractSelectedPackages without search trigger
	choicesWithoutSearch := []string{"plugin-one"}
	selected, wantsSearch = tui.ExtractSelectedPackages(choicesWithoutSearch)
	if wantsSearch {
		t.Errorf("expected wantsSearch to be false")
	}
	if len(selected) != 1 || selected[0] != "plugin-one" {
		t.Errorf("unexpected selected packages: %v", selected)
	}
}
