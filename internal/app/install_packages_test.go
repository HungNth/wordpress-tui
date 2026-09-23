package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wptui/internal/app"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/packages"
	"wptui/internal/tui"
)

func TestApp_InstallPackagesExecution_Wiring(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}

	application := app.New(app.Options{HomeDir: tempHome})
	_, err := application.InitConfig(false)
	if err != nil {
		t.Fatal(err)
	}

	appModel := application.BuildAppModel(context.Background())
	if appModel.WebsitesHub().OnInstallPackages == nil {
		t.Fatalf("expected OnInstallPackages callback to be wired on WebsitesHub")
	}

	cand := deprovision.Candidate{
		Slug: "my-site",
		Path: filepath.Join(tempHome, "sites", "my-site"),
	}

	cmd := appModel.WebsitesHub().OnInstallPackages(cand, packages.PackageTypePlugin, []string{"woocommerce"}, true)
	if cmd == nil {
		t.Fatalf("expected non-nil cmd when OnInstallPackages is called")
	}

	view := appModel.ViewString()
	if !strings.Contains(view, "Installing Plugins on my-site") {
		t.Errorf("expected view to contain progress title, got:\n%s", view)
	}
}

func TestApp_InstallPackagesExecution_ProgressAndSummary(t *testing.T) {
	steps := []tui.ProgressStep{
		{ID: "preflight", Title: "Verify WordPress Site", Status: tui.StepStatusPending},
		{ID: "resolve", Title: "Resolve Package Archives", Status: tui.StepStatusPending},
		{ID: "install", Title: "Install Plugins", Status: tui.StepStatusPending},
	}
	monitor := tui.NewProgressMonitorModel("Installing Plugins on test-site", steps)
	monitor.SetSize(80, 24)

	monitor.Update(tui.StepStartMsg{ID: "preflight", Title: "Verifying WordPress directory and configuration..."})
	monitor.Update(tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusSuccess, Detail: "Validated"})
	monitor.Update(tui.StepStartMsg{ID: "resolve", Title: "Resolving Plugins archives..."})
	monitor.Update(tui.LogLineMsg("Resolving plugin woocommerce..."))
	monitor.Update(tui.StepCompleteMsg{ID: "resolve", Status: tui.StepStatusSuccess, Detail: "1 resolved"})
	monitor.Update(tui.StepStartMsg{ID: "install", Title: "Installing Plugins via WP-CLI..."})
	monitor.Update(tui.LogLineMsg("✓ woocommerce: installed (activated)"))
	monitor.Update(tui.StepCompleteMsg{ID: "install", Status: tui.StepStatusSuccess, Detail: "1 installed"})
	monitor.Update(tui.OperationCompleteMsg{
		Title:   "Plugins Installed Successfully",
		Success: true,
		Summary: "URL:          https://test-site.test\nDirectory:    F:/sites/test-site\nPackage Type: Plugins\nInstalled:    1",
	})

	view := monitor.Render(80, 24)

	if !strings.Contains(view, "Plugins Installed Successfully") {
		t.Errorf("expected view to contain success title, got:\n%s", view)
	}
	if !strings.Contains(view, "Plugins") {
		t.Errorf("expected summary to contain package type, got:\n%s", view)
	}
	if !strings.Contains(view, "1") {
		t.Errorf("expected summary to contain installed count, got:\n%s", view)
	}
}

func TestApp_InstallPackagesExecution_PreflightValidation(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	_ = config.Save(cfgPath, cfg)

	application := app.New(app.Options{HomeDir: tempHome})
	_, _ = application.InitConfig(false)

	// Missing directory
	missingDir := filepath.Join(tempHome, "sites", "nonexistent")
	candMissing := deprovision.Candidate{Slug: "nonexistent", Path: missingDir}
	appModel := application.BuildAppModel(context.Background())
	cmd := appModel.WebsitesHub().OnInstallPackages(candMissing, packages.PackageTypePlugin, []string{"akismet"}, true)
	if cmd == nil {
		t.Fatalf("expected non-nil cmd")
	}

	// Missing wp-config.php
	siteDir := filepath.Join(tempHome, "sites", "no-config")
	_ = os.MkdirAll(siteDir, 0755)
	candNoConfig := deprovision.Candidate{Slug: "no-config", Path: siteDir}
	cmd2 := appModel.WebsitesHub().OnInstallPackages(candNoConfig, packages.PackageTypeTheme, []string{"twentytwentyfour"}, false)
	if cmd2 == nil {
		t.Fatalf("expected non-nil cmd")
	}
}
