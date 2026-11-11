package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"wptui/internal/config"
)

func configured(t *testing.T) *config.Config {
	t.Helper()
	cfg := config.DefaultConfig(t.TempDir())
	cfg.AppName, cfg.AppCommand = config.AppNameFlock, "flock"
	return cfg
}

func TestEnvironmentSelectionRequired(t *testing.T) {
	cfg := config.DefaultConfig(t.TempDir())
	if err := config.Validate(cfg); err == nil {
		t.Fatal("unselected environment must not be saved")
	}
	for _, name := range []string{"herd", "flock", "laragon"} {
		cfg.AppName, cfg.AppCommand = name, name
		if err := config.Validate(cfg); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	for _, name := range []string{"", "HERD", "flock ", "unknown"} {
		cfg.AppName = name
		if err := config.Validate(cfg); err == nil {
			t.Errorf("accepted invalid environment %q", name)
		}
	}
	cfg.AppName, cfg.AppCommand = "flock", "  "
	if err := config.Validate(cfg); err == nil {
		t.Fatal("accepted empty executable")
	}
}

func TestConfigRoundTrip(t *testing.T) {
	cfg := configured(t)
	cfg.AppCommand = `C:\Program Files\Flock\flock.exe`
	cfg.BackupPath = filepath.Join(t.TempDir(), "custom-backups")
	cfg.DeleteExcludes = []string{}
	path := filepath.Join(t.TempDir(), "settings", "config.json")
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AppName != cfg.AppName || loaded.AppCommand != cfg.AppCommand || loaded.BackupPath != cfg.BackupPath {
		t.Fatalf("configuration changed: %+v", loaded)
	}
	if loaded.DeleteExcludes == nil || len(loaded.DeleteExcludes) != 0 {
		t.Fatal("explicit empty exclusions changed")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"used_herd", "themes", "plugins", "default_theme_slug"} {
		if _, exists := raw[key]; exists {
			t.Errorf("obsolete key serialized: %s", key)
		}
	}
}

func TestRemovedConfigurationRejectedWithoutMutation(t *testing.T) {
	cfg := configured(t)
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"used_herd", "themes", "plugins", "default_theme_slug"} {
		t.Run(key, func(t *testing.T) {
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatal(err)
			}
			raw[key] = json.RawMessage("null")
			obsolete, err := json.Marshal(raw)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, obsolete, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := config.Load(path); err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("missing clear rejection: %v", err)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != string(obsolete) {
				t.Fatal("loading obsolete configuration modified it")
			}
		})
	}
}

func TestValidateConfigurationBoundaries(t *testing.T) {
	cases := []struct {
		name   string
		change func(*config.Config)
		valid  bool
	}{
		{"empty websites path", func(c *config.Config) { c.WebsitesPath = "" }, false},
		{"invalid port", func(c *config.Config) { c.DatabasePort = 65536 }, false},
		{"socket bypasses TCP", func(c *config.Config) { c.DBSocket = "/tmp/mysql.sock"; c.DatabasePort = 0; c.DatabaseHost = "" }, true},
		{"empty database user", func(c *config.Config) { c.DBUsername = "" }, false},
		{"empty admin user", func(c *config.Config) { c.DefaultAdminUsername = "" }, false},
		{"empty admin password", func(c *config.Config) { c.DefaultAdminPassword = "" }, false},
		{"invalid email", func(c *config.Config) { c.DefaultAdminEmail = "admin@" }, false},
		{"invalid URL", func(c *config.Config) { c.PackagesAPIURL = "ftp://example.com" }, false},
		{"anonymous API", func(c *config.Config) { c.PackagesAPIURL = "https://example.com/api"; c.PackagesAPIKey = "" }, true},
		{"unsupported tweak", func(c *config.Config) { c.WPTweaks = []config.WPTweak{{Type: "invalid"}} }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := configured(t)
			tc.change(cfg)
			if err := config.Validate(cfg); (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}

func TestInvalidConfigDoesNotOverwriteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	original := []byte("preserve me")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	cfg := configured(t)
	cfg.AppName = "unknown"
	if err := config.Save(path, cfg); err == nil {
		t.Fatal("saved invalid configuration")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Fatal("invalid save overwrote file")
	}
}
