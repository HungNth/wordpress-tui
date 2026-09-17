package siteconfig_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"testing"

	"wptui/internal/config"
	"wptui/internal/packages"
	"wptui/internal/siteconfig"
)

type mockWPClient struct {
	runFn              func(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error)
	configSetFn        func(ctx context.Context, dir, key, value string, raw bool) error
	rewriteStructureFn func(ctx context.Context, dir, value string) error
	optionUpdateFn     func(ctx context.Context, dir, key, value string) error
	languageCoreFn     func(ctx context.Context, dir, action, value string) error
	pluginInstallFn    func(ctx context.Context, dir, pathOrSlug string, activate bool) error
	themeInstallFn     func(ctx context.Context, dir, pathOrSlug string, activate bool) error
	calls              []string
}

func (m *mockWPClient) Run(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
	call := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
	if stdin != "" {
		call += fmt.Sprintf(" [stdin:%s]", stdin)
	}
	m.calls = append(m.calls, call)
	if m.runFn != nil {
		return m.runFn(ctx, dir, name, args, stdin)
	}
	return "", "", nil
}

func (m *mockWPClient) ConfigSet(ctx context.Context, dir, key, value string, raw bool) error {
	m.calls = append(m.calls, fmt.Sprintf("config set %s %s raw:%v", key, value, raw))
	if m.configSetFn != nil {
		return m.configSetFn(ctx, dir, key, value, raw)
	}
	return nil
}

func (m *mockWPClient) RewriteStructure(ctx context.Context, dir, value string) error {
	m.calls = append(m.calls, fmt.Sprintf("rewrite structure %s", value))
	if m.rewriteStructureFn != nil {
		return m.rewriteStructureFn(ctx, dir, value)
	}
	return nil
}

func (m *mockWPClient) OptionUpdate(ctx context.Context, dir, key, value string) error {
	m.calls = append(m.calls, fmt.Sprintf("option update %s %s", key, value))
	if m.optionUpdateFn != nil {
		return m.optionUpdateFn(ctx, dir, key, value)
	}
	return nil
}

func (m *mockWPClient) LanguageCore(ctx context.Context, dir, action, value string) error {
	m.calls = append(m.calls, fmt.Sprintf("language core %s %s", action, value))
	if m.languageCoreFn != nil {
		return m.languageCoreFn(ctx, dir, action, value)
	}
	return nil
}

func (m *mockWPClient) PluginInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error {
	m.calls = append(m.calls, fmt.Sprintf("plugin install %s activate:%v", pathOrSlug, activate))
	if m.pluginInstallFn != nil {
		return m.pluginInstallFn(ctx, dir, pathOrSlug, activate)
	}
	return nil
}

func (m *mockWPClient) ThemeInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error {
	m.calls = append(m.calls, fmt.Sprintf("theme install %s activate:%v", pathOrSlug, activate))
	if m.themeInstallFn != nil {
		return m.themeInstallFn(ctx, dir, pathOrSlug, activate)
	}
	return nil
}

