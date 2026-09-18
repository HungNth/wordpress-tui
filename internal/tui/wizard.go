package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/huh/v2"
	"wptui/internal/config"
)

// WizardInputs holds the values collected during first-run setup.
type WizardInputs struct {
	UsedHerd             bool
	WebsitesPath         string
	BackupPath           string
	PackagesAPIURL       string
	PackagesAPIKey       string
	DefaultAdminUsername string
	DefaultAdminPassword string
	DefaultAdminEmail    string
	DatabaseHost         string
	DatabasePortStr      string
	DBUsername           string
	DBPassword           string
	DBSocket             string
}

// BuildEnvironmentForm creates the first step asking about Laravel Herd.
func BuildEnvironmentForm(inputs *WizardInputs) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Laravel Herd Environment").
				Description("Do you use Laravel Herd for local web serving?").
				Value(&inputs.UsedHerd),
		),
	).WithTheme(CustomTheme())
}

// BuildMainWizardForm creates the remaining groups once Herd preference is chosen.
func BuildMainWizardForm(inputs *WizardInputs, homeDir string) *huh.Form {
	if inputs.WebsitesPath == "" {
		if inputs.UsedHerd {
			inputs.WebsitesPath = filepath.Join(homeDir, "Herd")
		} else {
			inputs.WebsitesPath = filepath.Join(homeDir, "Sites")
		}
	}
	if inputs.DefaultAdminUsername == "" {
		inputs.DefaultAdminUsername = "admin"
	}
	if inputs.DefaultAdminPassword == "" {
		inputs.DefaultAdminPassword = "admin"
	}
	if inputs.DefaultAdminEmail == "" {
		inputs.DefaultAdminEmail = "admin@admin.com"
	}
	if inputs.DatabaseHost == "" {
		inputs.DatabaseHost = "localhost"
	}
	if inputs.DatabasePortStr == "" {
		inputs.DatabasePortStr = "3306"
	}
	if inputs.DBUsername == "" {
		inputs.DBUsername = "root"
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Websites Storage Path").
				Description(func() string {
					if !inputs.UsedHerd {
						return "Directory where sites will be stored. Prerequisite: an external wildcard stack must serve this directory's children at *.test hostnames."
					}
					return "Directory where local WordPress sites will be stored"
				}()).
				Value(&inputs.WebsitesPath).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("websites storage path cannot be empty")
					}
					return nil
				}),

			huh.NewInput().
				Title("Backup Storage Path (Optional)").
				Description("Directory to store website backup archives (leave blank for <websites_path>/backups)").
				Value(&inputs.BackupPath),
		),

		huh.NewGroup(
			huh.NewInput().
				Title("Package API URL (Optional)").
				Description("Base URL to fetch themes and plugins catalog").
				Value(&inputs.PackagesAPIURL),

			huh.NewInput().
				Title("Package API License Key (Optional)").
				Description("License key for package authentication (stored securely)").
				EchoMode(huh.EchoModePassword).
				Value(&inputs.PackagesAPIKey),
		),

		huh.NewGroup(
			huh.NewInput().
				Title("Default Administrator Username").
				Description("Username for initial WordPress administrator account").
				Value(&inputs.DefaultAdminUsername).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("admin username cannot be empty")
					}
					return nil
				}),

			huh.NewInput().
				Title("Default Administrator Password").
				Description("Password for initial WordPress administrator account").
				EchoMode(huh.EchoModePassword).
				Value(&inputs.DefaultAdminPassword).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("admin password cannot be empty")
					}
					return nil
				}),

			huh.NewInput().
				Title("Default Administrator Email").
				Description("Email for initial WordPress administrator account").
				Value(&inputs.DefaultAdminEmail).
				Validate(config.ValidateEmail),
		),

		huh.NewGroup(
			huh.NewInput().
				Title("Database Host").
				Description("MySQL host (default: localhost)").
				Value(&inputs.DatabaseHost),

			huh.NewInput().
				Title("Database Port").
				Description("MySQL port (default: 3306)").
				Value(&inputs.DatabasePortStr).
				Validate(func(s string) error {
					if strings.TrimSpace(inputs.DBSocket) != "" {
						return nil
					}
					p, err := strconv.Atoi(strings.TrimSpace(s))
					if err != nil || p <= 0 || p > 65535 {
						return errors.New("port must be a number between 1 and 65535")
					}
					return nil
				}),

			huh.NewInput().
				Title("Database Username").
				Description("MySQL user (default: root)").
				Value(&inputs.DBUsername),

			huh.NewInput().
				Title("Database Password").
				Description("MySQL password (leave empty if none)").
				EchoMode(huh.EchoModePassword).
				Value(&inputs.DBPassword),

			huh.NewInput().
				Title("Database Socket (Optional)").
				Description("Unix socket path if connecting via socket instead of TCP").
				Value(&inputs.DBSocket),
		),
	).WithTheme(CustomTheme())
}

// ConvertInputsToConfig merges collected inputs into a full default config.
func ConvertInputsToConfig(inputs WizardInputs, homeDir string) (*config.Config, error) {
	port := 3306
	if strings.TrimSpace(inputs.DatabasePortStr) != "" {
		if p, err := strconv.Atoi(strings.TrimSpace(inputs.DatabasePortStr)); err == nil && p > 0 {
			port = p
		}
	}

	cfg := config.DefaultConfig(homeDir)
	cfg.UsedHerd = inputs.UsedHerd

	// If websites path was untouched or empty, assign based on UsedHerd
	targetPath := strings.TrimSpace(inputs.WebsitesPath)
	if targetPath == "" || (targetPath == filepath.Join(homeDir, "Herd") && !inputs.UsedHerd) {
		if inputs.UsedHerd {
			targetPath = filepath.Join(homeDir, "Herd")
		} else {
			targetPath = filepath.Join(homeDir, "Sites")
		}
	}
	cfg.WebsitesPath = targetPath
	backupPath := strings.TrimSpace(inputs.BackupPath)
	if backupPath == "" {
		backupPath = filepath.Join(targetPath, "backups")
	}
	cfg.BackupPath = backupPath
	cfg.PackagesAPIURL = strings.TrimSpace(inputs.PackagesAPIURL)
	cfg.PackagesAPIKey = strings.TrimSpace(inputs.PackagesAPIKey)
	cfg.DefaultAdminUsername = strings.TrimSpace(inputs.DefaultAdminUsername)
	cfg.DefaultAdminPassword = inputs.DefaultAdminPassword
	cfg.DefaultAdminEmail = strings.TrimSpace(inputs.DefaultAdminEmail)
	cfg.DatabaseHost = strings.TrimSpace(inputs.DatabaseHost)
	cfg.DatabasePort = port
	cfg.DBUsername = strings.TrimSpace(inputs.DBUsername)
	cfg.DBPassword = inputs.DBPassword
	cfg.DBSocket = strings.TrimSpace(inputs.DBSocket)

	if err := config.Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid wizard configuration: %w", err)
	}

	return cfg, nil
}

// RunConfigWizard runs the interactive first-run configuration wizard and returns the filled Config.
func RunConfigWizard(homeDir string) (*config.Config, error) {
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
	}

	inputs := WizardInputs{
		UsedHerd: true,
	}

	envForm := BuildEnvironmentForm(&inputs)
	if err := envForm.Run(); err != nil {
		return nil, err
	}

	mainForm := BuildMainWizardForm(&inputs, homeDir)
	if err := mainForm.Run(); err != nil {
		return nil, err
	}

	return ConvertInputsToConfig(inputs, homeDir)
}
