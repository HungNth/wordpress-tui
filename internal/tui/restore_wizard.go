package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"wptui/internal/config"
	"wptui/internal/restore"
)

const CustomPathOption = "[Enter custom path...]"

type RestoreInputs = CreateInputs

// PromptRestoreStrategy displays an interactive sub-menu to choose between Full Zip and AI1WM.
func PromptRestoreStrategy() (restore.Strategy, error) {
	options := []huh.Option[restore.Strategy]{
		huh.NewOption("1. Full source code & database (.zip)", restore.StrategyFullZIP),
		huh.NewOption("2. All-in-One WP Migration (.wpress)", restore.StrategyAI1WM),
	}

	var choice restore.Strategy
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[restore.Strategy]().
				Title("Select Restore Strategy").
				Description("Choose the backup format you want to restore").
				Options(options...).
				Value(&choice),
		),
	).WithTheme(CustomTheme())

	if err := form.Run(); err != nil {
		return "", err
	}

	return choice, nil
}

// PromptRestoreInputs prompts for website name and admin credential overrides without tweaks question.
func PromptRestoreInputs(cfg *config.Config, checker ...SlugAvailabilityChecker) (*RestoreInputs, error) {
	inputs := &CreateInputs{}
	form := BuildRestoreForm(inputs, cfg, checker...)
	if err := form.Run(); err != nil {
		return nil, err
	}

	resolvedSlug, err := ResolveAndValidateSlug(inputs.WebsiteName, inputs.WebsiteSlug, nil)
	if err != nil {
		return nil, err
	}
	inputs.WebsiteSlug = resolvedSlug
	return inputs, nil
}

// PromptDumpSelection prompts the user to select one database dump from multiple candidates.
func PromptDumpSelection(candidates []string) (string, error) {
	if len(candidates) == 0 {
		return "", errors.New("no SQL dump candidates available")
	}

	options := make([]huh.Option[string], 0, len(candidates))
	for _, c := range candidates {
		options = append(options, huh.NewOption(filepath.Base(c), c))
	}

	var choice string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Multiple SQL Dumps Detected").
				Description("Select which database dump to import").
				Options(options...).
				Value(&choice),
		),
	).WithTheme(CustomTheme())

	if err := form.Run(); err != nil {
		return "", err
	}

	return choice, nil
}

// ScanBackupArchives finds valid backup files (.zip, .wpress) in backupPath.
func ScanBackupArchives(backupPath string, strategy restore.Strategy) ([]string, error) {
	entries, err := os.ReadDir(backupPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var archives []string
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if strategy == restore.StrategyFullZIP && ext == ".zip" {
			archives = append(archives, filepath.Join(backupPath, name))
		} else if strategy == restore.StrategyAI1WM && ext == ".wpress" {
			archives = append(archives, filepath.Join(backupPath, name))
		} else if strategy == "" && (ext == ".zip" || ext == ".wpress") {
			archives = append(archives, filepath.Join(backupPath, name))
		}
	}
	return archives, nil
}

// ValidateArchivePath validates that the given path exists, is a regular file, is readable, and has expected extension.
func ValidateArchivePath(path string, strategy restore.Strategy) error {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "" {
		return errors.New("file path cannot be empty")
	}

	info, err := os.Lstat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", clean)
		}
		return fmt.Errorf("cannot access file: %w", err)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("path is not a regular file: %s", clean)
	}

	ext := strings.ToLower(filepath.Ext(clean))
	if strategy == restore.StrategyFullZIP && ext != ".zip" {
		return fmt.Errorf("expected .zip archive for Full ZIP restore, got %s", ext)
	}
	if strategy == restore.StrategyAI1WM && ext != ".wpress" {
		return fmt.Errorf("expected .wpress archive for AI1WM restore, got %s", ext)
	}
	if strategy == "" && ext != ".zip" && ext != ".wpress" {
		return fmt.Errorf("unsupported backup format %s (must be .zip or .wpress)", ext)
	}

	f, err := os.Open(clean)
	if err != nil {
		return fmt.Errorf("file is not readable: %w", err)
	}
	_ = f.Close()

	return nil
}

// PromptArchiveSelection displays the archive picker with [Enter custom path...] first.
func PromptArchiveSelection(backupPath string, strategy restore.Strategy) (string, error) {
	files, err := ScanBackupArchives(backupPath, strategy)
	if err != nil {
		return "", fmt.Errorf("failed to scan backup archives: %w", err)
	}

	options := []huh.Option[string]{
		huh.NewOption(CustomPathOption, CustomPathOption),
	}

	for _, f := range files {
		options = append(options, huh.NewOption(filepath.Base(f), f))
	}

	var choice string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Backup Archive").
				Description("Choose an archive from backup storage or enter a custom path").
				Options(options...).
				Value(&choice),
		),
	).WithTheme(CustomTheme())

	if err := form.Run(); err != nil {
		return "", err
	}

	if choice == CustomPathOption {
		var inputPath string
		placeholder := "C:\\path\\to\\backup.zip"
		if strategy == restore.StrategyAI1WM {
			placeholder = "C:\\path\\to\\backup.wpress"
		}

		pathForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Archive File Path").
					Description("Enter the full absolute path to your backup archive").
					Placeholder(placeholder).
					Value(&inputPath).
					Validate(func(v string) error {
						return ValidateArchivePath(v, strategy)
					}),
			),
		).WithTheme(CustomTheme())

		if err := pathForm.Run(); err != nil {
			return "", err
		}
		return strings.TrimSpace(inputPath), nil
	}

	if err := ValidateArchivePath(choice, strategy); err != nil {
		return "", err
	}
	return choice, nil
}

// PrintRestoreSummary prints the completed restoration summary.
func PrintRestoreSummary(res *restore.Result) {
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Bold(true)
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFCC00")).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	fmt.Println("\n" + cyan.Render("=== Website Restoration Complete ==="))
	fmt.Printf("  %s %s: %s\n", green.Render("[✓]"), "Website Name", res.WebsiteName)
	fmt.Printf("  %s %s: %s\n", green.Render("[✓]"), "Website Directory", res.SitePath)
	fmt.Printf("  %s %s: %s\n", green.Render("[✓]"), "Website URL", res.SiteURL)
	fmt.Printf("  %s %s: %s\n", green.Render("[✓]"), "Database Name", res.Database)
	fmt.Printf("  %s %s: %s\n", green.Render("[✓]"), "Admin User", res.AdminUser)

	if res.TLSError != "" {
		fmt.Printf("  %s %s\n", yellow.Render("[!]"), res.TLSError)
	}

	fmt.Println(dim.Render("  Your restored website is ready for local development."))
	fmt.Println()
}

// PrintRestoreProgress prints a single progress step line.
func PrintRestoreProgress(step, description string) {
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Bold(true)
	fmt.Printf("  %s %s\n", cyan.Render("[-]"), description)
}
