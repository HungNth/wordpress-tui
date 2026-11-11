package app_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"wptui/internal/app"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/packages"
	"wptui/internal/restore"
	"wptui/internal/tui"
)

func TestApp_LoadExistingConfig(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")

	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.DefaultAdminUsername = "tester"
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	application := app.New(app.Options{
		HomeDir: tempHome,
	})

	loadedCfg, err := application.InitConfig(false)
	if err != nil {
		t.Fatalf("InitConfig() failed: %v", err)
	}

	if loadedCfg.DefaultAdminUsername != "tester" {
		t.Errorf("expected loaded config admin username tester, got %s", loadedCfg.DefaultAdminUsername)
	}
}

func TestApp_InvalidConfigHalts(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("{ malformed json }"), 0600); err != nil {
		t.Fatal(err)
	}

	application := app.New(app.Options{
		HomeDir: tempHome,
	})

	_, err := application.InitConfig(false)
	if err == nil {
		t.Errorf("expected error when loading malformed config, got nil")
	}
}

func TestRunCreateFlow_CollisionCheckRejectsBeforePackageSelection(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	// Pre-create directory to trigger collision
	collidedDir := filepath.Join(cfg.WebsitesPath, "taken-site")
	if err := os.MkdirAll(collidedDir, 0755); err != nil {
		t.Fatal(err)
	}

	var packagePromptCalled bool
	deps := app.CreateFlowDependencies{
		PromptCreate: func(c *config.Config, checker ...tui.SlugAvailabilityChecker) (*tui.CreateInputs, error) {
			// Exercise the actual injected availabilityChecker!
			if len(checker) > 0 && checker[0] != nil {
				if err := checker[0]("taken-site"); err != nil {
					return nil, err
				}
			}
			return &tui.CreateInputs{
				WebsiteName: "Taken Site",
				WebsiteSlug: "taken-site",
			}, nil
		},
		PromptPackages: func(ctx context.Context, c *config.Config, items []packages.CatalogItem) ([]string, []string, error) {
			packagePromptCalled = true
			return nil, nil, nil
		},
	}

	err := app.RunCreateFlowWithDeps(context.Background(), cfg, deps)
	if err == nil {
		t.Fatal("expected error on colliding directory, got nil")
	}
	if packagePromptCalled {
		t.Fatal("PromptPackages should not have been called when slug collided")
	}
}

func TestRunCreateFlow_CustomPromptBypassCaughtBeforePackages(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	// Pre-create directory to trigger collision
	collidedDir := filepath.Join(cfg.WebsitesPath, "bypassed-slug")
	if err := os.MkdirAll(collidedDir, 0755); err != nil {
		t.Fatal(err)
	}

	var packagePromptCalled bool
	deps := app.CreateFlowDependencies{
		// Custom prompt callback that completely ignores the injected checker and returns a colliding slug
		PromptCreate: func(c *config.Config, checker ...tui.SlugAvailabilityChecker) (*tui.CreateInputs, error) {
			return &tui.CreateInputs{
				WebsiteName: "Bypassed Site",
				WebsiteSlug: "bypassed-slug",
			}, nil
		},
		PromptPackages: func(ctx context.Context, c *config.Config, items []packages.CatalogItem) ([]string, []string, error) {
			packagePromptCalled = true
			return nil, nil, nil
		},
	}

	err := app.RunCreateFlowWithDeps(context.Background(), cfg, deps)
	if err == nil {
		t.Fatal("expected collision error when custom prompt bypasses checker, got nil")
	}
	if packagePromptCalled {
		t.Fatal("PromptPackages should not have been called when custom prompt returned colliding slug")
	}
}

type mockDeleteWPClient struct {
	mu                         sync.Mutex
	configGetCalls             []string
	dbDropCalls                []string
	environmentUnsecureCalls   []string
	environmentUnregisterCalls []string
}

func (m *mockDeleteWPClient) ConfigGet(ctx context.Context, dir, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configGetCalls = append(m.configGetCalls, dir+":"+key)
	return "site_one_db", nil
}

func (m *mockDeleteWPClient) DBDrop(ctx context.Context, dir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dbDropCalls = append(m.dbDropCalls, dir)
	return nil
}

func (m *mockDeleteWPClient) EnvironmentUnsecure(ctx context.Context, dir, slug string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.environmentUnsecureCalls = append(m.environmentUnsecureCalls, slug)
	return nil
}

func (m *mockDeleteWPClient) EnvironmentUnregister(ctx context.Context, dir, slug string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.environmentUnregisterCalls = append(m.environmentUnregisterCalls, slug)
	return nil
}

