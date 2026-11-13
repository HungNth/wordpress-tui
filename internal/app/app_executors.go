package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/backup"
	"wptui/internal/config"
	"wptui/internal/create"
	"wptui/internal/deprovision"
	"wptui/internal/launcher"
	"wptui/internal/localenv"
	"wptui/internal/packages"
	"wptui/internal/restore"
	"wptui/internal/siteconfig"
	"wptui/internal/tui"
	"wptui/internal/wpcli"
)

func defaultAppRunner(ctx context.Context, model tea.Model) error {
	p := tea.NewProgram(model, tea.WithContext(ctx))
	_, err := p.Run()
	return err
}

func (a *App) siteURL(slug string) string {
	return localenv.SiteURL(a.config, slug)
}

// BuildAppModel constructs and configures the root Bubble Tea AppModel with all operational handlers.
func (a *App) BuildAppModel(ctx context.Context) *tui.AppModel {
	m := tui.NewAppModel(a.config)

	client := wpcli.NewClientForConfig(a.config)
	dbConn := wpcli.DBConnection{
		Host:   a.config.DatabaseHost,
		Port:   a.config.DatabasePort,
		User:   a.config.DBUsername,
		Pass:   a.config.DBPassword,
		Socket: a.config.DBSocket,
	}

	checker := func(slug string) error {
		targetPath := filepath.Join(a.config.WebsitesPath, slug)
		if fi, err := os.Stat(targetPath); err == nil && fi != nil {
			return fmt.Errorf("directory %s already exists", targetPath)
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to check directory %s: %w", targetPath, err)
		}

		exists, err := client.CheckDatabaseExists(ctx, dbConn, slug)
		if err != nil {
			return fmt.Errorf("database check failed: %w", err)
		}
		if exists {
			return fmt.Errorf("database %s already exists", slug)
		}
		return nil
	}

	var catalog []packages.CatalogItem
	var catalogWarning string
	if strings.TrimSpace(a.config.PackagesAPIURL) != "" {
		cat, err := packages.FetchCatalog(ctx, nil, a.config.PackagesAPIURL, a.config.PackagesAPIKey)
		if err != nil {
			catalogWarning = fmt.Sprintf("Package Catalog warning: %s", packages.SanitizeError(err))
		} else if len(cat) == 0 {
			catalogWarning = "Package Catalog warning: catalog returned 0 packages"
		} else {
			catalog = cat
		}
	}

	m.SetCreateDependencies(catalog, checker)
	if m.WebsitesHub() != nil {
		m.WebsitesHub().SetCatalog(catalog)
		m.WebsitesHub().SetCatalogWarning(catalogWarning)
	}
	m.SetRestoreDependencies(checker)

	defaultRunner := &launcher.DefaultRunner{}
	cacheDir, _ := packages.DefaultCacheDir()
	m.SetSettingsDependencies(a.cfgPath, defaultRunner, cacheDir)
	if m.WebsitesHub() != nil {
		m.WebsitesHub().SetRunner(defaultRunner)
	}

	if m.SettingsModel() != nil {
		m.SettingsModel().OnReload = func(newCfg *config.Config) {
			oldURL := a.config.PackagesAPIURL
			oldKey := a.config.PackagesAPIKey
			a.config = newCfg
			m.SetConfig(newCfg)
			if strings.TrimSpace(newCfg.PackagesAPIURL) == "" || newCfg.PackagesAPIURL != oldURL || newCfg.PackagesAPIKey != oldKey {
				catalog = nil
				m.InvalidateCatalog()
			}
		}
	}

	m.SetCreateHandler(func(inputs tui.CreateInputs, pkgs []string) tea.Cmd {
		ch := make(chan tea.Msg, 128)
		steps := BuildCreateSteps(inputs, pkgs, a.config)
		cmd := m.StartProgressWithChannel(fmt.Sprintf("Provisioning %s", inputs.WebsiteName), steps, ch)
		go a.runCreateExecution(ctx, inputs, pkgs, steps, ch)
		return cmd
	})

	m.SetRestoreHandler(func(strategy restore.Strategy, archivePath string, inputs tui.CreateInputs) tea.Cmd {
		ch := make(chan tea.Msg, 128)
		steps := []tui.ProgressStep{
			{ID: "preflight", Title: "Preflight Checks", Status: tui.StepStatusPending},
			{ID: "extract", Title: "Extract Archive", Status: tui.StepStatusPending},
			{ID: "database", Title: "Import Database", Status: tui.StepStatusPending},
			{ID: "credentials", Title: "Configure Administrator", Status: tui.StepStatusPending},
		}
		cmd := m.StartProgressWithChannel(fmt.Sprintf("Restoring %s", inputs.WebsiteSlug), steps, ch)
		go a.runRestoreExecution(ctx, strategy, archivePath, inputs, ch)
		return cmd
	})

	m.SetBackupHandler(func(cand deprovision.Candidate, strategy backup.BackupStrategy) tea.Cmd {
		ch := make(chan tea.Msg, 128)
		steps := backupStepsForStrategy(strategy)
		title := fmt.Sprintf("Full Backup for %s", cand.Slug)
		if strategy == backup.StrategyAI1WM {
			title = fmt.Sprintf("AI1WM Backup for %s", cand.Slug)
		}
		opCtx, cancel := context.WithCancel(ctx)
		cmd := m.StartProgressWithCancel(title, steps, ch, cancel)
		go func() {
			defer cancel()
			a.runBackupExecution(opCtx, cand, strategy, ch)
		}()
		return cmd
	})

	m.SetBatchBackupHandler(func(cands []deprovision.Candidate, strategy backup.BackupStrategy) tea.Cmd {
		if len(cands) == 0 {
			return nil
		}
		selected := slices.Clone(cands)
		ch := make(chan tea.Msg, 256)
		steps := backupStepsForStrategy(strategy)
		title := batchBackupTitle(strategy, 1, len(selected), selected[0].Slug)
		opCtx, cancel := context.WithCancel(ctx)
		cmd := m.StartProgressWithCancel(title, steps, ch, cancel)
		go func() {
			defer cancel()
			a.runBatchBackupExecution(opCtx, selected, strategy, ch)
		}()
		return cmd
	})

	m.SetDeleteHandler(func(cands []deprovision.Candidate) tea.Cmd {
		ch := make(chan tea.Msg, 128)
		steps := []tui.ProgressStep{
			{ID: "deprovision", Title: "De-provision Websites & Databases", Status: tui.StepStatusPending},
		}
		cmd := m.StartProgressWithChannel(fmt.Sprintf("Deleting %d website(s)", len(cands)), steps, ch)
		go a.runDeleteExecution(ctx, cands, ch)
		return cmd
	})

	m.SetConfigHandler(func(cand deprovision.Candidate) tea.Cmd {
		ch := make(chan tea.Msg, 128)
		steps := []tui.ProgressStep{
			{ID: "preflight", Title: "Verify WordPress Site", Status: tui.StepStatusPending},
			{ID: "tweaks", Title: "Apply WordPress Tweaks", Status: tui.StepStatusPending},
		}
		cmd := m.StartProgressWithChannel(fmt.Sprintf("Configuring %s", cand.Slug), steps, ch)
		go a.runConfigExecution(ctx, cand, ch)
		return cmd
	})

	m.SetToggleDebugHandler(func(cand deprovision.Candidate) tea.Cmd {
		ch := make(chan tea.Msg, 128)
		steps := []tui.ProgressStep{
			{ID: "preflight", Title: "Verify WordPress Website", Status: tui.StepStatusPending},
			{ID: "debug", Title: "Toggle Website Debug Mode", Status: tui.StepStatusPending},
		}
		opCtx, cancel := context.WithCancel(ctx)
		cmd := m.StartProgressWithCancel(fmt.Sprintf("Toggling Debug Mode for %s", cand.Slug), steps, ch, cancel)
		go func() {
			defer cancel()
			a.runToggleDebugExecution(opCtx, cand, ch)
		}()
		return cmd
	})

	m.SetChangeAdminHandler(func(cand deprovision.Candidate, username, password, email string) tea.Cmd {
		ch := make(chan tea.Msg, 128)
		steps := []tui.ProgressStep{
			{ID: "preflight", Title: "Verify WordPress Site", Status: tui.StepStatusPending},
			{ID: "discover", Title: "Discover Administrator", Status: tui.StepStatusPending},
			{ID: "dbconfig", Title: "Read Database Configuration", Status: tui.StepStatusPending},
			{ID: "update", Title: "Update Administrator Credentials", Status: tui.StepStatusPending},
		}
		cmd := m.StartProgressWithChannel(fmt.Sprintf("Updating credentials for %s", cand.Slug), steps, ch)
		go a.runChangeAdminExecution(ctx, cand, username, password, email, ch)
		return cmd
	})

	m.SetInstallPackagesHandler(func(cand deprovision.Candidate, pkgType packages.PackageType, slugs []string, activate bool, activeTheme ...string) tea.Cmd {
		ch := make(chan tea.Msg, 128)
		typeLabel := "Plugins"
		if pkgType == packages.PackageTypeTheme {
			typeLabel = "Themes"
		}
		steps := []tui.ProgressStep{
			{ID: "preflight", Title: "Verify WordPress Site", Status: tui.StepStatusPending},
			{ID: "resolve", Title: fmt.Sprintf("Resolve %s Archives", typeLabel), Status: tui.StepStatusPending},
			{ID: "install", Title: fmt.Sprintf("Install %s", typeLabel), Status: tui.StepStatusPending},
		}
		cmd := m.StartProgressWithChannel(fmt.Sprintf("Installing %s on %s", typeLabel, cand.Slug), steps, ch)
		actTheme := ""
		if len(activeTheme) > 0 {
			actTheme = activeTheme[0]
		}
		go a.runInstallPackagesExecution(ctx, cand, pkgType, slugs, activate, actTheme, ch)
		return cmd
	})

	return m
}

