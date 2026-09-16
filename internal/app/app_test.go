package app_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"wptui/internal/app"
	"wptui/internal/config"
	"wptui/internal/packages"
	"wptui/internal/tui"
)

func TestApp_LoadExistingConfig(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")

	cfg := config.DefaultConfig(tempHome)
	cfg.DefaultAdminUsername = "tester"
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	application := app.New(app.Options{
		HomeDir: tempHome,
	})

	loadedCfg, err := application.InitConfig(false)
	if err != nil {
		t.Fatalf("InitConfig() failed: %v", err)
	}

	if loadedCfg.DefaultAdminUsername != "tester" {
		t.Errorf("expected loaded config admin username tester, got %s", loadedCfg.DefaultAdminUsername)
	}
}

func TestApp_InvalidConfigHalts(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("{ malformed json }"), 0600); err != nil {
		t.Fatal(err)
	}

	application := app.New(app.Options{
		HomeDir: tempHome,
	})

	_, err := application.InitConfig(false)
	if err == nil {
		t.Errorf("expected error when loading malformed config, got nil")
	}
}

func TestRunCreateFlow_CollisionCheckRejectsBeforePackageSelection(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	// Pre-create directory to trigger collision
	collidedDir := filepath.Join(cfg.WebsitesPath, "taken-site")
	if err := os.MkdirAll(collidedDir, 0755); err != nil {
		t.Fatal(err)
	}

	var packagePromptCalled bool
	deps := app.CreateFlowDependencies{
		PromptCreate: func(c *config.Config, checker ...tui.SlugAvailabilityChecker) (*tui.CreateInputs, error) {
			// Exercise the actual injected availabilityChecker!
			if len(checker) > 0 && checker[0] != nil {
				if err := checker[0]("taken-site"); err != nil {
					return nil, err
				}
			}
			return &tui.CreateInputs{
				WebsiteName: "Taken Site",
				WebsiteSlug: "taken-site",
			}, nil
		},
		PromptPackages: func(ctx context.Context, c *config.Config, items []packages.CatalogItem) ([]string, []string, error) {
			packagePromptCalled = true
			return nil, nil, nil
		},
	}

	err := app.RunCreateFlowWithDeps(context.Background(), cfg, deps)
	if err == nil {
		t.Fatal("expected error on colliding directory, got nil")
	}
	if packagePromptCalled {
		t.Fatal("PromptPackages should not have been called when slug collided")
	}
}

func TestRunCreateFlow_CustomPromptBypassCaughtBeforePackages(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	// Pre-create directory to trigger collision
	collidedDir := filepath.Join(cfg.WebsitesPath, "bypassed-slug")
	if err := os.MkdirAll(collidedDir, 0755); err != nil {
		t.Fatal(err)
	}

	var packagePromptCalled bool
	deps := app.CreateFlowDependencies{
		// Custom prompt callback that completely ignores the injected checker and returns a colliding slug
		PromptCreate: func(c *config.Config, checker ...tui.SlugAvailabilityChecker) (*tui.CreateInputs, error) {
			return &tui.CreateInputs{
				WebsiteName: "Bypassed Site",
				WebsiteSlug: "bypassed-slug",
			}, nil
		},
		PromptPackages: func(ctx context.Context, c *config.Config, items []packages.CatalogItem) ([]string, []string, error) {
			packagePromptCalled = true
			return nil, nil, nil
		},
	}

	err := app.RunCreateFlowWithDeps(context.Background(), cfg, deps)
	if err == nil {
		t.Fatal("expected collision error when custom prompt bypasses checker, got nil")
	}
	if packagePromptCalled {
		t.Fatal("PromptPackages should not have been called when custom prompt returned colliding slug")
	}
}
