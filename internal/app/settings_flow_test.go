package app_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wptui/internal/app"
	"wptui/internal/config"
	"wptui/internal/tui"
)

type mockSettingsLauncher struct {
	lookPathFn func(string) (string, error)
	runFn      func(context.Context, string, ...string) error
	startFn    func(context.Context, string, ...string) error
	runs       []string
}

func (m *mockSettingsLauncher) LookPath(file string) (string, error) {
	if m.lookPathFn != nil {
		return m.lookPathFn(file)
	}
	return "/bin/" + file, nil
}

func (m *mockSettingsLauncher) Run(ctx context.Context, name string, args ...string) error {
	m.runs = append(m.runs, name+" "+strings.Join(args, " "))
	if m.runFn != nil {
		return m.runFn(ctx, name, args...)
	}
	return nil
}

func (m *mockSettingsLauncher) Start(ctx context.Context, name string, args ...string) error {
	m.runs = append(m.runs, "start:"+name+" "+strings.Join(args, " "))
	if m.startFn != nil {
		return m.startFn(ctx, name, args...)
	}
	return nil
}

func TestApp_SettingsMenuRouting(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		WebsitesPath: tempDir,
	}
	cfgPath, _ := config.ConfigPath(tempDir)
	_ = config.Save(cfgPath, cfg)

	settingsCalled := false
	menuCalls := 0

	application := app.New(app.Options{
		HomeDir: tempDir,
		MenuFn: func() (string, error) {
			menuCalls++
			if menuCalls == 1 {
				return "settings", nil
			}
			return "exit", nil
		},
		SettingsFn: func(ctx context.Context, c *config.Config, onReload func(*config.Config)) error {
			settingsCalled = true
			return nil
		},
	})

	err := application.RunWithContext(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !settingsCalled {
		t.Errorf("expected settingsFn to be called on 'settings' menu selection")
	}
}

