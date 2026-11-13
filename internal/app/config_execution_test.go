package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/app"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/tui"
)

func TestApp_ConfigExecution_PreflightValidation(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	if err := config.Save(cfgPath, cfg); err != nil {
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

func TestApp_ToggleDebug_MenuDispatchAndPreflight(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.WebsitesPath = filepath.Join(tempHome, "sites")
	siteDir := filepath.Join(cfg.WebsitesPath, "mysite")
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(siteDir, "wp-config.php"), []byte("<?php"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}

	application := app.New(app.Options{HomeDir: tempHome})
	if _, err := application.InitConfig(false); err != nil {
		t.Fatal(err)
	}

	appModel := application.BuildAppModel(t.Context())
	appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // focus Content Pane
	appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // open Action Menu
	for i := 0; i < len(tui.DefaultWebsiteActions) && appModel.WebsitesHub().SelectedAction() != tui.WebsiteActionToggleDebug; i++ {
		appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if appModel.WebsitesHub().SelectedAction() != tui.WebsiteActionToggleDebug {
		t.Fatalf("expected Toggle Debug Mode action in menu")
	}

	_, cmd := appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected non-nil cmd when selecting Toggle Debug Mode")
	}
	if !appModel.IsExecuting() {
		t.Fatalf("expected appModel.IsExecuting() to be true")
	}
	if !strings.Contains(appModel.ViewString(), "Toggling Debug Mode for mysite") {
		t.Errorf("expected progress monitor title for debug toggle, got:\n%s", appModel.ViewString())
	}

	// Navigation is locked while executing
	appModel.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !appModel.IsExecuting() {
		t.Errorf("expected execution lock to keep progress monitor active on Esc")
	}

	// Missing wp-config.php preflight emits failure report
	noCfgDir := filepath.Join(cfg.WebsitesPath, "no-wpconfig")
	_ = os.MkdirAll(noCfgDir, 0755)
	appModel2 := application.BuildAppModel(t.Context())
	appModel2.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	cmdMissing := appModel2.WebsitesHub().OnAction(tui.WebsiteActionToggleDebug, deprovision.Candidate{Slug: "no-wpconfig", Path: noCfgDir})
	if cmdMissing == nil {
		t.Fatalf("expected non-nil cmd for missing wp-config.php preflight")
	}
	drainModelCmds(t, appModel2, cmdMissing)
	summary := appModel2.ProgressMonitor().Render(120, 30)
	if !strings.Contains(summary, "Debug Mode Toggle Failed") || !strings.Contains(summary, "wp-config.php not found") {
		t.Errorf("expected preflight failure report for missing wp-config.php, got:\n%s", summary)
	}
}

func TestApp_ToggleDebug_ReportOutcomes(t *testing.T) {
	cases := []struct {
		name       string
		cancelKey  bool
		msg        tui.OperationCompleteMsg
		wantMarker string
		wantText   string
	}{
		{
			name: "success",
			msg: tui.OperationCompleteMsg{
				Title:   "Website Debug Mode Enabled",
				Success: true,
				Summary: "Website:   mysite\nDirectory: F:/sites/mysite\nState:     Enabled",
			},
			wantMarker: "✓ Website Debug Mode Enabled",
			wantText:   "F:/sites/mysite",
		},
		{
			name: "partial warning",
			msg: tui.OperationCompleteMsg{
				Title:   "Website Debug Mode Completed with Warnings",
				Success: false,
				Warning: true,
				Summary: "Website:   mysite\nDirectory: F:/sites/mysite\n[✗] Failed: WP_DEBUG_LOG: error",
			},
			wantMarker: "! Website Debug Mode Completed with Warnings",
			wantText:   "WP_DEBUG_LOG",
		},
		{
			name:      "cancelled",
			cancelKey: true,
			msg: tui.OperationCompleteMsg{
				Title:   "Website Debug Mode Cancelled",
				Success: false,
				Warning: true,
				Summary: "Website:   mysite\nDirectory: F:/sites/mysite\nTarget:    Enabled\nUpdated:   1 constant(s)",
			},
			wantMarker: "! Website Debug Mode Cancelled",
			wantText:   "Target:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tempHome := t.TempDir()
			sitesDir := filepath.Join(tempHome, "sites")
			siteDir := filepath.Join(sitesDir, "mysite")
			if err := os.MkdirAll(siteDir, 0755); err != nil {
				t.Fatal(err)
			}
			cfg := config.DefaultConfig(tempHome)
			cfg.AppName = config.AppNameHerd
			cfg.AppCommand = "herd"
			cfg.WebsitesPath = sitesDir

			appModel := tui.NewAppModel(cfg)
			appModel.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

			cancelled := false
			ch := make(chan tea.Msg, 4)
			appModel.SetToggleDebugHandler(func(cand deprovision.Candidate) tea.Cmd {
				steps := []tui.ProgressStep{
					{ID: "preflight", Title: "Verify WordPress Website", Status: tui.StepStatusSuccess},
					{ID: "debug", Title: "Toggle Website Debug Mode", Status: tui.StepStatusRunning},
				}
				return appModel.StartProgressWithCancel("Toggling Debug Mode for "+cand.Slug, steps, ch, func() {
					cancelled = true
				})
			})

			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // focus Websites Hub
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // open Action Menu
			for i := 0; i < len(tui.DefaultWebsiteActions) && appModel.WebsitesHub().SelectedAction() != tui.WebsiteActionToggleDebug; i++ {
				appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			}
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // execute Toggle Debug Mode
			if !appModel.IsExecuting() {
				t.Fatalf("expected appModel.IsExecuting() after triggering Toggle Debug Mode")
			}
			if tc.cancelKey {
				appModel.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
				if !cancelled {
					t.Fatalf("expected pressing q during execution to invoke cancel callback")
				}
			}

			appModel.Update(tc.msg)
			view := appModel.ViewString()
			if !strings.Contains(view, tc.wantMarker) || !strings.Contains(view, tc.wantText) {
				t.Errorf("expected marker %q and %q in view:\n%s", tc.wantMarker, tc.wantText, view)
			}

			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if appModel.IsExecuting() {
				t.Errorf("expected Enter on completed report to exit execution mode")
			}
		})
	}
}