func TestRunDeleteFlow_EndToEndSingleSite(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	// 1. When websites_path is empty, returns gracefully
	mockClient := &mockDeleteWPClient{}
	deps := app.DeleteFlowDependencies{
		WPClient: mockClient,
		Select: func(candidates []deprovision.Candidate) ([]deprovision.Candidate, error) {
			return candidates, nil
		},
		Confirm: func(selected []deprovision.Candidate) (bool, error) {
			return true, nil
		},
	}

	if err := app.RunDeleteFlowWithDeps(context.Background(), cfg, deps); err != nil {
		t.Fatalf("expected nil error on empty websites_path, got %v", err)
	}

	// 2. Pre-create a website directory with wp-config.php
	siteDir := filepath.Join(cfg.WebsitesPath, "site-one")
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(siteDir, "wp-config.php"), []byte("<?php"), 0644); err != nil {
		t.Fatal(err)
	}

	var confirmedSelected []deprovision.Candidate
	deps.Confirm = func(selected []deprovision.Candidate) (bool, error) {
		confirmedSelected = selected
		return true, nil
	}

	if err := app.RunDeleteFlowWithDeps(context.Background(), cfg, deps); err != nil {
		t.Fatalf("RunDeleteFlowWithDeps failed: %v", err)
	}

	// Verify candidate details
	if len(confirmedSelected) != 1 || confirmedSelected[0].Slug != "site-one" {
		t.Errorf("expected confirmed slug 'site-one', got %+v", confirmedSelected)
	}
	if confirmedSelected[0].DetectedDB != "site_one_db" {
		t.Errorf("expected detected DB 'site_one_db', got %q", confirmedSelected[0].DetectedDB)
	}

	// Verify WPClient calls
	mockClient.mu.Lock()
	defer mockClient.mu.Unlock()
	if len(mockClient.environmentUnsecureCalls) != 1 || mockClient.environmentUnsecureCalls[0] != "site-one" {
		t.Errorf("expected environment unsecure call for site-one, got %+v", mockClient.environmentUnsecureCalls)
	}
	if len(mockClient.dbDropCalls) != 1 || mockClient.dbDropCalls[0] != siteDir {
		t.Errorf("expected db drop call for %s, got %+v", siteDir, mockClient.dbDropCalls)
	}

	// Verify directory was removed from disk
	if _, err := os.Stat(siteDir); !os.IsNotExist(err) {
		t.Errorf("expected site directory %s to be deleted from disk, but it still exists", siteDir)
	}
}

func TestRunDeleteFlow_ComposedMultiSite(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	// Create 3 sites
	siteNames := []string{"site-alpha", "site-beta", "site-gamma"}
	var siteDirs []string
	for _, name := range siteNames {
		dir := filepath.Join(cfg.WebsitesPath, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "wp-config.php"), []byte("<?php"), 0644); err != nil {
			t.Fatal(err)
		}
		siteDirs = append(siteDirs, dir)
	}

	mockClient := &mockDeleteWPClient{}
	deps := app.DeleteFlowDependencies{
		WPClient: mockClient,
		Select: func(candidates []deprovision.Candidate) ([]deprovision.Candidate, error) {
			// Select all 3 candidates
			return candidates, nil
		},
		Confirm: func(selected []deprovision.Candidate) (bool, error) {
			if len(selected) != 3 {
				t.Errorf("expected 3 selected candidates in confirmation, got %d", len(selected))
			}
			return true, nil
		},
	}

	if err := app.RunDeleteFlowWithDeps(context.Background(), cfg, deps); err != nil {
		t.Fatalf("RunDeleteFlowWithDeps multi-site failed: %v", err)
	}

	// Verify all 3 directories removed from disk
	for _, dir := range siteDirs {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("expected site directory %s to be deleted from disk", dir)
		}
	}

	mockClient.mu.Lock()
	defer mockClient.mu.Unlock()
	if len(mockClient.environmentUnsecureCalls) != 3 {
		t.Errorf("expected 3 environment unsecure calls, got %d", len(mockClient.environmentUnsecureCalls))
	}
	if len(mockClient.dbDropCalls) != 3 {
		t.Errorf("expected 3 db drop calls, got %d", len(mockClient.dbDropCalls))
	}
}

func TestApp_RunWithContext_DispatchesDelete(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}

	var deleteCalled bool
	menuSelections := []string{"delete", "exit"}
	menuIdx := 0

	application := app.New(app.Options{
		HomeDir: tempHome,
		MenuFn: func() (string, error) {
			sel := menuSelections[menuIdx]
			menuIdx++
			return sel, nil
		},
		DeleteFn: func(ctx context.Context, c *config.Config) error {
			deleteCalled = true
			return nil
		},
	})

	if err := application.RunWithContext(context.Background()); err != nil {
		t.Fatalf("RunWithContext failed: %v", err)
	}

	if !deleteCalled {
		t.Errorf("expected DeleteFn to be dispatched by RunWithContext when menu returned 'delete'")
	}
}

func TestApp_RunWithContext_DispatchesRestore(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	var restoreCalled bool
	menuSelections := []string{"restore", "exit"}
	menuIdx := 0

	application := app.New(app.Options{
		HomeDir: tempHome,
		MenuFn: func() (string, error) {
			sel := menuSelections[menuIdx]
			menuIdx++
			return sel, nil
		},
		RestoreFn: func(ctx context.Context, c *config.Config) error {
			restoreCalled = true
			return nil
		},
	})

	if err := application.RunWithContext(context.Background()); err != nil {
		t.Fatalf("RunWithContext failed: %v", err)
	}

	if !restoreCalled {
		t.Errorf("expected RestoreFn to be dispatched by RunWithContext when menu returned 'restore'")
	}
}

