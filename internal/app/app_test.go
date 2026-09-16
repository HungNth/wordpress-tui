package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"wptui/internal/app"
	"wptui/internal/config"
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