func (a *App) runCreateExecution(ctx context.Context, inputs tui.CreateInputs, chosenPackages []string, steps []tui.ProgressStep, ch chan<- tea.Msg) {
	defer close(ch)

	client := wpcli.NewClientForConfig(a.config)
	dbConn := wpcli.DBConnection{
		Host:   a.config.DatabaseHost,
		Port:   a.config.DatabasePort,
		User:   a.config.DBUsername,
		Pass:   a.config.DBPassword,
		Socket: a.config.DBSocket,
	}
	dbChecker := func(c context.Context, dbName string) (bool, error) {
		return client.CheckDatabaseExists(c, dbConn, dbName)
	}

	creator := create.NewCreator(a.config, client, dbChecker)

	stageDir, err := os.MkdirTemp("", "wptui-create-*")
	if err != nil {
		ch <- tui.OperationCompleteMsg{
			Title:   "Creation Failed",
			Success: false,
			Summary: fmt.Sprintf("Failed to create temporary directory: %v", err),
		}
		return
	}
	defer os.RemoveAll(stageDir)

	var pkgResolver PackageResolver
	if strings.TrimSpace(a.config.PackagesAPIURL) != "" {
		pkgCache, err := packages.NewCacheWithContext(ctx)
		if err == nil {
			pkgResolver = packages.NewResolver(a.config, pkgCache)
		}
	}

	ch <- tui.StepStartMsg{ID: "resolve", Title: "Resolving packages..."}
	var pluginArts []packages.Artifact
	var themeArts []packages.Artifact

	if len(chosenPackages) > 0 {
		if pkgResolver == nil {
			ch <- tui.StepCompleteMsg{ID: "resolve", Status: tui.StepStatusFailed, Detail: "Package service unavailable"}
			ch <- tui.OperationCompleteMsg{
				Title:   "Website Provisioning Failed",
				Success: false,
				Summary: "Package resolver is not configured but packages were selected",
			}
			return
		}
		for _, slug := range chosenPackages {
			ch <- tui.LogLineMsg(fmt.Sprintf("Resolving package %s...", slug))
			art, err := pkgResolver.ResolvePackage(ctx, packages.PackageRef{Type: packages.PackageTypePlugin, Slug: slug}, stageDir)
			if err != nil {
				art, err = pkgResolver.ResolvePackage(ctx, packages.PackageRef{Type: packages.PackageTypeTheme, Slug: slug}, stageDir)
			}
			if err != nil || art == nil {
				ch <- tui.StepCompleteMsg{ID: "resolve", Status: tui.StepStatusFailed, Detail: fmt.Sprintf("Failed to resolve %s", slug)}
				ch <- tui.OperationCompleteMsg{
					Title:   "Website Provisioning Failed",
					Success: false,
					Summary: fmt.Sprintf("Failed to resolve package %q: %v", slug, err),
				}
				return
			}
			if art.Ref.Type == packages.PackageTypeTheme {
				themeArts = append(themeArts, *art)
			} else {
				pluginArts = append(pluginArts, *art)
			}
		}
	}
	ch <- tui.StepCompleteMsg{ID: "resolve", Status: tui.StepStatusSuccess, Detail: "Resolved"}

	tracker := NewCreateProgressTracker(ch, steps)
	progress := tracker.StepFunc()

	req := create.Request{
		WebsiteName:     inputs.WebsiteName,
		WebsiteSlug:     inputs.WebsiteSlug,
		AdminUsername:   inputs.AdminUsername,
		AdminPassword:   inputs.AdminPassword,
		AdminEmail:      inputs.AdminEmail,
		ApplyTweaks:     inputs.ApplyTweaks,
		ActiveThemeSlug: inputs.ActiveThemeSlug,
		PluginArtifacts: pluginArts,
		ThemeArtifacts:  themeArts,
	}

	result, err := creator.Create(ctx, req, progress)
	if err != nil {
		tracker.MarkFailed(err)
		ch <- tui.OperationCompleteMsg{
			Title:   "Website Provisioning Failed",
			Success: false,
			Summary: err.Error(),
		}
		return
	}

	tracker.CompleteAllSuccess()
	ch <- tui.OperationCompleteMsg{
		Title:   "Website Provisioned Successfully",
		Success: true,
		Summary: FormatCreateSummary(inputs, result, a.config.DefaultAdminUsername),
	}
}

