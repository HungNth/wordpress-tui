package config_test

import (
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
