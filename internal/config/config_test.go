package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"wptui/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	tempHome := t.TempDir()
	cfg := config.DefaultConfig(tempHome)

	if !cfg.UsedHerd {
		t.Errorf("expected UsedHerd to be true by default")
	}
	expectedHerdPath := filepath.Join(tempHome, "Herd")
	if cfg.WebsitesPath != expectedHerdPath {
		t.Errorf("expected WebsitesPath %s, got %s", expectedHerdPath, cfg.WebsitesPath)
	}
	if cfg.PackagesAPIURL != "" || cfg.PackagesAPIKey != "" {
		t.Errorf("expected PackagesAPIURL and PackagesAPIKey to be empty by default")
	}
	if cfg.DefaultAdminUsername != "admin" || cfg.DefaultAdminPassword != "admin" || cfg.DefaultAdminEmail != "admin@admin.com" {
		t.Errorf("admin defaults do not match expected admin credentials")
	}
	if cfg.DatabaseHost != "localhost" || cfg.DatabasePort != 3306 || cfg.DBUsername != "root" {
		t.Errorf("database defaults do not match expected root@localhost:3306")
	}
	if cfg.DefaultThemeSlug != "flatsome" {
		t.Errorf("expected default theme slug flatsome, got %s", cfg.DefaultThemeSlug)
	}
	if len(cfg.Themes) == 0 || len(cfg.Plugins) == 0 || len(cfg.WPTweaks) == 0 {
		t.Errorf("expected seeded themes, plugins, and wp_tweaks")
	}
	if err := config.Validate(cfg); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
	if len(cfg.DeleteExcludes) != 1 || cfg.DeleteExcludes[0] != "backups" {
		t.Errorf("expected default DeleteExcludes to be ['backups'], got %v", cfg.DeleteExcludes)
	}
	expectedBackupPath := filepath.Join(expectedHerdPath, "backups")
	if cfg.BackupPath != expectedBackupPath {
		t.Errorf("expected default BackupPath %s, got %s", expectedBackupPath, cfg.BackupPath)
	}
}

func TestConfig_LegacyLoadWithoutBackupPath(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	// Legacy config.json authored before backup_path existed, with no backup_path key
	legacyJSON := `{
  "used_herd": true,
  "websites_path": "C:\\sites",
  "packages_api_url": "",
  "packages_api_key": "",
  "default_admin_username": "admin",
  "default_admin_password": "admin",
  "default_admin_email": "admin@admin.com",
  "database_host": "localhost",
  "database_port": 3306,
  "db_username": "root",
  "db_password": "",
  "db_socket": "",
  "default_theme_slug": "flatsome"
}`
	if err := os.WriteFile(cfgPath, []byte(legacyJSON), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("legacy config without backup_path must still load: %v", err)
	}
	if cfg.BackupPath != "" {
		t.Errorf("Load must not inject a backup_path fallback, got %q", cfg.BackupPath)
	}
}

func TestConfig_BackupPathSerialization(t *testing.T) {
	tempHome := t.TempDir()
	cfg := config.DefaultConfig(tempHome)
	cfg.BackupPath = "custom/backup/dir"

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	var loaded config.Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	if loaded.BackupPath != "custom/backup/dir" {
		t.Errorf("expected BackupPath 'custom/backup/dir', got %q", loaded.BackupPath)
	}
}