func (a *App) runRestoreExecution(ctx context.Context, strategy restore.Strategy, archivePath string, inputs tui.CreateInputs, ch chan<- tea.Msg) {
	defer close(ch)

	client := wpcli.NewClientForConfig(a.config)
	ch <- tui.StepStartMsg{ID: "preflight", Title: "Running preflight checks..."}

	targetDir := filepath.Join(a.config.WebsitesPath, inputs.WebsiteSlug)
	if _, err := os.Stat(targetDir); err == nil {
		ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusFailed, Detail: "Directory exists"}
		ch <- tui.OperationCompleteMsg{
			Title:   "Restoration Failed",
			Success: false,
			Summary: fmt.Sprintf("Website directory %s already exists", targetDir),
		}
		return
	}

	dbConn := wpcli.DBConnection{
		Host:   a.config.DatabaseHost,
		Port:   a.config.DatabasePort,
		User:   a.config.DBUsername,
		Pass:   a.config.DBPassword,
		Socket: a.config.DBSocket,
	}
	dbExists, err := client.CheckDatabaseExists(ctx, dbConn, inputs.WebsiteSlug)
	if err != nil {
		ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusFailed, Detail: err.Error()}
		ch <- tui.OperationCompleteMsg{
			Title:   "Restoration Failed",
			Success: false,
			Summary: fmt.Sprintf("Database check failed: %v", err),
		}
		return
	}
	if dbExists {
		ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusFailed, Detail: "Database exists"}
		ch <- tui.OperationCompleteMsg{
			Title:   "Restoration Failed",
			Success: false,
			Summary: fmt.Sprintf("Database %s already exists", inputs.WebsiteSlug),
		}
		return
	}
	ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusSuccess, Detail: "Passed"}

	restorer := restore.NewRestorer(a.config, client)
	progress := func(step, description string) {
		ch <- tui.LogLineMsg(description)
		switch step {
		case "extract_staging", "create_directory", "extract_core":
			ch <- tui.StepStartMsg{ID: "extract", Title: description}
		case "create_db", "import_db", "ai1wm_restore":
			ch <- tui.StepCompleteMsg{ID: "extract", Status: tui.StepStatusSuccess, Detail: "Extracted"}
			ch <- tui.StepStartMsg{ID: "database", Title: description}
		case "update_admin", "secure_herd":
			ch <- tui.StepCompleteMsg{ID: "database", Status: tui.StepStatusSuccess, Detail: "Complete"}
			ch <- tui.StepStartMsg{ID: "credentials", Title: description}
		}
	}

	req := restore.Request{
		Strategy:    strategy,
		ArchivePath: archivePath,
		WebsiteName: inputs.WebsiteName,
		WebsiteSlug: inputs.WebsiteSlug,
		AdminUser:   inputs.AdminUsername,
		AdminPass:   inputs.AdminPassword,
		AdminEmail:  inputs.AdminEmail,
	}

	result, err := restorer.Restore(ctx, req, progress)
	if err != nil {
		ch <- tui.OperationCompleteMsg{
			Title:   "Website Restoration Failed",
			Success: false,
			Summary: err.Error(),
		}
		return
	}

	ch <- tui.StepCompleteMsg{ID: "credentials", Status: tui.StepStatusSuccess, Detail: "Updated"}
	ch <- tui.OperationCompleteMsg{
		Title:   "Website Restored Successfully",
		Success: true,
		Summary: fmt.Sprintf("Path: %s\nURL:  %s", result.SitePath, result.SiteURL),
	}
}

func backupStepsForStrategy(strategy backup.BackupStrategy) []tui.ProgressStep {
	if strategy == backup.StrategyAI1WM {
		return []tui.ProgressStep{
			{ID: "preflight", Title: "Verify WordPress Site", Status: tui.StepStatusPending},
			{ID: "extension", Title: "Verify AI1WM Extension", Status: tui.StepStatusPending},
			{ID: "ai1wm_backup", Title: "Generate AI1WM Backup", Status: tui.StepStatusPending},
			{ID: "relocate", Title: "Relocate Archive", Status: tui.StepStatusPending},
		}
	}
	return []tui.ProgressStep{
		{ID: "preflight", Title: "Verify WordPress Site", Status: tui.StepStatusPending},
		{ID: "db_export", Title: "Export Database", Status: tui.StepStatusPending},
		{ID: "archive", Title: "Create Backup Archive", Status: tui.StepStatusPending},
		{ID: "cleanup", Title: "Cleanup Staging", Status: tui.StepStatusPending},
	}
}

func batchBackupTitle(strategy backup.BackupStrategy, idx, total int, slug string) string {
	if strategy == backup.StrategyAI1WM {
		return fmt.Sprintf("AI1WM Backup (%d/%d): %s", idx, total, slug)
	}
	return fmt.Sprintf("Full Backup (%d/%d): %s", idx, total, slug)
}

func backupFormatLabel(strategy backup.BackupStrategy) string {
	if strategy == backup.StrategyAI1WM {
		return "All-in-One WP Migration (.wpress)"
	}
	return "Full Source Code & Database (.zip)"
}

