package restore_test

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wptui/internal/config"
	"wptui/internal/restore"
)

type mockRestoreWPClient struct {
	calls        []string
	failOnOp     string
	siteURL      string
	dbCreated    bool
	dbImported   bool
	siteSecured  bool
	isSecuredVal bool
}

func (m *mockRestoreWPClient) Run(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
	call := fmt.Sprintf("%s %s in %s", name, strings.Join(args, " "), dir)
	m.calls = append(m.calls, call)

	if m.failOnOp != "" && strings.Contains(call, m.failOnOp) {
		return "", "simulated failure", fmt.Errorf("mock error for %s", m.failOnOp)
	}

	if name == "wp" && len(args) >= 3 && args[0] == "option" && args[1] == "get" && args[2] == "siteurl" {
		url := m.siteURL
		if url == "" {
			url = "http://old-domain.com"
		}
		return url + "\n", "", nil
	}

	if name == "wp" && len(args) >= 2 && args[0] == "db" && args[1] == "create" {
		m.dbCreated = true
		return "Success: Database created.\n", "", nil
	}

	if name == "wp" && len(args) >= 3 && args[0] == "db" && args[1] == "import" {
		m.dbImported = true
		return "Success: Imported from database.\n", "", nil
	}

	if name == "herd" && len(args) >= 2 && args[0] == "secure" {
		m.siteSecured = true
		return "Secured\n", "", nil
	}

	if name == "herd" && len(args) >= 1 && args[0] == "secured" {
		if m.isSecuredVal {
			return "+-----------------------+\n| Site                  |\n+-----------------------+\n| my-site.test          |\n+-----------------------+\n", "", nil
		}
		return "+-----------------------+\n| Site                  |\n+-----------------------+\n", "", nil
	}

	return "", "", nil
}

func (m *mockRestoreWPClient) DBCreate(ctx context.Context, dir string) (bool, error) {
	m.calls = append(m.calls, "DBCreate in "+dir)
	if m.failOnOp == "db_create" {
		return false, fmt.Errorf("simulated db create error")
	}
	m.dbCreated = true
	return true, nil
}

func (m *mockRestoreWPClient) DBDrop(ctx context.Context, dir string) error {
	m.calls = append(m.calls, "DBDrop in "+dir)
	m.dbCreated = false
	return nil
}

func (m *mockRestoreWPClient) DBImport(ctx context.Context, dir, sqlFile string) error {
	m.calls = append(m.calls, fmt.Sprintf("DBImport %s in %s", sqlFile, dir))
	if m.failOnOp == "db_import" {
		return fmt.Errorf("simulated db import error")
	}
	m.dbImported = true
	return nil
}

func (m *mockRestoreWPClient) SearchReplace(ctx context.Context, dir, search, replace string) error {
	m.calls = append(m.calls, fmt.Sprintf("SearchReplace %s -> %s in %s", search, replace, dir))
	if m.failOnOp == "search_replace" {
		return fmt.Errorf("simulated search replace error")
	}
	return nil
}

func (m *mockRestoreWPClient) OptionGet(ctx context.Context, dir, key string) (string, error) {
	m.calls = append(m.calls, fmt.Sprintf("OptionGet %s in %s", key, dir))
	if key == "siteurl" {
		url := m.siteURL
		if url == "" {
			url = "http://old-domain.com"
		}
		return url, nil
	}
	return "", nil
}
func (m *mockRestoreWPClient) AI1WMRestore(ctx context.Context, dir, archiveFileName string) error {
	m.calls = append(m.calls, fmt.Sprintf("AI1WMRestore %s in %s", archiveFileName, dir))
	if m.failOnOp == "ai1wm restore" {
		return fmt.Errorf("simulated ai1wm restore error")
	}
	return nil
}

func (m *mockRestoreWPClient) CoreInstall(ctx context.Context, dir, url, title, adminUser, adminPass, adminEmail string) error {
	m.calls = append(m.calls, fmt.Sprintf("CoreInstall %s in %s", url, dir))
	if m.failOnOp == "core_install" {
		return fmt.Errorf("simulated core install error")
	}
	return nil
}
func (m *mockRestoreWPClient) HerdSecure(ctx context.Context, dir, slug string) error {
	m.calls = append(m.calls, fmt.Sprintf("HerdSecure %s in %s", slug, dir))
	if m.failOnOp == "herd_secure" {
		return fmt.Errorf("simulated herd secure error")
	}
	m.siteSecured = true
	return nil
}