func TestValidateConfig(t *testing.T) {
	tempHome := t.TempDir()

	tests := []struct {
		name    string
		modify  func(*config.Config)
		wantErr bool
	}{
		{
			name:    "valid",
			modify:  func(c *config.Config) {},
			wantErr: false,
		},
		{
			name: "empty websites path",
			modify: func(c *config.Config) {
				c.WebsitesPath = ""
			},
			wantErr: true,
		},
		{
			name: "invalid port with no socket",
			modify: func(c *config.Config) {
				c.DatabasePort = 0
			},
			wantErr: true,
		},
		{
			name: "socket provided allows zero port",
			modify: func(c *config.Config) {
				c.DBSocket = "/tmp/mysql.sock"
				c.DatabasePort = 0
				c.DatabaseHost = ""
			},
			wantErr: false,
		},
		{
			name: "empty db username",
			modify: func(c *config.Config) {
				c.DBUsername = ""
			},
			wantErr: true,
		},
		{
			name: "empty admin username",
			modify: func(c *config.Config) {
				c.DefaultAdminUsername = ""
			},
			wantErr: true,
		},
		{
			name: "empty admin password",
			modify: func(c *config.Config) {
				c.DefaultAdminPassword = ""
			},
			wantErr: true,
		},
		{
			name: "invalid admin email missing domain",
			modify: func(c *config.Config) {
				c.DefaultAdminEmail = "admin@"
			},
			wantErr: true,
		},
		{
			name: "invalid admin email not an email",
			modify: func(c *config.Config) {
				c.DefaultAdminEmail = "not-an-email"
			},
			wantErr: true,
		},
		{
			name: "empty default theme slug",
			modify: func(c *config.Config) {
				c.DefaultThemeSlug = ""
			},
			wantErr: true,
		},
		{
			name: "invalid packages api url",
			modify: func(c *config.Config) {
				c.PackagesAPIURL = "ftp://invalid-scheme.com"
			},
			wantErr: true,
		},
		{
			name: "unsupported tweak type",
			modify: func(c *config.Config) {
				c.WPTweaks = append(c.WPTweaks, config.WPTweak{
					Type:  "unsupported_action",
					Key:   "foo",
					Value: "bar",
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig(tempHome)
			tt.modify(cfg)
			err := config.Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadAndSaveConfig(t *testing.T) {
	tempHome := t.TempDir()
	configPath := filepath.Join(tempHome, ".config", "wptui", "config.json")

	cfg := config.DefaultConfig(tempHome)
	cfg.DefaultAdminUsername = "custom_admin"

	if err := config.Save(configPath, cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loaded.DefaultAdminUsername != "custom_admin" {
		t.Errorf("expected custom_admin, got %s", loaded.DefaultAdminUsername)
	}

	// Corrupt file
	if err := os.WriteFile(configPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}
	_, err = config.Load(configPath)
	if err == nil {
		t.Errorf("expected error loading invalid JSON, got nil")
	}
}
func TestLoad_DeleteExcludesMigration(t *testing.T) {
	tempHome := t.TempDir()
	configDir := filepath.Join(tempHome, ".config", "wptui")
	configPath := filepath.Join(configDir, "config.json")

	// 1. Existing legacy config with delete_excludes completely absent (missing key)
	cfg := config.DefaultConfig(tempHome)
	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		t.Fatal(err)
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		t.Fatal(err)
	}
	delete(rawMap, "delete_excludes") // explicitly remove key to simulate legacy file

	legacyBytes, err := json.MarshalIndent(rawMap, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, legacyBytes, 0600); err != nil {
		t.Fatal(err)
	}

	// Load should auto-migrate missing key to ["backups"] and persist to file
	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Load failed on config with missing delete_excludes: %v", err)
	}
	if len(loaded.DeleteExcludes) != 1 || loaded.DeleteExcludes[0] != "backups" {
		t.Fatalf("expected migrated delete_excludes to be ['backups'], got %v", loaded.DeleteExcludes)
	}

	// Verify it was physically written back to the disk file
	persistedBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var persistedMap map[string]interface{}
	if err := json.Unmarshal(persistedBytes, &persistedMap); err != nil {
		t.Fatal(err)
	}
	deVal, exists := persistedMap["delete_excludes"]
	if !exists {
		t.Fatalf("expected delete_excludes to be persisted into config.json, but was absent")
	}
	deSlice, ok := deVal.([]interface{})
	if !ok || len(deSlice) != 1 || deSlice[0] != "backups" {
		t.Fatalf("unexpected persisted delete_excludes: %v", deVal)
	}

	// 2. Existing config with explicit null value should also migrate and persist
	rawMap["delete_excludes"] = nil
	nullBytes, err := json.MarshalIndent(rawMap, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, nullBytes, 0600); err != nil {
		t.Fatal(err)
	}
	loadedNull, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Load failed on null delete_excludes: %v", err)
	}
	if len(loadedNull.DeleteExcludes) != 1 || loadedNull.DeleteExcludes[0] != "backups" {
		t.Fatalf("expected null delete_excludes to migrate to ['backups'], got %v", loadedNull.DeleteExcludes)
	}

	// 3. Explicitly empty delete_excludes [] should be preserved as empty and not re-seeded
	loaded.DeleteExcludes = []string{}
	if err := config.Save(configPath, loaded); err != nil {
		t.Fatal(err)
	}

	reloaded, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.DeleteExcludes == nil {
		t.Fatalf("expected empty slice, got nil")
	}
	if len(reloaded.DeleteExcludes) != 0 {
		t.Fatalf("expected empty delete_excludes [] to be preserved, got %v", reloaded.DeleteExcludes)
	}
}
