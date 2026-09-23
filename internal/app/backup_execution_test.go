package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wptui/internal/app"
	"wptui/internal/backup"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/tui"
)

func TestApp_BackupExecution_Wiring(t *testing.T) {
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
	if appModel.WebsitesHub().OnBackup == nil {
		t.Fatalf("expected OnBackup callback to be wired on WebsitesHub")
	}

	cand := deprovision.Candidate{
		Slug: "backup-site",
		Path: filepath.Join(tempHome, "sites", "backup-site"),
	}

	// Test StrategyFull wiring
	cmd := appModel.WebsitesHub().OnBackup(cand, backup.StrategyFull)
	if cmd == nil {
		t.Fatalf("expected non-nil cmd when OnBackup(StrategyFull) is called")
	}
	view := appModel.ViewString()
	if !strings.Contains(view, "Full Backup for backup-site") {
		t.Errorf("expected view to contain Full backup progress title, got:\n%s", view)
	}

	// Reset and test StrategyAI1WM wiring
	appModel2 := application.BuildAppModel(context.Background())
	cmd2 := appModel2.WebsitesHub().OnBackup(cand, backup.StrategyAI1WM)
	if cmd2 == nil {
		t.Fatalf("expected non-nil cmd when OnBackup(StrategyAI1WM) is called")
	}
	view2 := appModel2.ViewString()
	if !strings.Contains(view2, "AI1WM Backup for backup-site") {
		t.Errorf("expected view to contain AI1WM backup progress title, got:\n%s", view2)
	}
}

func TestApp_BackupExecution_Full_ProgressAndSummary(t *testing.T) {
	steps := []tui.ProgressStep{
		{ID: "preflight", Title: "Verify WordPress Site", Status: tui.StepStatusPending},
		{ID: "db_export", Title: "Export Database", Status: tui.StepStatusPending},
		{ID: "archive", Title: "Create Backup Archive", Status: tui.StepStatusPending},
		{ID: "cleanup", Title: "Cleanup Temporary Files", Status: tui.StepStatusPending},
	}
	monitor := tui.NewProgressMonitorModel("Backing up test-site (Full)", steps)
	monitor.SetSize(80, 24)

	monitor.Update(tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusSuccess, Detail: "Validated"})
	monitor.Update(tui.StepCompleteMsg{ID: "db_export", Status: tui.StepStatusSuccess, Detail: "Exported"})
	monitor.Update(tui.StepCompleteMsg{ID: "archive", Status: tui.StepStatusSuccess, Detail: "Archived"})
	monitor.Update(tui.StepCompleteMsg{ID: "cleanup", Status: tui.StepStatusSuccess, Detail: "Cleaned"})

	monitor.Update(tui.OperationCompleteMsg{
		Title:   "Full Backup Created Successfully",
		Success: true,
		Summary: "URL:          https://test-site.test\nDirectory:    F:/sites/test-site\nFormat:       Full Source Code & Database (.zip)\nArchive:      F:/backups/full_test-site.zip\nSize:         1.2 MB (1258291 bytes)",
	})

	view := monitor.Render(80, 24)

	if !strings.Contains(view, "[✓]") || !strings.Contains(view, "Verify WordPress Site") {
		t.Errorf("expected preflight step completed, got:\n%s", view)
	}
	if !strings.Contains(view, "Export Database") || !strings.Contains(view, "Create Backup Archive") {
		t.Errorf("expected export and archive steps, got:\n%s", view)
	}
	if !strings.Contains(view, "Full Backup Created Successfully") {
		t.Errorf("expected completion title, got:\n%s", view)
	}
	if !strings.Contains(view, "Full Source Code & Database (.zip)") {
		t.Errorf("expected format in summary, got:\n%s", view)
	}
}

func TestApp_BackupExecution_AI1WM_ProgressAndSummary(t *testing.T) {
	steps := []tui.ProgressStep{
		{ID: "preflight", Title: "Verify WordPress Site", Status: tui.StepStatusPending},
		{ID: "extension", Title: "Verify AI1WM Extension", Status: tui.StepStatusPending},
		{ID: "ai1wm_backup", Title: "Generate AI1WM Backup", Status: tui.StepStatusPending},
		{ID: "relocate", Title: "Relocate Archive", Status: tui.StepStatusPending},
	}
	monitor := tui.NewProgressMonitorModel("Backing up test-site (AI1WM)", steps)
	monitor.SetSize(80, 24)

	monitor.Update(tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusSuccess, Detail: "Validated"})
	monitor.Update(tui.StepCompleteMsg{ID: "extension", Status: tui.StepStatusSuccess, Detail: "Verified"})
	monitor.Update(tui.StepCompleteMsg{ID: "ai1wm_backup", Status: tui.StepStatusSuccess, Detail: "Generated"})
	monitor.Update(tui.StepCompleteMsg{ID: "relocate", Status: tui.StepStatusSuccess, Detail: "Relocated"})

	monitor.Update(tui.OperationCompleteMsg{
		Title:   "AI1WM Backup Created Successfully",
		Success: true,
		Summary: "URL:          https://test-site.test\nDirectory:    F:/sites/test-site\nFormat:       All-in-One WP Migration (.wpress)\nArchive:      F:/backups/ai1wm_test-site.wpress\nSize:         2.4 MB (2516582 bytes)",
	})

	view := monitor.Render(80, 24)

	if !strings.Contains(view, "[✓]") || !strings.Contains(view, "Verify AI1WM Extension") {
		t.Errorf("expected extension step completed, got:\n%s", view)
	}
	if !strings.Contains(view, "Generate AI1WM Backup") || !strings.Contains(view, "Relocate Archive") {
		t.Errorf("expected ai1wm steps in view, got:\n%s", view)
	}
	if !strings.Contains(view, "AI1WM Backup Created Successfully") {
		t.Errorf("expected completion title, got:\n%s", view)
	}
	if !strings.Contains(view, "All-in-One WP Migration (.wpress)") {
		t.Errorf("expected format in summary, got:\n%s", view)
	}
}

func TestApp_BackupExecution_PreflightValidation(t *testing.T) {
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

	// 1. Non-existent directory
	candMissingDir := deprovision.Candidate{
		Slug: "nonexistent",
		Path: filepath.Join(tempHome, "sites", "nonexistent"),
	}

	cmd1 := appModel.WebsitesHub().OnBackup(candMissingDir, backup.StrategyFull)
	if cmd1 == nil {
		t.Fatalf("expected non-nil cmd for Full backup on missing dir")
	}

	// 2. Directory exists but wp-config.php is missing
	siteDir := filepath.Join(tempHome, "sites", "missing-config")
	_ = os.MkdirAll(siteDir, 0755)
	candMissingConfig := deprovision.Candidate{
		Slug: "missing-config",
		Path: siteDir,
	}

	cmd2 := appModel.WebsitesHub().OnBackup(candMissingConfig, backup.StrategyAI1WM)
	if cmd2 == nil {
		t.Fatalf("expected non-nil cmd for AI1WM backup on site without wp-config.php")
	}
}
