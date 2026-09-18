package app_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wptui/internal/app"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/packages"
	"wptui/internal/tui"
)

type mockBackupPackageResolver struct{}

func (m *mockBackupPackageResolver) ResolvePackage(ctx context.Context, ref packages.PackageRef, stageDir string) (*packages.Artifact, error) {
	return &packages.Artifact{Ref: ref, Path: filepath.Join(stageDir, ref.Slug+".zip")}, nil
}
func (m *mockBackupPackageResolver) ResolveAll(ctx context.Context, refs []packages.PackageRef, stageDir string) ([]packages.Artifact, error) {
	arts := make([]packages.Artifact, 0, len(refs))
	for _, r := range refs {
		arts = append(arts, packages.Artifact{Ref: r, Path: filepath.Join(stageDir, r.Slug+".zip")})
	}
	return arts, nil
}

type mockBackupWPClient struct {
	runs []string
}

func (m *mockBackupWPClient) ConfigSet(ctx context.Context, dir, key, value string, raw bool) error {
	return nil
}
func (m *mockBackupWPClient) RewriteStructure(ctx context.Context, dir, value string) error {
	return nil
}
func (m *mockBackupWPClient) OptionUpdate(ctx context.Context, dir, key, value string) error {
	return nil
}
func (m *mockBackupWPClient) LanguageCore(ctx context.Context, dir, action, value string) error {
	return nil
}
func (m *mockBackupWPClient) PluginInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error {
	m.runs = append(m.runs, fmt.Sprintf("plugin install %s activate:%v", pathOrSlug, activate))
	return nil
}
func (m *mockBackupWPClient) ThemeInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error {
	return nil
}
func (m *mockBackupWPClient) Run(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
	call := fmt.Sprintf("%s %s in %s", name, strings.Join(args, " "), dir)
	m.runs = append(m.runs, call)

	if len(args) >= 3 && args[0] == "db" && args[1] == "export" {
		sqlFile := filepath.Join(dir, args[2])
		_ = os.WriteFile(sqlFile, []byte("-- dump"), 0644)
		return "Success: Exported to " + args[2], "", nil
	}

	if len(args) >= 3 && args[0] == "plugin" && args[1] == "get" {
		return "1.0.0", "", nil
	}

	if len(args) >= 2 && args[0] == "ai1wm" && args[1] == "backup" {
		wpressPath := filepath.Join(dir, "wp-content", "ai1wm-backups", "site.wpress")
		_ = os.MkdirAll(filepath.Dir(wpressPath), 0755)
		_ = os.WriteFile(wpressPath, []byte("wpress"), 0644)
		return "Backup in progress...\nSuccess: Backup complete.\nBackup location: " + wpressPath, "", nil
	}

	return "", "", nil
}

func TestApp_BackupMenuRouting(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = tempDir
	cfg.BackupPath = filepath.Join(tempDir, "backups")
	cfgPath, _ := config.ConfigPath(tempDir)
	_ = config.Save(cfgPath, cfg)

	backupCalled := false
	menuCalls := 0

	application := app.New(app.Options{
		HomeDir: tempDir,
		MenuFn: func() (string, error) {
			menuCalls++
			if menuCalls == 1 {
				return "backup", nil
			}
			return "exit", nil
		},
		BackupFn: func(ctx context.Context, c *config.Config) error {
			backupCalled = true
			return nil
		},
	})

	err := application.RunWithContext(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !backupCalled {
		t.Errorf("expected backupFn to be called on 'backup' menu selection")
	}
}

func TestRunBackupFlowWithDeps_MissingBackupPathReturnsError(t *testing.T) {
	cfg := &config.Config{
		WebsitesPath: t.TempDir(),
		BackupPath:   "", // missing backup_path must error out without fallback
	}

	err := app.RunBackupFlowWithDeps(context.Background(), cfg, app.BackupFlowDependencies{})
	if err == nil {
		t.Fatal("expected error when backup_path is empty, got nil")
	}
	if !strings.Contains(err.Error(), "backup_path is not configured") {
		t.Errorf("expected missing backup_path error, got: %v", err)
	}
}

func TestRunBackupFlowWithDeps_FullBackupExecution(t *testing.T) {
	tempDir := t.TempDir()
	siteDir := filepath.Join(tempDir, "mysite")
	_ = os.MkdirAll(siteDir, 0755)
	_ = os.WriteFile(filepath.Join(siteDir, "index.php"), []byte("<?php"), 0644)

	backupDir := filepath.Join(tempDir, "mybackups")
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = tempDir
	cfg.BackupPath = backupDir

	mockCli := &mockBackupWPClient{}
	siteSelectionCount := 0

	deps := app.BackupFlowDependencies{
		WPClient: mockCli,
		SelectWebsite: func(candidates []deprovision.Candidate) (*deprovision.Candidate, error) {
			siteSelectionCount++
			if siteSelectionCount > 1 {
				return nil, nil // user backed out to main menu
			}
			return &candidates[0], nil
		},
		SelectStrategy: func(slug string) (tui.BackupStrategyAction, error) {
			return tui.ActionBackupFull, nil
		},
	}

	err := app.RunBackupFlowWithDeps(context.Background(), cfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify temporary SQL dump is cleaned up
	expectedSQL := filepath.Join(siteDir, "mysite.sql")
	if _, err := os.Stat(expectedSQL); !os.IsNotExist(err) {
		t.Errorf("expected %s to be deleted after backup", expectedSQL)
	}

	// Verify backup archive exists in backupDir
	entries, _ := os.ReadDir(backupDir)
	foundZip := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "full_mysite_") && strings.HasSuffix(e.Name(), ".zip") {
			foundZip = true
		}
	}
	if !foundZip {
		t.Errorf("expected full_mysite_*.zip in %s, found: %v", backupDir, entries)
	}
}

func TestRunBackupFlowWithDeps_AI1WMBackupExecution(t *testing.T) {
	tempDir := t.TempDir()
	siteDir := filepath.Join(tempDir, "ai1site")
	_ = os.MkdirAll(siteDir, 0755)
	_ = os.WriteFile(filepath.Join(siteDir, "index.php"), []byte("<?php"), 0644)

	backupDir := filepath.Join(tempDir, "ai1backups")
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = tempDir
	cfg.BackupPath = backupDir

	mockCli := &mockBackupWPClient{}
	mockRes := &mockBackupPackageResolver{}
	siteSelectionCount := 0

	deps := app.BackupFlowDependencies{
		WPClient: mockCli,
		Resolver: mockRes,
		SelectWebsite: func(candidates []deprovision.Candidate) (*deprovision.Candidate, error) {
			siteSelectionCount++
			if siteSelectionCount > 1 {
				return nil, nil // user backed out to main menu
			}
			return &candidates[0], nil
		},
		SelectStrategy: func(slug string) (tui.BackupStrategyAction, error) {
			return tui.ActionBackupAI1WM, nil
		},
	}

	err := app.RunBackupFlowWithDeps(context.Background(), cfg, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify .wpress archive exists in backupDir
	entries, _ := os.ReadDir(backupDir)
	foundWpress := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "ai1wm_ai1site_") && strings.HasSuffix(e.Name(), ".wpress") {
			foundWpress = true
		}
	}
	if !foundWpress {
		t.Errorf("expected ai1wm_ai1site_*.wpress in %s, found: %v", backupDir, entries)
	}
}
