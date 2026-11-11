package tui_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/backup"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/packages"
	"wptui/internal/tui"
)

func createTestWebsites(t *testing.T, count int) (string, []string) {
	t.Helper()
	dir := t.TempDir()
	var slugs []string
	for i := 1; i <= count; i++ {
		slugName := fmt.Sprintf("site%d", i)
		slug := filepath.Join(dir, slugName)
		if err := os.MkdirAll(slug, 0755); err != nil {
			t.Fatal(err)
		}
		// Create mock wp-config.php so it detects as valid candidate
		if err := os.WriteFile(filepath.Join(slug, "wp-config.php"), []byte("<?php"), 0644); err != nil {
			t.Fatal(err)
		}
		slugs = append(slugs, slugName)
	}
	return dir, slugs
}

func TestWebsitesHub_DiscoveryAndEmptyState(t *testing.T) {
	emptyDir := t.TempDir()
	cfg := config.DefaultConfig(emptyDir)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.WebsitesPath = emptyDir
	hub := tui.NewWebsitesHubModel(cfg)
	view := hub.Render(60, 20)
	if !strings.Contains(view, "No websites found") {
		t.Errorf("expected empty state message when no websites exist, got:\n%s", view)
	}

	// Now with 2 sites
	sitesDir, _ := createTestWebsites(t, 2)
	cfg2 := config.DefaultConfig(sitesDir)
	cfg2.AppName = config.AppNameHerd
	cfg2.AppCommand = "herd"
	cfg2.WebsitesPath = sitesDir

	hub2 := tui.NewWebsitesHubModel(cfg2)
	view2 := hub2.Render(60, 20)
	if !strings.Contains(view2, "site1") || !strings.Contains(view2, "site2") {
		t.Errorf("expected view to contain site1 and site2, got:\n%s", view2)
	}
	if !strings.Contains(view2, "https://site1.test") {
		t.Errorf("expected view to contain https://site1.test, got:\n%s", view2)
	}
}

func TestWebsitesHub_NavigationAndMultiSelect(t *testing.T) {
	sitesDir, slugs := createTestWebsites(t, 2)
	cfg := config.DefaultConfig(sitesDir)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.WebsitesPath = sitesDir

	hub := tui.NewWebsitesHubModel(cfg)

	// Initially cursor on site1
	if hub.SelectedCandidate().Slug != slugs[0] {
		t.Fatalf("expected initial cursor on %s, got %s", slugs[0], hub.SelectedCandidate().Slug)
	}

	// Press Space to toggle selection on site1
	hub.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if !hub.IsSelected(slugs[0]) {
		t.Errorf("expected %s to be selected after Space", slugs[0])
	}
	view := hub.Render(60, 20)
	if !strings.Contains(view, "[x]") {
		t.Errorf("expected view to contain [x] after Space, got:\n%s", view)
	}

	// Move Down to site2 with 'j'
	hub.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if hub.SelectedCandidate().Slug != slugs[1] {
		t.Fatalf("expected cursor on %s after 'j', got %s", slugs[1], hub.SelectedCandidate().Slug)
	}

	// Press 'd' with site1 selected -> enters BatchDeleteConfirm
	hub.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if hub.State() != tui.WebsitesHubBatchDeleteConfirm {
		t.Fatalf("expected state WebsitesHubBatchDeleteConfirm after 'd', got %v", hub.State())
	}
	deleteView := hub.Render(60, 20)
	if !strings.Contains(deleteView, "Delete Selected Websites?") || !strings.Contains(deleteView, slugs[0]) {
		t.Errorf("expected delete confirmation to list site1, got:\n%s", deleteView)
	}

	// Press Esc to cancel confirmation
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if hub.State() != tui.WebsitesHubList {
		t.Fatalf("expected state WebsitesHubList after Esc, got %v", hub.State())
	}
}