func TestApp_RunWithContext_MasterDetail_AllSections(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	var runnerInvoked bool
	application := app.New(app.Options{
		HomeDir: tempHome,
		AppRunner: func(ctx context.Context, model tea.Model) error {
			runnerInvoked = true
			appModel, ok := model.(*tui.AppModel)
			if !ok {
				t.Fatalf("expected model to be *tui.AppModel, got %T", model)
			}

			// 1. Initial section is Websites
			if appModel.ActiveSection() != tui.SectionWebsites {
				t.Errorf("expected initial section SectionWebsites, got %v", appModel.ActiveSection())
			}
			if appModel.Focus() != tui.FocusSidebar {
				t.Errorf("expected initial focus FocusSidebar, got %v", appModel.Focus())
			}

			// 2. Navigate down to Create
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			if appModel.ActiveSection() != tui.SectionCreate {
				t.Errorf("expected section SectionCreate after down arrow, got %v", appModel.ActiveSection())
			}

			// 3. Navigate down to Delete
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			if appModel.ActiveSection() != tui.SectionDelete {
				t.Errorf("expected section SectionDelete after down arrow, got %v", appModel.ActiveSection())
			}

			// 4. Navigate down to Restore
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			if appModel.ActiveSection() != tui.SectionRestore {
				t.Errorf("expected section SectionRestore after down arrow, got %v", appModel.ActiveSection())
			}

			// 5. Navigate down to Settings
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			if appModel.ActiveSection() != tui.SectionSettings {
				t.Errorf("expected section SectionSettings after down arrow, got %v", appModel.ActiveSection())
			}

			// 6. Navigate down to Exit
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			if appModel.ActiveSection() != tui.SectionExit {
				t.Errorf("expected section SectionExit after down arrow, got %v", appModel.ActiveSection())
			}

			// 7. Enter on Exit returns tea.Quit
			_, cmd := appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if cmd == nil {
				t.Errorf("expected tea.Quit cmd on Exit enter, got nil")
			}

			// 8. Verify clean teardown with 'q'
			_, qCmd := appModel.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
			if qCmd == nil {
				t.Errorf("expected tea.Quit cmd on 'q', got nil")
			}

			return nil
		},
	})

	if err := application.RunWithContext(context.Background()); err != nil {
		t.Fatalf("RunWithContext failed: %v", err)
	}

	if !runnerInvoked {
		t.Errorf("expected AppRunner to be invoked by RunWithContext")
	}
}

func TestApp_RunWithContext_MasterDetail_RestoreFlow(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.BackupPath = filepath.Join(tempHome, "backups")
	if err := os.MkdirAll(cfg.BackupPath, 0755); err != nil {
		t.Fatal(err)
	}
	// Create dummy zip
	dummyZip := filepath.Join(cfg.BackupPath, "test-backup.zip")
	if err := os.WriteFile(dummyZip, []byte("dummy zip content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	application := app.New(app.Options{
		HomeDir: tempHome,
		AppRunner: func(ctx context.Context, model tea.Model) error {
			appModel := model.(*tui.AppModel)

			// Navigate down to Restore
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Create
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Delete
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Restore
			if appModel.ActiveSection() != tui.SectionRestore {
				t.Fatalf("expected SectionRestore, got %v", appModel.ActiveSection())
			}

			// Enter content pane
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if appModel.Focus() != tui.FocusContent {
				t.Fatalf("expected FocusContent, got %v", appModel.Focus())
			}

			rw := appModel.RestoreWizard()
			if rw == nil {
				t.Fatalf("expected RestoreWizard to be initialized")
			}
			if rw.Step() != tui.RestoreWizardStepFormat {
				t.Errorf("expected RestoreWizardStepFormat, got %v", rw.Step())
			}

			// Step 1: Select format (Enter)
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if rw.Step() != tui.RestoreWizardStepArchive {
				t.Errorf("expected RestoreWizardStepArchive, got %v", rw.Step())
			}

			// Step 2: First option is [Enter custom path...]; move down once to select scanned archive
			viewStep2 := appModel.ViewString()
			if !strings.Contains(viewStep2, tui.CustomPathOption) {
				t.Fatalf("expected %s in archive list, got:\n%s", tui.CustomPathOption, viewStep2)
			}
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if rw.Step() != tui.RestoreWizardStepInputs {
				t.Errorf("expected RestoreWizardStepInputs, got %v", rw.Step())
			}
			// Esc returns back to archive
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
			if rw.Step() != tui.RestoreWizardStepArchive {
				t.Errorf("expected RestoreWizardStepArchive after Esc, got %v", rw.Step())
			}

			// Esc returns to format
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
			if rw.Step() != tui.RestoreWizardStepFormat {
				t.Errorf("expected RestoreWizardStepFormat after Esc, got %v", rw.Step())
			}

			// Esc returns to Sidebar
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
			if appModel.Focus() != tui.FocusSidebar {
				t.Errorf("expected focus to return to FocusSidebar, got %v", appModel.Focus())
			}

			return nil
		},
	})

	if err := application.RunWithContext(t.Context()); err != nil {
		t.Fatalf("RunWithContext failed: %v", err)
	}
}

