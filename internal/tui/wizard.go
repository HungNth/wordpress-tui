package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"wptui/internal/config"
)

// WizardInputs holds the values collected during first-run setup.
type WizardInputs struct {
	AppName              string
	AppCommand           string
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

// EnvironmentOptions returns the available local development environment options filtered by platform.
func EnvironmentOptions(targetOS ...string) []huh.Option[string] {
	goos := runtime.GOOS
	if len(targetOS) > 0 && targetOS[0] != "" {
		goos = targetOS[0]
	}

	opts := []huh.Option[string]{
		huh.NewOption("Laravel Herd", config.AppNameHerd),
		huh.NewOption("Flock", config.AppNameFlock),
	}
	if goos == "windows" {
		opts = append(opts, huh.NewOption("Laragon", config.AppNameLaragon))
	}
	return opts
}

// DefaultWebsitesPath returns the suggested websites directory for the given environment and home dir.
func DefaultWebsitesPath(appName, homeDir string) string {
	switch strings.ToLower(strings.TrimSpace(appName)) {
	case config.AppNameHerd:
		return filepath.Join(homeDir, "Herd")
	case config.AppNameFlock:
		return filepath.Join(homeDir, "Sites")
	case config.AppNameLaragon:
		return `C:\laragon\www`
	default:
		return ""
	}
}

// DefaultAppCommand returns the suggested executable command for the given environment.
func DefaultAppCommand(appName string) string {
	switch strings.ToLower(strings.TrimSpace(appName)) {
	case config.AppNameHerd:
		return "herd"
	case config.AppNameFlock:
		return "flock"
	case config.AppNameLaragon:
		return `C:\laragon\laragon.exe`
	default:
		return ""
	}
}

func websitesPathDescription(appName string) string {
	switch strings.ToLower(strings.TrimSpace(appName)) {
	case config.AppNameLaragon:
		return "Laragon Document Root (C:\\laragon\\www). Prerequisites: Websites must be under Document Root, hostname pattern must be {name}.test, and Auto SSL must be enabled in Laragon."
	case config.AppNameFlock:
		return "Directory where local WordPress sites will be stored (linked and secured via Flock)"
	case config.AppNameHerd:
		return "Directory where local WordPress sites will be stored (parked and secured via Herd)"
	default:
		return "Directory where local WordPress sites will be stored"
	}
}

func appCommandDescription(appName string) string {
	switch strings.ToLower(strings.TrimSpace(appName)) {
	case config.AppNameLaragon:
		return "Path to Laragon executable used to reload virtual hosts (default: C:\\laragon\\laragon.exe)"
	case config.AppNameFlock:
		return "Executable name or path for Flock CLI (default: flock)"
	case config.AppNameHerd:
		return "Executable name or path for Herd CLI (default: herd)"
	default:
		return "Executable name or path for the local development environment"
	}
}

// EnvironmentSentinel is the label for the placeholder option requiring explicit selection.
const EnvironmentSentinel = "Choose an environment…"

// BuildEnvironmentForm creates the first step asking for the local development environment.
func BuildEnvironmentForm(inputs *WizardInputs, targetOS ...string) *huh.Form {
	opts := append([]huh.Option[string]{
		huh.NewOption(EnvironmentSentinel, ""),
	}, EnvironmentOptions(targetOS...)...)

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Local Development Environment").
				Description("Choose your local WordPress development environment").
				Options(opts...).
				Value(&inputs.AppName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("environment selection is required")
					}
					return nil
				}),
		),
	).WithTheme(CustomTheme())
}

// BuildMainWizardForm creates the remaining groups once the environment is chosen.
func BuildMainWizardForm(inputs *WizardInputs, homeDir string) *huh.Form {
	if inputs.WebsitesPath == "" {
		inputs.WebsitesPath = DefaultWebsitesPath(inputs.AppName, homeDir)
	}
	if inputs.AppCommand == "" {
		inputs.AppCommand = DefaultAppCommand(inputs.AppName)
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
				Description(websitesPathDescription(inputs.AppName)).
				Value(&inputs.WebsitesPath).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("websites storage path cannot be empty")
					}
					return nil
				}),

			huh.NewInput().
				Title("Environment Executable Command").
				Description(appCommandDescription(inputs.AppName)).
				Value(&inputs.AppCommand).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("environment executable command cannot be empty")
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
	socket := strings.TrimSpace(inputs.DBSocket)
	portStr := strings.TrimSpace(inputs.DatabasePortStr)
	var port int

	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil || p <= 0 || p > 65535 {
			if socket == "" {
				return nil, fmt.Errorf("invalid database port %q: must be a number between 1 and 65535", portStr)
			}
		} else {
			port = p
		}
	} else if socket == "" {
		port = 3306
	}

	cfg := config.DefaultConfig(homeDir)
	cfg.AppName = strings.TrimSpace(inputs.AppName)
	cfg.AppCommand = strings.TrimSpace(inputs.AppCommand)
	targetPath := strings.TrimSpace(inputs.WebsitesPath)
	if targetPath == "" && cfg.AppName != "" {
		targetPath = DefaultWebsitesPath(cfg.AppName, homeDir)
	}
	cfg.WebsitesPath = targetPath

	backupPath := strings.TrimSpace(inputs.BackupPath)
	if backupPath == "" && targetPath != "" {
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

	var inputs WizardInputs

	altScreenHook := func(v tea.View) tea.View {
		v.AltScreen = true
		return v
	}

	envForm := BuildEnvironmentForm(&inputs).WithViewHook(altScreenHook)
	if err := envForm.Run(); err != nil {
		return nil, err
	}

	mainForm := BuildMainWizardForm(&inputs, homeDir).WithViewHook(altScreenHook)
	if err := mainForm.Run(); err != nil {
		return nil, err
	}

	return ConvertInputsToConfig(inputs, homeDir)
}

// PrintProgress prints a step-prefixed status line for non-Bubble-Tea flows.
func PrintProgress(step, total int, message string) {
	if total > 0 {
		fmt.Printf("  %s %s\n", StyleHighlight.Render(fmt.Sprintf("[%d/%d]", step, total)), message)
	} else {
		fmt.Printf("  %s %s\n", StyleHighlight.Render("[-]"), message)
	}
}