func (m *mockRestoreWPClient) HerdUnsecure(ctx context.Context, dir, slug string) error {
	m.calls = append(m.calls, fmt.Sprintf("HerdUnsecure %s in %s", slug, dir))
	m.siteSecured = false
	return nil
}

func (m *mockRestoreWPClient) IsSiteSecured(ctx context.Context, slug string) (bool, error) {
	m.calls = append(m.calls, "IsSiteSecured "+slug)
	return m.isSecuredVal, nil
}

func (m *mockRestoreWPClient) ConfigCreateWithOptions(ctx context.Context, dir, dbName string, conn restore.DBConnection, opts restore.ConfigCreateOptions) error {
	m.calls = append(m.calls, fmt.Sprintf("ConfigCreateWithOptions dbName=%s prefix=%s in %s", dbName, opts.DBPrefix, dir))
	if m.failOnOp == "config_create" {
		return fmt.Errorf("simulated config create error")
	}
	return nil
}
func (m *mockRestoreWPClient) ConfigSet(ctx context.Context, dir, key, value string, raw bool) error {
	m.calls = append(m.calls, fmt.Sprintf("ConfigSet %s=%s in %s", key, value, dir))
	return nil
}

func (m *mockRestoreWPClient) RewriteStructure(ctx context.Context, dir, value string) error {
	m.calls = append(m.calls, fmt.Sprintf("RewriteStructure %s in %s", value, dir))
	return nil
}

func (m *mockRestoreWPClient) OptionUpdate(ctx context.Context, dir, key, value string) error {
	m.calls = append(m.calls, fmt.Sprintf("OptionUpdate %s=%s in %s", key, value, dir))
	return nil
}

func (m *mockRestoreWPClient) LanguageCore(ctx context.Context, dir, action, value string) error {
	m.calls = append(m.calls, fmt.Sprintf("LanguageCore %s %s in %s", action, value, dir))
	return nil
}

func (m *mockRestoreWPClient) PluginInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error {
	m.calls = append(m.calls, fmt.Sprintf("PluginInstall %s in %s", pathOrSlug, dir))
	return nil
}

func (m *mockRestoreWPClient) ThemeInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error {
	m.calls = append(m.calls, fmt.Sprintf("ThemeInstall %s in %s", pathOrSlug, dir))
	return nil
}

type mockAdminUpdater struct {
	updatedWith []string
	fail        bool
}

func (m *mockAdminUpdater) UpdateAdminCredentials(ctx context.Context, siteDir, newUsername, newPassword, newEmail string) error {
	if m.fail {
		return fmt.Errorf("simulated admin update error")
	}
	m.updatedWith = []string{siteDir, newUsername, newPassword, newEmail}
	return nil
}

func TestRestorer_FullZIP_Success(t *testing.T) {
	tempRoot := t.TempDir()
	websitesPath := filepath.Join(tempRoot, "websites")
	_ = os.MkdirAll(websitesPath, 0755)

	// Create a dummy mock zip archive that contains a valid WordPress root
	archivePath := filepath.Join(tempRoot, "full_mysite_2026-09-18_12-00-00.zip")
	createValidZipArchive(t, archivePath, "mysite.sql", "wptest_")

	cfg := &config.Config{
		UsedHerd:     true,
		WebsitesPath: websitesPath,
		DatabaseHost: "localhost",
		DatabasePort: 3306,
		DBUsername:   "root",
	}

	mockClient := &mockRestoreWPClient{
		siteURL: "http://old-domain.com",
	}
	adminUpdater := &mockAdminUpdater{}

	res := restore.NewRestorer(cfg, mockClient)
	res.SetAdminUpdater(adminUpdater)
	req := restore.Request{
		Strategy:    restore.StrategyFullZIP,
		ArchivePath: archivePath,
		WebsiteName: "My Restored Site",
		WebsiteSlug: "my-restored-site",
		AdminUser:   "newadmin",
		AdminPass:   "newpass",
		AdminEmail:  "newadmin@example.com",
	}

	result, err := res.Restore(t.Context(), req, nil)
	if err != nil {
		t.Fatalf("unexpected restore error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.SiteURL != "https://my-restored-site.test" {
		t.Errorf("got URL %q, want 'https://my-restored-site.test'", result.SiteURL)
	}

	// Verify target site directory exists
	targetDir := filepath.Join(websitesPath, "my-restored-site")
	if _, err := os.Stat(targetDir); err != nil {
		t.Errorf("expected target site directory to exist: %v", err)
	}

	// Verify selected SQL dump was cleaned up from target directory
	dumpInTarget := filepath.Join(targetDir, "mysite.sql")
	if _, err := os.Stat(dumpInTarget); !os.IsNotExist(err) {
		t.Errorf("expected SQL dump in target directory to be deleted, but found it")
	}

	// Verify original archive still exists untouched
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("expected original archive to remain intact: %v", err)
	}
}

