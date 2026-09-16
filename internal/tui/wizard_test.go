package tui_test

import (
	"path/filepath"
	"testing"

	"wptui/internal/tui"
)

func TestWizard_NonHerdUntouchedPathDefaultsToSites(t *testing.T) {
	tempHome := t.TempDir()

	inputs := tui.WizardInputs{
		UsedHerd:             false,
		WebsitesPath:         "", // untouched
		DefaultAdminUsername: "admin",
		DefaultAdminPassword: "admin",
		DefaultAdminEmail:    "admin@admin.com",
		DatabaseHost:         "localhost",
		DatabasePortStr:      "3306",
		DBUsername:           "root",
	}

	cfg, err := tui.ConvertInputsToConfig(inputs, tempHome)
	if err != nil {
		t.Fatalf("ConvertInputsToConfig failed: %v", err)
	}

	expectedSites := filepath.Join(tempHome, "Sites")
	if cfg.WebsitesPath != expectedSites {
		t.Errorf("expected WebsitesPath %s when UsedHerd is false, got %s", expectedSites, cfg.WebsitesPath)
	}
	if cfg.UsedHerd {
		t.Errorf("expected UsedHerd to be false")
	}
}

func TestWizard_HerdUntouchedPathDefaultsToHerd(t *testing.T) {
	tempHome := t.TempDir()

	inputs := tui.WizardInputs{
		UsedHerd:             true,
		WebsitesPath:         "", // untouched
		DefaultAdminUsername: "admin",
		DefaultAdminPassword: "admin",
		DefaultAdminEmail:    "admin@admin.com",
		DatabaseHost:         "localhost",
		DatabasePortStr:      "3306",
		DBUsername:           "root",
	}

	cfg, err := tui.ConvertInputsToConfig(inputs, tempHome)
	if err != nil {
		t.Fatalf("ConvertInputsToConfig failed: %v", err)
	}

	expectedHerd := filepath.Join(tempHome, "Herd")
	if cfg.WebsitesPath != expectedHerd {
		t.Errorf("expected WebsitesPath %s when UsedHerd is true, got %s", expectedHerd, cfg.WebsitesPath)
	}
	if !cfg.UsedHerd {
		t.Errorf("expected UsedHerd to be true")
	}
}