func (a *App) executeWebsiteBackup(
	ctx context.Context,
	cand deprovision.Candidate,
	strategy backup.BackupStrategy,
	backupPath string,
	pkgResolver backup.PackageResolver,
	client *wpcli.Client,
	ch chan<- tea.Msg,
) (res *backup.BackupResult, err error, preflightFailed bool) {
	if err := ctx.Err(); err != nil {
		return nil, err, false
	}
	ch <- tui.StepStartMsg{ID: "preflight", Title: "Verifying WordPress directory and configuration..."}
	if _, err := os.Stat(cand.Path); err != nil {
		ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusFailed, Detail: "Directory missing"}
		return nil, errors.New("Website directory not found"), true
	}
	wpConfigPath := filepath.Join(cand.Path, "wp-config.php")
	if _, err := os.Stat(wpConfigPath); err != nil {
		ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusFailed, Detail: "wp-config.php missing"}
		return nil, errors.New("wp-config.php not found"), true
	}
	ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusSuccess, Detail: "Validated"}

	runner := a.backupRunnerFn
	if runner == nil {
		runner = func(runCtx context.Context, target deprovision.Candidate, strat backup.BackupStrategy, destDir string, onProgress backup.ProgressFunc) (*backup.BackupResult, error) {
			if strat == backup.StrategyAI1WM {
				return backup.RunAI1WMBackup(runCtx, target.Path, target.Slug, destDir, a.config.BackupExcludes, pkgResolver, client, onProgress)
			}
			return backup.RunFullBackup(runCtx, target.Path, target.Slug, destDir, a.config.BackupExcludes, client, onProgress)
		}
	}

	activeStep := "db_export"
	initTitle := "Exporting database..."
	finalStep, finalDetail := "cleanup", "Cleaned"
	if strategy == backup.StrategyAI1WM {
		activeStep = "extension"
		initTitle = "Verifying AI1WM extension plugin..."
		finalStep, finalDetail = "relocate", "Relocated"
	}

	ch <- tui.StepStartMsg{ID: activeStep, Title: initTitle}
	progress := func(step, total int, message string) {
		ch <- tui.LogLineMsg(a.redactSecrets(message))
		if strategy == backup.StrategyAI1WM {
			switch step {
			case 1:
				activeStep = "extension"
				ch <- tui.StepStartMsg{ID: "extension", Title: message}
			case 2:
				ch <- tui.StepCompleteMsg{ID: "extension", Status: tui.StepStatusSuccess, Detail: "Verified"}
				activeStep = "ai1wm_backup"
				ch <- tui.StepStartMsg{ID: "ai1wm_backup", Title: message}
			case 3:
				ch <- tui.StepCompleteMsg{ID: "ai1wm_backup", Status: tui.StepStatusSuccess, Detail: "Generated"}
				activeStep = "relocate"
				ch <- tui.StepStartMsg{ID: "relocate", Title: message}
			}
			return
		}
		switch step {
		case 1:
			activeStep = "db_export"
			ch <- tui.StepStartMsg{ID: "db_export", Title: message}
		case 2:
			ch <- tui.StepCompleteMsg{ID: "db_export", Status: tui.StepStatusSuccess, Detail: "Exported"}
			activeStep = "archive"
			ch <- tui.StepStartMsg{ID: "archive", Title: message}
		case 3, 4:
			ch <- tui.StepCompleteMsg{ID: "archive", Status: tui.StepStatusSuccess, Detail: "Archived"}
			activeStep = "cleanup"
			ch <- tui.StepStartMsg{ID: "cleanup", Title: message}
		}
	}

	res, err = runner(ctx, cand, strategy, backupPath, progress)
	if err != nil {
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			ch <- tui.StepCompleteMsg{ID: activeStep, Status: tui.StepStatusWarning, Detail: "Cancelled"}
		} else {
			cause := a.redactSecrets(packages.SanitizeError(err).Error())
			ch <- tui.StepCompleteMsg{ID: activeStep, Status: tui.StepStatusFailed, Detail: cause}
		}
		return nil, err, false
	}
	ch <- tui.StepCompleteMsg{ID: finalStep, Status: tui.StepStatusSuccess, Detail: finalDetail}
	return res, nil, false
}

func (a *App) resolveBackupEnv(ctx context.Context) (string, backup.PackageResolver, *wpcli.Client) {
	client := wpcli.NewClientForConfig(a.config)
	backupPath := strings.TrimSpace(a.config.BackupPath)
	if backupPath == "" {
		backupPath = filepath.Join(a.config.WebsitesPath, "backups")
	}
	var pkgResolver backup.PackageResolver
	if strings.TrimSpace(a.config.PackagesAPIURL) != "" {
		if pkgCache, err := packages.NewCacheWithContext(ctx); err == nil {
			pkgResolver = packages.NewResolver(a.config, pkgCache)
		}
	}
	return backupPath, pkgResolver, client
}

func (a *App) runBackupExecution(ctx context.Context, cand deprovision.Candidate, strategy backup.BackupStrategy, ch chan<- tea.Msg) {
	defer close(ch)

	backupPath, pkgResolver, client := a.resolveBackupEnv(ctx)
	res, err, preflightFailed := a.executeWebsiteBackup(ctx, cand, strategy, backupPath, pkgResolver, client, ch)
	if err != nil {
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			ch <- tui.OperationCompleteMsg{
				Title:   "Backup Cancelled",
				Success: false,
				Warning: true,
				Summary: a.formatSingleBackupFailureSummary(cand, err),
			}
			return
		}
		failTitle := "Full Backup Failed"
		if preflightFailed {
			failTitle = "Backup Failed"
		} else if strategy == backup.StrategyAI1WM {
			failTitle = "AI1WM Backup Failed"
		}
		ch <- tui.OperationCompleteMsg{
			Title:   failTitle,
			Success: false,
			Summary: a.formatSingleBackupFailureSummary(cand, err),
		}
		return
	}

	successTitle := "Full Backup Created Successfully"
	if strategy == backup.StrategyAI1WM {
		successTitle = "AI1WM Backup Created Successfully"
	}
	summaryLines := []string{
		fmt.Sprintf("URL:          %s", a.siteURL(cand.Slug)),
		fmt.Sprintf("Directory:    %s", cand.Path),
		fmt.Sprintf("Format:       %s", backupFormatLabel(strategy)),
		fmt.Sprintf("Archive:      %s", res.FilePath),
		fmt.Sprintf("Size:         %s (%d bytes)", tui.FormatBytes(res.FileSize), res.FileSize),
	}
	ch <- tui.OperationCompleteMsg{
		Title:   successTitle,
		Success: true,
		Summary: strings.Join(summaryLines, "\n"),
	}
}

func (a *App) runBatchBackupExecution(ctx context.Context, cands []deprovision.Candidate, strategy backup.BackupStrategy, ch chan<- tea.Msg) {
	defer close(ch)

	backupPath, pkgResolver, client := a.resolveBackupEnv(ctx)
	total := len(cands)
	summaryLines := []string{
		fmt.Sprintf("Format:       %s", backupFormatLabel(strategy)),
	}

	succeeded := 0
	failed := 0
	cancelled := false

	for i, cand := range cands {
		if ctx.Err() != nil {
			cancelled = true
			for j := i; j < total; j++ {
				summaryLines = append(summaryLines, fmt.Sprintf("[-] %s — Skipped (batch cancelled)", cands[j].Slug))
			}
			break
		}
		if i > 0 {
			ch <- tui.ProgressResetStepsMsg{
				Title: batchBackupTitle(strategy, i+1, total, cand.Slug),
				Steps: backupStepsForStrategy(strategy),
			}
		}
		ch <- tui.LogLineMsg(fmt.Sprintf("=== (%d/%d) %s ===", i+1, total, cand.Slug))
		res, err, _ := a.executeWebsiteBackup(ctx, cand, strategy, backupPath, pkgResolver, client, ch)
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				cancelled = true
				summaryLines = append(summaryLines, fmt.Sprintf("[!] %s — Cancelled", cand.Slug))
				summaryLines = a.appendBatchArtifactDetails(summaryLines, err)
				ch <- tui.LogLineMsg(fmt.Sprintf("Cancelled %s; skipping %d queued website(s).", cand.Slug, total-i-1))
				for j := i + 1; j < total; j++ {
					summaryLines = append(summaryLines, fmt.Sprintf("[-] %s — Skipped (batch cancelled)", cands[j].Slug))
				}
				break
			}
			failed++
			cause := a.redactSecrets(packages.SanitizeError(err).Error())
			summaryLines = append(summaryLines, fmt.Sprintf("[✗] %s — %s", cand.Slug, cause))
			summaryLines = a.appendBatchArtifactDetails(summaryLines, err)
			continue
		}
		succeeded++
		summaryLines = append(summaryLines, fmt.Sprintf("[✓] %s — %s (%s, %d bytes)", cand.Slug, res.FilePath, tui.FormatBytes(res.FileSize), res.FileSize))
	}

	if ctx.Err() != nil {
		cancelled = true
	}

	var title string
	var success, warning bool
	switch {
	case cancelled:
		title = "Batch Backup Cancelled"
		warning = true
	case succeeded > 0 && failed == 0:
		title = "Batch Backup Complete"
		success = true
	case succeeded > 0 && failed > 0:
		title = "Batch Backup Completed with Warnings"
		warning = true
	default:
		title = "Batch Backup Failed"
	}

	ch <- tui.OperationCompleteMsg{
		Title:   title,
		Success: success,
		Warning: warning,
		Summary: strings.Join(summaryLines, "\n"),
	}
}

