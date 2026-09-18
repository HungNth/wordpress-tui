package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2/spinner"
	"wptui/internal/config"
	"wptui/internal/restore"
	"wptui/internal/tui"
	"wptui/internal/wpcli"
)

// RunDefaultRestoreFlow orchestrates the interactive Website Restoration workflow.
func RunDefaultRestoreFlow(ctx context.Context, cfg *config.Config) error {
	client := wpcli.NewClient()
	if err := client.CheckDependencies(cfg.UsedHerd); err != nil {
		return err
	}

	// Step 1: Choose Strategy (Full ZIP or AI1WM)
	strategy, err := tui.PromptRestoreStrategy()
	if err != nil {
		return err
	}

	if strategy == restore.StrategyAI1WM && strings.TrimSpace(cfg.PackagesAPIURL) == "" {
		return errors.New("cannot perform AI1WM restore: packages_api_url is not configured in config.json")
	}
	// Step 2: Archive Picker
	archivePath, err := tui.PromptArchiveSelection(cfg.BackupPath, strategy)
	if err != nil {
		return err
	}

	// Step 3: Prompt Website Name, Slug & Admin overrides (no tweaks question)
	inputs, err := tui.PromptRestoreInputs(cfg)
	if err != nil {
		return err
	}

	// Step 4: Preflight non-collision validation
	targetDir := filepath.Join(cfg.WebsitesPath, inputs.WebsiteSlug)
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("website directory already exists: %s", targetDir)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("cannot inspect target directory: %w", err)
	}

	dbConn := wpcli.DBConnection{
		Host:   cfg.DatabaseHost,
		Port:   cfg.DatabasePort,
		User:   cfg.DBUsername,
		Pass:   cfg.DBPassword,
		Socket: cfg.DBSocket,
	}
	dbExists, err := client.CheckDatabaseExists(ctx, dbConn, inputs.WebsiteSlug)
	if err != nil {
		return fmt.Errorf("database preflight check failed: %w", err)
	}
	if dbExists {
		return fmt.Errorf("database already exists: %s", inputs.WebsiteSlug)
	}

	restorer := restore.NewRestorer(cfg, client)
	restorer.SetPromptDumpFunc(tui.PromptDumpSelection)

	req := restore.Request{
		Strategy:    strategy,
		ArchivePath: archivePath,
		WebsiteName: inputs.WebsiteName,
		WebsiteSlug: inputs.WebsiteSlug,
		AdminUser:   inputs.AdminUsername,
		AdminPass:   inputs.AdminPassword,
		AdminEmail:  inputs.AdminEmail,
	}

	var result *restore.Result

	action := func(actionCtx context.Context) error {
		res, err := restorer.Restore(actionCtx, req, func(step, description string) {
			tui.PrintRestoreProgress(step, description)
		})
		if err != nil {
			return err
		}
		result = res
		return nil
	}

	if err := spinner.New().
		Title("Restoring website from backup archive...").
		Context(ctx).
		ActionWithErr(action).
		Run(); err != nil {
		return err
	}

	tui.PrintRestoreSummary(result)

	return nil
}
