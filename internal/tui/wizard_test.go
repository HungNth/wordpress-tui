package tui_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"wptui/internal/config"
	"wptui/internal/tui"
)

func TestWizard_ClearedAppCommandFailsValidation(t *testing.T) {
	tempHome := t.TempDir()

	inputs := tui.WizardInputs{
		AppName:              config.AppNameHerd,
		WebsitesPath:         filepath.Join(tempHome, "Herd"),
		AppCommand:           "", // explicitly cleared
		DefaultAdminUsername: "admin",
		DefaultAdminPassword: "admin",
		DefaultAdminEmail:    "admin@admin.com",
		DatabaseHost:         "localhost",
		DatabasePortStr:      "3306",
		DBUsername:           "root",
	}

	_, err := tui.ConvertInputsToConfig(inputs, tempHome)
	if err == nil {
		t.Fatal("expected error when app_command is cleared, got nil")
	}
	if !strings.Contains(err.Error(), "app_command is required") {
		t.Errorf("expected 'app_command is required' error, got: %v", err)
	}
}

func TestWizard_MalformedDatabasePortReturnsError(t *testing.T) {
	tempHome := t.TempDir()

	inputs := tui.WizardInputs{
		AppName:              config.AppNameFlock,
		WebsitesPath:         filepath.Join(tempHome, "Sites"),
		AppCommand:           "flock",
		DefaultAdminUsername: "admin",
		DefaultAdminPassword: "admin",
		DefaultAdminEmail:    "admin@admin.com",
		DatabaseHost:         "localhost",
		DatabasePortStr:      "not-a-number",
		DBUsername:           "root",
	}

	_, err := tui.ConvertInputsToConfig(inputs, tempHome)
	if err == nil {
		t.Fatal("expected error for nonnumeric database port, got nil")
	}
	if !strings.Contains(err.Error(), "invalid database port") {
		t.Errorf("expected 'invalid database port' error, got: %v", err)
	}
}

func TestWizard_OutOfRangeDatabasePortReturnsError(t *testing.T) {
	tempHome := t.TempDir()

	for _, invalidPort := range []string{"0", "-1", "70000"} {
		inputs := tui.WizardInputs{
			AppName:              config.AppNameFlock,
			WebsitesPath:         filepath.Join(tempHome, "Sites"),
			AppCommand:           "flock",
			DefaultAdminUsername: "admin",
			DefaultAdminPassword: "admin",
			DefaultAdminEmail:    "admin@admin.com",
			DatabaseHost:         "localhost",
			DatabasePortStr:      invalidPort,
			DBUsername:           "root",
		}

		_, err := tui.ConvertInputsToConfig(inputs, tempHome)
		if err == nil {
			t.Fatalf("expected error for out-of-range database port %q, got nil", invalidPort)
		}
		if !strings.Contains(err.Error(), "invalid database port") {
			t.Errorf("expected 'invalid database port' error for %q, got: %v", invalidPort, err)
		}
	}
}

func TestWizard_SocketMayOmitPort(t *testing.T) {
	tempHome := t.TempDir()

	inputs := tui.WizardInputs{
		AppName:              config.AppNameFlock,
		WebsitesPath:         filepath.Join(tempHome, "Sites"),
		AppCommand:           "flock",
		DefaultAdminUsername: "admin",
		DefaultAdminPassword: "admin",
		DefaultAdminEmail:    "admin@admin.com",
		DBSocket:             "/tmp/mysql.sock",
		DatabasePortStr:      "", // omitted
		DBUsername:           "root",
	}

	cfg, err := tui.ConvertInputsToConfig(inputs, tempHome)
	if err != nil {
		t.Fatalf("expected socket without port to succeed, got error: %v", err)
	}
	if cfg.DBSocket != "/tmp/mysql.sock" {
		t.Errorf("expected DBSocket '/tmp/mysql.sock', got %q", cfg.DBSocket)
	}
}

