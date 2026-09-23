package tui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/tui"
)

func setupTestWebsites(t *testing.T) (*config.Config, []string) {
	t.Helper()
	tmpDir := t.TempDir()
	sitesDir := filepath.Join(tmpDir, "sites")
	if err := os.MkdirAll(sitesDir, 0755); err != nil {
		t.Fatalf("failed to create sites dir: %v", err)
	}

	sites := []string{"alpha-site", "beta-site", "gamma-site"}
	for _, s := range sites {
		sDir := filepath.Join(sitesDir, s)
		if err := os.MkdirAll(sDir, 0755); err != nil {
			t.Fatalf("failed to create site dir: %v", err)
		}
		_ = os.WriteFile(filepath.Join(sDir, "wp-config.php"), []byte("<?php"), 0644)
	}

	cfg := config.DefaultConfig(tmpDir)
	cfg.WebsitesPath = sitesDir
	return cfg, sites
}

func TestDeleteModel_InitialListAndNavigation(t *testing.T) {
	cfg, sites := setupTestWebsites(t)
	m := tui.NewDeleteModel(cfg)

	if len(m.Candidates()) != len(sites) {
		t.Fatalf("expected %d candidates, got %d", len(sites), len(m.Candidates()))
	}
	if m.Cursor() != 0 {
		t.Errorf("expected initial cursor 0, got %d", m.Cursor())
	}
	if m.State() != tui.DeleteModelList {
		t.Errorf("expected initial state DeleteModelList, got %v", m.State())
	}

	// Navigate down with "j"
	newM, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	m = newM.(*tui.DeleteModel)
	if m.Cursor() != 1 {
		t.Errorf("expected cursor 1 after 'j', got %d", m.Cursor())
	}

	// Navigate down with "down"
	newM, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = newM.(*tui.DeleteModel)
	if m.Cursor() != 2 {
		t.Errorf("expected cursor 2 after 'down', got %d", m.Cursor())
	}

	// Navigate up with "k"
	newM, _ = m.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	m = newM.(*tui.DeleteModel)
	if m.Cursor() != 1 {
		t.Errorf("expected cursor 1 after 'k', got %d", m.Cursor())
	}
}

func TestDeleteModel_SelectionAndSelectAllToggle(t *testing.T) {
	cfg, _ := setupTestWebsites(t)
	m := tui.NewDeleteModel(cfg)

	// Toggle selection on item 0 with Space
	newM, _ := m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	m = newM.(*tui.DeleteModel)
	if m.SelectedCount() != 1 {
		t.Fatalf("expected 1 selected candidate, got %d", m.SelectedCount())
	}
	cand0 := m.Candidates()[0].Slug
	if !m.IsSelected(cand0) {
		t.Errorf("expected %s to be selected", cand0)
	}

	// Press 'a' should select all (since not all were selected)
	newM, _ = m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m = newM.(*tui.DeleteModel)
	if m.SelectedCount() != 3 {
		t.Fatalf("expected all 3 selected after 'a', got %d", m.SelectedCount())
	}

	// Press 'a' again should deselect all (since all were selected)
	newM, _ = m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	m = newM.(*tui.DeleteModel)
	if m.SelectedCount() != 0 {
		t.Fatalf("expected 0 selected after second 'a', got %d", m.SelectedCount())
	}
}

func TestDeleteModel_EnterWithoutSelectionShowsWarning(t *testing.T) {
	cfg, _ := setupTestWebsites(t)
	m := tui.NewDeleteModel(cfg)

	// Enter with 0 selected
	newM, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newM.(*tui.DeleteModel)

	if m.State() != tui.DeleteModelList {
		t.Fatalf("expected to stay in DeleteModelList when 0 selected, got %v", m.State())
	}

	view := m.Render(80, 20)
	if !strings.Contains(view, "No websites selected") {
		t.Errorf("expected warning message in view, got:\n%s", view)
	}
}

func TestDeleteModel_ConfirmationAndExecution(t *testing.T) {
	cfg, _ := setupTestWebsites(t)
	m := tui.NewDeleteModel(cfg)

	// Select first two candidates
	m.Update(tea.KeyPressMsg{Code: ' ', Text: " "}) // cand 0
	m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"}) // down
	m.Update(tea.KeyPressMsg{Code: ' ', Text: " "}) // cand 1

	if m.SelectedCount() != 2 {
		t.Fatalf("expected 2 selected candidates, got %d", m.SelectedCount())
	}

	// Press Enter to go to confirmation screen
	newM, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newM.(*tui.DeleteModel)

	if m.State() != tui.DeleteModelConfirm {
		t.Fatalf("expected state DeleteModelConfirm, got %v", m.State())
	}

	// Default choice must be false ([ No, Cancel ])
	if m.ConfirmChoice() != false {
		t.Errorf("expected default confirmChoice to be false, got true")
	}

	view := m.Render(80, 20)
	if !strings.Contains(view, "⚠ Delete Selected Websites?") {
		t.Errorf("expected warning header in confirm view, got:\n%s", view)
	}
	if !strings.Contains(view, "[ Yes, Delete All ]") || !strings.Contains(view, "[ No, Cancel ]") {
		t.Errorf("expected confirmation buttons in confirm view, got:\n%s", view)
	}

	// Press Enter while No is selected -> should return to DeleteModelList without deleting
	var deleted []deprovision.Candidate
	m.OnDelete = func(cands []deprovision.Candidate) tea.Cmd {
		deleted = cands
		return nil
	}

	newM, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newM.(*tui.DeleteModel)
	if m.State() != tui.DeleteModelList {
		t.Errorf("expected to return to DeleteModelList on No, got %v", m.State())
	}
	if len(deleted) != 0 {
		t.Errorf("expected OnDelete not to be called when No, but was called with %d items", len(deleted))
	}

	// Return to confirm screen
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.State() != tui.DeleteModelConfirm {
		t.Fatalf("expected state DeleteModelConfirm, got %v", m.State())
	}

	// Toggle choice to Yes using Right arrow
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m.ConfirmChoice() != true {
		t.Errorf("expected confirmChoice to be true after KeyRight, got false")
	}

	// Press Enter to confirm deletion
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(deleted) != 2 {
		t.Fatalf("expected OnDelete to be called with 2 candidates, got %d", len(deleted))
	}
}
