package app

import (
	"context"
	"fmt"
	"strings"

	"wptui/internal/backup"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/packages"
	"wptui/internal/siteconfig"
	"wptui/internal/tui"
	"wptui/internal/wpcli"
)

type BackupFlowDependencies struct {
	WPClient       siteconfig.WPClient
	Resolver       backup.PackageResolver
	SelectWebsite  func([]deprovision.Candidate) (*deprovision.Candidate, error)
	SelectStrategy func(string) (tui.BackupStrategyAction, error)
}

func RunDefaultBackupFlow(ctx context.Context, cfg *config.Config) error {
	client := wpcli.NewClient()
	var resolver backup.PackageResolver

	if strings.TrimSpace(cfg.PackagesAPIURL) != "" {
		cache, err := packages.NewCacheWithContext(ctx)
		if err == nil {
			resolver = packages.NewResolver(cfg, cache)
		}
	}

	return RunBackupFlowWithDeps(ctx, cfg, BackupFlowDependencies{
		WPClient: client,
		Resolver: resolver,
	})
}

func RunBackupFlowWithDeps(ctx context.Context, cfg *config.Config, deps BackupFlowDependencies) error {
	selectWebsite := deps.SelectWebsite
	if selectWebsite == nil {
		selectWebsite = tui.SelectWebsiteForBackup
	}

	selectStrategy := deps.SelectStrategy
	if selectStrategy == nil {
		selectStrategy = tui.PromptBackupStrategy
	}

	backupPath := strings.TrimSpace(cfg.BackupPath)
	if backupPath == "" {
		return fmt.Errorf("backup_path is not configured in config.json; please set it via Settings or re-run setup wizard")
	}

	for {
		candidates, err := deprovision.DiscoverCandidates(ctx, cfg.WebsitesPath, cfg.DeleteExcludes)
		if err != nil {
			return fmt.Errorf("failed to discover websites: %w", err)
		}
		if len(candidates) == 0 {
			fmt.Println("No existing websites found to backup.")
			return nil
		}

		selectedSite, err := selectWebsite(candidates)
		if err != nil {
			return err
		}
		if selectedSite == nil {
			return nil // User chose Back to main menu
		}

		strategy, err := selectStrategy(selectedSite.Slug)
		if err != nil {
			return err
		}
		if strategy == tui.ActionBackupBack || strategy == "" {
			continue // pick another website
		}

		var res *backup.BackupResult
		switch strategy {
		case tui.ActionBackupFull:
			res, err = backup.RunFullBackup(ctx, selectedSite.Path, selectedSite.Slug, backupPath, cfg.BackupExcludes, deps.WPClient, tui.PrintProgress)
			if err != nil {
				fmt.Printf("Full backup failed: %v\n", err)
				continue
			}
			tui.PrintBackupSummary(res)

		case tui.ActionBackupAI1WM:
			res, err = backup.RunAI1WMBackup(ctx, selectedSite.Path, selectedSite.Slug, backupPath, cfg.BackupExcludes, deps.Resolver, deps.WPClient, tui.PrintProgress)
			if err != nil {
				fmt.Printf("All-in-One WP Migration backup failed: %v\n", err)
				continue
			}
			tui.PrintBackupSummary(res)
		}
	}
}
