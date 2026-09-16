package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type ThemeItem struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type PluginItem struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type TweakType string

const (
	TweakTypeConfigSet        TweakType = "config_set"
	TweakTypeRewriteStructure TweakType = "rewrite_structure"
	TweakTypeOptionUpdate     TweakType = "option_update"
	TweakTypeLanguageCore     TweakType = "language_core"
)

type WPTweak struct {
	Type  TweakType `json:"type"`
	Key   string    `json:"key,omitempty"`
	Value string    `json:"value"`
	Raw   bool      `json:"raw,omitempty"`
}

type Config struct {
	UsedHerd              bool         `json:"used_herd"`
	WebsitesPath          string       `json:"websites_path"`
	PackagesAPIURL        string       `json:"packages_api_url"`
	PackagesAPIKey        string       `json:"packages_api_key"`
	DefaultAdminUsername  string       `json:"default_admin_username"`
	DefaultAdminPassword  string       `json:"default_admin_password"`
	DefaultAdminEmail     string       `json:"default_admin_email"`
	DatabaseHost          string       `json:"database_host"`
	DatabasePort          int          `json:"database_port"`
	DBUsername            string       `json:"db_username"`
	DBPassword            string       `json:"db_password"`
	DBSocket              string       `json:"db_socket"`
	DefaultThemeSlug      string       `json:"default_theme_slug"`
	Themes                []ThemeItem  `json:"themes"`
	Plugins               []PluginItem `json:"plugins"`
	WPTweaks              []WPTweak    `json:"wp_tweaks"`
	BackupExcludes        []string     `json:"backup_excludes"`
	WPContentCopyExcludes []string     `json:"wp_content_copy_excludes"`
}

var ValidTweakTypes = map[TweakType]bool{
	TweakTypeConfigSet:        true,
	TweakTypeRewriteStructure: true,
	TweakTypeOptionUpdate:     true,
	TweakTypeLanguageCore:     true,
}

func DefaultConfig(homeDir string) *Config {
	websitesPath := filepath.Join(homeDir, "Herd")

	return &Config{
		UsedHerd:             true,
		WebsitesPath:         websitesPath,
		PackagesAPIURL:       "",
		PackagesAPIKey:       "",
		DefaultAdminUsername: "admin",
		DefaultAdminPassword: "admin",
		DefaultAdminEmail:    "admin@admin.com",
		DatabaseHost:         "localhost",
		DatabasePort:         3306,
		DBUsername:           "root",
		DBPassword:           "",
		DBSocket:             "",
		DefaultThemeSlug:     "flatsome",
		Themes: []ThemeItem{
			{Name: "Flatsome", Slug: "flatsome"},
			{Name: "Bricks", Slug: "bricks"},
			{Name: "Etch Theme", Slug: "etch-theme"},
			{Name: "Woodmart", Slug: "woodmart"},
			{Name: "Avada", Slug: "Avada"},
			{Name: "Jannah", Slug: "jannah"},
		},
		Plugins: []PluginItem{
			{Name: "Advanced Custom Fields PRO", Slug: "advanced-custom-fields-pro"},
			{Name: "All-in-One WP Migration Unlimited Extension", Slug: "all-in-one-wp-migration-unlimited-extension"},
			{Name: "Rank Math SEO PRO", Slug: "seo-by-rank-math-pro"},
			{Name: "UpdraftPlus - Backup/Restore", Slug: "updraftplus"},
			{Name: "WP Mail SMTP Pro", Slug: "wp-mail-smtp-pro"},
			{Name: "Admin and Site Enhancements (ASE) Pro", Slug: "admin-site-enhancements-pro"},
			{Name: "WP Rocket", Slug: "wp-rocket"},
			{Name: "Perfmatters", Slug: "perfmatters"},
			{Name: "Duplicator Pro", Slug: "duplicator-pro"},
			{Name: "FluentCart Pro", Slug: "fluent-cart-pro"},
			{Name: "Etch", Slug: "etch"},
			{Name: "Automatic.css", Slug: "automaticcss-plugin"},
		},
		WPTweaks: []WPTweak{
			{Type: TweakTypeConfigSet, Key: "WP_DEBUG", Value: "true", Raw: true},
			{Type: TweakTypeConfigSet, Key: "WP_DEBUG_LOG", Value: "true", Raw: true},
			{Type: TweakTypeConfigSet, Key: "WP_DEBUG_DISPLAY", Value: "true", Raw: true},
			{Type: TweakTypeConfigSet, Key: "WP_MEMORY_LIMIT", Value: "256M", Raw: false},
			{Type: TweakTypeConfigSet, Key: "AUTOSAVE_INTERVAL", Value: "600", Raw: true},
			{Type: TweakTypeConfigSet, Key: "WP_POST_REVISIONS", Value: "5", Raw: true},
			{Type: TweakTypeConfigSet, Key: "EMPTY_TRASH_DAYS", Value: "21", Raw: true},
			{Type: TweakTypeRewriteStructure, Value: "/%category%/%postname%/"},
			{Type: TweakTypeOptionUpdate, Key: "timezone_string", Value: "Asia/Ho_Chi_Minh"},
			{Type: TweakTypeOptionUpdate, Key: "time_format", Value: "H:i"},
			{Type: TweakTypeOptionUpdate, Key: "date_format", Value: "d/m/Y"},
			{Type: TweakTypeOptionUpdate, Key: "large_size_w", Value: "0"},
			{Type: TweakTypeOptionUpdate, Key: "large_size_h", Value: "0"},
			{Type: TweakTypeOptionUpdate, Key: "medium_large_size_w", Value: "0"},
			{Type: TweakTypeOptionUpdate, Key: "medium_large_size_h", Value: "0"},
			{Type: TweakTypeOptionUpdate, Key: "medium_size_w", Value: "0"},
			{Type: TweakTypeOptionUpdate, Key: "medium_size_h", Value: "0"},
			{Type: TweakTypeOptionUpdate, Key: "thumbnail_size_w", Value: "0"},
			{Type: TweakTypeOptionUpdate, Key: "thumbnail_size_h", Value: "0"},
			{Type: TweakTypeOptionUpdate, Key: "thumbnail_crop", Value: "0"},
			{Type: TweakTypeOptionUpdate, Key: "comment_moderation", Value: "1"},
			{Type: TweakTypeOptionUpdate, Key: "default_pingback_flag", Value: "0"},
			{Type: TweakTypeOptionUpdate, Key: "default_ping_status", Value: "closed"},
			{Type: TweakTypeOptionUpdate, Key: "posts_per_page", Value: "30"},
			{Type: TweakTypeOptionUpdate, Key: "posts_per_rss", Value: "210"},
			{Type: TweakTypeOptionUpdate, Key: "rss_use_excerpt", Value: "1"},
			{Type: TweakTypeOptionUpdate, Key: "avatar_default", Value: "identicon"},
			{Type: TweakTypeLanguageCore, Key: "install", Value: "vi"},
			{Type: TweakTypeLanguageCore, Key: "activate", Value: "vi"},
		},
		BackupExcludes: []string{
			".idea",
			".vscode",
			"node_modules",
			"__MACOSX",
		},
		WPContentCopyExcludes: []string{
			".idea",
			".vscode",
			"__MACOSX",
			"node_modules",
			"cache",
		},
	}
}

