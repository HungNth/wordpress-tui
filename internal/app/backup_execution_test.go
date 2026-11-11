package app_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
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
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
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

func TestApp_SingleBackup_RetainedArtifactAndSecretRedaction(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.DBPassword = "super-secret-db-pass"
	cfg.PackagesAPIKey = "super-secret-api-key"

	siteDir := filepath.Join(cfg.WebsitesPath, "alpha")
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(siteDir, "wp-config.php"), []byte("<?php"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}

	retainedSQL := filepath.Join(siteDir, "alpha.sql")
	conflictZip := filepath.Join(cfg.BackupPath, "full_alpha_2026-11-11_10-00-00.zip")

	application := app.New(app.Options{
		HomeDir: tempHome,
		BackupRunnerFn: func(ctx context.Context, cand deprovision.Candidate, strategy backup.BackupStrategy, backupPath string, onProgress backup.ProgressFunc) (*backup.BackupResult, error) {
			onProgress(1, 4, "Exporting database...")
			onProgress(2, 4, "Creating zip...")
			onProgress(3, 4, "Relocating...")
			return nil, &backup.ArtifactError{
				Cause:        fmt.Errorf("failed to move archive using pass=%s key=%s: %w", cfg.DBPassword, cfg.PackagesAPIKey, os.ErrExist),
				ConflictPath: conflictZip,
				Artifact: backup.RetainedArtifact{
					Kind:           backup.ArtifactSQLDump,
					Path:           retainedSQL,
					CleanupOutcome: "Temporary archive workspace removed; database dump retained (" + cfg.DBPassword + ")",
				},
			}
		},
	})
	if _, err := application.InitConfig(false); err != nil {
		t.Fatal(err)
	}

	appModel := application.BuildAppModel(t.Context())
	appModel.Update(tea.WindowSizeMsg{Width: 240, Height: 40})

	cand := deprovision.Candidate{Slug: "alpha", Path: siteDir}
	cmd := appModel.WebsitesHub().OnBackup(cand, backup.StrategyFull)
	drainModelCmds(t, appModel, cmd)

	view := appModel.ViewString()
	wideSummary := appModel.ProgressMonitor().Render(240, 40)
	if !strings.Contains(view, "Full Backup Failed") {
		t.Errorf("expected Full Backup Failed heading, got:\n%s", view)
	}
	if !strings.Contains(wideSummary, retainedSQL) {
		t.Errorf("expected retained SQL path %q in summary, got:\n%s", retainedSQL, wideSummary)
	}
	if !strings.Contains(wideSummary, conflictZip) {
		t.Errorf("expected conflict archive path %q in summary, got:\n%s", conflictZip, wideSummary)
	}
	if !strings.Contains(wideSummary, "Temporary archive workspace removed") {
		t.Errorf("expected cleanup outcome in summary, got:\n%s", wideSummary)
	}
	if strings.Contains(view, "super-secret-db-pass") || strings.Contains(view, "super-secret-api-key") || strings.Contains(wideSummary, "super-secret-db-pass") || strings.Contains(wideSummary, "super-secret-api-key") {
		t.Errorf("expected secrets to be redacted from backup failure output, got:\n%s", wideSummary)
	}
}

func drainModelCmds(t *testing.T, m *tui.AppModel, cmd tea.Cmd) {
	t.Helper()
	queue := []tea.Cmd{cmd}
	for range 256 {
		if m.ProgressMonitor() != nil && m.ProgressMonitor().IsCompleted() {
			return
		}
		if len(queue) == 0 {
			return
		}
		next := queue[0]
		queue = queue[1:]
		if next == nil {
			continue
		}
		msg := next()
		if msg == nil {
			continue
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			queue = append(queue, batch...)
			continue
		}
		switch msg.(type) {
		case tui.StepStartMsg, tui.StepCompleteMsg, tui.LogLineMsg, tui.OperationCompleteMsg, tui.ProgressResetStepsMsg:
			_, followUp := m.Update(msg)
			if followUp != nil {
				queue = append(queue, followUp)
			}
		}
	}
	t.Fatalf("exceeded command drain limit")
}

