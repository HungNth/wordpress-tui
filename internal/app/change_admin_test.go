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

func TestApp_ChangeAdminExecution_Wiring(t *testing.T) {
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
	if appModel.WebsitesHub().OnChangeAdmin == nil {
		t.Fatalf("expected OnChangeAdmin callback to be wired on WebsitesHub")
	}

	cand := deprovision.Candidate{
		Slug: "my-site",
		Path: filepath.Join(tempHome, "sites", "my-site"),
	}

	cmd := appModel.WebsitesHub().OnChangeAdmin(cand, "newadmin", "newpass123", "newadmin@example.com")
	if cmd == nil {
		t.Fatalf("expected non-nil cmd when OnChangeAdmin is called")
	}

	view := appModel.ViewString()
	if !strings.Contains(view, "Updating credentials for my-site") {
		t.Errorf("expected view to contain progress title, got:\n%s", view)
	}
}

func TestApp_ChangeAdminExecution_ProgressAndSummary(t *testing.T) {
	steps := []tui.ProgressStep{
		{ID: "preflight", Title: "Verify WordPress Site", Status: tui.StepStatusPending},
		{ID: "discover", Title: "Discover Administrator", Status: tui.StepStatusPending},
		{ID: "dbconfig", Title: "Read Database Configuration", Status: tui.StepStatusPending},
		{ID: "update", Title: "Update Administrator Credentials", Status: tui.StepStatusPending},
	}
	monitor := tui.NewProgressMonitorModel("Updating credentials for test-site", steps)
	monitor.SetSize(80, 24)

	monitor.Update(tui.StepStartMsg{ID: "preflight", Title: "Verifying WordPress directory and configuration..."})
	monitor.Update(tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusSuccess, Detail: "Validated"})
	monitor.Update(tui.StepStartMsg{ID: "discover", Title: "Discovering administrator account..."})
	monitor.Update(tui.StepCompleteMsg{ID: "discover", Status: tui.StepStatusSuccess, Detail: "Admin ID 1 (admin)"})
	monitor.Update(tui.StepStartMsg{ID: "dbconfig", Title: "Reading database configuration..."})
	monitor.Update(tui.StepCompleteMsg{ID: "dbconfig", Status: tui.StepStatusSuccess, Detail: "Database: test_db"})
	monitor.Update(tui.StepStartMsg{ID: "update", Title: "Updating administrator credentials..."})
	monitor.Update(tui.LogLineMsg("Updating username to \"newadmin\" in database..."))
	monitor.Update(tui.LogLineMsg("Updating administrator password via WP-CLI..."))
	monitor.Update(tui.LogLineMsg("Updating email and site admin_email to \"newadmin@example.com\"..."))
	monitor.Update(tui.StepCompleteMsg{ID: "update", Status: tui.StepStatusSuccess, Detail: "Updated successfully"})
	monitor.Update(tui.OperationCompleteMsg{
		Title:   "Admin Credentials Updated Successfully",
		Success: true,
		Summary: "URL:          https://test-site.test\nDirectory:    F:/sites/test-site\nAdmin User:   newadmin (ID: 1)\nAdmin Email:  newadmin@example.com\nPassword:     (updated successfully)",
	})

	view := monitor.Render(80, 24)

	if !strings.Contains(view, "Admin Credentials Updated Successfully") {
		t.Errorf("expected view to contain success title, got:\n%s", view)
	}
	if !strings.Contains(view, "newadmin (ID: 1)") {
		t.Errorf("expected summary to contain newadmin, got:\n%s", view)
	}
	if !strings.Contains(view, "(updated successfully)") {
		t.Errorf("expected summary to show password masked/safe, got:\n%s", view)
	}
}

func TestApp_ChangeAdminExecution_PreflightValidation(t *testing.T) {
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
	cmd := appModel.WebsitesHub().OnChangeAdmin(candMissing, "admin", "pass", "admin@test.com")
	if cmd == nil {
		t.Fatalf("expected non-nil cmd")
	}

	// Missing wp-config.php
	siteDir := filepath.Join(tempHome, "sites", "no-config")
	_ = os.MkdirAll(siteDir, 0755)
	candNoConfig := deprovision.Candidate{Slug: "no-config", Path: siteDir}
	cmd2 := appModel.WebsitesHub().OnChangeAdmin(candNoConfig, "admin", "pass", "admin@test.com")
	if cmd2 == nil {
		t.Fatalf("expected non-nil cmd")
	}
}
