package tui_test

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/config"
	"wptui/internal/tui"
)

func TestAppModel_InitialLayoutAndDimensions(t *testing.T) {
	cfg := config.DefaultConfig(t.TempDir())
	m := tui.NewAppModel(cfg)

	// Dispatch initial window size (100x30 >= 80x24)
	newM, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model := newM.(*tui.AppModel)

	if model.Focus() != tui.FocusSidebar {
		t.Fatalf("expected initial focus to be FocusSidebar, got %v", model.Focus())
	}
	if model.ActiveSection() != tui.SectionWebsites {
		t.Fatalf("expected initial active section to be SectionWebsites, got %v", model.ActiveSection())
	}

	if !model.View().AltScreen {
		t.Errorf("expected model.View().AltScreen to be true")
	}

	view := model.ViewString()
	if !strings.Contains(view, "WPTUI — WordPress Local Manager") {
		t.Errorf("expected view to contain header title, got:\n%s", view)
	}
	if !strings.Contains(view, "Websites") || !strings.Contains(view, "Create") || !strings.Contains(view, "Delete") || !strings.Contains(view, "Restore") || !strings.Contains(view, "Settings") || !strings.Contains(view, "Exit") {
		t.Errorf("expected view to contain all 6 sidebar items, got:\n%s", view)
	}
}

func TestAppModel_TerminalTooSmallNotice(t *testing.T) {
	cfg := config.DefaultConfig(t.TempDir())
	m := tui.NewAppModel(cfg)

	// 1. Width too small
	newM, _ := m.Update(tea.WindowSizeMsg{Width: 70, Height: 30})
	view := newM.(*tui.AppModel).ViewString()
	if !strings.Contains(view, "Terminal window too small (minimum 80x24 required)") {
		t.Errorf("expected terminal too small warning for width=70, got:\n%s", view)
	}

	// 2. Height too small
	newM, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	view = newM.(*tui.AppModel).ViewString()
	if !strings.Contains(view, "Terminal window too small (minimum 80x24 required)") {
		t.Errorf("expected terminal too small warning for height=20, got:\n%s", view)
	}

	// 3. Normal size restores view
	newM, _ = m.Update(tea.WindowSizeMsg{Width: 85, Height: 26})
	view = newM.(*tui.AppModel).ViewString()
	if strings.Contains(view, "Terminal window too small") {
		t.Errorf("expected normal view when dimensions >= 80x24, got:\n%s", view)
	}
}

func TestAppModel_SidebarNavigationAndFocusSwitching(t *testing.T) {
	cfg := config.DefaultConfig(t.TempDir())
	m := tui.NewAppModel(cfg)
	newM, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model := newM.(*tui.AppModel)

	// Navigate down with "j" -> SectionCreate
	newM, _ = model.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	model = newM.(*tui.AppModel)
	if model.ActiveSection() != tui.SectionCreate {
		t.Errorf("expected active section to be SectionCreate after 'j', got %v", model.ActiveSection())
	}

	// Navigate down with "down" -> SectionDelete
	newM, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model = newM.(*tui.AppModel)
	if model.ActiveSection() != tui.SectionDelete {
		t.Errorf("expected active section to be SectionDelete after 'down', got %v", model.ActiveSection())
	}

	// Navigate down with "down" -> SectionRestore
	newM, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model = newM.(*tui.AppModel)
	if model.ActiveSection() != tui.SectionRestore {
		t.Errorf("expected active section to be SectionRestore after second 'down', got %v", model.ActiveSection())
	}

	// Navigate up with "k" -> SectionDelete
	newM, _ = model.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	model = newM.(*tui.AppModel)
	if model.ActiveSection() != tui.SectionDelete {
		t.Errorf("expected active section to be SectionDelete after 'k', got %v", model.ActiveSection())
	}

	// Navigate up with "k" -> SectionCreate
	newM, _ = model.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	model = newM.(*tui.AppModel)
	if model.ActiveSection() != tui.SectionCreate {
		t.Errorf("expected active section to be SectionCreate after second 'k', got %v", model.ActiveSection())
	}

	// Press Enter to focus Content Pane
	newM, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = newM.(*tui.AppModel)
	if model.Focus() != tui.FocusContent {
		t.Fatalf("expected focus to switch to FocusContent after Enter, got %v", model.Focus())
	}

	view := model.ViewString()
	if !strings.Contains(view, "Esc") {
		t.Errorf("expected footer to show Esc in FocusContent, got:\n%s", view)
	}

	// Press Esc to return to Sidebar
	newM, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	model = newM.(*tui.AppModel)
	if model.Focus() != tui.FocusSidebar {
		t.Fatalf("expected focus to return to FocusSidebar after Esc, got %v", model.Focus())
	}

	// Press Tab to toggle to Content Pane
	newM, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	model = newM.(*tui.AppModel)
	if model.Focus() != tui.FocusContent {
		t.Fatalf("expected focus to switch to FocusContent after Tab, got %v", model.Focus())
	}

	// Press Shift+Tab to toggle back to Sidebar
	newM, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	model = newM.(*tui.AppModel)
	if model.Focus() != tui.FocusSidebar {
		t.Fatalf("expected focus to switch to FocusSidebar after Shift+Tab, got %v", model.Focus())
	}
}