func TestApp_BatchBackup_SequentialAllSuccessAndDismissal(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"

	sitesDir := cfg.WebsitesPath
	for _, slug := range []string{"alpha", "bravo", "charlie"} {
		siteDir := filepath.Join(sitesDir, slug)
		if err := os.MkdirAll(siteDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(siteDir, "wp-config.php"), []byte("<?php"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}

	var timeline []string
	sizes := map[string]int64{
		"alpha":   1536,
		"bravo":   2048,
		"charlie": 4096,
	}

	application := app.New(app.Options{
		HomeDir: tempHome,
		BackupRunnerFn: func(ctx context.Context, cand deprovision.Candidate, strategy backup.BackupStrategy, backupPath string, onProgress backup.ProgressFunc) (*backup.BackupResult, error) {
			timeline = append(timeline, "start:"+cand.Slug)
			onProgress(1, 4, "Exporting "+cand.Slug+" database...")
			onProgress(2, 4, "Creating "+cand.Slug+" archive...")
			onProgress(3, 4, "Relocating "+cand.Slug+" archive...")
			archivePath := filepath.Join(backupPath, "full_"+cand.Slug+"_2026-11-11_10-00-00.zip")
			timeline = append(timeline, "end:"+cand.Slug)
			return &backup.BackupResult{
				FilePath: archivePath,
				FileSize: sizes[cand.Slug],
			}, nil
		},
	})
	if _, err := application.InitConfig(false); err != nil {
		t.Fatal(err)
	}

	appModel := application.BuildAppModel(t.Context())
	appModel.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Focus Websites Hub content pane -> Select All -> Enter (BatchActions) -> Enter (Backup) -> Enter (Full ZIP)
	appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	appModel.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if appModel.WebsitesHub().SelectedCount() != 3 {
		t.Fatalf("expected 3 selected websites, got %d", appModel.WebsitesHub().SelectedCount())
	}
	appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // BatchActions
	appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // Strategy chooser
	_, cmd := appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // Select Full ZIP

	var observedTitles []string
	queue := []tea.Cmd{cmd}
	for range 512 {
		if pm := appModel.ProgressMonitor(); pm != nil {
			rendered := pm.Render(240, 40)
			for _, expectedTitle := range []string{
				"Full Backup (1/3): alpha",
				"Full Backup (2/3): bravo",
				"Full Backup (3/3): charlie",
			} {
				if strings.Contains(rendered, expectedTitle) && (len(observedTitles) == 0 || observedTitles[len(observedTitles)-1] != expectedTitle) {
					observedTitles = append(observedTitles, expectedTitle)
				}
			}
			if pm.IsCompleted() {
				break
			}
		}
		if len(queue) == 0 {
			break
		}
		next := queue[0]
		queue = queue[1:]
		if next == nil {
			continue
		}
		msg := next()
		if msg == nil {
			continue
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			queue = append(queue, batch...)
			continue
		}
		switch msg.(type) {
		case tui.StepStartMsg, tui.StepCompleteMsg, tui.LogLineMsg, tui.OperationCompleteMsg, tui.ProgressResetStepsMsg:
			_, followUp := appModel.Update(msg)
			if followUp != nil {
				queue = append(queue, followUp)
			}
		}
	}

	wantTimeline := []string{"start:alpha", "end:alpha", "start:bravo", "end:bravo", "start:charlie", "end:charlie"}
	if strings.Join(timeline, ",") != strings.Join(wantTimeline, ",") {
		t.Fatalf("expected strict sequential timeline %v, got %v", wantTimeline, timeline)
	}
	wantTitles := []string{"Full Backup (1/3): alpha", "Full Backup (2/3): bravo", "Full Backup (3/3): charlie"}
	if strings.Join(observedTitles, "|") != strings.Join(wantTitles, "|") {
		t.Fatalf("expected progress titles %v, got %v", wantTitles, observedTitles)
	}

	pm := appModel.ProgressMonitor()
	if pm == nil || !pm.IsCompleted() {
		t.Fatalf("expected completed progress monitor")
	}
	wideSummary := pm.Render(240, 50)
	if !strings.Contains(wideSummary, "Batch Backup Complete") {
		t.Errorf("expected Batch Backup Complete heading, got:\n%s", wideSummary)
	}
	if strings.Count(wideSummary, "Format:") != 1 {
		t.Errorf("expected Format: displayed once in summary, got:\n%s", wideSummary)
	}
	for _, check := range []string{
		"Exporting charlie database...",
		"[✓] alpha",
		"1.50 KB",
		"1536 bytes",
		"[✓] bravo",
		"2.00 KB",
		"2048 bytes",
		"[✓] charlie",
		"4.00 KB",
		"4096 bytes",
	} {
		if !strings.Contains(wideSummary, check) {
			t.Errorf("expected %q in completed batch report, got:\n%s", check, wideSummary)
		}
	}

	// Scroll log viewport up to prove logs from earlier websites were retained
	appModel.Update(tea.WindowSizeMsg{Width: 240, Height: 60})
	fullLogView := appModel.ProgressMonitor().Render(240, 60)
	for _, logLine := range []string{"Exporting alpha database...", "Exporting bravo database...", "Exporting charlie database..."} {
		if !strings.Contains(fullLogView, logLine) {
			t.Errorf("expected retained log line %q in batch log viewport, got:\n%s", logLine, fullLogView)
		}
	}
	// Create a new website on disk before dismissing to prove dismissal refreshes discovery and clears selection
	deltaDir := filepath.Join(sitesDir, "delta")
	if err := os.MkdirAll(deltaDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deltaDir, "wp-config.php"), []byte("<?php"), 0644); err != nil {
		t.Fatal(err)
	}

	appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if appModel.IsExecuting() {
		t.Fatalf("expected progress monitor dismissed after Enter")
	}
	if appModel.WebsitesHub().State() != tui.WebsitesHubList {
		t.Fatalf("expected WebsitesHubList after dismissal, got %v", appModel.WebsitesHub().State())
	}
	if appModel.WebsitesHub().SelectedCount() != 0 {
		t.Fatalf("expected selection cleared after dismissal, got %d", appModel.WebsitesHub().SelectedCount())
	}
	hubView := appModel.WebsitesHub().Render(80, 24)
	if !strings.Contains(hubView, "delta") {
		t.Errorf("expected refreshed WebsitesHub to include newly created delta site, got:\n%s", hubView)
	}
}

func TestApp_BatchBackup_PartialFailureCollisionAndNoRetry(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.DBPassword = "batch-secret-db-pass"

	for _, slug := range []string{"alpha", "bravo", "charlie"} {
		siteDir := filepath.Join(cfg.WebsitesPath, slug)
		if err := os.MkdirAll(siteDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(siteDir, "wp-config.php"), []byte("<?php"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}

	bravoSQL := filepath.Join(cfg.WebsitesPath, "bravo", "bravo.sql")
	bravoConflict := filepath.Join(cfg.BackupPath, "full_bravo_2026-11-11_10-00-00.zip")
	var calls []string

	application := app.New(app.Options{
		HomeDir: tempHome,
		BackupRunnerFn: func(ctx context.Context, cand deprovision.Candidate, strategy backup.BackupStrategy, backupPath string, onProgress backup.ProgressFunc) (*backup.BackupResult, error) {
			calls = append(calls, cand.Slug)
			onProgress(1, 4, "Exporting "+cand.Slug+"...")
			if cand.Slug == "bravo" {
				return nil, &backup.ArtifactError{
					Cause:        fmt.Errorf("destination archive already exists (pass=%s): %w", cfg.DBPassword, os.ErrExist),
					ConflictPath: bravoConflict,
					Artifact: backup.RetainedArtifact{
						Kind:           backup.ArtifactSQLDump,
						Path:           bravoSQL,
						CleanupOutcome: "Temporary archive workspace removed; database dump retained",
					},
				}
			}
			return &backup.BackupResult{
				FilePath: filepath.Join(backupPath, "full_"+cand.Slug+"_2026-11-11_10-00-00.zip"),
				FileSize: 1024,
			}, nil
		},
	})
	if _, err := application.InitConfig(false); err != nil {
		t.Fatal(err)
	}

	appModel := application.BuildAppModel(t.Context())
	appModel.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	appModel.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	_, cmd := appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	drainModelCmds(t, appModel, cmd)

	if strings.Join(calls, ",") != "alpha,bravo,charlie" {
		t.Fatalf("expected alpha,bravo,charlie with zero retries, got %v", calls)
	}

	wideSummary := appModel.ProgressMonitor().Render(240, 50)
	if !strings.Contains(wideSummary, "Batch Backup Completed with Warnings") {
		t.Errorf("expected Batch Backup Completed with Warnings heading, got:\n%s", wideSummary)
	}
	if strings.Count(wideSummary, "Format:") != 1 {
		t.Errorf("expected single Format: line, got:\n%s", wideSummary)
	}
	if strings.Contains(wideSummary, "batch-secret-db-pass") {
		t.Errorf("expected DB password redacted in batch report, got:\n%s", wideSummary)
	}
	idxAlpha := strings.Index(wideSummary, "[✓] alpha")
	idxBravo := strings.Index(wideSummary, "[✗] bravo")
	idxCharlie := strings.Index(wideSummary, "[✓] charlie")
	if idxAlpha == -1 || idxBravo == -1 || idxCharlie == -1 || !(idxAlpha < idxBravo && idxBravo < idxCharlie) {
		t.Errorf("expected [✓] alpha < [✗] bravo < [✓] charlie in batch order, got:\n%s", wideSummary)
	}
	for _, detail := range []string{bravoConflict, bravoSQL, "Temporary archive workspace removed; database dump retained"} {
		if !strings.Contains(wideSummary, detail) {
			t.Errorf("expected %q in failed row details, got:\n%s", detail, wideSummary)
		}
	}
}

func TestApp_BatchBackup_AllFailed(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"

	// alpha has wp-config.php, bravo is missing wp-config.php (preflight failure)
	alphaDir := filepath.Join(cfg.WebsitesPath, "alpha")
	bravoDir := filepath.Join(cfg.WebsitesPath, "bravo")
	if err := os.MkdirAll(alphaDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(alphaDir, "wp-config.php"), []byte("<?php"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bravoDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}

	application := app.New(app.Options{
		HomeDir: tempHome,
		BackupRunnerFn: func(ctx context.Context, cand deprovision.Candidate, strategy backup.BackupStrategy, backupPath string, onProgress backup.ProgressFunc) (*backup.BackupResult, error) {
			return nil, errors.New("mysqldump connection refused")
		},
	})
	if _, err := application.InitConfig(false); err != nil {
		t.Fatal(err)
	}

	appModel := application.BuildAppModel(t.Context())
	cands := []deprovision.Candidate{
		{Slug: "alpha", Path: alphaDir},
		{Slug: "bravo", Path: bravoDir},
	}
	cmd := appModel.WebsitesHub().OnBatchBackup(cands, backup.StrategyFull)
	drainModelCmds(t, appModel, cmd)

	wideSummary := appModel.ProgressMonitor().Render(240, 50)
	if !strings.Contains(wideSummary, "Batch Backup Failed") {
		t.Errorf("expected Batch Backup Failed heading when 0 websites succeed, got:\n%s", wideSummary)
	}
	if !strings.Contains(wideSummary, "[✗] alpha") || !strings.Contains(wideSummary, "[✗] bravo") {
		t.Errorf("expected both alpha and bravo marked [✗], got:\n%s", wideSummary)
	}
}

func TestApp_BatchBackup_CancellationPrecedenceCleanupLockAndSkippedQueue(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"

	for _, slug := range []string{"alpha", "bravo", "charlie"} {
		siteDir := filepath.Join(cfg.WebsitesPath, slug)
		if err := os.MkdirAll(siteDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(siteDir, "wp-config.php"), []byte("<?php"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}

	bravoStarted := make(chan struct{})
	allowBravoCleanup := make(chan struct{})
	bravoRetained := filepath.Join(cfg.WebsitesPath, "bravo", "wp-content", "ai1wm-backups", "bravo.wpress")
	var calls []string

	application := app.New(app.Options{
		HomeDir: tempHome,
		BackupRunnerFn: func(ctx context.Context, cand deprovision.Candidate, strategy backup.BackupStrategy, backupPath string, onProgress backup.ProgressFunc) (*backup.BackupResult, error) {
			calls = append(calls, cand.Slug)
			onProgress(1, 4, "Working on "+cand.Slug+"...")
			if cand.Slug == "alpha" {
				return &backup.BackupResult{
					FilePath: filepath.Join(backupPath, "ai1wm_alpha_2026-11-11_10-00-00.wpress"),
					FileSize: 2048,
				}, nil
			}
			if cand.Slug == "bravo" {
				close(bravoStarted)
				<-ctx.Done()
				<-allowBravoCleanup
				return nil, &backup.ArtifactError{
					Cause: ctx.Err(),
					Artifact: backup.RetainedArtifact{
						Kind:           backup.ArtifactWpress,
						Path:           bravoRetained,
						CleanupOutcome: "Source .wpress archive retained after cancellation",
					},
				}
			}
			t.Errorf("charlie should never be started after cancellation")
			return nil, errors.New("unexpected execution")
		},
	})
	if _, err := application.InitConfig(false); err != nil {
		t.Fatal(err)
	}

	appModel := application.BuildAppModel(t.Context())
	appModel.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	appModel.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})          // BatchActions
	appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})          // Strategy
	appModel.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})        // AI1WM
	_, startCmd := appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // Start batch

	<-bravoStarted

	// Request cancellation with Ctrl+C while bravo is active
	_, quitCmd := appModel.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if quitCmd != nil {
		t.Fatalf("expected Ctrl+C during active batch backup not to quit TUI")
	}
	// Focus must remain locked while bravo cleanup is still blocked
	appModel.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !appModel.IsExecuting() {
		t.Fatalf("expected execution lock maintained while bravo cleanup is in flight")
	}

	// Allow bravo cleanup to finish and drain events until completion
	close(allowBravoCleanup)
	drainModelCmds(t, appModel, startCmd)

	if strings.Join(calls, ",") != "alpha,bravo" {
		t.Fatalf("expected only alpha and bravo started, got %v", calls)
	}

	appModel.Update(tea.WindowSizeMsg{Width: 240, Height: 60})
	wideSummary := appModel.ProgressMonitor().Render(240, 60)
	if !strings.Contains(wideSummary, "Batch Backup Cancelled") {
		t.Errorf("expected Batch Backup Cancelled heading precedence, got:\n%s", wideSummary)
	}
	idxAlpha := strings.Index(wideSummary, "[✓] alpha")
	idxBravo := strings.Index(wideSummary, "[!] bravo")
	idxCharlie := strings.Index(wideSummary, "[-] charlie")
	if idxAlpha == -1 || idxBravo == -1 || idxCharlie == -1 || !(idxAlpha < idxBravo && idxBravo < idxCharlie) {
		t.Errorf("expected [✓] alpha < [!] bravo < [-] charlie in original order, got:\n%s", wideSummary)
	}
	for _, check := range []string{
		"Working on alpha...",
		"Working on bravo...",
		bravoRetained,
		"Source .wpress archive retained after cancellation",
		"Skipped (batch cancelled)",
	} {
		if !strings.Contains(wideSummary, check) {
			t.Errorf("expected %q in cancelled batch report, got:\n%s", check, wideSummary)
		}
	}

	// Dismiss report -> selection cleared
	appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if appModel.IsExecuting() || appModel.WebsitesHub().SelectedCount() != 0 {
		t.Fatalf("expected dismissed report and cleared selection")
	}
}
