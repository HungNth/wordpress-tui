package tui_test

import (
	"context"
	"testing"

	"charm.land/huh/v2"
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
	// 1. When catalog is absent, no search option is appended and defaults remain intact
	defaultOpts := []huh.Option[string]{
		huh.NewOption("Plugin A", "plugin-a"),
		huh.NewOption("Plugin B", "plugin-b"),
	}
	optsWithoutCat := tui.BuildPackageOptions(defaultOpts, false)
	if len(optsWithoutCat) != 2 || optsWithoutCat[0].Value != "plugin-a" || optsWithoutCat[1].Value != "plugin-b" {
		t.Errorf("expected defaults preserved without search option, got %+v", optsWithoutCat)
	}

	// 2. When catalog is present, search option is prepended at index 0 and defaults follow in order
	optsWithCat := tui.BuildPackageOptions(defaultOpts, true)
	if len(optsWithCat) != 3 {
		t.Fatalf("expected 3 options, got %d", len(optsWithCat))
	}
	if optsWithCat[0].Value != tui.SearchOptionKey {
		t.Errorf("expected head option value %q, got %q", tui.SearchOptionKey, optsWithCat[0].Value)
	}
	if optsWithCat[1].Value != "plugin-a" || optsWithCat[2].Value != "plugin-b" {
		t.Errorf("expected defaults preserved in order following search, got %+v", optsWithCat)
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

func TestUpdateAccumulatedSelection(t *testing.T) {
	// Start with site defaults already selected
	accumulated := map[string]bool{
		"default-plugin": true,
	}

	// Query 1: User searched "acf" matching ["acf-pro", "acf-extended"]
	// User selects "acf-pro"
	view1Matches := []string{"acf-pro", "acf-extended"}
	view1Chosen := []string{"acf-pro"}
	tui.UpdateAccumulatedSelection(accumulated, view1Matches, view1Chosen)

	if !accumulated["default-plugin"] {
		t.Errorf("expected default-plugin to be retained")
	}
	if !accumulated["acf-pro"] {
		t.Errorf("expected acf-pro to be accumulated")
	}
	if accumulated["acf-extended"] {
		t.Errorf("expected acf-extended to not be selected")
	}

	// Query 2: User clears and searches "smtp" matching ["wp-mail-smtp", "easy-smtp"]
	// User selects "wp-mail-smtp"
	view2Matches := []string{"wp-mail-smtp", "easy-smtp"}
	view2Chosen := []string{"wp-mail-smtp"}
	tui.UpdateAccumulatedSelection(accumulated, view2Matches, view2Chosen)

	// Both previous selections MUST be retained!
	if !accumulated["default-plugin"] {
		t.Errorf("expected default-plugin to still be retained across query 2")
	}
	if !accumulated["acf-pro"] {
		t.Errorf("expected acf-pro from query 1 to be preserved after query 2")
	}
	if !accumulated["wp-mail-smtp"] {
		t.Errorf("expected wp-mail-smtp to be accumulated in query 2")
	}

	// Query 3: User searches "acf" again and UNCHECKS "acf-pro"
	view3Matches := []string{"acf-pro", "acf-extended"}
	view3Chosen := []string{} // unchecked in this view
	tui.UpdateAccumulatedSelection(accumulated, view3Matches, view3Chosen)

	if accumulated["acf-pro"] {
		t.Errorf("expected acf-pro to be unselected when unchecked in matching view")
	}
	if !accumulated["default-plugin"] || !accumulated["wp-mail-smtp"] {
		t.Errorf("expected unrelated selections to remain intact")
	}
}
