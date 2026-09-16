package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"wptui/internal/app"
	"wptui/internal/config"
)

func TestTicket01_DemoIsolatedHome(t *testing.T) {
	tempHome := t.TempDir()

	wizardRunCount := 0
	mockWizard := func(home string) (*config.Config, error) {
		wizardRunCount++
		cfg := config.DefaultConfig(home)
		cfg.DefaultAdminUsername = "demo_admin"
		cfg.PackagesAPIURL = "https://example.com/api/v1"
		return cfg, nil
	}

	menuAction := "exit"
	mockMenu := func() (string, error) {
		return menuAction, nil
	}

	// Step 1: First-run creation
	app1 := app.New(app.Options{
		HomeDir:  tempHome,
		WizardFn: mockWizard,
		MenuFn:   mockMenu,
	})

	if err := app1.Run(); err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if wizardRunCount != 1 {
		t.Fatalf("expected wizard to run once, ran %d times", wizardRunCount)
	}

	cfgFile := filepath.Join(tempHome, ".config", "wptui", "config.json")
	if _, err := os.Stat(cfgFile); err != nil {
		t.Fatalf("config file was not created: %v", err)
	}

	// Step 2: Second-run loading without wizard
	app2 := app.New(app.Options{
		HomeDir: tempHome,
		WizardFn: func(home string) (*config.Config, error) {
			t.Fatal("wizard should not be called when config already exists")
			return nil, nil
		},
		MenuFn: mockMenu,
	})

	if err := app2.Run(); err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	if app2.Config().DefaultAdminUsername != "demo_admin" {
		t.Fatalf("expected loaded admin username demo_admin, got %s", app2.Config().DefaultAdminUsername)
	}

	// Step 3: Corrupt config refusal without rewriting
	if err := os.WriteFile(cfgFile, []byte("{ malformed-json"), 0600); err != nil {
		t.Fatal(err)
	}

	app3 := app.New(app.Options{
		HomeDir: tempHome,
		MenuFn:  mockMenu,
	})

	if err := app3.Run(); err == nil {
		t.Fatalf("expected app to halt on corrupt config, but got nil")
	}

	// Verify file was left unchanged and not overwritten
	content, _ := os.ReadFile(cfgFile)
	if string(content) != "{ malformed-json" {
		t.Fatalf("corrupt config was overwritten!")
	}
}
