package tui_test

import (
	"errors"
	"path/filepath"
	"testing"

	"wptui/internal/config"
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

func TestBuildCreateForm_UnifiedInputs(t *testing.T) {
	tempHome := t.TempDir()
	cfg := config.DefaultConfig(tempHome)

	inputs := &tui.CreateInputs{
		WebsiteName: "My Unified Site",
		WebsiteSlug: "", // blank to auto-derive
	}

	form := tui.BuildCreateForm(inputs, cfg)
	if form == nil {
		t.Fatal("expected non-nil form from BuildCreateForm")
	}
}

func TestResolveAndValidateSlug(t *testing.T) {
	var checkedSlug string
	checker := func(slug string) error {
		checkedSlug = slug
		if slug == "colliding-slug" {
			return errors.New("already exists")
		}
		return nil
	}

	// 1. Blank slug automatically derives from name and calls checker with derived slug
	derived, err := tui.ResolveAndValidateSlug("Cool New Site", "", checker)
	if err != nil {
		t.Fatalf("expected blank slug to succeed, got %v", err)
	}
	if derived != "cool-new-site" {
		t.Errorf("expected derived slug 'cool-new-site', got %q", derived)
	}
	if checkedSlug != "cool-new-site" {
		t.Errorf("expected checker called with 'cool-new-site', got %q", checkedSlug)
	}

	// 2. Explicit slug overrides name and calls checker
	checkedSlug = ""
	custom, err := tui.ResolveAndValidateSlug("Cool New Site", "  custom-slug  ", checker)
	if err != nil {
		t.Fatalf("expected trimmed custom slug to succeed, got %v", err)
	}
	if custom != "custom-slug" {
		t.Errorf("expected trimmed custom slug 'custom-slug', got %q", custom)
	}
	if checkedSlug != "custom-slug" {
		t.Errorf("expected checker called with 'custom-slug', got %q", checkedSlug)
	}

	// 3. Collision on derived slug returns error
	_, err = tui.ResolveAndValidateSlug("Colliding Slug", "", checker)
	if err == nil || err.Error() != "already exists" {
		t.Fatalf("expected collision error on derived slug, got %v", err)
	}

	// 4. Invalid explicit slug rejects on syntax without calling checker
	checkedSlug = ""
	_, err = tui.ResolveAndValidateSlug("Cool New Site", "-invalid-edge-hyphen-", checker)
	if err == nil {
		t.Fatal("expected syntax error on invalid explicit slug")
	}
	if checkedSlug != "" {
		t.Fatalf("checker should not be called on invalid slug, called with %q", checkedSlug)
	}
}
