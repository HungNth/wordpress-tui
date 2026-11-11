package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"wptui/internal/config"
	"wptui/internal/restore"
)

const CustomPathOption = "[Enter custom path...]"

type RestoreInputs = CreateInputs

const StrategyBack restore.Strategy = "back"

// PromptRestoreStrategy displays an interactive sub-menu to choose between Full Zip and AI1WM.
func PromptRestoreStrategy() (restore.Strategy, error) {
	options := []huh.Option[restore.Strategy]{
		huh.NewOption("Full source code & database (.zip)", restore.StrategyFullZIP),
		huh.NewOption("All-in-One WP Migration (.wpress)", restore.StrategyAI1WM),
		huh.NewOption("Back to Main Menu", StrategyBack),
	}

	var choice restore.Strategy
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[restore.Strategy]().
				Title("WPTUI / Restore").
				Description("Choose the backup format you want to restore").
				Options(options...).
				Value(&choice),
		),
	).WithTheme(CustomTheme())

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return StrategyBack, nil
		}
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

func expectedArchiveExtension(strategy restore.Strategy) (ext, engineLabel string) {
	switch strategy {
	case restore.StrategyFullZIP:
		return ".zip", "Full ZIP"
	case restore.StrategyAI1WM:
		return ".wpress", "AI1WM"
	default:
		return "", ""
	}
}

// ScanBackupArchives finds valid Restore Archives (.zip, .wpress) in backupPath.
func ScanBackupArchives(backupPath string, strategy restore.Strategy) ([]string, error) {
	trimmedBackupPath := strings.TrimSpace(backupPath)
	if trimmedBackupPath == "" {
		return nil, errors.New("backup directory is not configured")
	}

	entries, err := os.ReadDir(trimmedBackupPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("backup directory does not exist: %s", trimmedBackupPath)
		}
		if errors.Is(err, os.ErrPermission) {
			return nil, fmt.Errorf("backup directory is not readable: %s", trimmedBackupPath)
		}
		return nil, fmt.Errorf("cannot scan backup directory %s: %w", trimmedBackupPath, err)
	}

	wantExt, _ := expectedArchiveExtension(strategy)
	var archives []string
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if wantExt != "" && ext == wantExt {
			archives = append(archives, filepath.Join(trimmedBackupPath, name))
		} else if strategy == "" && (ext == ".zip" || ext == ".wpress") {
			archives = append(archives, filepath.Join(trimmedBackupPath, name))
		}
	}
	return archives, nil
}

// NormalizeArchivePath trims surrounding whitespace, strips at most one balanced pair
// of outer double quotes, normalizes path separators, and requires an absolute path.
func NormalizeArchivePath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("restore archive path cannot be empty")
	}

	hasLeadingQuote := strings.HasPrefix(trimmed, `"`)
	hasTrailingQuote := strings.HasSuffix(trimmed, `"`)
	if hasLeadingQuote != hasTrailingQuote || trimmed == `"` {
		return "", errors.New("restore archive path has unmatched double quotes")
	}
	if hasLeadingQuote && hasTrailingQuote {
		trimmed = trimmed[1 : len(trimmed)-1]
		if strings.TrimSpace(trimmed) == "" {
			return "", errors.New("restore archive path cannot be empty")
		}
		if trimmed == `"` || strings.HasPrefix(trimmed, `"`) != strings.HasSuffix(trimmed, `"`) {
			return "", errors.New("restore archive path has unmatched double quotes")
		}
	}

	clean := filepath.Clean(trimmed)
	if !filepath.IsAbs(clean) {
		return "", errors.New("restore archive path must be an absolute path")
	}
	return clean, nil
}

// NormalizeAndValidateArchivePath normalizes raw input and verifies that the target
// exists, is a regular readable file, and matches the selected Restore Engine extension.
func NormalizeAndValidateArchivePath(path string, strategy restore.Strategy) (string, error) {
	clean, err := NormalizeArchivePath(path)
	if err != nil {
		return "", err
	}

	info, err := os.Lstat(clean)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("restore archive does not exist: %s", clean)
		}
		if errors.Is(err, os.ErrPermission) {
			return "", fmt.Errorf("restore archive is not accessible: %s", clean)
		}
		return "", fmt.Errorf("cannot access restore archive: %w", err)
	}

	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("restore archive path is not a regular file: %s", clean)
	}

	ext := strings.ToLower(filepath.Ext(clean))
	gotExt := ext
	if gotExt == "" {
		gotExt = "no extension"
	}
	wantExt, engineLabel := expectedArchiveExtension(strategy)
	if wantExt != "" && ext != wantExt {
		return "", fmt.Errorf("expected %s archive for %s restore, got %s", wantExt, engineLabel, gotExt)
	}
	if strategy == "" && ext != ".zip" && ext != ".wpress" {
		return "", fmt.Errorf("expected .zip or .wpress restore archive, got %s", gotExt)
	}

	f, err := os.Open(clean)
	if err != nil {
		return "", fmt.Errorf("restore archive is not readable: %w", err)
	}
	_ = f.Close()

	return clean, nil
}

// ValidateArchivePath validates that the given path exists, is a regular file, is readable, and has expected extension.
func ValidateArchivePath(path string, strategy restore.Strategy) error {
	_, err := NormalizeAndValidateArchivePath(path, strategy)
	return err
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
	title := "=== Website Restoration Complete ==="
	if res.TLSError != "" {
		title = "=== Website Restoration Completed with Warnings ==="
	}

	fmt.Println("\n" + StyleHighlight.Render(title))
	fmt.Printf("  Website Name:      %s\n", res.WebsiteName)
	fmt.Printf("  Website Directory: %s\n", StyleHighlight.Render(res.SitePath))
	fmt.Printf("  Website URL:       %s\n", StyleHighlight.Render(res.SiteURL))
	fmt.Printf("  Database Name:     %s\n", res.Database)
	fmt.Printf("  Admin User:        %s\n", res.AdminUser)

	if res.TLSError != "" {
		fmt.Printf("  %s %s\n", StyleWarning.Render("[!]"), res.TLSError)
	}

	fmt.Println(StyleMuted.Render("  Your restored website is ready for local development."))
	fmt.Println()
}

// PrintRestoreProgress prints a single progress step line.
func PrintRestoreProgress(step, description string) {
	fmt.Printf("  %s %s\n", StyleHighlight.Render("[-]"), description)
}
