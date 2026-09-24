package tui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/config"
	"wptui/internal/restore"
	"wptui/internal/tui"
)

func TestRestoreWizard_MultiStepFlow(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.BackupPath = tempDir

	// Create a dummy backup archive
	archivePath := filepath.Join(tempDir, "site1_2026-09-20.zip")
	if err := os.WriteFile(archivePath, []byte("PK..."), 0644); err != nil {
		t.Fatal(err)
	}

	wizard := tui.NewRestoreWizardModel(cfg, nil)

	// Step 1: Format selection (Full ZIP vs AI1WM)
	if wizard.Step() != tui.RestoreWizardStepFormat {
		t.Fatalf("expected step RestoreWizardStepFormat, got %v", wizard.Step())
	}
	view := wizard.Render(70, 20)
	if !strings.Contains(view, "Full source code & database (.zip)") {
		t.Errorf("expected view to contain format options, got:\n%s", view)
	}

	// Press Enter to select Full ZIP and advance to Step 2
	wizard.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if wizard.Step() != tui.RestoreWizardStepArchive {
		t.Fatalf("expected step RestoreWizardStepArchive, got %v", wizard.Step())
	}
	if wizard.Strategy() != restore.StrategyFullZIP {
		t.Errorf("expected strategy FullZIP, got %v", wizard.Strategy())
	}

	// Step 2: Archive selection
	view2 := wizard.Render(70, 20)
	if !strings.Contains(view2, "site1_2026-09-20.zip") {
		t.Errorf("expected archive list to show site1_2026-09-20.zip, got:\n%s", view2)
	}

	// Press Enter on the first archive to select it and advance to Step 3
	wizard.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if wizard.Step() != tui.RestoreWizardStepInputs {
		t.Fatalf("expected step RestoreWizardStepInputs, got %v", wizard.Step())
	}
	if wizard.SelectedArchive() != archivePath {
		t.Errorf("expected selected archive %s, got %s", archivePath, wizard.SelectedArchive())
	}

	// Step 3: Enter website name
	for _, ch := range "Restored Site" {
		wizard.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
	}

	// Navigate to Submit button and press Enter
	for range 5 {
		wizard.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	}
	wizard.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if !wizard.IsSubmitted() {
		t.Fatalf("expected wizard to be submitted")
	}
	if wizard.Inputs().WebsiteSlug != "restored-site" {
		t.Errorf("expected slug 'restored-site', got %s", wizard.Inputs().WebsiteSlug)
	}

	// Esc in step 2 returns to step 1
	wizard2 := tui.NewRestoreWizardModel(cfg, nil)
	wizard2.Update(tea.KeyPressMsg{Code: tea.KeyEnter})  // to step 2
	wizard2.Update(tea.KeyPressMsg{Code: tea.KeyEscape}) // back to step 1
	if wizard2.Step() != tui.RestoreWizardStepFormat {
		t.Errorf("expected return to Step 1 on Esc")
	}
}

func TestRestoreWizard_ArchiveScrolling(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.BackupPath = tempDir

	for i := 1; i <= 25; i++ {
		name := filepath.Join(tempDir, fmt.Sprintf("backup%02d.zip", i))
		_ = os.WriteFile(name, []byte("PK..."), 0644)
	}

	wizard := tui.NewRestoreWizardModel(cfg, nil)
	// Go to step 2 (archives)
	wizard.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Navigate down 18 times to reach backup19.zip
	for i := 0; i < 18; i++ {
		wizard.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	}

	viewScrolled := wizard.Render(70, 20)
	if !strings.Contains(viewScrolled, "backup19.zip") {
		t.Errorf("expected scrolled view to contain backup19.zip, but got:\n%s", viewScrolled)
	}
	if strings.Contains(viewScrolled, "backup01.zip") {
		t.Errorf("expected backup01.zip to have scrolled out of view, got:\n%s", viewScrolled)
	}
}

