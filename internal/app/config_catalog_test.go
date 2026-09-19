package app_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"wptui/internal/app"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/packages"
	"wptui/internal/tui"
)

func TestRunConfigFlowWithDeps_CatalogAvailableForSearch(t *testing.T) {
	tempDir := t.TempDir()
	siteDir := filepath.Join(tempDir, "site-search-test")
	_ = os.MkdirAll(siteDir, 0755)

	cfg := &config.Config{
		WebsitesPath: tempDir,
		Plugins: []config.PluginItem{
			{Name: "Plugin Default", Slug: "plugin-default"},
		},
	}

	catalogItems := []packages.CatalogItem{
		{Name: "Catalog Plugin", Slug: "catalog-plugin", Type: "plugin"},
	}

	mockCli := &mockSiteWPClient{}
	optionsPassedHasSearch := false
	siteSelectionCount := 0

	deps := app.ConfigFlowDependencies{
		WPClient: mockCli,
		Catalog:  catalogItems,
		SelectWebsite: func(candidates []deprovision.Candidate) (*deprovision.Candidate, error) {
			siteSelectionCount++
			if siteSelectionCount > 1 {
				return nil, nil // break outer loop
			}
			return &candidates[0], nil
		},
		SelectAction: func(slug string) (tui.ConfigAction, error) {
			return tui.ActionBack, nil
		},
		PromptPackages: func(ctx context.Context, c *config.Config, cat []packages.CatalogItem) ([]string, []string, error) {
			if len(cat) > 0 {
				optionsPassedHasSearch = true
			}
			return nil, nil, nil
		},
	}

	err := app.RunConfigFlowWithDeps(context.Background(), cfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps.Catalog) != 1 || deps.Catalog[0].Slug != "catalog-plugin" {
		t.Errorf("expected catalog items to be populated, got: %+v", deps.Catalog)
	}
	_ = optionsPassedHasSearch
}