func TestRestorer_FullZIP_RollbackOnImportFailure(t *testing.T) {
	tempRoot := t.TempDir()
	websitesPath := filepath.Join(tempRoot, "websites")
	_ = os.MkdirAll(websitesPath, 0755)

	archivePath := filepath.Join(tempRoot, "full_mysite_2026-09-18_12-00-00.zip")
	createValidZipArchive(t, archivePath, "mysite.sql", "wptest_")

	cfg := &config.Config{
		UsedHerd:     false,
		WebsitesPath: websitesPath,
		DatabaseHost: "localhost",
		DatabasePort: 3306,
		DBUsername:   "root",
	}

	mockClient := &mockRestoreWPClient{
		failOnOp: "db_import",
	}

	res := restore.NewRestorer(cfg, mockClient)
	res.SetAdminUpdater(&mockAdminUpdater{})
	req := restore.Request{
		Strategy:    restore.StrategyFullZIP,
		ArchivePath: archivePath,
		WebsiteName: "My Restored Site",
		WebsiteSlug: "my-restored-site",
	}

	_, err := res.Restore(t.Context(), req, nil)
	if err == nil {
		t.Fatal("expected restore error on import failure, got nil")
	}

	// Target directory should be rolled back and deleted
	targetDir := filepath.Join(websitesPath, "my-restored-site")
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		t.Errorf("expected target site directory to be cleaned up after failure, but still exists")
	}

	// Database should be dropped
	if mockClient.dbCreated {
		t.Errorf("expected database to be rolled back/dropped after failure")
	}

	// Source archive MUST remain intact
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("expected source archive to be preserved: %v", err)
	}
}

func TestRestorer_FullZIP_HerdWarningDoesNotRollback(t *testing.T) {
	tempRoot := t.TempDir()
	websitesPath := filepath.Join(tempRoot, "websites")
	_ = os.MkdirAll(websitesPath, 0755)

	archivePath := filepath.Join(tempRoot, "full_mysite_2026-09-18_12-00-00.zip")
	createValidZipArchive(t, archivePath, "mysite.sql", "wptest_")

	cfg := &config.Config{
		UsedHerd:     true,
		WebsitesPath: websitesPath,
		DatabaseHost: "localhost",
		DatabasePort: 3306,
		DBUsername:   "root",
	}

	mockClient := &mockRestoreWPClient{
		failOnOp: "herd_secure",
	}

	res := restore.NewRestorer(cfg, mockClient)
	res.SetAdminUpdater(&mockAdminUpdater{})
	req := restore.Request{
		Strategy:    restore.StrategyFullZIP,
		ArchivePath: archivePath,
		WebsiteName: "My Restored Site",
		WebsiteSlug: "my-restored-site",
	}

	result, err := res.Restore(t.Context(), req, nil)
	if err != nil {
		t.Fatalf("expected restore to succeed with warning, but got error: %v", err)
	}

	if result.TLSError == "" {
		t.Error("expected TLSError warning in result")
	}
	if result.SiteURL != "http://my-restored-site.test" {
		t.Errorf("got SiteURL %q, want 'http://my-restored-site.test'", result.SiteURL)
	}

	// Target directory must remain intact
	targetDir := filepath.Join(websitesPath, "my-restored-site")
	if _, err := os.Stat(targetDir); err != nil {
		t.Errorf("expected target site directory to exist: %v", err)
	}
}

func createValidZipArchive(t *testing.T, archivePath, sqlName, prefix string) {
	t.Helper()
	zf, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer zf.Close()

	zw := zip.NewWriter(zf)
	defer zw.Close()

	// Write WordPress core markers
	w, _ := zw.Create("wp-content/index.php")
	_, _ = w.Write([]byte("<?php"))
	w, _ = zw.Create("wp-includes/version.php")
	_, _ = w.Write([]byte("<?php"))
	w, _ = zw.Create("wp-settings.php")
	_, _ = w.Write([]byte("<?php"))

	// Write wp-config.php
	w, _ = zw.Create("wp-config.php")
	_, _ = w.Write([]byte(fmt.Sprintf("<?php\n$table_prefix = '%s';\n", prefix)))

	// Write SQL dump
	w, _ = zw.Create(sqlName)
	_, _ = w.Write([]byte("-- SQL Dump content\nCREATE TABLE `" + prefix + "users` (id int);\n"))
}