func TestApp_RunWithContext_RestoreCustomArchivePathFlow(t *testing.T) {
	tempHome := t.TempDir()
	externalDir := filepath.Join(t.TempDir(), "Downloads", "Client Exports")
	if err := os.MkdirAll(externalDir, 0755); err != nil {
		t.Fatal(err)
	}
	externalZip := filepath.Join(externalDir, "Production Export (2026).ZIP")
	origBytes := []byte("PK\x03\x04external-archive-payload")
	if err := os.WriteFile(externalZip, origBytes, 0644); err != nil {
		t.Fatal(err)
	}
	wrongExtFile := filepath.Join(externalDir, "migration.wpress")
	if err := os.WriteFile(wrongExtFile, []byte("wpress-payload"), 0644); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	// Configure a missing backup directory to prove non-blocking scan feedback
	cfg.BackupPath = filepath.Join(tempHome, "missing-backups-dir")
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	var capturedStrategy string
	var capturedArchive string
	var capturedSlug string
	restoreCalls := 0

	application := app.New(app.Options{
		HomeDir: tempHome,
		AppRunner: func(ctx context.Context, model tea.Model) error {
			appModel := model.(*tui.AppModel)
			appModel.SetRestoreDependencies(func(slug string) error { return nil })
			appModel.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
			appModel.SetRestoreHandler(func(strategy restore.Strategy, archivePath string, inputs tui.CreateInputs) tea.Cmd {
				restoreCalls++
				capturedStrategy = string(strategy)
				capturedArchive = archivePath
				capturedSlug = inputs.WebsiteSlug
				return appModel.StartProgress("Restoring Website: "+inputs.WebsiteName, nil)
			})

			// Navigate Sidebar to Restore and enter Content Pane
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

			// Select Full ZIP format -> Archive step
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			viewArchiveList := appModel.ViewString()
			if !strings.Contains(viewArchiveList, tui.CustomPathOption) {
				t.Fatalf("expected custom path option in view despite missing backup_path, got:\n%s", viewArchiveList)
			}
			if !strings.Contains(strings.ToLower(viewArchiveList), "backup directory") {
				t.Errorf("expected non-blocking missing backup directory notice in view, got:\n%s", viewArchiveList)
			}

			// Press Enter on [Enter custom path...] -> enters custom path text mode within Step 2
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if appModel.RestoreWizard().Step() != tui.RestoreWizardStepArchive {
				t.Fatalf("expected custom entry to stay in RestoreWizardStepArchive, got %v", appModel.RestoreWizard().Step())
			}

			// Typing partial input must NOT trigger filesystem validation before Enter
			appModel.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
			appModel.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
			appModel.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
			if strings.Contains(appModel.ViewString(), "✗") {
				t.Fatalf("expected no validation error before pressing Enter, got:\n%s", appModel.ViewString())
			}

			// Pressing Enter on relative path shows inline error and stays in custom input
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if appModel.RestoreWizard().Step() != tui.RestoreWizardStepArchive {
				t.Fatalf("expected invalid relative path to remain on archive step")
			}
			if !strings.Contains(strings.ToLower(appModel.ViewString()), "absolute") {
				t.Errorf("expected absolute-path error in view, got:\n%s", appModel.ViewString())
			}

			// Clear typed characters and paste wrong-extension file -> names .zip on Enter
			for range 3 {
				appModel.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
			}
			appModel.Update(tea.PasteMsg{Content: wrongExtFile})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if appModel.RestoreWizard().Step() != tui.RestoreWizardStepArchive {
				t.Fatalf("expected wrong extension to remain on archive step")
			}
			if !strings.Contains(appModel.ViewString(), ".zip") {
				t.Errorf("expected error naming .zip extension, got:\n%s", appModel.ViewString())
			}

			// Clear field (Ctrl+U) and paste Windows Copy-as-path quoted archive path
			appModel.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
			quotedCopyAsPath := `"` + externalZip + `"`
			appModel.Update(tea.PasteMsg{Content: quotedCopyAsPath})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

			if appModel.RestoreWizard().Step() != tui.RestoreWizardStepInputs {
				t.Fatalf("expected valid quoted path to advance to RestoreWizardStepInputs, got %v;\nview:\n%s",
					appModel.RestoreWizard().Step(), appModel.ViewString())
			}

			// Full normalized path (including parent directory, without outer quotes) is shown
			viewInputs := appModel.ViewString()
			if !strings.Contains(viewInputs, "Client Exports") ||
				!strings.Contains(viewInputs, "Production Export") ||
				!strings.Contains(viewInputs, "(2026).ZIP") {
				t.Errorf("expected full normalized archive path in target parameters view, got:\n%s", viewInputs)
			}
			if strings.Contains(viewInputs, `"`+externalZip+`"`) {
				t.Errorf("expected outer double quotes stripped from displayed archive path, got:\n%s", viewInputs)
			}

			// Enter Website Name and submit restoration
			for _, ch := range "Imported Client" {
				appModel.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
			}
			for range 5 {
				appModel.Update(tea.KeyPressMsg{Code: tea.KeyTab})
			}
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

			if restoreCalls != 1 {
				t.Fatalf("expected restore handler called once, got %d", restoreCalls)
			}
			if capturedStrategy != string(restore.StrategyFullZIP) {
				t.Errorf("expected strategy %q, got %q", restore.StrategyFullZIP, capturedStrategy)
			}
			if capturedArchive != externalZip {
				t.Errorf("expected normalized archive path %q, got %q", externalZip, capturedArchive)
			}
			if capturedSlug != "imported-client" {
				t.Errorf("expected slug 'imported-client', got %q", capturedSlug)
			}
			if !appModel.IsExecuting() {
				t.Errorf("expected appModel.IsExecuting() to be true after valid restore start")
			}
			return nil
		},
	})

	if err := application.RunWithContext(t.Context()); err != nil {
		t.Fatalf("RunWithContext failed: %v", err)
	}

	// External archive must remain untouched in place and not copied into backup_path
	gotBytes, err := os.ReadFile(externalZip)
	if err != nil || string(gotBytes) != string(origBytes) {
		t.Fatalf("expected external archive to remain untouched, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.BackupPath, filepath.Base(externalZip))); !os.IsNotExist(err) {
		t.Errorf("expected external archive not to be copied into backup_path")
	}
}

func TestApp_RunWithContext_RestoreRevalidationAndPreservation(t *testing.T) {
	tempHome := t.TempDir()
	backupDir := filepath.Join(tempHome, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}
	listedZip := filepath.Join(backupDir, "listed-site.zip")
	if err := os.WriteFile(listedZip, []byte("PK\x03\x04listed"), 0644); err != nil {
		t.Fatal(err)
	}
	secondListedZip := filepath.Join(backupDir, "second-listed.zip")
	if err := os.WriteFile(secondListedZip, []byte("PK\x03\x04second"), 0644); err != nil {
		t.Fatal(err)
	}

	externalDir := t.TempDir()
	transientCustomZip := filepath.Join(externalDir, "transient-custom.zip")
	if err := os.WriteFile(transientCustomZip, []byte("PK\x03\x04custom"), 0644); err != nil {
		t.Fatal(err)
	}
	replacementCustomZip := filepath.Join(externalDir, "replacement-custom.zip")
	if err := os.WriteFile(replacementCustomZip, []byte("PK\x03\x04replacement"), 0644); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.BackupPath = backupDir
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	restoreCalls := 0
	checkerCalls := 0
	var finalArchive string
	var finalInputs tui.CreateInputs

	application := app.New(app.Options{
		HomeDir: tempHome,
		AppRunner: func(ctx context.Context, model tea.Model) error {
			appModel := model.(*tui.AppModel)
			appModel.SetRestoreDependencies(func(slug string) error {
				checkerCalls++
				return nil
			})
			appModel.SetRestoreHandler(func(strategy restore.Strategy, archivePath string, inputs tui.CreateInputs) tea.Cmd {
				restoreCalls++
				finalArchive = archivePath
				finalInputs = inputs
				return appModel.StartProgress("Restoring "+inputs.WebsiteSlug, nil)
			})

			// Navigate to Restore -> Format -> Archive list
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

			// 1. Stale listed archive deleted after scan fails inline on Enter
			if err := os.Remove(listedZip); err != nil {
				t.Fatal(err)
			}
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // listed-site.zip
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if appModel.RestoreWizard().Step() != tui.RestoreWizardStepArchive {
				t.Fatalf("expected stale listed archive to remain on archive step")
			}
			if !strings.Contains(appModel.ViewString(), "[✗]") {
				t.Errorf("expected non-color error indicator [✗] for stale listed archive, got:\n%s", appModel.ViewString())
			}

			// 2. Select valid listed archive, fill all 5 fields, then delete archive before final submit
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // second-listed.zip
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if appModel.RestoreWizard().Step() != tui.RestoreWizardStepInputs {
				t.Fatalf("expected valid listed archive to advance to inputs step")
			}
			for _, ch := range "Preserved Site" {
				appModel.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
			}
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // Slug
			for _, ch := range "preserved-slug" {
				appModel.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
			}
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // Admin Username
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
			appModel.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // Admin Password
			appModel.Update(tea.KeyPressMsg{Code: '!', Text: "!"})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // Admin Email
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // Submit button

			if err := os.Remove(secondListedZip); err != nil {
				t.Fatal(err)
			}
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

			if checkerCalls != 0 {
				t.Fatalf("expected missing archive revalidation to run before slug/database checker, checkerCalls=%d", checkerCalls)
			}
			if restoreCalls != 0 || appModel.IsExecuting() {
				t.Fatalf("expected deleted archive to block restore execution, calls=%d executing=%v", restoreCalls, appModel.IsExecuting())
			}
			if appModel.RestoreWizard().Step() != tui.RestoreWizardStepArchive || appModel.RestoreWizard().IsEnteringCustomPath() {
				t.Fatalf("expected return to archive list mode for deleted listed archive")
			}
			if !strings.Contains(strings.ToLower(appModel.ViewString()), "does not exist") {
				t.Errorf("expected missing-archive error on return to archive step, got:\n%s", appModel.ViewString())
			}

			// 3. Switch to custom archive, advance, delete custom archive before submit -> returns to custom input preserving all target fields
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyUp})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyUp}) // [Enter custom path...]
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			appModel.Update(tea.PasteMsg{Content: `"` + transientCustomZip + `"`})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if appModel.RestoreWizard().Step() != tui.RestoreWizardStepInputs {
				t.Fatalf("expected transient custom archive to advance to inputs")
			}

			gotInputs := appModel.RestoreWizard().Inputs()
			if gotInputs.WebsiteName != "Preserved Site" || gotInputs.WebsiteSlug != "preserved-slug" || gotInputs.AdminUsername != "admix" || gotInputs.AdminPassword != "admin!" {
				t.Fatalf("expected all target inputs preserved, got %+v", gotInputs)
			}

			if err := os.Remove(transientCustomZip); err != nil {
				t.Fatal(err)
			}
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // Submit button still focused
			if restoreCalls != 0 || appModel.IsExecuting() || checkerCalls != 0 {
				t.Fatalf("expected deleted custom archive to block restore execution and checker")
			}
			if !appModel.RestoreWizard().IsEnteringCustomPath() {
				t.Fatalf("expected final revalidation failure on custom source to return to custom path input")
			}
			if !strings.Contains(strings.ToLower(appModel.ViewString()), "does not exist") {
				t.Errorf("expected missing custom archive error inline, got:\n%s", appModel.ViewString())
			}

			// Replace with valid custom archive and submit without retyping target parameters
			appModel.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
			appModel.Update(tea.PasteMsg{Content: replacementCustomZip})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if appModel.RestoreWizard().Step() != tui.RestoreWizardStepInputs {
				t.Fatalf("expected replacement custom archive to return to target inputs")
			}
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // Submit

			if restoreCalls != 1 || checkerCalls != 1 {
				t.Fatalf("expected restore and checker called once after fixing archive, got restore=%d checker=%d", restoreCalls, checkerCalls)
			}
			if finalArchive != replacementCustomZip {
				t.Errorf("expected replacement archive %q, got %q", replacementCustomZip, finalArchive)
			}
			if finalInputs.WebsiteName != "Preserved Site" || finalInputs.WebsiteSlug != "preserved-slug" || finalInputs.AdminUsername != "admix" || finalInputs.AdminPassword != "admin!" {
				t.Errorf("expected preserved inputs submitted intact, got %+v", finalInputs)
			}
			return nil
		},
	})

	if err := application.RunWithContext(t.Context()); err != nil {
		t.Fatalf("RunWithContext failed: %v", err)
	}
}

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func assertFitsViewport(t *testing.T, label, view string, maxWidth, maxHeight int) {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(view, "\n"), "\n")
	if len(lines) > maxHeight {
		t.Errorf("%s exceeded viewport height %d (got %d lines):\n%s", label, maxHeight, len(lines), view)
	}
	for idx, line := range lines {
		if w := lipgloss.Width(line); w > maxWidth {
			t.Errorf("%s line %d exceeded viewport width %d (got %d): %q", label, idx+1, maxWidth, w, line)
		}
	}
}