func TestWebsitesHub_SingleDeleteConfirm(t *testing.T) {
	sitesDir, slugs := createTestWebsites(t, 1)
	cfg := config.DefaultConfig(sitesDir)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.WebsitesPath = sitesDir

	hub := tui.NewWebsitesHubModel(cfg)

	// Press Enter to enter Action Menu
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	for hub.SelectedAction() != tui.WebsiteActionDelete {
		hub.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if hub.State() != tui.WebsitesHubSingleDeleteConfirm {
		t.Fatalf("expected state WebsitesHubSingleDeleteConfirm, got %v", hub.State())
	}

	view := hub.Render(60, 20)
	if !strings.Contains(view, "Delete Website "+slugs[0]+"?") {
		t.Errorf("expected single delete confirmation for %s, got:\n%s", slugs[0], view)
	}
	// Default confirmation should be No
	if hub.DeleteConfirmChoice() != false {
		t.Errorf("expected default confirmation to be false (No)")
	}

	// Toggle choice with Left
	hub.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if hub.DeleteConfirmChoice() != true {
		t.Errorf("expected confirmation to be true (Yes) after Left")
	}

	// Cancel with Esc
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if hub.State() != tui.WebsitesHubActions {
		t.Fatalf("expected return to WebsitesHubActions on Esc, got %v", hub.State())
	}
}

func TestWebsitesHub_ChangeAdminCredentials(t *testing.T) {
	sitesDir, slugs := createTestWebsites(t, 1)
	cfg := config.DefaultConfig(sitesDir)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.WebsitesPath = sitesDir

	hub := tui.NewWebsitesHubModel(cfg)

	var (
		capturedCand     deprovision.Candidate
		capturedUser     string
		capturedPass     string
		capturedEmail    string
		callbackExecuted bool
	)

	hub.OnChangeAdmin = func(cand deprovision.Candidate, username, password, email string) tea.Cmd {
		capturedCand = cand
		capturedUser = username
		capturedPass = password
		capturedEmail = email
		callbackExecuted = true
		return nil
	}

	// Press Enter to enter Action Menu
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	for hub.SelectedAction() != tui.WebsiteActionChangeAdmin {
		hub.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if hub.State() != tui.WebsitesHubChangeAdmin {
		t.Fatalf("expected state WebsitesHubChangeAdmin, got %v", hub.State())
	}

	view := hub.Render(80, 24)
	if !strings.Contains(view, "Change Administrator Credentials") {
		t.Errorf("expected view to contain 'Change Administrator Credentials', got:\n%s", view)
	}

	// Submit form with default values (enter through fields to submit)
	// Field 0 (Username) -> enter -> Field 1 (Password) -> enter -> Field 2 (Email) -> enter -> Field 3 (Submit) -> enter
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if !callbackExecuted {
		t.Fatalf("expected OnChangeAdmin callback to be executed")
	}
	if capturedCand.Slug != slugs[0] {
		t.Errorf("expected slug %s, got %s", slugs[0], capturedCand.Slug)
	}
	if capturedUser != cfg.DefaultAdminUsername {
		t.Errorf("expected default username %s, got %s", cfg.DefaultAdminUsername, capturedUser)
	}
	if capturedPass != cfg.DefaultAdminPassword {
		t.Errorf("expected default password %s, got %s", cfg.DefaultAdminPassword, capturedPass)
	}
	if capturedEmail != cfg.DefaultAdminEmail {
		t.Errorf("expected default email %s, got %s", cfg.DefaultAdminEmail, capturedEmail)
	}
}

func TestWebsitesHub_InstallPluginsAndThemes(t *testing.T) {
	sitesDir, slugs := createTestWebsites(t, 1)
	cfg := config.DefaultConfig(sitesDir)
	cfg.WebsitesPath = sitesDir

	catalog := []packages.CatalogItem{
		{Name: "WooCommerce", Slug: "woocommerce", Type: "plugin"},
		{Name: "Yoast SEO", Slug: "wordpress-seo", Type: "plugin"},
		{Name: "Astra", Slug: "astra", Type: "theme"},
	}

	hub := tui.NewWebsitesHubModel(cfg)
	hub.SetCatalog(catalog)

	var (
		capturedCand        deprovision.Candidate
		capturedType        packages.PackageType
		capturedSlugs       []string
		capturedActivate    bool
		capturedActiveTheme string
		callbackExecuted    bool
	)

	hub.OnInstallPackages = func(cand deprovision.Candidate, pkgType packages.PackageType, slugs []string, activate bool, activeTheme ...string) tea.Cmd {
		capturedCand = cand
		capturedType = pkgType
		capturedSlugs = slugs
		capturedActivate = activate
		if len(activeTheme) > 0 {
			capturedActiveTheme = activeTheme[0]
		}
		callbackExecuted = true
		return nil
	}

	// 1. Test Install Plugins (index 1: Config(0), Install Plugins(1))
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // enter action menu
	hub.Update(tea.KeyPressMsg{Code: tea.KeyDown})  // index 1
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // open Install Plugins

	if hub.State() != tui.WebsitesHubInstallPlugins {
		t.Fatalf("expected state WebsitesHubInstallPlugins, got %v", hub.State())
	}

	pluginView := hub.Render(80, 24)
	if !strings.Contains(pluginView, "Install Plugins") || !strings.Contains(pluginView, "WooCommerce") {
		t.Errorf("expected plugin picker view with WooCommerce, got:\n%s", pluginView)
	}

	// Select WooCommerce (space) and submit (enter)
	hub.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if !callbackExecuted {
		t.Fatalf("expected OnInstallPackages callback executed for plugins")
	}
	if capturedCand.Slug != slugs[0] {
		t.Errorf("expected cand slug %s, got %s", slugs[0], capturedCand.Slug)
	}
	if capturedType != packages.PackageTypePlugin {
		t.Errorf("expected PackageTypePlugin, got %v", capturedType)
	}
	if len(capturedSlugs) != 1 || capturedSlugs[0] != "woocommerce" {
		t.Errorf("expected [woocommerce], got %v", capturedSlugs)
	}

	// 2. Test Install Themes (index 2: Config(0), Install Plugins(1), Install Theme(2))
	callbackExecuted = false
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // enter action menu
	hub.Update(tea.KeyPressMsg{Code: tea.KeyDown})  // index 1
	hub.Update(tea.KeyPressMsg{Code: tea.KeyDown})  // index 2
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // open Install Themes

	if hub.State() != tui.WebsitesHubInstallThemes {
		t.Fatalf("expected state WebsitesHubInstallThemes, got %v", hub.State())
	}

	themeView := hub.Render(80, 24)
	if !strings.Contains(themeView, "Install Themes") || !strings.Contains(themeView, "Astra") {
		t.Errorf("expected theme picker view with Astra, got:\n%s", themeView)
	}

	// Select Astra (space) and submit (enter)
	hub.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if hub.State() != tui.WebsitesHubThemeActivateConfirm {
		t.Fatalf("expected WebsitesHubThemeActivateConfirm after selecting theme, got %v", hub.State())
	}
	// Navigate to Astra (option 1: 0 is None, 1 is astra)
	hub.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	// Confirm activation (enter)
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if !callbackExecuted {
		t.Fatalf("expected OnInstallPackages callback executed for themes")
	}
	if capturedType != packages.PackageTypeTheme {
		t.Errorf("expected PackageTypeTheme, got %v", capturedType)
	}
	if len(capturedSlugs) != 1 || capturedSlugs[0] != "astra" {
		t.Errorf("expected [astra], got %v", capturedSlugs)
	}
	if !capturedActivate {
		t.Errorf("expected activate=true for theme installation")
	}
	if capturedActiveTheme != "astra" {
		t.Errorf("expected activeTheme='astra', got %q", capturedActiveTheme)
	}
}

func TestWebsitesHub_BackupStrategySelection(t *testing.T) {
	sitesDir, slugs := createTestWebsites(t, 1)
	cfg := config.DefaultConfig(sitesDir)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.WebsitesPath = sitesDir

	hub := tui.NewWebsitesHubModel(cfg)

	var (
		capturedCand     deprovision.Candidate
		capturedStrategy backup.BackupStrategy
		callbackExecuted bool
	)

	hub.OnBackup = func(cand deprovision.Candidate, strategy backup.BackupStrategy) tea.Cmd {
		capturedCand = cand
		capturedStrategy = strategy
		callbackExecuted = true
		return nil
	}

	// 1. Enter Action Menu on website
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	for hub.SelectedAction() != tui.WebsiteActionBackup {
		hub.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Assert state is WebsitesHubBackupStrategy
	if hub.State() != tui.WebsitesHubBackupStrategy {
		t.Fatalf("expected state WebsitesHubBackupStrategy, got %v", hub.State())
	}

	view := hub.Render(80, 24)
	if !strings.Contains(view, "Full source code & database (.zip)") {
		t.Errorf("expected view to contain 'Full source code & database (.zip)', got:\n%s", view)
	}
	if !strings.Contains(view, "All-in-One WP Migration (.wpress)") {
		t.Errorf("expected view to contain 'All-in-One WP Migration (.wpress)', got:\n%s", view)
	}

	// 2. Select All-in-One WP Migration (down to index 1 and enter)
	hub.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if !callbackExecuted {
		t.Fatalf("expected OnBackup callback to be executed")
	}
	if capturedCand.Slug != slugs[0] {
		t.Errorf("expected cand slug %s, got %s", slugs[0], capturedCand.Slug)
	}
	if capturedStrategy != backup.StrategyAI1WM {
		t.Errorf("expected StrategyAI1WM, got %v", capturedStrategy)
	}
}

type mockLauncherRunner struct {
	started []string
}

func (m *mockLauncherRunner) LookPath(file string) (string, error) {
	return "/bin/" + file, nil
}

func (m *mockLauncherRunner) Run(ctx context.Context, name string, args ...string) error {
	return nil
}

func (m *mockLauncherRunner) Start(ctx context.Context, name string, args ...string) error {
	m.started = append(m.started, name+" "+strings.Join(args, " "))
	return nil
}

func TestWebsitesHub_OpenInBrowser_RespectsEnvironment(t *testing.T) {
	sitesDir, slugs := createTestWebsites(t, 1)

	for _, appName := range []string{config.AppNameHerd, config.AppNameFlock, config.AppNameLaragon} {
		cfg := config.DefaultConfig(sitesDir)
		cfg.WebsitesPath = sitesDir
		cfg.AppName = appName
		cfg.AppCommand = appName
		hub := tui.NewWebsitesHubModel(cfg)
		runner := &mockLauncherRunner{}
		hub.SetRunner(runner)

		hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // Open actions
		for hub.SelectedAction() != tui.WebsiteActionBrowser {
			hub.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		}
		hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

		if len(runner.started) != 1 {
			t.Fatalf("expected 1 browser launch for %s, got %d", appName, len(runner.started))
		}
		expectedURL := "https://" + slugs[0] + ".test"
		if !strings.Contains(runner.started[0], expectedURL) {
			t.Errorf("expected launch command for %s to contain %s, got: %s", appName, expectedURL, runner.started[0])
		}
	}
}

func TestWebsitesHub_Render_RespectsEnvironment(t *testing.T) {
	sitesDir, slugs := createTestWebsites(t, 1)

	for _, appName := range []string{config.AppNameHerd, config.AppNameFlock, config.AppNameLaragon} {
		cfg := config.DefaultConfig(sitesDir)
		cfg.WebsitesPath = sitesDir
		cfg.AppName = appName
		cfg.AppCommand = appName
		hub := tui.NewWebsitesHubModel(cfg)
		view := hub.Render(80, 24)
		if !strings.Contains(view, "https://"+slugs[0]+".test") {
			t.Errorf("expected view for %s to contain https://%s.test, got:\n%s", appName, slugs[0], view)
		}
	}
}

func TestWebsitesHub_SelectAllAndEnterBatchDelete(t *testing.T) {
	sitesDir, slugs := createTestWebsites(t, 3)
	cfg := config.DefaultConfig(sitesDir)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.WebsitesPath = sitesDir
	hub := tui.NewWebsitesHubModel(cfg)

	// 1. Press 'a' to select all
	hub.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if hub.SelectedCount() != 3 {
		t.Fatalf("expected 3 selected after 'a', got %d", hub.SelectedCount())
	}
	for _, slug := range slugs {
		if !hub.IsSelected(slug) {
			t.Errorf("expected %s to be selected", slug)
		}
	}

	// 2. Press 'a' again to deselect all
	hub.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if hub.SelectedCount() != 0 {
		t.Fatalf("expected 0 selected after second 'a', got %d", hub.SelectedCount())
	}

	// 3. When SelectedCount <= 1, Enter opens single website details
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if hub.State() != tui.WebsitesHubActions {
		t.Fatalf("expected WebsitesHubActions when SelectedCount <= 1, got %v", hub.State())
	}

	// Return to list
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if hub.State() != tui.WebsitesHubList {
		t.Fatalf("expected WebsitesHubList after Esc, got %v", hub.State())
	}

	// 4. Select 2 sites with Space
	hub.Update(tea.KeyPressMsg{Code: ' ', Text: " "}) // site 1
	hub.Update(tea.KeyPressMsg{Code: 'j', Text: "j"}) // down
	hub.Update(tea.KeyPressMsg{Code: ' ', Text: " "}) // site 2
	if hub.SelectedCount() != 2 {
		t.Fatalf("expected 2 selected sites, got %d", hub.SelectedCount())
	}

	// 5. Press Enter with 2 sites selected -> directly goes to WebsitesHubBatchDeleteConfirm
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if hub.State() != tui.WebsitesHubBatchDeleteConfirm {
		t.Fatalf("expected WebsitesHubBatchDeleteConfirm when Enter pressed with 2 sites selected, got %v", hub.State())
	}

	// Verify single confirmation execution
	var deleted []deprovision.Candidate
	hub.OnBatchDelete = func(cands []deprovision.Candidate) tea.Cmd {
		deleted = cands
		return nil
	}

	// Toggle choice to Yes (Right arrow) and press Enter
	hub.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	hub.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if len(deleted) != 2 {
		t.Fatalf("expected OnBatchDelete to be called with 2 candidates, got %d", len(deleted))
	}
}

func TestWebsitesHub_ScrollingAndCompactLayout(t *testing.T) {
	sitesDir, _ := createTestWebsites(t, 25)
	cfg := config.DefaultConfig(sitesDir)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.WebsitesPath = sitesDir
	hub := tui.NewWebsitesHubModel(cfg)

	// 1. Initial view at (80, 20)
	viewInitial := hub.Render(80, 20)

	// Must be compact 1-line: no "Path: " in the list
	if strings.Contains(viewInitial, "Path: ") {
		t.Errorf("expected compact 1-line format without 'Path: ', but found 'Path: ' in:\n%s", viewInitial)
	}
	// Must contain first site URL
	if !strings.Contains(viewInitial, "https://site1.test") {
		t.Errorf("expected view to contain URL https://site1.test, got:\n%s", viewInitial)
	}

	// 2. Navigate down 18 times to reach site19 (0-indexed cursor = 18)
	for i := 0; i < 18; i++ {
		hub.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	}

	// Also check in AppModel with terminal height 24 (bodyHeight = 20)
	app := tui.NewAppModel(cfg)
	// Focus content
	app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	// Press down 18 times in Websites list
	for i := 0; i < 18; i++ {
		app.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	}
	appView := app.ViewString()
	if !strings.Contains(appView, "site19") {
		t.Logf("CONFIRMED BUG: in AppModel view (height 30/24), site19 is clipped and invisible!")
	}
}