func (a *App) appendBatchArtifactDetails(lines []string, err error) []string {
	artErr, ok := errors.AsType[*backup.ArtifactError](err)
	if !ok {
		return lines
	}
	if artErr.ConflictPath != "" {
		lines = append(lines, fmt.Sprintf("  Conflict: %s", a.redactSecrets(artErr.ConflictPath)))
	}
	if artErr.Artifact.Path != "" {
		lines = append(lines, fmt.Sprintf("  Retained: %s", a.redactSecrets(artErr.Artifact.Path)))
	}
	if artErr.Artifact.CleanupOutcome != "" {
		lines = append(lines, fmt.Sprintf("  Cleanup: %s", a.redactSecrets(artErr.Artifact.CleanupOutcome)))
	}
	return lines
}

func (a *App) redactSecrets(text string) string {
	if a.config == nil || text == "" {
		return text
	}
	for _, secret := range []string{a.config.DBPassword, a.config.PackagesAPIKey, a.config.DefaultAdminPassword} {
		if s := strings.TrimSpace(secret); s != "" {
			text = strings.ReplaceAll(text, s, "REDACTED")
		}
	}
	return text
}

func (a *App) formatSingleBackupFailureSummary(cand deprovision.Candidate, err error) string {
	cause := a.redactSecrets(packages.SanitizeError(err).Error())
	lines := []string{
		fmt.Sprintf("Directory:    %s", a.redactSecrets(cand.Path)),
		fmt.Sprintf("Error:        %s", cause),
	}
	if artErr, ok := errors.AsType[*backup.ArtifactError](err); ok {
		if artErr.ConflictPath != "" {
			lines = append(lines, fmt.Sprintf("Conflict:     %s", a.redactSecrets(artErr.ConflictPath)))
		}
		if artErr.Artifact.Path != "" {
			lines = append(lines, fmt.Sprintf("Retained:     %s", a.redactSecrets(artErr.Artifact.Path)))
		}
		if artErr.Artifact.CleanupOutcome != "" {
			lines = append(lines, fmt.Sprintf("Cleanup:      %s", a.redactSecrets(artErr.Artifact.CleanupOutcome)))
		}
	}
	return strings.Join(lines, "\n")
}

func (a *App) runDeleteExecution(ctx context.Context, cands []deprovision.Candidate, ch chan<- tea.Msg) {
	defer close(ch)

	client := wpcli.NewClientForConfig(a.config)
	ch <- tui.StepStartMsg{ID: "deprovision", Title: "Resolving databases and de-provisioning..."}

	n := len(cands)
	limit := 4
	if limit > n {
		limit = n
	}
	if limit > 0 {
		sem := make(chan struct{}, limit)
		var wg sync.WaitGroup
		for i := range cands {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				deprovision.ResolveCandidateDB(ctx, &cands[idx], client)
				dbDesc := cands[idx].DetectedDB
				if dbDesc == "" {
					dbDesc = "none"
				}
				ch <- tui.LogLineMsg(fmt.Sprintf("Target: %s (DB: %s)", cands[idx].Slug, dbDesc))
			}(i)
		}
		wg.Wait()
	}

	results := deprovision.Deprovision(ctx, cands, client, deprovision.DeprovisionOptions{
		Config:      a.config,
		Concurrency: 4,
		OnProgress: func(r deprovision.Result) {
			if r.EnvironmentErr != nil || r.DBErr != nil || r.DirErr != nil {
				var errMsgs []string
				if r.EnvironmentErr != nil {
					errMsgs = append(errMsgs, fmt.Sprintf("environment: %v", r.EnvironmentErr))
				}
				if r.DBErr != nil {
					errMsgs = append(errMsgs, fmt.Sprintf("db: %v", r.DBErr))
				}
				if r.DirErr != nil {
					errMsgs = append(errMsgs, fmt.Sprintf("dir: %v", r.DirErr))
				}
				ch <- tui.LogLineMsg(fmt.Sprintf("Error deleting %s: %s", r.Candidate.Slug, strings.Join(errMsgs, ", ")))
			} else {
				ch <- tui.LogLineMsg(fmt.Sprintf("Successfully deleted %s", r.Candidate.Slug))
			}
		},
	})

	failedCount := 0
	for _, r := range results {
		if r.EnvironmentErr != nil || r.DBErr != nil || r.DirErr != nil {
			failedCount++
		}
	}

	if failedCount > 0 {
		ch <- tui.StepCompleteMsg{ID: "deprovision", Status: tui.StepStatusWarning, Detail: fmt.Sprintf("%d errors", failedCount)}
		ch <- tui.OperationCompleteMsg{
			Title:   "De-provisioning Finished With Warnings",
			Success: false,
			Warning: true,
			Summary: fmt.Sprintf("%d of %d site(s) had errors during deletion", failedCount, len(cands)),
		}
	} else {
		ch <- tui.StepCompleteMsg{ID: "deprovision", Status: tui.StepStatusSuccess, Detail: "Deleted"}
		ch <- tui.OperationCompleteMsg{
			Title:   "De-provisioning Complete",
			Success: true,
			Summary: fmt.Sprintf("Successfully deleted %d website(s)", len(cands)),
		}
	}
}

