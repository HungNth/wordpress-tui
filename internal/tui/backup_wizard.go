package tui

import (
	"fmt"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"wptui/internal/backup"
)

type BackupStrategyAction string

const (
	ActionBackupFull  BackupStrategyAction = "full"
	ActionBackupAI1WM BackupStrategyAction = "ai1wm"
	ActionBackupBack  BackupStrategyAction = "back"
)

// PromptBackupStrategy displays an interactive sub-menu to choose between Full Zip and AI1WM.
func PromptBackupStrategy(siteSlug string) (BackupStrategyAction, error) {
	options := []huh.Option[BackupStrategyAction]{
		huh.NewOption("1. Full source code & database (.zip)", ActionBackupFull),
		huh.NewOption("2. All-in-One WP Migration (.wpress)", ActionBackupAI1WM),
		huh.NewOption("← Back (Select another website)", ActionBackupBack),
	}

	var choice BackupStrategyAction
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[BackupStrategyAction]().
				Title(fmt.Sprintf("Backup Website: %s", siteSlug)).
				Description("Choose a backup strategy").
				Options(options...).
				Value(&choice),
		),
	).WithTheme(CustomTheme())

	if err := form.Run(); err != nil {
		return "", err
	}

	return choice, nil
}

// PrintBackupSummary prints the completed backup archive details in color.
func PrintBackupSummary(res *backup.BackupResult) {
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Bold(true)
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	strategyTitle := "Full Source & Database"
	if res.Strategy == backup.StrategyAI1WM {
		strategyTitle = "All-in-One WP Migration"
	}

	fmt.Println("\n" + cyan.Render(fmt.Sprintf("=== Backup Complete: %s ===", strategyTitle)))
	fmt.Printf("  %s %s: %s\n", green.Render("[✓]"), "Archive Location", res.FilePath)
	fmt.Printf("  %s %s: %s\n", green.Render("[✓]"), "Archive Size", formatBytes(res.FileSize))
	fmt.Printf("  %s %s: %v\n", green.Render("[✓]"), "Time Elapsed", res.Duration.Round(100*1000000))
	fmt.Println(dim.Render("  Artifact is ready in your backup storage."))
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
