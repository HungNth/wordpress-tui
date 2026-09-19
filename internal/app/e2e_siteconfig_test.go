package app_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wptui/internal/app"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/packages"
	"wptui/internal/siteconfig"
	"wptui/internal/tui"
)

type e2eConfigMockClient struct {
	runs []string
}

func (m *e2eConfigMockClient) ConfigSet(ctx context.Context, dir, key, value string, raw bool) error {
	m.runs = append(m.runs, fmt.Sprintf("config set %s=%s", key, value))
	return nil
}
func (m *e2eConfigMockClient) RewriteStructure(ctx context.Context, dir, value string) error {
	m.runs = append(m.runs, "rewrite structure "+value)
	return nil
}
func (m *e2eConfigMockClient) OptionUpdate(ctx context.Context, dir, key, value string) error {
	m.runs = append(m.runs, fmt.Sprintf("option update %s=%s", key, value))
	return nil
}
func (m *e2eConfigMockClient) LanguageCore(ctx context.Context, dir, action, value string) error {
	m.runs = append(m.runs, fmt.Sprintf("language core %s %s", action, value))
	return nil
}
func (m *e2eConfigMockClient) PluginInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error {
	m.runs = append(m.runs, fmt.Sprintf("plugin install %s activate:%v", pathOrSlug, activate))
	return nil
}
func (m *e2eConfigMockClient) ThemeInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error {
	m.runs = append(m.runs, fmt.Sprintf("theme install %s activate:%v", pathOrSlug, activate))
	return nil
}
func (m *e2eConfigMockClient) Run(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
	call := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
	if stdin != "" {
		call += fmt.Sprintf(" [stdin:%s]", stdin)
	}
	m.runs = append(m.runs, call)

	if len(args) >= 3 && args[0] == "config" && args[1] == "get" {
		switch args[2] {
		case "DB_NAME":
			return "e2e_db", "", nil
		case "DB_USER":
			return "e2e_user", "", nil
		case "DB_PASSWORD":
			return "e2e_pass", "", nil
		case "DB_HOST":
			return "127.0.0.1:3306", "", nil
		case "table_prefix":
			return "wp_e2e_", "", nil
		}
	}
	if len(args) >= 2 && args[0] == "user" && args[1] == "list" {
		return `[{"ID":1,"user_login":"old_admin","user_email":"old@example.com"}]`, "", nil
	}
	return "", "", nil
}

type fakeE2EDriver struct {
	query string
	args  []driver.Value
}

func (f *fakeE2EDriver) Open(name string) (driver.Conn, error) {
	return &fakeE2EConn{f: f}, nil
}

type fakeE2EConn struct {
	f *fakeE2EDriver
}

func (c *fakeE2EConn) Prepare(query string) (driver.Stmt, error) {
	return &fakeE2EStmt{f: c.f, query: query}, nil
}
func (c *fakeE2EConn) Close() error              { return nil }
func (c *fakeE2EConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }

type fakeE2EStmt struct {
	f     *fakeE2EDriver
	query string
}

func (s *fakeE2EStmt) Close() error { return nil }
func (s *fakeE2EStmt) NumInput() int {
	return -1
}
func (s *fakeE2EStmt) Exec(args []driver.Value) (driver.Result, error) {
	s.f.query = s.query
	s.f.args = args
	return driver.RowsAffected(1), nil
}
func (s *fakeE2EStmt) Query(args []driver.Value) (driver.Rows, error) {
	return nil, errors.New("not implemented")
}

type mockE2EResolver struct{}

func (m *mockE2EResolver) ResolvePackage(ctx context.Context, ref packages.PackageRef, stageDir string) (*packages.Artifact, error) {
	return &packages.Artifact{Ref: ref, Path: filepath.Join(stageDir, ref.Slug+".zip")}, nil
}
func (m *mockE2EResolver) ResolveAll(ctx context.Context, refs []packages.PackageRef, stageDir string) ([]packages.Artifact, error) {
	artifacts := make([]packages.Artifact, 0, len(refs))
	for _, r := range refs {
		artifacts = append(artifacts, packages.Artifact{Ref: r, Path: filepath.Join(stageDir, r.Slug+".zip")})
	}
	return artifacts, nil
}

