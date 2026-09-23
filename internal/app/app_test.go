package app_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/app"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/packages"
	"wptui/internal/tui"
)

func TestApp_LoadExistingConfig(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")

	cfg := config.DefaultConfig(tempHome)
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
	mu                sync.Mutex
	configGetCalls    []string
	dbDropCalls       []string
	herdUnsecureCalls []string
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

func (m *mockDeleteWPClient) HerdUnsecure(ctx context.Context, dir, slug string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.herdUnsecureCalls = append(m.herdUnsecureCalls, slug)
	return nil
}

func TestRunDeleteFlow_EndToEndSingleSite(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
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
	if len(mockClient.herdUnsecureCalls) != 1 || mockClient.herdUnsecureCalls[0] != "site-one" {
		t.Errorf("expected herd unsecure call for site-one, got %+v", mockClient.herdUnsecureCalls)
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
	if len(mockClient.herdUnsecureCalls) != 3 {
		t.Errorf("expected 3 herd unsecure calls, got %d", len(mockClient.herdUnsecureCalls))
	}
	if len(mockClient.dbDropCalls) != 3 {
		t.Errorf("expected 3 db drop calls, got %d", len(mockClient.dbDropCalls))
	}
}

func TestApp_RunWithContext_DispatchesDelete(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
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

			// Step 2: Select archive (Enter)
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

	if err := application.RunWithContext(context.Background()); err != nil {
		t.Fatalf("RunWithContext failed: %v", err)
	}
}

func TestApp_RunWithContext_MasterDetail_DeleteFlow(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
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

