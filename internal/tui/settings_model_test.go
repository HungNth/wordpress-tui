package tui_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/config"
	"wptui/internal/tui"
)

type mockProcessRunner struct {
	runs []string
}

func (m *mockProcessRunner) LookPath(file string) (string, error) {
	return "/bin/" + file, nil
}

func (m *mockProcessRunner) Run(ctx context.Context, name string, args ...string) error {
	m.runs = append(m.runs, name+" "+strings.Join(args, " "))
	return nil
}

func (m *mockProcessRunner) Start(ctx context.Context, name string, args ...string) error {
	m.runs = append(m.runs, "start:"+name+" "+strings.Join(args, " "))
	return nil
}

func TestSettingsModel_NavigationAndActions(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfgPath := filepath.Join(tempDir, "config.json")
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}

	// Create cache directory so Open Cache succeeds
	cacheDir := filepath.Join(tempDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	runner := &mockProcessRunner{}
	model := tui.NewSettingsModel(cfg, cfgPath, runner, cacheDir)

	view := model.Render(70, 20)
	if !strings.Contains(view, "Open config.json in VS Code") || !strings.Contains(view, "Open Cache Directory") {
		t.Errorf("expected view to contain settings options, got:\n%s", view)
	}

	// 1. Enter on option 0: VS Code
	model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(runner.runs) != 1 || !strings.Contains(runner.runs[0], "code") {
		t.Errorf("expected VS Code launch, got runs: %v", runner.runs)
	}
	view = model.Render(70, 20)
	if !strings.Contains(view, "config.json in VS Code") {
		t.Errorf("expected success status message, got:\n%s", view)
	}

	// 2. Move down to Option 1: Open Cache Directory
	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(runner.runs) != 2 {
		t.Errorf("expected 2 launcher runs, got %v", runner.runs)
	}

	// 3. Move down to Option 2: Reload Configuration
	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	view = model.Render(70, 20)
	if !strings.Contains(view, "Configuration reloaded") {
		t.Errorf("expected reload status message, got:\n%s", view)
	}
}
