package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/backup"
	"wptui/internal/config"
	"wptui/internal/create"
	"wptui/internal/deprovision"
	"wptui/internal/launcher"
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
	if a.config != nil && a.config.UsedHerd {
		return fmt.Sprintf("https://%s.test", slug)
	}
	return fmt.Sprintf("http://%s.test", slug)
}

// BuildAppModel constructs and configures the root Bubble Tea AppModel with all operational handlers.
func (a *App) BuildAppModel(ctx context.Context) *tui.AppModel {
	m := tui.NewAppModel(a.config)

	client := wpcli.NewClient()
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
	if strings.TrimSpace(a.config.PackagesAPIURL) != "" {
		cat, err := packages.FetchCatalog(ctx, nil, a.config.PackagesAPIURL, a.config.PackagesAPIKey)
		if err == nil {
			catalog = cat
		}
	}

	slugSet := make(map[string]bool)
	for _, it := range catalog {
		slugSet[it.Slug] = true
	}
	for _, p := range a.config.Plugins {
		if !slugSet[p.Slug] {
			slugSet[p.Slug] = true
			catalog = append(catalog, packages.CatalogItem{
				Name: p.Name,
				Slug: p.Slug,
				Type: "plugin",
			})
		}
	}
	for _, t := range a.config.Themes {
		if !slugSet[t.Slug] {
			slugSet[t.Slug] = true
			catalog = append(catalog, packages.CatalogItem{
				Name: t.Name,
				Slug: t.Slug,
				Type: "theme",
			})
		}
	}

	m.SetCreateDependencies(catalog, checker)
	m.SetRestoreDependencies(checker)

	defaultRunner := &launcher.DefaultRunner{}
	cacheDir, _ := packages.DefaultCacheDir()
	m.SetSettingsDependencies(a.cfgPath, defaultRunner, cacheDir)
	if m.WebsitesHub() != nil {
		m.WebsitesHub().SetRunner(defaultRunner)
	}

	if m.SettingsModel() != nil {
		m.SettingsModel().OnReload = func(newCfg *config.Config) {
			a.config = newCfg
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
		var steps []tui.ProgressStep
		var title string
		if strategy == backup.StrategyAI1WM {
			steps = []tui.ProgressStep{
				{ID: "preflight", Title: "Verify WordPress Site", Status: tui.StepStatusPending},
				{ID: "extension", Title: "Verify AI1WM Extension", Status: tui.StepStatusPending},
				{ID: "ai1wm_backup", Title: "Generate AI1WM Backup", Status: tui.StepStatusPending},
				{ID: "relocate", Title: "Relocate Archive", Status: tui.StepStatusPending},
			}
			title = fmt.Sprintf("AI1WM Backup for %s", cand.Slug)
		} else {
			steps = []tui.ProgressStep{
				{ID: "preflight", Title: "Verify WordPress Site", Status: tui.StepStatusPending},
				{ID: "db_export", Title: "Export Database", Status: tui.StepStatusPending},
				{ID: "archive", Title: "Create Backup Archive", Status: tui.StepStatusPending},
				{ID: "cleanup", Title: "Cleanup Staging", Status: tui.StepStatusPending},
			}
			title = fmt.Sprintf("Full Backup for %s", cand.Slug)
		}
		cmd := m.StartProgressWithChannel(title, steps, ch)
		go a.runBackupExecution(ctx, cand, strategy, ch)
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

	m.SetInstallPackagesHandler(func(cand deprovision.Candidate, pkgType packages.PackageType, slugs []string, activate bool) tea.Cmd {
		ch := make(chan tea.Msg, 128)
		typeLabel := "Plugins"
		if pkgType == packages.PackageTypeTheme {
			typeLabel = "Themes"
		}
		steps := []tui.ProgressStep{
			{ID: "preflight", Title: "Verify WordPress Site", Status: tui.StepStatusPending},
			{ID: "resolve", Title: "Resolve Package Archives", Status: tui.StepStatusPending},
			{ID: "install", Title: fmt.Sprintf("Install %s", typeLabel), Status: tui.StepStatusPending},
		}
		cmd := m.StartProgressWithChannel(fmt.Sprintf("Installing %s on %s", typeLabel, cand.Slug), steps, ch)
		go a.runInstallPackagesExecution(ctx, cand, pkgType, slugs, activate, ch)
		return cmd
	})

	return m
}

func (a *App) runCreateExecution(ctx context.Context, inputs tui.CreateInputs, chosenPackages []string, steps []tui.ProgressStep, ch chan<- tea.Msg) {
	defer close(ch)

	client := wpcli.NewClient()
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
	var defaultThemeArt *packages.Artifact
	defaultThemeSkipped := false

	if a.config.DefaultThemeSlug != "" {
		if strings.TrimSpace(a.config.PackagesAPIURL) != "" && pkgResolver != nil {
			ch <- tui.LogLineMsg(fmt.Sprintf("Resolving default theme %s...", a.config.DefaultThemeSlug))
			art, err := pkgResolver.ResolvePackage(ctx, packages.PackageRef{Type: packages.PackageTypeTheme, Slug: a.config.DefaultThemeSlug}, stageDir)
			if err != nil {
				ch <- tui.StepCompleteMsg{ID: "resolve", Status: tui.StepStatusWarning, Detail: err.Error()}
				ch <- tui.LogLineMsg(fmt.Sprintf("Warning: failed to resolve default theme: %v", err))
			} else {
				defaultThemeArt = art
			}
		} else {
			defaultThemeSkipped = true
		}
	}

	if pkgResolver != nil && len(chosenPackages) > 0 {
		for _, slug := range chosenPackages {
			ch <- tui.LogLineMsg(fmt.Sprintf("Resolving package %s...", slug))
			art, err := pkgResolver.ResolvePackage(ctx, packages.PackageRef{Type: packages.PackageTypePlugin, Slug: slug}, stageDir)
			if err != nil {
				art, err = pkgResolver.ResolvePackage(ctx, packages.PackageRef{Type: packages.PackageTypeTheme, Slug: slug}, stageDir)
			}
			if err == nil && art != nil {
				if art.Ref.Type == packages.PackageTypeTheme {
					themeArts = append(themeArts, *art)
				} else {
					pluginArts = append(pluginArts, *art)
				}
			} else {
				ch <- tui.LogLineMsg(fmt.Sprintf("Warning: could not resolve %s: %v", slug, err))
			}
		}
	}
	ch <- tui.StepCompleteMsg{ID: "resolve", Status: tui.StepStatusSuccess, Detail: "Resolved"}

	tracker := NewCreateProgressTracker(ch, steps)
	ch <- tui.StepStartMsg{ID: "download", Title: "Downloading WordPress core..."}
	progress := tracker.StepFunc()

	req := create.Request{
		WebsiteName:         inputs.WebsiteName,
		WebsiteSlug:         inputs.WebsiteSlug,
		AdminUsername:       inputs.AdminUsername,
		AdminPassword:       inputs.AdminPassword,
		AdminEmail:          inputs.AdminEmail,
		ApplyTweaks:         inputs.ApplyTweaks,
		DefaultTheme:        defaultThemeArt,
		DefaultThemeSkipped: defaultThemeSkipped,
		PluginArtifacts:     pluginArts,
		ThemeArtifacts:      themeArts,
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

	client := wpcli.NewClient()
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

func (a *App) runBackupExecution(ctx context.Context, cand deprovision.Candidate, strategy backup.BackupStrategy, ch chan<- tea.Msg) {
	defer close(ch)

	client := wpcli.NewClient()
	backupPath := strings.TrimSpace(a.config.BackupPath)
	if backupPath == "" {
		backupPath = filepath.Join(a.config.WebsitesPath, "backups")
	}

	// 1. Preflight
	ch <- tui.StepStartMsg{ID: "preflight", Title: "Verifying WordPress directory and configuration..."}
	if _, err := os.Stat(cand.Path); err != nil {
		ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusFailed, Detail: "Directory missing"}
		ch <- tui.OperationCompleteMsg{
			Title:   "Backup Failed",
			Success: false,
			Summary: fmt.Sprintf("Directory:    %s\nError:        Website directory not found", cand.Path),
		}
		return
	}

	wpConfigPath := filepath.Join(cand.Path, "wp-config.php")
	if _, err := os.Stat(wpConfigPath); err != nil {
		ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusFailed, Detail: "wp-config.php missing"}
		ch <- tui.OperationCompleteMsg{
			Title:   "Backup Failed",
			Success: false,
			Summary: fmt.Sprintf("Directory:    %s\nError:        wp-config.php not found", cand.Path),
		}
		return
	}
	ch <- tui.StepCompleteMsg{ID: "preflight", Status: tui.StepStatusSuccess, Detail: "Validated"}

	var pkgResolver backup.PackageResolver
	if strings.TrimSpace(a.config.PackagesAPIURL) != "" {
		pkgCache, err := packages.NewCacheWithContext(ctx)
		if err == nil {
			pkgResolver = packages.NewResolver(a.config, pkgCache)
		}
	}

	if strategy == backup.StrategyAI1WM {
		ch <- tui.StepStartMsg{ID: "extension", Title: "Verifying AI1WM extension plugin..."}
		progress := func(step, total int, message string) {
			ch <- tui.LogLineMsg(message)
			switch step {
			case 1:
				ch <- tui.StepStartMsg{ID: "extension", Title: message}
			case 2:
				ch <- tui.StepCompleteMsg{ID: "extension", Status: tui.StepStatusSuccess, Detail: "Verified"}
				ch <- tui.StepStartMsg{ID: "ai1wm_backup", Title: message}
			case 3:
				ch <- tui.StepCompleteMsg{ID: "ai1wm_backup", Status: tui.StepStatusSuccess, Detail: "Generated"}
				ch <- tui.StepStartMsg{ID: "relocate", Title: message}
			}
		}

		res, err := backup.RunAI1WMBackup(ctx, cand.Path, cand.Slug, backupPath, a.config.BackupExcludes, pkgResolver, client, progress)
		if err != nil {
			ch <- tui.StepCompleteMsg{ID: "ai1wm_backup", Status: tui.StepStatusFailed, Detail: err.Error()}
			ch <- tui.OperationCompleteMsg{
				Title:   "AI1WM Backup Failed",
				Success: false,
				Summary: err.Error(),
			}
			return
		}

		ch <- tui.StepCompleteMsg{ID: "relocate", Status: tui.StepStatusSuccess, Detail: "Relocated"}
		summaryLines := []string{
			fmt.Sprintf("URL:          %s", a.siteURL(cand.Slug)),
			fmt.Sprintf("Directory:    %s", cand.Path),
			"Format:       All-in-One WP Migration (.wpress)",
			fmt.Sprintf("Archive:      %s", res.FilePath),
			fmt.Sprintf("Size:         %s (%d bytes)", tui.FormatBytes(res.FileSize), res.FileSize),
		}
		ch <- tui.OperationCompleteMsg{
			Title:   "AI1WM Backup Created Successfully",
			Success: true,
			Summary: strings.Join(summaryLines, "\n"),
		}
		return
	}

	// StrategyFull
	ch <- tui.StepStartMsg{ID: "db_export", Title: "Exporting database..."}
	progress := func(step, total int, message string) {
		ch <- tui.LogLineMsg(message)
		switch step {
		case 1:
			ch <- tui.StepStartMsg{ID: "db_export", Title: message}
		case 2:
			ch <- tui.StepCompleteMsg{ID: "db_export", Status: tui.StepStatusSuccess, Detail: "Exported"}
			ch <- tui.StepStartMsg{ID: "archive", Title: message}
		case 3, 4:
			ch <- tui.StepCompleteMsg{ID: "archive", Status: tui.StepStatusSuccess, Detail: "Archived"}
			ch <- tui.StepStartMsg{ID: "cleanup", Title: message}
		}
	}

	res, err := backup.RunFullBackup(ctx, cand.Path, cand.Slug, backupPath, a.config.BackupExcludes, client, progress)
	if err != nil {
		ch <- tui.StepCompleteMsg{ID: "archive", Status: tui.StepStatusFailed, Detail: err.Error()}
		ch <- tui.OperationCompleteMsg{
			Title:   "Full Backup Failed",
			Success: false,
			Summary: err.Error(),
		}
		return
	}

	ch <- tui.StepCompleteMsg{ID: "cleanup", Status: tui.StepStatusSuccess, Detail: "Cleaned"}
	summaryLines := []string{
		fmt.Sprintf("URL:          %s", a.siteURL(cand.Slug)),
		fmt.Sprintf("Directory:    %s", cand.Path),
		"Format:       Full Source Code & Database (.zip)",
		fmt.Sprintf("Archive:      %s", res.FilePath),
		fmt.Sprintf("Size:         %s (%d bytes)", tui.FormatBytes(res.FileSize), res.FileSize),
	}
	ch <- tui.OperationCompleteMsg{
		Title:   "Full Backup Created Successfully",
		Success: true,
		Summary: strings.Join(summaryLines, "\n"),
	}
}

func (a *App) runDeleteExecution(ctx context.Context, cands []deprovision.Candidate, ch chan<- tea.Msg) {
	defer close(ch)

	client := wpcli.NewClient()
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
		UsedHerd:    a.config.UsedHerd,
		Concurrency: 4,
		OnProgress: func(r deprovision.Result) {
			if r.HerdErr != nil || r.DBErr != nil || r.DirErr != nil {
				var errMsgs []string
				if r.HerdErr != nil {
					errMsgs = append(errMsgs, fmt.Sprintf("herd: %v", r.HerdErr))
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
		if r.HerdErr != nil || r.DBErr != nil || r.DirErr != nil {
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

	client := wpcli.NewClient()
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

func (a *App) runChangeAdminExecution(ctx context.Context, cand deprovision.Candidate, username, password, email string, ch chan<- tea.Msg) {
	defer close(ch)

	client := wpcli.NewClient()

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

func (a *App) runInstallPackagesExecution(ctx context.Context, cand deprovision.Candidate, pkgType packages.PackageType, slugs []string, activate bool, ch chan<- tea.Msg) {
	defer close(ch)

	client := wpcli.NewClient()
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
	for _, slug := range slugs {
		ch <- tui.LogLineMsg(fmt.Sprintf("Resolving %s %s...", pkgType, slug))
		if pkgResolver != nil {
			art, err := pkgResolver.ResolvePackage(ctx, packages.PackageRef{Type: pkgType, Slug: slug}, stageDir)
			if err != nil {
				ch <- tui.LogLineMsg(fmt.Sprintf("Warning: could not download %s: %v, falling back to WP.org slug", slug, err))
				artifacts = append(artifacts, packages.Artifact{
					Ref:  packages.PackageRef{Type: pkgType, Slug: slug},
					Path: slug,
				})
			} else {
				artifacts = append(artifacts, *art)
			}
		} else {
			artifacts = append(artifacts, packages.Artifact{
				Ref:  packages.PackageRef{Type: pkgType, Slug: slug},
				Path: slug,
			})
		}
	}
	ch <- tui.StepCompleteMsg{ID: "resolve", Status: tui.StepStatusSuccess, Detail: fmt.Sprintf("%d resolved", len(artifacts))}

	// 3. Install
	ch <- tui.StepStartMsg{ID: "install", Title: fmt.Sprintf("Installing %s via WP-CLI...", typeLabel)}
	onProgress := func(step, total int, message string) {
		ch <- tui.LogLineMsg(message)
	}

	results := siteconfig.InstallPackages(ctx, cand.Path, pkgType, artifacts, activate, client, onProgress)

	var successCount, skippedCount, failedCount int
	var failedNames []string
	for _, r := range results {
		if r.Skipped {
			skippedCount++
			ch <- tui.LogLineMsg(fmt.Sprintf("[-] %s: %s (skipped)", r.Slug, r.SkipReason))
		} else if r.Success {
			successCount++
			act := ""
			if r.Activated {
				act = " (activated)"
			}
			ch <- tui.LogLineMsg(fmt.Sprintf("✓ %s: installed%s", r.Slug, act))
		} else {
			failedCount++
			failedNames = append(failedNames, r.Slug)
			ch <- tui.LogLineMsg(fmt.Sprintf("✗ %s: %v", r.Slug, r.Err))
		}
	}

	summaryLines := []string{
		fmt.Sprintf("URL:          %s", a.siteURL(cand.Slug)),
		fmt.Sprintf("Directory:    %s", cand.Path),
		fmt.Sprintf("Package Type: %s", typeLabel),
		fmt.Sprintf("Installed:    %d", successCount),
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