func TestApp_RunWithContext_RestoreAI1WM80x24AndDirtyNavigation(t *testing.T) {
	tempHome := t.TempDir()
	emptyBackupDir := filepath.Join(tempHome, "empty-backups")
	if err := os.MkdirAll(emptyBackupDir, 0755); err != nil {
		t.Fatal(err)
	}
	externalDir := filepath.Join(t.TempDir(), "external-exports", "agency-deliverables-2026", "wordpress-migrations")
	if err := os.MkdirAll(externalDir, 0755); err != nil {
		t.Fatal(err)
	}
	ai1wmFile := filepath.Join(externalDir, "third-party-migration-export.WPRESS")
	if err := os.WriteFile(ai1wmFile, []byte("ai1wm-payload"), 0644); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.BackupPath = emptyBackupDir
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	var capturedStrategy restore.Strategy
	var capturedArchive string

	application := app.New(app.Options{
		HomeDir: tempHome,
		AppRunner: func(ctx context.Context, model tea.Model) error {
			appModel := model.(*tui.AppModel)
			appModel.SetRestoreDependencies(func(slug string) error { return nil })
			appModel.SetRestoreHandler(func(strategy restore.Strategy, archivePath string, inputs tui.CreateInputs) tea.Cmd {
				capturedStrategy = strategy
				capturedArchive = archivePath
				return appModel.StartProgress("Restoring "+inputs.WebsiteSlug, nil)
			})
			appModel.Update(tea.WindowSizeMsg{Width: tui.MinTerminalWidth, Height: tui.MinTerminalHeight})

			// Navigate to Restore -> select AI1WM (.wpress)
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // AI1WM
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

			// Empty existing backup_path shows custom option without scan warning [!]
			viewEmpty := appModel.ViewString()
			assertFitsViewport(t, "80x24 empty archive list", viewEmpty, tui.MinTerminalWidth, tui.MinTerminalHeight)
			if !strings.Contains(viewEmpty, tui.CustomPathOption) {
				t.Fatalf("expected custom path option when backup directory is empty")
			}
			if strings.Contains(viewEmpty, "[!]") {
				t.Errorf("expected no scan error [!] for existing empty directory, got:\n%s", viewEmpty)
			}

			// Open custom path, test invalid unmatched quote in colorless output, then paste quoted .WPRESS path
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			appModel.Update(tea.PasteMsg{Content: `"` + ai1wmFile})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			viewInvalid := appModel.ViewString()
			assertFitsViewport(t, "80x24 custom path error", viewInvalid, tui.MinTerminalWidth, tui.MinTerminalHeight)
			plainInvalid := ansiEscapePattern.ReplaceAllString(viewInvalid, "")
			if !strings.Contains(plainInvalid, "> Archive Path:") || !strings.Contains(plainInvalid, "[✗]") || !strings.Contains(plainInvalid, "Esc Back to Archives") {
				t.Errorf("expected colorless readability with >, [✗], and contextual footer, got:\n%s", plainInvalid)
			}

			// Fix closing quote and confirm -> Step 3 at 80x24
			appModel.Update(tea.KeyPressMsg{Code: '"', Text: `"`})
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if appModel.RestoreWizard().Step() != tui.RestoreWizardStepInputs {
				t.Fatalf("expected AI1WM custom archive to advance to inputs")
			}

			// Trigger inline validation error on Step 3 with long wrapped archive path at 80x24
			for range 5 {
				appModel.Update(tea.KeyPressMsg{Code: tea.KeyTab})
			}
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // empty Website Name -> inline error
			viewInputsErr := appModel.ViewString()
			assertFitsViewport(t, "80x24 inputs with long archive and error", viewInputsErr, tui.MinTerminalWidth, tui.MinTerminalHeight)
			if strings.Contains(viewInputsErr, "...") {
				t.Errorf("expected full normalized archive path wrapped without ellipsis truncation, got:\n%s", viewInputsErr)
			}

			// Focus Website Name, type name, verify Esc chain preserves values and prompts Discard changes? (y/n)
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // wraps back to field 0 (Website Name)
			for _, ch := range "AI1WM Site" {
				appModel.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
			}
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEscape}) // back to custom path input
			if !appModel.RestoreWizard().IsEnteringCustomPath() {
				t.Fatalf("expected Esc from inputs to return to custom path input")
			}
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEscape}) // back to archive list
			if appModel.RestoreWizard().IsEnteringCustomPath() || appModel.RestoreWizard().Step() != tui.RestoreWizardStepArchive {
				t.Fatalf("expected Esc from custom path input to return to archive list")
			}
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEscape}) // back to format selection
			if appModel.RestoreWizard().Step() != tui.RestoreWizardStepFormat {
				t.Fatalf("expected Esc from archive list to return to format selection")
			}

			// Esc on dirty wizard prompts Discard changes? (y/n); 'n' preserves state and lets user complete AI1WM restore
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
			viewDiscard := appModel.ViewString()
			assertFitsViewport(t, "80x24 discard confirmation", viewDiscard, tui.MinTerminalWidth, tui.MinTerminalHeight)
			plainDiscard := ansiEscapePattern.ReplaceAllString(viewDiscard, "")
			if !strings.Contains(plainDiscard, "[!] Discard changes? (y/n)") {
				t.Fatalf("expected non-color [!] Discard changes? (y/n) prompt, got:\n%s", plainDiscard)
			}
			appModel.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
			if appModel.Focus() != tui.FocusContent || appModel.RestoreWizard().Inputs().WebsiteName != "AI1WM Site" {
				t.Fatalf("expected 'n' to keep focus and preserve WebsiteName")
			}

			// Re-advance through AI1WM -> custom path (already preserved!) -> submit AI1WM restore
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // Step 1 -> Step 2 (AI1WM cursor preserved)
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // [Enter custom path...] -> preserved custom path
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // Confirm preserved custom path -> Step 3
			for range 5 {
				appModel.Update(tea.KeyPressMsg{Code: tea.KeyTab})
			}
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // Submit

			if capturedStrategy != restore.StrategyAI1WM || capturedArchive != ai1wmFile {
				t.Fatalf("expected AI1WM restore with %q, got strategy=%q archive=%q", ai1wmFile, capturedStrategy, capturedArchive)
			}
			return nil
		},
	})

	if err := application.RunWithContext(t.Context()); err != nil {
		t.Fatalf("RunWithContext failed: %v", err)
	}
}

