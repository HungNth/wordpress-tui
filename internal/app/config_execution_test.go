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
	"wptui/internal/tui"
)

func TestApp_ConfigExecution_PreflightValidation(t *testing.T) {
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

	// 1. Non-existent directory
	candMissingDir := deprovision.Candidate{
		Slug: "nonexistent",
		Path: filepath.Join(tempHome, "sites", "nonexistent"),
	}

	appModel := application.BuildAppModel(context.Background())
	cmd := appModel.WebsitesHub().OnAction(tui.WebsiteActionConfig, candMissingDir)
	if cmd == nil {
		t.Fatalf("expected non-nil cmd for Config action")
	}

	// 2. Directory exists but wp-config.php is missing
	siteDir := filepath.Join(tempHome, "sites", "missing-config")
	_ = os.MkdirAll(siteDir, 0755)
	candMissingConfig := deprovision.Candidate{
		Slug: "missing-config",
		Path: siteDir,
	}

	cmd2 := appModel.WebsitesHub().OnAction(tui.WebsiteActionConfig, candMissingConfig)
	if cmd2 == nil {
		t.Fatalf("expected non-nil cmd for Config action on site without wp-config.php")
	}

	// 3. Valid directory with wp-config.php and empty tweaks
	validSiteDir := filepath.Join(tempHome, "sites", "valid-site")
	_ = os.MkdirAll(validSiteDir, 0755)
	_ = os.WriteFile(filepath.Join(validSiteDir, "wp-config.php"), []byte("<?php"), 0644)
	candValid := deprovision.Candidate{
		Slug: "valid-site",
		Path: validSiteDir,
	}

	cmd3 := appModel.WebsitesHub().OnAction(tui.WebsiteActionConfig, candValid)
	if cmd3 == nil {
		t.Fatalf("expected non-nil cmd for valid site config")
	}

	view := appModel.ViewString()
	if !strings.Contains(view, "Configuring valid-site") {
		t.Errorf("expected view to display progress monitor for configuring valid-site, got:\n%s", view)
	}
}

func TestApp_ConfigExecution_ProgressAndSummary(t *testing.T) {
	steps := []tui.ProgressStep{
		{ID: "preflight", Title: "Verify WordPress Site", Status: tui.StepStatusPending},
		{ID: "tweaks", Title: "Apply WordPress Tweaks", Status: tui.StepStatusPending},
	}
	monitor := tui.NewProgressMonitorModel("Configuring test-site", steps)
	monitor.SetSize(80, 24)

	// Simulate messages emitted by runConfigExecution
	monitor.Update(tui.StepStartMsg{ID: "preflight", Title: "Verifying WordPress site..."})
	monitor.Update(tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusSuccess, Detail: "Validated"})
	monitor.Update(tui.StepStartMsg{ID: "tweaks", Title: "Applying WordPress tweaks..."})
	monitor.Update(tui.LogLineMsg("Applying tweak: [option_update] blogdescription=Testing"))
	monitor.Update(tui.StepCompleteMsg{ID: "tweaks", Status: tui.StepStatusSuccess, Detail: "Complete"})
	monitor.Update(tui.OperationCompleteMsg{
		Title:   "Website Configured Successfully",
		Success: true,
		Summary: "URL:          https://test-site.test\nDirectory:    F:/sites/test-site\nTweaks:       1 applied",
	})

	view := monitor.Render(80, 24)

	if !strings.Contains(view, "[✓]") || !strings.Contains(view, "Verifying WordPress site...") {
		t.Errorf("expected preflight step to be completed [✓], got:\n%s", view)
	}
	if !strings.Contains(view, "[✓]") || !strings.Contains(view, "Applying WordPress tweaks...") {
		t.Errorf("expected tweaks step to be completed [✓], got:\n%s", view)
	}
	if !strings.Contains(view, "Website Configured Successfully") {
		t.Errorf("expected completion title in view, got:\n%s", view)
	}
	if !strings.Contains(view, "https://test-site.test") {
		t.Errorf("expected URL in summary, got:\n%s", view)
	}
	if !strings.Contains(view, "1 applied") {
		t.Errorf("expected tweaks count in summary, got:\n%s", view)
	}
}
