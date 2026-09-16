package tui_test

import (
	"errors"
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

func TestValidateSlugWithChecker(t *testing.T) {
	var checkerCalled bool
	mockChecker := func(slug string) error {
		checkerCalled = true
		if slug == "existing-slug" {
			return errors.New("already exists")
		}
		return nil
	}

	// 1. Syntactically invalid slug must fail and NEVER call checker
	checkerCalled = false
	err := tui.ValidateSlugWithChecker("", mockChecker)
	if err == nil {
		t.Fatal("expected empty slug to fail syntax validation")
	}
	if checkerCalled {
		t.Fatal("checker should not be called on syntactically invalid slug")
	}

	checkerCalled = false
	err = tui.ValidateSlugWithChecker("INVALID_UPPERCASE", mockChecker)
	if err == nil {
		t.Fatal("expected uppercase slug to fail syntax validation")
	}
	if checkerCalled {
		t.Fatal("checker should not be called on uppercase slug")
	}

	// 2. Syntactically valid colliding slug calls checker and returns error
	checkerCalled = false
	err = tui.ValidateSlugWithChecker("existing-slug", mockChecker)
	if err == nil || err.Error() != "already exists" {
		t.Fatalf("expected 'already exists' error, got %v", err)
	}
	if !checkerCalled {
		t.Fatal("checker was not called on valid slug")
	}

	// 3. Syntactically valid non-colliding slug succeeds
	checkerCalled = false
	err = tui.ValidateSlugWithChecker("available-slug", mockChecker)
	if err != nil {
		t.Fatalf("expected valid available slug to succeed, got %v", err)
	}
	if !checkerCalled {
		t.Fatal("checker was not called on valid slug")
	}
}
