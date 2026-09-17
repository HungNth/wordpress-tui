package app_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

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