func Validate(c *Config) error {
	if c == nil {
		return errors.New("config is nil")
	}
	if strings.TrimSpace(c.WebsitesPath) == "" {
		return errors.New("websites_path is required")
	}
	if strings.TrimSpace(c.DBSocket) == "" {
		if strings.TrimSpace(c.DatabaseHost) == "" {
			return errors.New("database_host is required when db_socket is not provided")
		}
		if c.DatabasePort <= 0 || c.DatabasePort > 65535 {
			return fmt.Errorf("invalid database_port: %d (must be between 1 and 65535)", c.DatabasePort)
		}
	}
	if strings.TrimSpace(c.DefaultAdminUsername) == "" {
		return errors.New("default_admin_username is required")
	}
	if strings.TrimSpace(c.DefaultAdminPassword) == "" {
		return errors.New("default_admin_password is required")
	}
	if !strings.Contains(c.DefaultAdminEmail, "@") || strings.TrimSpace(c.DefaultAdminEmail) == "" {
		return errors.New("default_admin_email must be a valid email address")
	}
	if strings.TrimSpace(c.PackagesAPIURL) != "" {
		u, err := url.Parse(c.PackagesAPIURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("invalid packages_api_url %q: must be a valid http or https URL", c.PackagesAPIURL)
		}
	}

	for i, tweak := range c.WPTweaks {
		if !ValidTweakTypes[tweak.Type] {
			return fmt.Errorf("wp_tweaks[%d]: unsupported tweak type %q", i, tweak.Type)
		}
		switch tweak.Type {
		case TweakTypeConfigSet:
			if strings.TrimSpace(tweak.Key) == "" {
				return fmt.Errorf("wp_tweaks[%d]: config_set requires a non-empty key", i)
			}
			if strings.TrimSpace(tweak.Value) == "" {
				return fmt.Errorf("wp_tweaks[%d]: config_set requires a non-empty value", i)
			}
		case TweakTypeRewriteStructure:
			if strings.TrimSpace(tweak.Value) == "" {
				return fmt.Errorf("wp_tweaks[%d]: rewrite_structure requires a non-empty value", i)
			}
		case TweakTypeOptionUpdate:
			if strings.TrimSpace(tweak.Key) == "" || strings.TrimSpace(tweak.Value) == "" {
				return fmt.Errorf("wp_tweaks[%d]: option_update requires non-empty key and value", i)
			}
		case TweakTypeLanguageCore:
			if tweak.Key != "install" && tweak.Key != "activate" {
				return fmt.Errorf("wp_tweaks[%d]: language_core key must be 'install' or 'activate', got %q", i, tweak.Key)
			}
			if strings.TrimSpace(tweak.Value) == "" {
				return fmt.Errorf("wp_tweaks[%d]: language_core requires a non-empty locale value", i)
			}
		}
	}
	return nil
}

func ConfigPath(customHome ...string) (string, error) {
	var home string
	if len(customHome) > 0 && customHome[0] != "" {
		home = customHome[0]
	} else {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine user home directory: %w", err)
		}
	}
	return filepath.Join(home, ".config", "wptui", "config.json"), nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config JSON in %s: %w", path, err)
	}

	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config in %s: %w", path, err)
	}

	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	if err := Validate(cfg); err != nil {
		return fmt.Errorf("cannot save invalid config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", path, err)
	}

	return nil
}