func TestRunSettingsFlowWithDeps_OpenVSCodeAndHotReload(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	initialCfg := config.DefaultConfig(tempDir)
	initialCfg.WebsitesPath = tempDir
	initialCfg.DefaultAdminUsername = "initial_admin"
	_ = config.Save(cfgPath, initialCfg)

	mockLaunch := &mockSettingsLauncher{
		runFn: func(ctx context.Context, name string, args ...string) error {
			// Simulate user editing config.json in VS Code before closing
			updatedCfg := config.DefaultConfig(tempDir)
			updatedCfg.WebsitesPath = tempDir
			updatedCfg.DefaultAdminUsername = "updated_admin"
			return config.Save(cfgPath, updatedCfg)
		},
	}

	var reloadedConfig *config.Config
	actionCalls := 0

	deps := app.SettingsFlowDependencies{
		ConfigPath: cfgPath,
		Launcher:   mockLaunch,
		PromptAction: func() (tui.SettingsAction, error) {
			actionCalls++
			if actionCalls == 1 {
				return tui.ActionOpenVSCode, nil
			}
			return tui.ActionSettingsBack, nil
		},
		OnReloadConfig: func(newCfg *config.Config) {
			reloadedConfig = newCfg
		},
	}

	err := app.RunSettingsFlowWithDeps(context.Background(), initialCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if reloadedConfig == nil {
		t.Fatal("expected reloadedConfig to be populated via OnReloadConfig callback")
	}
	if reloadedConfig.DefaultAdminUsername != "updated_admin" {
		t.Errorf("expected updated username 'updated_admin', got %q", reloadedConfig.DefaultAdminUsername)
	}
}

func TestRunSettingsFlowWithDeps_InvalidJSONRetainsOldConfig(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	initialCfg := config.DefaultConfig(tempDir)
	initialCfg.WebsitesPath = tempDir
	initialCfg.DefaultAdminUsername = "stable_admin"
	_ = config.Save(cfgPath, initialCfg)

	mockLaunch := &mockSettingsLauncher{
		runFn: func(ctx context.Context, name string, args ...string) error {
			// Simulate user writing invalid JSON
			return os.WriteFile(cfgPath, []byte(`{invalid-json`), 0644)
		},
	}

	reloadCalled := false
	actionCalls := 0

	deps := app.SettingsFlowDependencies{
		ConfigPath: cfgPath,
		Launcher:   mockLaunch,
		PromptAction: func() (tui.SettingsAction, error) {
			actionCalls++
			if actionCalls == 1 {
				return tui.ActionOpenVSCode, nil
			}
			return tui.ActionSettingsBack, nil
		},
		OnReloadConfig: func(newCfg *config.Config) {
			reloadCalled = true
		},
	}

	err := app.RunSettingsFlowWithDeps(context.Background(), initialCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if reloadCalled {
		t.Errorf("reload callback must NOT be called when JSON is invalid")
	}
}

func TestRunSettingsFlowWithDeps_MissingCodeCLI(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	mockLaunch := &mockSettingsLauncher{
		lookPathFn: func(file string) (string, error) {
			return "", errors.New("not found")
		},
	}

	actionCalls := 0
	deps := app.SettingsFlowDependencies{
		ConfigPath: cfgPath,
		Launcher:   mockLaunch,
		PromptAction: func() (tui.SettingsAction, error) {
			actionCalls++
			if actionCalls == 1 {
				return tui.ActionOpenVSCode, nil
			}
			return tui.ActionSettingsBack, nil
		},
	}

	err := app.RunSettingsFlowWithDeps(context.Background(), &config.Config{}, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not execute code
	if len(mockLaunch.runs) != 0 {
		t.Errorf("expected no execution calls on missing code CLI, got: %v", mockLaunch.runs)
	}
}

func TestRunSettingsFlowWithDeps_VSCodeProcessFails_NoReload(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	initialCfg := config.DefaultConfig(tempDir)
	initialCfg.WebsitesPath = tempDir
	initialCfg.DefaultAdminUsername = "stable_admin"
	_ = config.Save(cfgPath, initialCfg)

	mockLaunch := &mockSettingsLauncher{
		runFn: func(ctx context.Context, name string, args ...string) error {
			// Editor exits non-zero without the user saving anything
			return errors.New("exit status 1")
		},
	}

	reloadCalled := false
	actionCalls := 0

	deps := app.SettingsFlowDependencies{
		ConfigPath: cfgPath,
		Launcher:   mockLaunch,
		PromptAction: func() (tui.SettingsAction, error) {
			actionCalls++
			if actionCalls == 1 {
				return tui.ActionOpenVSCode, nil
			}
			return tui.ActionSettingsBack, nil
		},
		OnReloadConfig: func(newCfg *config.Config) {
			reloadCalled = true
		},
	}

	err := app.RunSettingsFlowWithDeps(context.Background(), initialCfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if reloadCalled {
		t.Errorf("reload callback must NOT be called when the editor process fails")
	}
	if len(mockLaunch.runs) != 1 {
		t.Errorf("expected exactly one 'code --wait' invocation, got: %v", mockLaunch.runs)
	}
}

func TestRunSettingsFlowWithDeps_OpenCache(t *testing.T) {
	tempDir := t.TempDir()
	mockLaunch := &mockSettingsLauncher{}

	// Pre-create wptui cache dir in user cache dir for test
	userCache, _ := os.UserCacheDir()
	cacheDir := filepath.Join(userCache, "wptui")
	_ = os.MkdirAll(cacheDir, 0755)

	actionCalls := 0
	deps := app.SettingsFlowDependencies{
		ConfigPath: filepath.Join(tempDir, "config.json"),
		Launcher:   mockLaunch,
		PromptAction: func() (tui.SettingsAction, error) {
			actionCalls++
			if actionCalls == 1 {
				return tui.ActionOpenCache, nil
			}
			return tui.ActionSettingsBack, nil
		},
	}

	err := app.RunSettingsFlowWithDeps(context.Background(), &config.Config{}, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundCacheLaunch := false
	for _, run := range mockLaunch.runs {
		if strings.Contains(run, "start:") && strings.Contains(run, "wptui") {
			foundCacheLaunch = true
			break
		}
	}
	if !foundCacheLaunch {
		t.Errorf("expected cache directory launcher call, got: %v", mockLaunch.runs)
	}
}