func (a *App) runConfigExecution(ctx context.Context, cand deprovision.Candidate, ch chan<- tea.Msg) {
	defer close(ch)

	client := wpcli.NewClientForConfig(a.config)
	ch <- tui.StepStartMsg{ID: "preflight", Title: "Verifying WordPress site..."}

	// 1. Check directory
	if fi, err := os.Stat(cand.Path); err != nil || !fi.IsDir() {
		ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusFailed, Detail: "Directory missing"}
		ch <- tui.OperationCompleteMsg{
			Title:   "Configuration Failed",
			Success: false,
			Summary: fmt.Sprintf("Directory:    %s\nError:        Website directory not found", cand.Path),
		}
		return
	}

	// 2. Check wp-config.php exists
	wpConfigPath := filepath.Join(cand.Path, "wp-config.php")
	if _, err := os.Stat(wpConfigPath); err != nil {
		ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusFailed, Detail: "wp-config.php missing"}
		ch <- tui.OperationCompleteMsg{
			Title:   "Configuration Failed",
			Success: false,
			Summary: fmt.Sprintf("Directory:    %s\nError:        wp-config.php not found", cand.Path),
		}
		return
	}

	ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusSuccess, Detail: "Validated"}

	// 3. Apply wp_tweaks
	ch <- tui.StepStartMsg{ID: "tweaks", Title: "Applying WordPress tweaks..."}
	if len(a.config.WPTweaks) == 0 {
		ch <- tui.LogLineMsg("No wp_tweaks configured in config.json.")
		ch <- tui.StepCompleteMsg{ID: "tweaks", Status: tui.StepStatusSuccess, Detail: "No tweaks configured"}
		ch <- tui.OperationCompleteMsg{
			Title:   "Website Configured Successfully",
			Success: true,
			Summary: fmt.Sprintf("URL:          %s\nDirectory:    %s\nTweaks:       0 configured in config.json", a.siteURL(cand.Slug), cand.Path),
		}
		return
	}

	onProgress := func(step, total int, message string) {
		ch <- tui.LogLineMsg(message)
		ch <- tui.StepStartMsg{ID: "tweaks", Title: fmt.Sprintf("Applying tweak %d of %d: %s", step, total, message)}
	}

	results := siteconfig.ApplyTweaks(ctx, cand.Path, a.config.WPTweaks, client, onProgress)

	var failed []string
	appliedCount := 0
	for _, res := range results {
		if res.Success {
			appliedCount++
			ch <- tui.LogLineMsg(fmt.Sprintf("✓ Applied [%s] %s=%s", res.Tweak.Type, res.Tweak.Key, res.Tweak.Value))
		} else {
			failed = append(failed, fmt.Sprintf("[%s] %s: %v", res.Tweak.Type, res.Tweak.Key, res.Err))
			ch <- tui.LogLineMsg(fmt.Sprintf("✗ Failed [%s] %s: %v", res.Tweak.Type, res.Tweak.Key, res.Err))
		}
	}

	summaryLines := []string{
		fmt.Sprintf("URL:          %s", a.siteURL(cand.Slug)),
		fmt.Sprintf("Directory:    %s", cand.Path),
		fmt.Sprintf("Tweaks:       %d applied", appliedCount),
	}

	if len(failed) > 0 {
		summaryLines = append(summaryLines, fmt.Sprintf("[!] Failed:    %d tweak(s) failed", len(failed)))
		ch <- tui.StepCompleteMsg{ID: "tweaks", Status: tui.StepStatusWarning, Detail: fmt.Sprintf("%d applied, %d failed", appliedCount, len(failed))}
		ch <- tui.OperationCompleteMsg{
			Title:   "Configuration Completed with Warnings",
			Success: true,
			Warning: true,
			Summary: strings.Join(summaryLines, "\n"),
		}
	} else {
		ch <- tui.StepCompleteMsg{ID: "tweaks", Status: tui.StepStatusSuccess, Detail: "Complete"}
		ch <- tui.OperationCompleteMsg{
			Title:   "Website Configured Successfully",
			Success: true,
			Summary: strings.Join(summaryLines, "\n"),
		}
	}
}

func (a *App) runToggleDebugExecution(ctx context.Context, cand deprovision.Candidate, ch chan<- tea.Msg) {
	defer close(ch)

	ch <- tui.StepStartMsg{ID: "preflight", Title: "Verifying Website directory and wp-config.php..."}

	if fi, err := os.Stat(cand.Path); err != nil || !fi.IsDir() {
		ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusFailed, Detail: "Directory missing"}
		ch <- tui.OperationCompleteMsg{
			Title:   "Debug Mode Toggle Failed",
			Success: false,
			Summary: fmt.Sprintf("Website:   %s\nDirectory: %s\nError:     Website directory not found", cand.Slug, cand.Path),
		}
		return
	}

	wpConfigPath := filepath.Join(cand.Path, "wp-config.php")
	if _, err := os.Stat(wpConfigPath); err != nil {
		ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusFailed, Detail: "wp-config.php missing"}
		ch <- tui.OperationCompleteMsg{
			Title:   "Debug Mode Toggle Failed",
			Success: false,
			Summary: fmt.Sprintf("Website:   %s\nDirectory: %s\nError:     wp-config.php not found", cand.Slug, cand.Path),
		}
		return
	}
	ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusSuccess, Detail: "Validated"}

	ch <- tui.StepStartMsg{ID: "debug", Title: "Inspecting and updating debug constants..."}
	client := wpcli.NewClientForConfig(a.config)
	onProgress := func(step, total int, message string) {
		ch <- tui.LogLineMsg(message)
		if step > 0 {
			ch <- tui.StepStartMsg{ID: "debug", Title: fmt.Sprintf("Updating debug constant %d of %d: %s", step, total, message)}
		}
	}

	res, err := siteconfig.ToggleDebugMode(ctx, cand.Path, client, onProgress)

	var failed []string
	appliedCount := 0
	for _, w := range res.Writes {
		if w.Success {
			appliedCount++
			ch <- tui.LogLineMsg(fmt.Sprintf("[✓] Set %s=%s", w.Key, w.Value))
		} else {
			failed = append(failed, fmt.Sprintf("[✗] Failed: %s=%s (%v)", w.Key, w.Value, w.Err))
			ch <- tui.LogLineMsg(fmt.Sprintf("[✗] Failed %s=%s: %v", w.Key, w.Value, w.Err))
		}
	}

	targetLabel := "Unknown"
	if res.TargetKnown {
		if res.TargetEnabled {
			targetLabel = "Enabled"
		} else {
			targetLabel = "Disabled"
		}
	}

	if res.Cancelled || errors.Is(err, context.Canceled) {
		ch <- tui.StepCompleteMsg{ID: "debug", Status: tui.StepStatusWarning, Detail: "Cancelled"}
		summaryLines := []string{
			fmt.Sprintf("Website:   %s", cand.Slug),
			fmt.Sprintf("Directory: %s", cand.Path),
			fmt.Sprintf("Target:    %s", targetLabel),
			fmt.Sprintf("Updated:   %d constant(s)", appliedCount),
		}
		summaryLines = append(summaryLines, failed...)
		ch <- tui.OperationCompleteMsg{
			Title:   "Website Debug Mode Cancelled",
			Success: false,
			Warning: true,
			Summary: strings.Join(summaryLines, "\n"),
		}
		return
	}

	if err != nil {
		ch <- tui.StepCompleteMsg{ID: "debug", Status: tui.StepStatusFailed, Detail: err.Error()}
		ch <- tui.OperationCompleteMsg{
			Title:   "Debug Mode Toggle Failed",
			Success: false,
			Summary: fmt.Sprintf("Website:   %s\nDirectory: %s\nError:     %v", cand.Slug, cand.Path, err),
		}
		return
	}

	if len(failed) > 0 {
		ch <- tui.StepCompleteMsg{ID: "debug", Status: tui.StepStatusWarning, Detail: fmt.Sprintf("%d updated, %d failed", appliedCount, len(failed))}
		summaryLines := []string{
			fmt.Sprintf("Website:   %s", cand.Slug),
			fmt.Sprintf("Directory: %s", cand.Path),
			fmt.Sprintf("Target:    %s", targetLabel),
			fmt.Sprintf("Updated:   %d constant(s)", appliedCount),
		}
		summaryLines = append(summaryLines, failed...)
		ch <- tui.OperationCompleteMsg{
			Title:   "Website Debug Mode Completed with Warnings",
			Success: false,
			Warning: true,
			Summary: strings.Join(summaryLines, "\n"),
		}
		return
	}

	stateDetail := "Disabled (WP_DEBUG=false, WP_DEBUG_LOG=false, WP_DEBUG_DISPLAY=false)"
	if res.TargetEnabled {
		stateDetail = "Enabled (WP_DEBUG=true, WP_DEBUG_LOG=true, WP_DEBUG_DISPLAY=false)"
	}
	ch <- tui.StepCompleteMsg{ID: "debug", Status: tui.StepStatusSuccess, Detail: targetLabel}
	ch <- tui.OperationCompleteMsg{
		Title:   "Website Debug Mode " + targetLabel,
		Success: true,
		Summary: strings.Join([]string{
			fmt.Sprintf("Website:   %s", cand.Slug),
			fmt.Sprintf("URL:       %s", a.siteURL(cand.Slug)),
			fmt.Sprintf("Directory: %s", cand.Path),
			fmt.Sprintf("State:     %s", stateDetail),
		}, "\n"),
	}
}