func TestExtractDBConfig(t *testing.T) {
	ctx := context.Background()
	mock := &mockWPClient{
		runFn: func(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
			if len(args) >= 3 && args[0] == "config" && args[1] == "get" {
				switch args[2] {
				case "DB_NAME":
					return "wordpress_db", "", nil
				case "DB_USER":
					return "wp_user", "", nil
				case "DB_PASSWORD":
					return "secret_pass", "", nil
				case "DB_HOST":
					return "127.0.0.1:3307", "", nil
				case "table_prefix":
					return "wp_custom_", "", nil
				}
			}
			return "", "", fmt.Errorf("unexpected arg %v", args)
		},
	}

	cfg, err := siteconfig.ExtractDBConfig(ctx, "/path/to/site", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "wordpress_db" {
		t.Errorf("expected DB Name 'wordpress_db', got %q", cfg.Name)
	}
	if cfg.User != "wp_user" {
		t.Errorf("expected DB User 'wp_user', got %q", cfg.User)
	}
	if cfg.Password != "secret_pass" {
		t.Errorf("expected DB Password 'secret_pass', got %q", cfg.Password)
	}
	if cfg.Host != "127.0.0.1" || cfg.Port != 3307 {
		t.Errorf("expected DB Host:Port 127.0.0.1:3307, got %s:%d", cfg.Host, cfg.Port)
	}
	if cfg.TablePrefix != "wp_custom_" {
		t.Errorf("expected TablePrefix 'wp_custom_', got %q", cfg.TablePrefix)
	}
}

func TestExtractDBConfig_InvalidPrefix(t *testing.T) {
	ctx := context.Background()
	mock := &mockWPClient{
		runFn: func(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
			if len(args) >= 3 && args[0] == "config" && args[1] == "get" && args[2] == "table_prefix" {
				return "wp;DROP TABLE users;--", "", nil
			}
			return "val", "", nil
		},
	}

	_, err := siteconfig.ExtractDBConfig(ctx, "/path/to/site", mock)
	if err == nil {
		t.Fatal("expected error on malicious table_prefix, got nil")
	}
	if !strings.Contains(err.Error(), "invalid table_prefix") {
		t.Errorf("expected 'invalid table_prefix' in error, got %v", err)
	}
}

func TestDiscoverAdministrators(t *testing.T) {
	ctx := context.Background()
	mock := &mockWPClient{
		runFn: func(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
			jsonOut := `[{"ID":1,"user_login":"admin1","user_email":"admin1@test.com"},{"ID":2,"user_login":"admin2","user_email":"admin2@test.com"}]`
			return jsonOut, "", nil
		},
	}

	admins, err := siteconfig.DiscoverAdministrators(ctx, "/path/to/site", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(admins) != 2 {
		t.Fatalf("expected 2 administrators, got %d", len(admins))
	}
	if admins[0].UserLogin != "admin1" || admins[1].UserLogin != "admin2" {
		t.Errorf("unexpected admins: %+v", admins)
	}
}

func TestDiscoverAdministrators_NoneFound(t *testing.T) {
	ctx := context.Background()
	mock := &mockWPClient{
		runFn: func(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
			return "[]", "", nil
		},
	}

	_, err := siteconfig.DiscoverAdministrators(ctx, "/path/to/site", mock)
	if err == nil {
		t.Fatal("expected error when 0 administrators found, got nil")
	}
	if !strings.Contains(err.Error(), "no users with administrator role found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Fake driver for database/sql mock in testing
type fakeDriver struct {
	executedQuery string
	args          []driver.Value
}

func (f *fakeDriver) Open(name string) (driver.Conn, error) {
	return &fakeConn{f: f}, nil
}

type fakeConn struct {
	f *fakeDriver
}

func (c *fakeConn) Prepare(query string) (driver.Stmt, error) {
	return &fakeStmt{f: c.f, query: query}, nil
}
func (c *fakeConn) Close() error              { return nil }
func (c *fakeConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }

type fakeStmt struct {
	f     *fakeDriver
	query string
}

func (s *fakeStmt) Close() error { return nil }
func (s *fakeStmt) NumInput() int {
	return -1
}
func (s *fakeStmt) Exec(args []driver.Value) (driver.Result, error) {
	s.f.executedQuery = s.query
	s.f.args = args
	return driver.RowsAffected(1), nil
}
func (s *fakeStmt) Query(args []driver.Value) (driver.Rows, error) {
	return nil, errors.New("not implemented")
}

func TestUpdateAdminCredentials(t *testing.T) {
	ctx := context.Background()
	mock := &mockWPClient{}

	dbCfg := &siteconfig.DBConfig{
		Name:        "wp_db",
		User:        "wp_user",
		Password:    "wp_pass",
		Host:        "localhost",
		Port:        3306,
		TablePrefix: "wp_",
	}

	input := siteconfig.AdminInput{
		UserID:      42,
		NewUsername: "super_admin",
		NewPassword: "new_super_secret_password",
		NewEmail:    "newadmin@example.com",
	}

	fd := &fakeDriver{}
	customConnector := func(driverName, dataSourceName string) (*sql.DB, error) {
		driverNameCustom := fmt.Sprintf("fake_driver_%p", fd)
		sql.Register(driverNameCustom, fd)
		return sql.Open(driverNameCustom, dataSourceName)
	}

	progressEvents := 0
	onProgress := func(step, total int, msg string) {
		progressEvents++
	}

	err := siteconfig.UpdateAdminCredentials(ctx, "/path/to/site", dbCfg, input, mock, customConnector, onProgress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if progressEvents == 0 {
		t.Errorf("expected progress events to be fired")
	}
	expectedQuery := "UPDATE `wp_users` SET user_login = ?, user_nicename = ? WHERE ID = ?"
	if fd.executedQuery != expectedQuery {
		t.Errorf("expected SQL query %q, got %q", expectedQuery, fd.executedQuery)
	}
	if len(fd.args) != 3 {
		t.Fatalf("expected 3 SQL parameters, got %v", fd.args)
	}
	if fd.args[0] != "super_admin" || fd.args[1] != "super_admin" || fd.args[2] != int64(42) {
		t.Errorf("unexpected SQL params: %v", fd.args)
	}

	// 2. Check Password WP-CLI call via stdin
	foundPassCall := false
	foundEmailCall := false
	foundOptionCall := false
	for _, call := range mock.calls {
		if strings.Contains(call, "wp user update 42 --prompt=user_pass [stdin:new_super_secret_password]") {
			foundPassCall = true
		}
		if strings.Contains(call, "wp user update 42 --user_email=newadmin@example.com") {
			foundEmailCall = true
		}
		if strings.Contains(call, "option update admin_email newadmin@example.com") {
			foundOptionCall = true
		}
	}

	if !foundPassCall {
		t.Errorf("expected password update via stdin prompt, got calls: %v", mock.calls)
	}
	if !foundEmailCall {
		t.Errorf("expected user_email update, got calls: %v", mock.calls)
	}
	if !foundOptionCall {
		t.Errorf("expected option update admin_email, got calls: %v", mock.calls)
	}
}

func TestApplyTweaks(t *testing.T) {
	ctx := context.Background()
	mock := &mockWPClient{}

	tweaks := []config.WPTweak{
		{Type: "config_set", Key: "WP_DEBUG", Value: "true", Raw: true},
		{Type: "rewrite_structure", Value: "/%postname%/"},
		{Type: "option_update", Key: "timezone_string", Value: "Asia/Ho_Chi_Minh"},
		{Type: "language_core", Key: "install", Value: "vi"},
	}
	results := siteconfig.ApplyTweaks(ctx, "/path/to/site", tweaks, mock, nil)
	for i, r := range results {
		if !r.Success {
			t.Errorf("tweak %d failed: %v", i, r.Err)
		}
	}
}

func TestInstallPackages(t *testing.T) {
	ctx := context.Background()
	mock := &mockWPClient{}

	plugins := []packages.Artifact{
		{Ref: packages.PackageRef{Slug: "plugin-a", Type: packages.PackageTypePlugin}, Path: "/cache/plugin-a.zip"},
		{Ref: packages.PackageRef{Slug: "plugin-b", Type: packages.PackageTypePlugin}, Path: ""},
	}
	results := siteconfig.InstallPackages(ctx, "/path/to/site", packages.PackageTypePlugin, plugins, true, mock, nil)
	if !results[0].Success || !results[0].Activated {
		t.Errorf("plugin-a failed or not activated")
	}

	themes := []packages.Artifact{
		{Ref: packages.PackageRef{Slug: "theme-a", Type: packages.PackageTypeTheme}, Path: "/cache/theme-a.zip"},
	}
	themeResults := siteconfig.InstallPackages(ctx, "/path/to/site", packages.PackageTypeTheme, themes, false, mock, nil)
	if !themeResults[0].Success || themeResults[0].Activated {
		t.Errorf("theme-a failed or unexpectedly activated")
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1, v2 string
		want   int
	}{
		{"6.8.10", "6.8.9", 1},
		{"6.8.9", "6.8.10", -1},
		{"6.8.10", "6.8.10", 0},
		{"v1.2.3", "1.2.3", 0},
		{"2.0", "1.9.9", 1},
		{"1.0.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.0.1", -1},
	}

	for _, tc := range tests {
		got := siteconfig.CompareVersions(tc.v1, tc.v2)
		if got != tc.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tc.v1, tc.v2, got, tc.want)
		}
	}
}

func TestInstallPackages_VersionAware(t *testing.T) {
	ctx := context.Background()
	mock := &mockWPClient{
		runFn: func(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
			if len(args) >= 3 && args[0] == "plugin" && args[1] == "get" {
				switch args[2] {
				case "up-to-date-plugin":
					return "2.0.0", "", nil
				case "outdated-plugin":
					return "1.0.0", "", nil
				case "newer-installed-plugin":
					return "3.0.0", "", nil
				case "brand-new-plugin":
					return "", "Error: not found", errors.New("not found")
				}
			}
			return "", "", nil
		},
	}

	artifacts := []packages.Artifact{
		{Ref: packages.PackageRef{Slug: "up-to-date-plugin", Type: packages.PackageTypePlugin}, Version: "2.0.0"},
		{Ref: packages.PackageRef{Slug: "outdated-plugin", Type: packages.PackageTypePlugin}, Version: "2.0.0"},
		{Ref: packages.PackageRef{Slug: "newer-installed-plugin", Type: packages.PackageTypePlugin}, Version: "2.0.0"},
		{Ref: packages.PackageRef{Slug: "brand-new-plugin", Type: packages.PackageTypePlugin}, Version: "1.0.0"},
	}

	results := siteconfig.InstallPackages(ctx, "/path/to/site", packages.PackageTypePlugin, artifacts, true, mock, nil)
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	// 1. up-to-date: skipped
	if !results[0].Skipped || !strings.Contains(results[0].SkipReason, "Already up to date") {
		t.Errorf("expected up-to-date plugin to be skipped, got: %+v", results[0])
	}

	// 2. outdated: upgraded with --force
	if results[1].Skipped || !results[1].Success {
		t.Errorf("expected outdated plugin to be upgraded, got: %+v", results[1])
	}

	// 3. newer installed: skipped
	if !results[2].Skipped || !strings.Contains(results[2].SkipReason, "Current installed version is newer") {
		t.Errorf("expected newer installed plugin to be skipped, got: %+v", results[2])
	}

	// 4. brand new: installed without --force
	if results[3].Skipped || !results[3].Success {
		t.Errorf("expected brand new plugin to be installed, got: %+v", results[3])
	}

	// Verify --force call in mock
	foundForce := false
	for _, call := range mock.calls {
		if strings.Contains(call, "plugin install outdated-plugin") && strings.Contains(call, "--force") {
			foundForce = true
			break
		}
	}
	if !foundForce {
		t.Errorf("expected --force flag on outdated plugin install, calls: %v", mock.calls)
	}
}
