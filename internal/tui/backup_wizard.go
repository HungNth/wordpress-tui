package tui

import (
	"errors"
	"fmt"

	"charm.land/huh/v2"
	"wptui/internal/backup"
	"wptui/internal/deprovision"
)

type BackupStrategyAction string

const (
	ActionBackupFull  BackupStrategyAction = "full"
	ActionBackupAI1WM BackupStrategyAction = "ai1wm"
	ActionBackupBack  BackupStrategyAction = "back"
)

// SelectWebsiteForBackup displays a single-select menu of candidate websites to back up, with Back to Main Menu at the end.
func SelectWebsiteForBackup(websites []deprovision.Candidate) (*deprovision.Candidate, error) {
	if len(websites) == 0 {
		return nil, errors.New("no existing websites found to backup")
	}

	options := make([]huh.Option[string], 0, len(websites)+1)
	for _, w := range websites {
		options = append(options, huh.NewOption(fmt.Sprintf("%s (%s)", w.Slug, w.Path), w.Slug))
	}
	options = append(options, huh.NewOption("Back to Main Menu", "back"))

	var choice string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("WPTUI / Backup").
				Description("Select a website to back up").
				Options(options...).
				Value(&choice),
		),
	).WithTheme(CustomTheme())

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, nil
		}
		return nil, err
	}

	if choice == "back" || choice == "" {
		return nil, nil // graceful back navigation
	}

	for _, w := range websites {
		if w.Slug == choice {
			return &w, nil
		}
	}
	return nil, nil
}

// PromptBackupStrategy displays an interactive sub-menu to choose between Full Zip and AI1WM.
func PromptBackupStrategy(siteSlug string) (BackupStrategyAction, error) {
	options := []huh.Option[BackupStrategyAction]{
		huh.NewOption("Full source code & database (.zip)", ActionBackupFull),
		huh.NewOption("All-in-One WP Migration (.wpress)", ActionBackupAI1WM),
		huh.NewOption("Back to Website Selection", ActionBackupBack),
	}

	var choice BackupStrategyAction
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[BackupStrategyAction]().
				Title("WPTUI / Backup").
				Description(fmt.Sprintf("Choose a backup format for %s", siteSlug)).
				Options(options...).
				Value(&choice),
		),
	).WithTheme(CustomTheme())

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return ActionBackupBack, nil
		}
		return "", err
	}

	return choice, nil
}

// PrintBackupSummary prints the completed backup archive details in color.
func PrintBackupSummary(res *backup.BackupResult) {
	strategyTitle := "Full Source & Database"
	if res.Strategy == backup.StrategyAI1WM {
		strategyTitle = "All-in-One WP Migration"
	}

	fmt.Println("\n" + StyleHighlight.Render(fmt.Sprintf("=== Backup Complete: %s ===", strategyTitle)))
	fmt.Printf("  Archive Location: %s\n", StyleHighlight.Render(res.FilePath))
	fmt.Printf("  Archive Size:     %s\n", formatBytes(res.FileSize))
	fmt.Printf("  Time Elapsed:     %v\n", res.Duration.Round(100*1000000))
	fmt.Println(StyleMuted.Render("  Artifact is ready in your backup storage."))
	fmt.Println()
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