func TestE2E_ComposedSiteConfiguration(t *testing.T) {
	tempDir := t.TempDir()
	siteDir := filepath.Join(tempDir, "sample-site")
	_ = os.MkdirAll(siteDir, 0755)

	cfg := &config.Config{
		WebsitesPath: tempDir,
		WPTweaks: []config.WPTweak{
			{Type: "config_set", Key: "WP_DEBUG", Value: "true"},
		},
		DefaultAdminUsername: "new_admin",
		DefaultAdminPassword: "new_password",
		DefaultAdminEmail:    "new@example.com",
	}

	client := &e2eConfigMockClient{}
	driverInstance := &fakeE2EDriver{}
	driverName := fmt.Sprintf("fake_e2e_%p", driverInstance)
	sql.Register(driverName, driverInstance)
	connector := func(d, dsn string) (*sql.DB, error) {
		return sql.Open(driverName, dsn)
	}

	actionsToRun := []tui.ConfigAction{
		tui.ActionApplyTweaks,
		tui.ActionChangeAdmin,
		tui.ActionInstallPlugins,
		tui.ActionInstallThemes,
	}
	actionIdx := 0
	siteSelectionCount := 0

	deps := app.ConfigFlowDependencies{
		WPClient: client,
		Resolver: &mockE2EResolver{},
		Connector: connector,
		SelectWebsite: func(candidates []deprovision.Candidate) (*deprovision.Candidate, error) {
			siteSelectionCount++
			if siteSelectionCount > 1 {
				return nil, nil // user backed out to main menu
			}
			return &candidates[0], nil
		},
		SelectAction: func(slug string) (tui.ConfigAction, error) {
			if actionIdx < len(actionsToRun) {
				act := actionsToRun[actionIdx]
				actionIdx++
				return act, nil
			}
			return tui.ActionBack, nil
		},
		PromptAdmin: func(current siteconfig.AdminUser, defaults *config.Config) (siteconfig.AdminInput, error) {
			return siteconfig.AdminInput{
				UserID:      current.ID,
				NewUsername: "updated_admin",
				NewPassword: "updated_pass",
				NewEmail:    "updated@example.com",
			}, nil
		},
		PromptPackages: func(ctx context.Context, c *config.Config, cat []packages.CatalogItem) ([]string, []string, error) {
			return []string{"plugin-one"}, nil, nil
		},
		PromptThemes: func(ctx context.Context, c *config.Config, cat []packages.CatalogItem) ([]string, []string, error) {
			return []string{"theme-one"}, nil, nil
		},
		PromptThemeAct: func() (bool, error) {
			return false, nil // install without activation
		},
	}

	err := app.RunConfigFlowWithDeps(context.Background(), cfg, deps)
	if err != nil {
		t.Fatalf("unexpected error during e2e config flow: %v", err)
	}

	// 1. Verify Tweaks
	tweakFound := false
	for _, r := range client.runs {
		if r == "config set WP_DEBUG=true" {
			tweakFound = true
			break
		}
	}
	if !tweakFound {
		t.Errorf("expected WP_DEBUG tweak, got: %v", client.runs)
	}

	// 2. Verify Admin MySQL direct update
	expectedSQL := "UPDATE `wp_e2e_users` SET user_login = ?, user_nicename = ? WHERE ID = ?"
	if driverInstance.query != expectedSQL {
		t.Errorf("expected SQL %q, got %q", expectedSQL, driverInstance.query)
	}
	if len(driverInstance.args) != 3 || driverInstance.args[0] != "updated_admin" {
		t.Errorf("unexpected SQL arguments: %v", driverInstance.args)
	}

	// 3. Verify Admin Password update via stdin
	passFound := false
	emailUserFound := false
	emailOptionFound := false
	for _, r := range client.runs {
		if strings.Contains(r, "wp user update 1 --prompt=user_pass [stdin:updated_pass]") {
			passFound = true
		}
		if strings.Contains(r, "wp user update 1 --user_email=updated@example.com") {
			emailUserFound = true
		}
		if r == "option update admin_email=updated@example.com" {
			emailOptionFound = true
		}
	}
	if !passFound {
		t.Errorf("expected password update via stdin, got: %v", client.runs)
	}
	if !emailUserFound || !emailOptionFound {
		t.Errorf("expected user and site admin email update, got: %v", client.runs)
	}

	// 4. Verify Plugin Installation (activated and version checked)
	pluginFound := false
	for _, r := range client.runs {
		if strings.Contains(r, "wp plugin install") && strings.Contains(r, "plugin-one.zip") && strings.Contains(r, "--activate") {
			pluginFound = true
			break
		}
	}
	if !pluginFound {
		t.Errorf("expected plugin install with activation, got: %v", client.runs)
	}

	// 5. Verify Theme Installation (not activated and version checked)
	themeFound := false
	for _, r := range client.runs {
		if strings.Contains(r, "wp theme install") && strings.Contains(r, "theme-one.zip") && !strings.Contains(r, "--activate") {
			themeFound = true
			break
		}
	}
	if !themeFound {
		t.Errorf("expected theme install without activation, got: %v", client.runs)
	}
}
