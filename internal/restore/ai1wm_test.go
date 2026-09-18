package restore_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"wptui/internal/config"
	"wptui/internal/restore"
)

type mockExtensionInstaller struct {
	called bool
	fail   bool
}

func (m *mockExtensionInstaller) EnsureExtension(ctx context.Context, siteDir string) error {
	m.called = true
	if m.fail {
		return &mockInstallerError{}
	}
	return nil
}

type mockInstallerError struct{}

func (e *mockInstallerError) Error() string {
	return "simulated extension install error"
}

type mockCoreExtractor struct {
	called bool
	fail   bool
}

func (m *mockCoreExtractor) ExtractCore(ctx context.Context, destDir string) error {
	m.called = true
	if m.fail {
		return &mockInstallerError{}
	}
	// Write dummy core files to destination
	_ = os.WriteFile(filepath.Join(destDir, "index.php"), []byte("<?php"), 0644)
	return nil
}

func TestRestorer_AI1WM_Success(t *testing.T) {
	tempRoot := t.TempDir()
	websitesPath := filepath.Join(tempRoot, "websites")
	_ = os.MkdirAll(websitesPath, 0755)

	archivePath := filepath.Join(tempRoot, "ai1wm_mysite_2026-09-18_12-00-00.wpress")
	if err := os.WriteFile(archivePath, []byte("wpress binary content"), 0644); err != nil {
		t.Fatal(err)
	}

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
	extInstaller := &mockExtensionInstaller{}
	coreExtractor := &mockCoreExtractor{}

	res := restore.NewRestorer(cfg, mockClient)
	res.SetAdminUpdater(adminUpdater)
	res.SetExtensionInstaller(extInstaller)
	res.SetCoreExtractor(coreExtractor)

	req := restore.Request{
		Strategy:    restore.StrategyAI1WM,
		ArchivePath: archivePath,
		WebsiteName: "My AI1WM Site",
		WebsiteSlug: "my-ai1wm-site",
		AdminUser:   "restoredadmin",
		AdminPass:   "restoredpass",
		AdminEmail:  "restored@example.com",
	}

	result, err := res.Restore(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected AI1WM restore error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.SiteURL != "https://my-ai1wm-site.test" {
		t.Errorf("got SiteURL %q, want 'https://my-ai1wm-site.test'", result.SiteURL)
	}

	if !extInstaller.called {
		t.Error("expected AI1WM extension to be installed prior to restore")
	}
	if !coreExtractor.called {
		t.Error("expected core foundation to be extracted")
	}

	// Verify target site directory exists
	targetDir := filepath.Join(websitesPath, "my-ai1wm-site")
	if _, err := os.Stat(targetDir); err != nil {
		t.Errorf("expected target site directory to exist: %v", err)
	}

	// Verify staged .wpress was deleted from wp-content/ai1wm-backups/
	stagedWpress := filepath.Join(targetDir, "wp-content", "ai1wm-backups", filepath.Base(archivePath))
	if _, err := os.Stat(stagedWpress); !os.IsNotExist(err) {
		t.Errorf("expected staged .wpress copy to be deleted, but found it")
	}

	// Verify admin credentials update was invoked with user overrides
	if len(adminUpdater.updatedWith) < 4 || adminUpdater.updatedWith[1] != "restoredadmin" {
		t.Errorf("expected admin update with restoredadmin, got: %v", adminUpdater.updatedWith)
	}

	// Verify source .wpress file is untouched
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("expected source .wpress file to be preserved: %v", err)
	}
}

func TestRestorer_AI1WM_RollbackOnRestoreFailure(t *testing.T) {
	tempRoot := t.TempDir()
	websitesPath := filepath.Join(tempRoot, "websites")
	_ = os.MkdirAll(websitesPath, 0755)

	archivePath := filepath.Join(tempRoot, "ai1wm_mysite_2026-09-18_12-00-00.wpress")
	_ = os.WriteFile(archivePath, []byte("wpress binary content"), 0644)

	cfg := &config.Config{
		UsedHerd:     false,
		WebsitesPath: websitesPath,
		DatabaseHost: "localhost",
		DatabasePort: 3306,
		DBUsername:   "root",
	}

	mockClient := &mockRestoreWPClient{
		failOnOp: "ai1wm restore",
	}
	extInstaller := &mockExtensionInstaller{}
	coreExtractor := &mockCoreExtractor{}

	res := restore.NewRestorer(cfg, mockClient)
	res.SetExtensionInstaller(extInstaller)
	res.SetCoreExtractor(coreExtractor)

	req := restore.Request{
		Strategy:    restore.StrategyAI1WM,
		ArchivePath: archivePath,
		WebsiteName: "My AI1WM Site",
		WebsiteSlug: "my-ai1wm-site",
	}

	_, err := res.Restore(context.Background(), req, nil)
	if err == nil {
		t.Fatal("expected restore error on AI1WM restore failure, got nil")
	}

	// Target directory should be deleted
	targetDir := filepath.Join(websitesPath, "my-ai1wm-site")
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		t.Errorf("expected target site directory to be cleaned up after failure, but still exists")
	}

	// Database should be dropped
	if mockClient.dbCreated {
		t.Errorf("expected database to be dropped on failure")
	}

	// Source archive MUST remain intact
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("expected source archive to be preserved: %v", err)
	}
}