func TestAppModel_QuitSignals(t *testing.T) {
	cfg := config.DefaultConfig(t.TempDir())

	// 1. 'q' in FocusSidebar quits
	m := tui.NewAppModel(cfg)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Errorf("expected non-nil cmd (tea.Quit) on 'q'")
	}

	// 2. 'ctrl+c' in FocusSidebar quits
	m2 := tui.NewAppModel(cfg)
	m2.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	_, cmd2 := m2.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd2 == nil {
		t.Errorf("expected non-nil cmd (tea.Quit) on 'ctrl+c'")
	}

	// 3. Selecting Exit item and pressing Enter quits
	m3 := tui.NewAppModel(cfg)
	m3.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	// Navigate to Exit (5 down steps)
	for range 5 {
		newM, _ := m3.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m3 = newM.(*tui.AppModel)
	}
	if m3.ActiveSection() != tui.SectionExit {
		t.Fatalf("expected active section SectionExit, got %v", m3.ActiveSection())
	}
	_, cmd3 := m3.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd3 == nil {
		t.Errorf("expected non-nil cmd (tea.Quit) when pressing Enter on Exit")
	}
}

func TestAppModel_ExecutionLockAndProgress(t *testing.T) {
	cfg := config.DefaultConfig(t.TempDir())
	m := tui.NewAppModel(cfg)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Start a background task progress
	steps := []tui.ProgressStep{
		{ID: "step1", Title: "Download Core"},
	}
	m.StartProgress("Creating Website", steps)

	if !m.IsExecuting() {
		t.Fatalf("expected m.IsExecuting() to be true")
	}

	// Try navigating with "j" - sidebar should be locked, active section stays Websites
	newM, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	model := newM.(*tui.AppModel)
	if model.ActiveSection() != tui.SectionWebsites {
		t.Errorf("expected sidebar navigation to be locked during execution")
	}

	// View should render progress monitor
	view := model.ViewString()
	if !strings.Contains(view, "Creating Website") || !strings.Contains(view, "Download Core") {
		t.Errorf("expected view to render progress stepper, got:\n%s", view)
	}

	// Complete task
	newM, _ = model.Update(tui.OperationCompleteMsg{
		Title:   "Done",
		Success: true,
	})
	model = newM.(*tui.AppModel)

	// Press Enter to finish
	newM, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = newM.(*tui.AppModel)

	if model.IsExecuting() {
		t.Errorf("expected execution to end after Enter")
	}
}

func TestAppModel_VisualAlignment(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	siteDir := tempDir + "/sites/flatsome"
	cfg.WebsitesPath = tempDir + "/sites"
	_ = os.MkdirAll(siteDir, 0755)

	m := tui.NewAppModel(cfg)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	// Enter content
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	// Enter action menu for flatsome
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	view := m.ViewString()
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "Name/Slug: flatsome") {
			spaces := contentLeadingSpaces(line)
			if spaces > 6 {
				t.Errorf("Websites action Name/Slug has excessive leading spaces (%d): %q", spaces, line)
			}
		}
		if strings.Contains(line, "Config") && strings.Contains(line, ">") {
			spaces := contentLeadingSpaces(line)
			if spaces > 6 {
				t.Errorf("Websites action Config has excessive leading spaces (%d): %q", spaces, line)
			}
		}
	}

	// 2. Test Create Wizard Alignment
	m2 := tui.NewAppModel(cfg)
	m2.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m2.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Go to Create
	m2.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // Focus Content

	createView := m2.ViewString()
	for _, line := range strings.Split(createView, "\n") {
		if strings.Contains(line, "Website Name") {
			spaces := contentLeadingSpaces(line)
			if spaces > 6 {
				t.Errorf("Create wizard Website Name has excessive leading spaces (%d): %q", spaces, line)
			}
		}
	}

	// 3. Test Delete Alignment
	mDelete := tui.NewAppModel(cfg)
	mDelete.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	mDelete.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Create
	mDelete.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Delete
	mDelete.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // Focus Content

	deleteView := mDelete.ViewString()
	for _, line := range strings.Split(deleteView, "\n") {
		if strings.Contains(line, "Batch Website De-provisioning") {
			spaces := contentLeadingSpaces(line)
			if spaces > 6 {
				t.Errorf("Delete view title has excessive leading spaces (%d): %q", spaces, line)
			}
		}
	}

	// 4. Test Restore Wizard Alignment
	m3 := tui.NewAppModel(cfg)
	m3.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m3.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Create
	m3.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Delete
	m3.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Restore
	m3.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // Focus Content

	restoreView := m3.ViewString()
	for _, line := range strings.Split(restoreView, "\n") {
		if strings.Contains(line, "Full ZIP") {
			spaces := contentLeadingSpaces(line)
			if spaces > 6 {
				t.Errorf("Restore wizard format option has excessive leading spaces (%d): %q", spaces, line)
			}
		}
	}

	// 5. Test Settings Alignment
	m4 := tui.NewAppModel(cfg)
	m4.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m4.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Create
	m4.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Delete
	m4.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Restore
	m4.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Settings
	m4.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // Focus Content

	settingsView := m4.ViewString()
	for _, line := range strings.Split(settingsView, "\n") {
		if strings.Contains(line, "VS Code") {
			spaces := contentLeadingSpaces(line)
			if spaces > 6 {
				t.Errorf("Settings option has excessive leading spaces (%d): %q", spaces, line)
			}
		}
	}
}

func contentLeadingSpaces(line string) int {
	parts := strings.Split(line, "│")
	if len(parts) < 3 {
		return 0
	}
	content := parts[2]
	var plain strings.Builder
	inEscape := false
	for _, r := range content {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		plain.WriteRune(r)
	}
	s := plain.String()
	trimmed := strings.TrimLeft(s, " ")
	return len(s) - len(trimmed)
}