func TestWizard_CustomPathAndCommandPreserved(t *testing.T) {
	tempHome := t.TempDir()

	customPath := filepath.Join(tempHome, "CustomSites")
	customCmd := "/custom/bin/flock"

	inputs := tui.WizardInputs{
		AppName:              config.AppNameFlock,
		WebsitesPath:         customPath,
		AppCommand:           customCmd,
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

	if cfg.WebsitesPath != customPath {
		t.Errorf("expected WebsitesPath %s, got %s", customPath, cfg.WebsitesPath)
	}
	if cfg.AppCommand != customCmd {
		t.Errorf("expected AppCommand %s, got %s", customCmd, cfg.AppCommand)
	}
}

func TestWizard_NoEnvironmentSelectedReturnsError(t *testing.T) {
	tempHome := t.TempDir()

	inputs := tui.WizardInputs{
		AppName:              "", // no environment selected
		DefaultAdminUsername: "admin",
		DefaultAdminPassword: "admin",
		DefaultAdminEmail:    "admin@admin.com",
		DatabaseHost:         "localhost",
		DatabasePortStr:      "3306",
		DBUsername:           "root",
	}

	_, err := tui.ConvertInputsToConfig(inputs, tempHome)
	if err == nil {
		t.Fatal("expected error when no environment is selected, got nil")
	}
}

func TestEnvironmentOptions_PlatformFiltering(t *testing.T) {
	// Windows exposes Herd, Flock, and Laragon
	winOpts := tui.EnvironmentOptions("windows")
	if len(winOpts) != 3 {
		t.Fatalf("expected 3 options on windows, got %d", len(winOpts))
	}
	expectedWin := []string{config.AppNameHerd, config.AppNameFlock, config.AppNameLaragon}
	for i, opt := range winOpts {
		if opt.Value != expectedWin[i] {
			t.Errorf("windows opt[%d]: expected %s, got %s", i, expectedWin[i], opt.Value)
		}
	}

	// macOS excludes Laragon
	macOpts := tui.EnvironmentOptions("darwin")
	if len(macOpts) != 2 {
		t.Fatalf("expected 2 options on darwin, got %d", len(macOpts))
	}
	expectedMac := []string{config.AppNameHerd, config.AppNameFlock}
	for i, opt := range macOpts {
		if opt.Value != expectedMac[i] {
			t.Errorf("darwin opt[%d]: expected %s, got %s", i, expectedMac[i], opt.Value)
		}
	}

	// Linux also excludes Laragon
	linuxOpts := tui.EnvironmentOptions("linux")
	if len(linuxOpts) != 2 {
		t.Fatalf("expected 2 options on linux, got %d", len(linuxOpts))
	}
}

func TestBuildEnvironmentForm(t *testing.T) {
	var inputs tui.WizardInputs
	form := tui.BuildEnvironmentForm(&inputs, "windows")
	if form == nil {
		t.Fatal("expected non-nil form from BuildEnvironmentForm")
	}
}

func TestBuildEnvironmentForm_SentinelBehavior(t *testing.T) {
	var inputs tui.WizardInputs
	form := tui.BuildEnvironmentForm(&inputs, "windows")
	if form == nil {
		t.Fatal("expected non-nil form from BuildEnvironmentForm")
	}

	// 1. Initial state: inputs.AppName must remain empty (sentinel selected, NOT preselecting Herd)
	if inputs.AppName != "" {
		t.Errorf("expected inputs.AppName to be empty on initialization, got %q", inputs.AppName)
	}

	// 2. Pressing Enter on sentinel option fails validation: does not complete, inputs.AppName remains empty
	m, cmd := form.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updatedForm, ok := m.(*huh.Form)
	if !ok {
		t.Fatalf("expected *huh.Form, got %T", m)
	}
	if updatedForm.State == huh.StateCompleted {
		t.Error("expected form not to complete on sentinel option")
	}
	if cmd != nil {
		t.Errorf("expected no completion command on failed sentinel validation, got %v", cmd)
	}
	if inputs.AppName != "" {
		t.Errorf("expected inputs.AppName to remain empty after rejected Enter, got %q", inputs.AppName)
	}

	// 3. Navigating down selects Herd, updating inputs.AppName
	m, _ = updatedForm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if inputs.AppName != config.AppNameHerd {
		t.Errorf("expected inputs.AppName to be %q after first KeyDown, got %q", config.AppNameHerd, inputs.AppName)
	}

	// 4. Navigating down again selects Flock
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if inputs.AppName != config.AppNameFlock {
		t.Errorf("expected inputs.AppName to be %q after second KeyDown, got %q", config.AppNameFlock, inputs.AppName)
	}

	// 5. Navigating down again on Windows selects Laragon
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if inputs.AppName != config.AppNameLaragon {
		t.Errorf("expected inputs.AppName to be %q after third KeyDown, got %q", config.AppNameLaragon, inputs.AppName)
	}
}

func TestBuildMainWizardForm_SuggestionsPopulated(t *testing.T) {
	tempHome := t.TempDir()

	inputs := &tui.WizardInputs{
		AppName: config.AppNameLaragon,
	}

	form := tui.BuildMainWizardForm(inputs, tempHome)
	if form == nil {
		t.Fatal("expected non-nil form from BuildMainWizardForm")
	}

	if inputs.WebsitesPath != `C:\laragon\www` {
		t.Errorf("expected WebsitesPath `C:\\laragon\\www`, got %s", inputs.WebsitesPath)
	}
	if inputs.AppCommand != `C:\laragon\laragon.exe` {
		t.Errorf("expected AppCommand `C:\\laragon\\laragon.exe`, got %s", inputs.AppCommand)
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
