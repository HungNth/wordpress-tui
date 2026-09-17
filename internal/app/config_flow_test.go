package app_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"wptui/internal/app"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/tui"
)

type mockSiteWPClient struct {
	runs []string
}

func (m *mockSiteWPClient) ConfigSet(ctx context.Context, dir, key, value string, raw bool) error {
	m.runs = append(m.runs, "config set "+key)
	return nil
}
func (m *mockSiteWPClient) RewriteStructure(ctx context.Context, dir, value string) error {
	m.runs = append(m.runs, "rewrite structure "+value)
	return nil
}
func (m *mockSiteWPClient) OptionUpdate(ctx context.Context, dir, key, value string) error {
	m.runs = append(m.runs, "option update "+key)
	return nil
}
func (m *mockSiteWPClient) LanguageCore(ctx context.Context, dir, action, value string) error {
	m.runs = append(m.runs, "language core "+action)
	return nil
}
func (m *mockSiteWPClient) PluginInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error {
	m.runs = append(m.runs, "plugin install "+pathOrSlug)
	return nil
}
func (m *mockSiteWPClient) ThemeInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error {
	m.runs = append(m.runs, "theme install "+pathOrSlug)
	return nil
}
func (m *mockSiteWPClient) Run(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
	if len(args) >= 3 && args[0] == "config" && args[1] == "get" {
		switch args[2] {
		case "DB_NAME":
			return "db_site", "", nil
		case "DB_USER":
			return "user_site", "", nil
		case "DB_PASSWORD":
			return "pass_site", "", nil
		case "DB_HOST":
			return "127.0.0.1:3306", "", nil
		case "table_prefix":
			return "wp_", "", nil
		}
	}
	if len(args) >= 2 && args[0] == "user" && args[1] == "list" {
		return `[{"ID":1,"user_login":"siteadmin","user_email":"admin@site.test"}]`, "", nil
	}
	if len(args) >= 3 && args[0] == "user" && args[1] == "update" {
		m.runs = append(m.runs, "user update")
		return "", "", nil
	}
	return "", "", nil
}

func TestApp_ConfigMenuRouting(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		WebsitesPath: tempDir,
	}
	cfgPath, _ := config.ConfigPath(tempDir)
	_ = config.Save(cfgPath, cfg)

	configCalled := false
	menuCalls := 0

	application := app.New(app.Options{
		HomeDir: tempDir,
		MenuFn: func() (string, error) {
			menuCalls++
			if menuCalls == 1 {
				return "config", nil
			}
			return "exit", nil
		},
		ConfigFn: func(ctx context.Context, c *config.Config) error {
			configCalled = true
			return nil
		},
	})

	err := application.RunWithContext(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !configCalled {
		t.Errorf("expected configFn to be called on 'config' menu selection")
	}
}

func TestRunConfigFlowWithDeps_ApplyTweaks(t *testing.T) {
	tempDir := t.TempDir()
	siteDir := filepath.Join(tempDir, "mysite")
	_ = os.MkdirAll(siteDir, 0755)

	cfg := &config.Config{
		WebsitesPath: tempDir,
		WPTweaks: []config.WPTweak{
			{Type: "config_set", Key: "WP_DEBUG", Value: "true"},
		},
	}

	mockCli := &mockSiteWPClient{}
	actionCalls := 0
	websiteCalls := 0

	deps := app.ConfigFlowDependencies{
		WPClient: mockCli,
		SelectWebsite: func(candidates []deprovision.Candidate) (*deprovision.Candidate, error) {
			websiteCalls++
			if websiteCalls > 1 {
				return nil, nil // user backed out to main menu
			}
			return &candidates[0], nil
		},
		SelectAction: func(slug string) (tui.ConfigAction, error) {
			actionCalls++
			if actionCalls == 1 {
				return tui.ActionApplyTweaks, nil
			}
			return tui.ActionBack, nil
		},
		PromptContinue: func() (bool, error) {
			return false, nil // exit site loop
		},
	}

	err := app.RunConfigFlowWithDeps(context.Background(), cfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundTweak := false
	for _, run := range mockCli.runs {
		if run == "config set WP_DEBUG" {
			foundTweak = true
			break
		}
	}
	if !foundTweak {
		t.Errorf("expected tweak to be executed, got: %v", mockCli.runs)
	}
}
