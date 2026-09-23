package tui_test

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/config"
	"wptui/internal/packages"
	"wptui/internal/tui"
)

func TestCreateWizard_InitialStateAndDefaults(t *testing.T) {
	cfg := config.DefaultConfig(t.TempDir())
	cfg.DefaultAdminUsername = "wptester"
	cfg.DefaultAdminPassword = "secretpassword"
	cfg.DefaultAdminEmail = "test@example.com"

	catalog := []packages.CatalogItem{
		{Name: "ACF Pro", Slug: "acf-pro", Type: "plugin", Version: "1.0"},
	}

	wizard := tui.NewCreateWizardModel(cfg, catalog, nil)

	if wizard.Step() != tui.CreateWizardStepInputs {
		t.Fatalf("expected initial step CreateWizardStepInputs, got %v", wizard.Step())
	}
	if wizard.Inputs().AdminUsername != "wptester" {
		t.Errorf("expected default username wptester, got %s", wizard.Inputs().AdminUsername)
	}
	if wizard.Inputs().AdminPassword != "secretpassword" {
		t.Errorf("expected default password secretpassword, got %s", wizard.Inputs().AdminPassword)
	}
	if wizard.Inputs().AdminEmail != "test@example.com" {
		t.Errorf("expected default email test@example.com, got %s", wizard.Inputs().AdminEmail)
	}

	view := wizard.Render(70, 20)
	if !strings.Contains(view, "Website Name") || !strings.Contains(view, "Website Slug") {
		t.Errorf("expected view to contain form fields, got:\n%s", view)
	}
}

func TestCreateWizard_FieldNavigationAndAutoSlug(t *testing.T) {
	cfg := config.DefaultConfig(t.TempDir())
	catalog := []packages.CatalogItem{
		{Name: "ACF Pro", Slug: "acf-pro", Type: "plugin", Version: "1.0"},
	}

	wizard := tui.NewCreateWizardModel(cfg, catalog, nil)

	// 1. Type "My Cool Site" into Website Name (field 0)
	for _, ch := range "My Cool Site" {
		wizard.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
	}
	if wizard.Inputs().WebsiteName != "My Cool Site" {
		t.Errorf("expected WebsiteName 'My Cool Site', got %s", wizard.Inputs().WebsiteName)
	}

	// 2. Press Tab to advance to Slug (leave it empty)
	wizard.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	// 3. Press Tab to advance to Username
	wizard.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	// 4. Press Tab to advance to Password
	wizard.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	// 5. Press Tab to advance to Email
	wizard.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	// 6. Press Tab to advance to Tweaks checkbox
	wizard.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !wizard.Inputs().ApplyTweaks {
		t.Errorf("expected tweaks to default to true")
	}
	// Press Space to toggle tweaks off
	wizard.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if wizard.Inputs().ApplyTweaks {
		t.Errorf("expected tweaks to be toggled to false")
	}

	// 7. Press Tab to advance to [ Next: Select Packages ] button
	wizard.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	// 8. Press Enter to submit Step 1
	wizard.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Slug should be auto-derived to "my-cool-site" and step should advance to Packages
	if wizard.Inputs().WebsiteSlug != "my-cool-site" {
		t.Errorf("expected auto-derived slug 'my-cool-site', got %s", wizard.Inputs().WebsiteSlug)
	}
	if wizard.Step() != tui.CreateWizardStepPackages {
		t.Fatalf("expected step CreateWizardStepPackages, got %v", wizard.Step())
	}

	// View in Step 2 should display package search
	view2 := wizard.Render(70, 20)
	if !strings.Contains(view2, "Select Packages") || !strings.Contains(view2, "ACF Pro") {
		t.Errorf("expected step 2 view to contain package picker, got:\n%s", view2)
	}

	// Press Esc in Step 2 -> returns to Step 1 with data intact
	wizard.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if wizard.Step() != tui.CreateWizardStepInputs {
		t.Fatalf("expected return to Step 1 on Esc, got %v", wizard.Step())
	}
	if wizard.Inputs().WebsiteName != "My Cool Site" || wizard.Inputs().WebsiteSlug != "my-cool-site" {
		t.Errorf("expected form inputs preserved on back, got name=%s slug=%s", wizard.Inputs().WebsiteName, wizard.Inputs().WebsiteSlug)
	}
}

func TestCreateWizard_SlugCollisionValidation(t *testing.T) {
	cfg := config.DefaultConfig(t.TempDir())
	mockChecker := func(slug string) error {
		if slug == "existing-site" {
			return errors.New("website directory already exists")
		}
		return nil
	}

	wizard := tui.NewCreateWizardModel(cfg, nil, mockChecker)

	// Type "existing-site" as Name
	for _, ch := range "existing-site" {
		wizard.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
	}

	// Navigate to Submit button and press Enter
	for range 6 {
		wizard.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	}
	wizard.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Step must remain Step 1 because validation failed
	if wizard.Step() != tui.CreateWizardStepInputs {
		t.Fatalf("expected wizard to remain in Step 1 on validation error, got %v", wizard.Step())
	}

	view := wizard.Render(70, 20)
	if !strings.Contains(view, "website directory already exists") {
		t.Errorf("expected view to contain error message, got:\n%s", view)
	}
}

func TestCreateWizard_DirtyStateAndDiscardPrompt(t *testing.T) {
	cfg := config.DefaultConfig(t.TempDir())
	wizard := tui.NewCreateWizardModel(cfg, nil, nil)

	// Clean wizard: Esc sets ShouldExitToSidebar = true
	wizard.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !wizard.ShouldExitToSidebar() {
		t.Errorf("expected clean form to exit to sidebar immediately on Esc")
	}

	// Modified wizard:
	wizard2 := tui.NewCreateWizardModel(cfg, nil, nil)
	wizard2.Update(tea.KeyPressMsg{Code: 'a', Text: "a"}) // modifies name

	// Press Esc -> should enter DiscardConfirm state
	wizard2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if wizard2.Step() != tui.CreateWizardConfirmDiscard {
		t.Fatalf("expected step CreateWizardConfirmDiscard on dirty Esc, got %v", wizard2.Step())
	}
	view := wizard2.Render(70, 20)
	if !strings.Contains(view, "Discard changes? (y/n)") {
		t.Errorf("expected view to prompt discard confirmation, got:\n%s", view)
	}

	// Press 'n' -> resumes editing in Step 1
	wizard2.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if wizard2.Step() != tui.CreateWizardStepInputs {
		t.Fatalf("expected return to Step 1 after 'n', got %v", wizard2.Step())
	}

	// Press Esc again -> confirm discard with 'y'
	wizard2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	wizard2.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if !wizard2.ShouldExitToSidebar() {
		t.Errorf("expected ShouldExitToSidebar after 'y'")
	}
}