func (a *App) runChangeAdminExecution(ctx context.Context, cand deprovision.Candidate, username, password, email string, ch chan<- tea.Msg) {
	defer close(ch)

	client := wpcli.NewClientForConfig(a.config)

	// 1. Preflight: Verify site directory & wp-config.php
	ch <- tui.StepStartMsg{ID: "preflight", Title: "Verifying WordPress directory and configuration..."}
	if _, err := os.Stat(cand.Path); err != nil {
		ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusFailed, Detail: "Directory missing"}
		ch <- tui.OperationCompleteMsg{
			Title:   "Credential Update Failed",
			Success: false,
			Summary: fmt.Sprintf("Directory:    %s\nError:        Website directory not found", cand.Path),
		}
		return
	}

	wpConfigPath := filepath.Join(cand.Path, "wp-config.php")
	if _, err := os.Stat(wpConfigPath); err != nil {
		ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusFailed, Detail: "wp-config.php missing"}
		ch <- tui.OperationCompleteMsg{
			Title:   "Credential Update Failed",
			Success: false,
			Summary: fmt.Sprintf("Directory:    %s\nError:        wp-config.php not found", cand.Path),
		}
		return
	}
	ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusSuccess, Detail: "Validated"}

	// 2. Discover Administrator
	ch <- tui.StepStartMsg{ID: "discover", Title: "Discovering administrator account..."}
	admins, err := siteconfig.DiscoverAdministrators(ctx, cand.Path, client)
	if err != nil {
		ch <- tui.StepCompleteMsg{ID: "discover", Status: tui.StepStatusFailed, Detail: err.Error()}
		ch <- tui.OperationCompleteMsg{
			Title:   "Credential Update Failed",
			Success: false,
			Summary: fmt.Sprintf("Directory:    %s\nError:        Failed to discover administrator: %v", cand.Path, err),
		}
		return
	}
	targetAdmin := admins[0]
	ch <- tui.StepCompleteMsg{ID: "discover", Status: tui.StepStatusSuccess, Detail: fmt.Sprintf("Admin ID %d (%s)", targetAdmin.ID, targetAdmin.UserLogin)}

	// 3. Extract DB Config
	ch <- tui.StepStartMsg{ID: "dbconfig", Title: "Reading database configuration..."}
	dbCfg, err := siteconfig.ExtractDBConfig(ctx, cand.Path, client)
	if err != nil {
		ch <- tui.StepCompleteMsg{ID: "dbconfig", Status: tui.StepStatusFailed, Detail: err.Error()}
		ch <- tui.OperationCompleteMsg{
			Title:   "Credential Update Failed",
			Success: false,
			Summary: fmt.Sprintf("Directory:    %s\nError:        Failed to read database configuration: %v", cand.Path, err),
		}
		return
	}
	ch <- tui.StepCompleteMsg{ID: "dbconfig", Status: tui.StepStatusSuccess, Detail: fmt.Sprintf("Database: %s", dbCfg.Name)}

	// 4. Update Credentials
	ch <- tui.StepStartMsg{ID: "update", Title: "Updating administrator credentials..."}
	onProgress := func(step, total int, message string) {
		ch <- tui.LogLineMsg(message)
	}

	input := siteconfig.AdminInput{
		UserID:      targetAdmin.ID,
		NewUsername: username,
		NewPassword: password,
		NewEmail:    email,
	}

	if err := siteconfig.UpdateAdminCredentials(ctx, cand.Path, dbCfg, input, client, nil, onProgress); err != nil {
		ch <- tui.StepCompleteMsg{ID: "update", Status: tui.StepStatusFailed, Detail: err.Error()}
		ch <- tui.OperationCompleteMsg{
			Title:   "Credential Update Failed",
			Success: false,
			Summary: fmt.Sprintf("Directory:    %s\nError:        %v", cand.Path, err),
		}
		return
	}

	ch <- tui.StepCompleteMsg{ID: "update", Status: tui.StepStatusSuccess, Detail: "Updated successfully"}

	summaryLines := []string{
		fmt.Sprintf("URL:          %s", a.siteURL(cand.Slug)),
		fmt.Sprintf("Directory:    %s", cand.Path),
		fmt.Sprintf("Admin User:   %s (ID: %d)", username, targetAdmin.ID),
		fmt.Sprintf("Admin Email:  %s", email),
		"Password:     (updated successfully)",
	}
	ch <- tui.OperationCompleteMsg{
		Title:   "Admin Credentials Updated Successfully",
		Success: true,
		Summary: strings.Join(summaryLines, "\n"),
	}
}