func TestApp_RunWithContext_MasterDetail_DeleteFlow(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	sitesDir := filepath.Join(tempHome, "sites")
	_ = os.MkdirAll(filepath.Join(sitesDir, "del-site"), 0755)
	_ = os.WriteFile(filepath.Join(sitesDir, "del-site", "wp-config.php"), []byte("<?php"), 0644)
	cfg.WebsitesPath = sitesDir
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	application := app.New(app.Options{
		HomeDir: tempHome,
		AppRunner: func(ctx context.Context, model tea.Model) error {
			appModel := model.(*tui.AppModel)

			// Navigate down to Delete
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Create
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Delete
			if appModel.ActiveSection() != tui.SectionDelete {
				t.Fatalf("expected SectionDelete, got %v", appModel.ActiveSection())
			}

			// Enter content pane
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if appModel.Focus() != tui.FocusContent {
				t.Fatalf("expected FocusContent, got %v", appModel.Focus())
			}

			dm := appModel.DeleteModel()
			if dm == nil {
				t.Fatalf("expected DeleteModel to be initialized")
			}
			if dm.State() != tui.DeleteModelList {
				t.Errorf("expected DeleteModelList state, got %v", dm.State())
			}

			// Esc returns to Sidebar
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
			if appModel.Focus() != tui.FocusSidebar {
				t.Errorf("expected focus to return to FocusSidebar, got %v", appModel.Focus())
			}

			return nil
		},
	})

	if err := application.RunWithContext(context.Background()); err != nil {
		t.Fatalf("RunWithContext failed: %v", err)
	}
}

func TestApp_RunWithContext_MasterDetail_SettingsFlow(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.DefaultAdminUsername = "initial_admin"
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	var application *app.App
	application = app.New(app.Options{
		HomeDir: tempHome,
		AppRunner: func(ctx context.Context, model tea.Model) error {
			appModel := model.(*tui.AppModel)

			// Navigate to Settings
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Create
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Delete
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Restore
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Settings
			if appModel.ActiveSection() != tui.SectionSettings {
				t.Fatalf("expected SectionSettings, got %v", appModel.ActiveSection())
			}

			// Enter content pane
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if appModel.Focus() != tui.FocusContent {
				t.Fatalf("expected FocusContent, got %v", appModel.Focus())
			}

			sm := appModel.SettingsModel()
			if sm == nil {
				t.Fatalf("expected SettingsModel to be initialized")
			}

			// Modify config file on disk to simulate external edit
			cfg.DefaultAdminUsername = "updated_admin"
			if err := config.Save(cfgPath, cfg); err != nil {
				t.Fatal(err)
			}

			// Move to Reload Configuration (index 2)
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Cache
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // Reload
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

			// Verify reloaded config
			if application.Config().DefaultAdminUsername != "updated_admin" {
				t.Errorf("expected application config to be reloaded with 'updated_admin', got %s", application.Config().DefaultAdminUsername)
			}

			// Esc returns to sidebar
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
			if appModel.Focus() != tui.FocusSidebar {
				t.Errorf("expected FocusSidebar after Esc, got %v", appModel.Focus())
			}

			return nil
		},
	})

	if err := application.RunWithContext(context.Background()); err != nil {
		t.Fatalf("RunWithContext failed: %v", err)
	}
}