func (a *App) runInstallPackagesExecution(ctx context.Context, cand deprovision.Candidate, pkgType packages.PackageType, slugs []string, activate bool, activeTheme string, ch chan<- tea.Msg) {
	defer close(ch)

	client := wpcli.NewClientForConfig(a.config)
	typeLabel := "Plugins"
	if pkgType == packages.PackageTypeTheme {
		typeLabel = "Themes"
	}

	// 1. Preflight
	ch <- tui.StepStartMsg{ID: "preflight", Title: "Verifying WordPress directory and configuration..."}
	if _, err := os.Stat(cand.Path); err != nil {
		ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusFailed, Detail: "Directory missing"}
		ch <- tui.OperationCompleteMsg{
			Title:   fmt.Sprintf("Failed to Install %s", typeLabel),
			Success: false,
			Summary: fmt.Sprintf("Directory:    %s\nError:        Website directory not found", cand.Path),
		}
		return
	}

	wpConfigPath := filepath.Join(cand.Path, "wp-config.php")
	if _, err := os.Stat(wpConfigPath); err != nil {
		ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusFailed, Detail: "wp-config.php missing"}
		ch <- tui.OperationCompleteMsg{
			Title:   fmt.Sprintf("Failed to Install %s", typeLabel),
			Success: false,
			Summary: fmt.Sprintf("Directory:    %s\nError:        wp-config.php not found", cand.Path),
		}
		return
	}
	ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusSuccess, Detail: "Validated"}

	// 2. Resolve
	stageDir, err := os.MkdirTemp("", fmt.Sprintf("wptui-%s-*", pkgType))
	if err != nil {
		ch <- tui.StepCompleteMsg{ID: "resolve", Status: tui.StepStatusFailed, Detail: err.Error()}
		ch <- tui.OperationCompleteMsg{
			Title:   fmt.Sprintf("Failed to Install %s", typeLabel),
			Success: false,
			Summary: fmt.Sprintf("Failed to create temporary directory: %v", err),
		}
		return
	}
	defer os.RemoveAll(stageDir)

	var pkgResolver PackageResolver
	if strings.TrimSpace(a.config.PackagesAPIURL) != "" {
		pkgCache, err := packages.NewCacheWithContext(ctx)
		if err == nil {
			pkgResolver = packages.NewResolver(a.config, pkgCache)
		}
	}

	ch <- tui.StepStartMsg{ID: "resolve", Title: fmt.Sprintf("Resolving %s archives...", typeLabel)}
	var artifacts []packages.Artifact
	var failedNames []string
	var failedCount int

	for _, slug := range slugs {
		ch <- tui.LogLineMsg(fmt.Sprintf("Resolving %s %s...", pkgType, slug))
		if pkgResolver == nil {
			failedCount++
			failedNames = append(failedNames, slug)
			ch <- tui.LogLineMsg(fmt.Sprintf("✗ %s: package service not configured", slug))
			continue
		}
		art, err := pkgResolver.ResolvePackage(ctx, packages.PackageRef{Type: pkgType, Slug: slug}, stageDir)
		if err != nil {
			failedCount++
			failedNames = append(failedNames, slug)
			ch <- tui.LogLineMsg(fmt.Sprintf("✗ %s: %v", slug, err))
			continue
		}
		artifacts = append(artifacts, *art)
	}

	if len(artifacts) == 0 && len(slugs) > 0 {
		ch <- tui.StepCompleteMsg{ID: "resolve", Status: tui.StepStatusFailed, Detail: "All package resolutions failed"}
		ch <- tui.OperationCompleteMsg{
			Title:   fmt.Sprintf("Failed to Install %s", typeLabel),
			Success: false,
			Summary: fmt.Sprintf("Failed to resolve packages: %s", strings.Join(failedNames, ", ")),
		}
		return
	}
	ch <- tui.StepCompleteMsg{ID: "resolve", Status: tui.StepStatusSuccess, Detail: fmt.Sprintf("%d resolved", len(artifacts))}

	// 3. Install
	ch <- tui.StepStartMsg{ID: "install", Title: fmt.Sprintf("Installing %s via WP-CLI...", typeLabel)}
	onProgress := func(step, total int, message string) {
		ch <- tui.LogLineMsg(message)
	}

	installActivate := activate
	if pkgType == packages.PackageTypeTheme {
		installActivate = false
	}
	results := siteconfig.InstallPackages(ctx, cand.Path, pkgType, artifacts, installActivate, client, onProgress)

	var successCount, skippedCount int
	for _, r := range results {
		if r.Skipped {
			skippedCount++
			ch <- tui.LogLineMsg(fmt.Sprintf("[-] %s: %s (skipped)", r.Slug, r.SkipReason))
		} else if r.Success {
			successCount++
			ch <- tui.LogLineMsg(fmt.Sprintf("✓ %s: installed", r.Slug))
		} else {
			failedCount++
			failedNames = append(failedNames, r.Slug)
			ch <- tui.LogLineMsg(fmt.Sprintf("✗ %s: %v", r.Slug, r.Err))
		}
	}

	if pkgType == packages.PackageTypeTheme && activeTheme != "" {
		ch <- tui.LogLineMsg(fmt.Sprintf("Activating theme %s...", activeTheme))
		_, stderr, actErr := client.Run(ctx, cand.Path, "wp", []string{"theme", "activate", activeTheme}, "")
		if actErr != nil {
			ch <- tui.LogLineMsg(fmt.Sprintf("✗ Failed to activate theme %s: %v (%s)", activeTheme, actErr, stderr))
		} else {
			ch <- tui.LogLineMsg(fmt.Sprintf("✓ Theme %s activated", activeTheme))
		}
	}

	summaryLines := []string{
		fmt.Sprintf("URL:          %s", a.siteURL(cand.Slug)),
		fmt.Sprintf("Directory:    %s", cand.Path),
		fmt.Sprintf("Package Type: %s", typeLabel),
		fmt.Sprintf("Installed:    %d", successCount),
	}
	if pkgType == packages.PackageTypeTheme {
		if activeTheme != "" {
			summaryLines = append(summaryLines, fmt.Sprintf("Active Theme: %s", activeTheme))
		} else {
			summaryLines = append(summaryLines, "Active Theme: (none)")
		}
	}
	if skippedCount > 0 {
		summaryLines = append(summaryLines, fmt.Sprintf("Skipped:      %d (already up to date)", skippedCount))
	}
	if failedCount > 0 {
		summaryLines = append(summaryLines, fmt.Sprintf("[!] Failed:    %d (%s)", failedCount, strings.Join(failedNames, ", ")))
		ch <- tui.StepCompleteMsg{ID: "install", Status: tui.StepStatusWarning, Detail: fmt.Sprintf("%d installed, %d failed", successCount, failedCount)}
		ch <- tui.OperationCompleteMsg{
			Title:   fmt.Sprintf("%s Installation Completed with Warnings", typeLabel),
			Success: true,
			Warning: true,
			Summary: strings.Join(summaryLines, "\n"),
		}
	} else {
		ch <- tui.StepCompleteMsg{ID: "install", Status: tui.StepStatusSuccess, Detail: fmt.Sprintf("%d installed", successCount)}
		ch <- tui.OperationCompleteMsg{
			Title:   fmt.Sprintf("%s Installed Successfully", typeLabel),
			Success: true,
			Summary: strings.Join(summaryLines, "\n"),
		}
	}
}