func TestApp_RunWithContext_FirstRun_MissingConfigRunsWizard(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")

	var wizardCalled bool
	var runnerCalled bool

	application := app.New(app.Options{
		HomeDir: tempHome,
		WizardFn: func(homeDir string) (*config.Config, error) {
			wizardCalled = true
			c := config.DefaultConfig(homeDir)
			c.AppName = config.AppNameHerd
			c.AppCommand = "herd"
			c.DefaultAdminUsername = "wizard_user"
			return c, nil
		},
		AppRunner: func(ctx context.Context, model tea.Model) error {
			runnerCalled = true
			return nil
		},
	})

	if err := application.RunWithContext(context.Background()); err != nil {
		t.Fatalf("RunWithContext failed: %v", err)
	}

	if !wizardCalled {
		t.Errorf("expected WizardFn to be called when config.json is missing")
	}
	if !runnerCalled {
		t.Errorf("expected AppRunner to be called after wizard saves config")
	}

	// Verify config file was saved to disk
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Errorf("expected config file to be saved to %s", cfgPath)
	}

	if application.Config().DefaultAdminUsername != "wizard_user" {
		t.Errorf("expected saved config admin username to be 'wizard_user', got %s", application.Config().DefaultAdminUsername)
	}
}

func TestApp_RunWithContext_MasterDetail_WebsitesHub_ConfigAction(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	siteDir := filepath.Join(tempHome, "sites", "mysite")
	cfg.WebsitesPath = filepath.Join(tempHome, "sites")
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(siteDir, "wp-config.php"), []byte("<?php // dummy"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	var executedConfig bool
	application := app.New(app.Options{
		HomeDir: tempHome,
		AppRunner: func(ctx context.Context, model tea.Model) error {
			appModel := model.(*tui.AppModel)

			// Initial state: SectionWebsites
			if appModel.ActiveSection() != tui.SectionWebsites {
				t.Fatalf("expected SectionWebsites, got %v", appModel.ActiveSection())
			}

			// Focus Content Pane
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

			// Focus Action Menu on mysite
			appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

			// Action menu cursor is at index 0 (Config). Press Enter to trigger Config.
			_, cmd := appModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

			if cmd == nil {
				t.Fatalf("expected non-nil cmd when selecting Config on website, but got nil (wptui standing still)")
			}
			if !appModel.IsExecuting() {
				t.Errorf("expected appModel.IsExecuting() to be true after selecting Config")
			}
			executedConfig = true
			return nil
		},
	})

	if err := application.RunWithContext(context.Background()); err != nil {
		t.Fatalf("RunWithContext failed: %v", err)
	}

	if !executedConfig {
		t.Errorf("expected AppRunner to execute Config action test")
	}
}
