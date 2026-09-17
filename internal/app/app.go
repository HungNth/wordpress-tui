package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"wptui/internal/config"
	"wptui/internal/create"
	"wptui/internal/deprovision"
	"wptui/internal/packages"
	"wptui/internal/tui"
	"wptui/internal/wpcli"
)

type Options struct {
	HomeDir  string
	WizardFn func(homeDir string) (*config.Config, error)
	MenuFn   func() (string, error)
	CreateFn func(ctx context.Context, cfg *config.Config) error
	DeleteFn func(ctx context.Context, cfg *config.Config) error
}

type App struct {
	homeDir  string
	cfgPath  string
	config   *config.Config
	wizardFn func(homeDir string) (*config.Config, error)
	menuFn   func() (string, error)
	createFn func(ctx context.Context, cfg *config.Config) error
	deleteFn func(ctx context.Context, cfg *config.Config) error
}

func New(opts Options) *App {
	home := opts.HomeDir
	if home == "" {
		h, err := os.UserHomeDir()
		if err == nil {
			home = h
		}
	}

	cfgPath, _ := config.ConfigPath(home)

	wFn := opts.WizardFn
	if wFn == nil {
		wFn = tui.RunConfigWizard
	}

	mFn := opts.MenuFn
	if mFn == nil {
		mFn = tui.RunMainMenu
	}

	cFn := opts.CreateFn
	if cFn == nil {
		cFn = RunDefaultCreateFlow
	}

	dFn := opts.DeleteFn
	if dFn == nil {
		dFn = RunDefaultDeleteFlow
	}
	return &App{
		homeDir:  home,
		cfgPath:  cfgPath,
		wizardFn: wFn,
		menuFn:   mFn,
		createFn: cFn,
		deleteFn: dFn,
	}
}

func (a *App) Config() *config.Config {
	return a.config
}

func (a *App) ConfigPath() string {
	return a.cfgPath
}

// InitConfig checks whether config.json exists. If missing, it runs the wizard and saves it.
// If it exists, it loads and validates it.
func (a *App) InitConfig(interactive bool) (*config.Config, error) {
	if _, err := os.Stat(a.cfgPath); errors.Is(err, os.ErrNotExist) {
		if !interactive && a.wizardFn == nil {
			return nil, fmt.Errorf("config file not found at %s", a.cfgPath)
		}

		fmt.Println("No existing configuration found. Starting setup wizard...")
		cfg, err := a.wizardFn(a.homeDir)
		if err != nil {
			return nil, fmt.Errorf("wizard failed or cancelled: %w", err)
		}

		if err := config.Save(a.cfgPath, cfg); err != nil {
			return nil, fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Configuration saved successfully to %s\n", a.cfgPath)
		a.config = cfg
		return cfg, nil
	}

	cfg, err := config.Load(a.cfgPath)
	if err != nil {
		return nil, fmt.Errorf("configuration error at %s: %w", a.cfgPath, err)
	}

	a.config = cfg
	return cfg, nil
}

type PackageResolver interface {
	ResolvePackage(ctx context.Context, ref packages.PackageRef, stageDir string) (*packages.Artifact, error)
	ResolveAll(ctx context.Context, refs []packages.PackageRef, stageDir string) ([]packages.Artifact, error)
}

type CreateFlowDependencies struct {
	Runner         wpcli.Runner
	Resolver       PackageResolver
	CoreResolver   create.CoreResolver
	CoreExtractor  create.CoreExtractor
	Catalog        []packages.CatalogItem
	PromptCreate   func(*config.Config, ...tui.SlugAvailabilityChecker) (*tui.CreateInputs, error)
	PromptPackages func(context.Context, *config.Config, []packages.CatalogItem) ([]string, []string, error)
}

func RunCreateFlowWithDeps(ctx context.Context, cfg *config.Config, deps CreateFlowDependencies) error {
	var client *wpcli.Client
	if deps.Runner != nil {
		client = wpcli.NewClientWithRunner(deps.Runner)
	} else {
		client = wpcli.NewClient()
	}

	dbConn := wpcli.DBConnection{
		Host:   cfg.DatabaseHost,
		Port:   cfg.DatabasePort,
		User:   cfg.DBUsername,
		Pass:   cfg.DBPassword,
		Socket: cfg.DBSocket,
	}
	dbChecker := func(c context.Context, dbName string) (bool, error) {
		return client.CheckDatabaseExists(c, dbConn, dbName)
	}
	availabilityChecker := func(slug string) error {
		targetPath := filepath.Join(cfg.WebsitesPath, slug)
		if fi, err := os.Stat(targetPath); err == nil && fi != nil {
			return fmt.Errorf("directory %s already exists", targetPath)
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to check directory %s: %w", targetPath, err)
		}

		exists, err := dbChecker(ctx, slug)
		if err != nil {
			return fmt.Errorf("database check failed: %w", err)
		}
		if exists {
			return fmt.Errorf("database %s already exists", slug)
		}
		return nil
	}

	var creatorOpts []create.CreatorOption
	if deps.CoreResolver != nil {
		creatorOpts = append(creatorOpts, create.WithCoreResolver(deps.CoreResolver))
	}
	if deps.CoreExtractor != nil {
		creatorOpts = append(creatorOpts, create.WithCoreExtractor(deps.CoreExtractor))
	}
	creator := create.NewCreator(cfg, client, dbChecker, creatorOpts...)

	promptCreate := deps.PromptCreate
	if promptCreate == nil {
		promptCreate = tui.PromptCreateInputs
	}
	promptPackages := deps.PromptPackages
	if promptPackages == nil {
		promptPackages = tui.PromptPackageSelections
	}
	// Fetch catalog once if package API is configured and reuse across collision retries
	catalog := deps.Catalog
	if len(catalog) == 0 && strings.TrimSpace(cfg.PackagesAPIURL) != "" {
		cat, err := packages.FetchCatalog(ctx, nil, cfg.PackagesAPIURL, cfg.PackagesAPIKey)
		if err == nil {
			catalog = cat
		}
	}

	for {
		inputs, err := promptCreate(cfg, availabilityChecker)
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil
			}
			return err
		}

		// Explicit defense: re-verify slug availability if custom or bypassed prompt returned a colliding slug
		if err := availabilityChecker(inputs.WebsiteSlug); err != nil {
			if deps.PromptCreate != nil {
				return err
			}
			fmt.Printf("\nCollision: %v\nPlease choose a different website name or slug.\n\n", err)
			continue
		}
		plugins, themes, err := promptPackages(ctx, cfg, catalog)
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil
			}
			return err
		}

		// Pre-mutation: resolve package artifacts before creating website or database
		stageDir, err := os.MkdirTemp("", "wptui-stage-*")
		if err != nil {
			return fmt.Errorf("failed to create package staging directory: %w", err)
		}
		defer os.RemoveAll(stageDir)

		var defaultThemeArt *packages.Artifact
		defaultThemeSkipped := false
		if cfg.DefaultThemeSlug != "" {
			if strings.TrimSpace(cfg.PackagesAPIURL) != "" && deps.Resolver != nil {
				fmt.Printf("→ Resolving default theme %s...\n", cfg.DefaultThemeSlug)
				art, err := deps.Resolver.ResolvePackage(ctx, packages.PackageRef{Type: packages.PackageTypeTheme, Slug: cfg.DefaultThemeSlug}, stageDir)
				if err != nil {
					return fmt.Errorf("failed to resolve default theme %q: %w", cfg.DefaultThemeSlug, err)
				}
				defaultThemeArt = art
			} else {
				defaultThemeSkipped = true
			}
		}

		var pluginArts []packages.Artifact
		if deps.Resolver != nil && len(plugins) > 0 {
			for _, p := range plugins {
				fmt.Printf("→ Resolving plugin %s...\n", p)
				art, err := deps.Resolver.ResolvePackage(ctx, packages.PackageRef{Type: packages.PackageTypePlugin, Slug: p}, stageDir)
				if err != nil {
					return fmt.Errorf("failed to resolve plugin %q: %w", p, err)
				}
				pluginArts = append(pluginArts, *art)
			}
		}

		var themeArts []packages.Artifact
		if deps.Resolver != nil && len(themes) > 0 {
			for _, th := range themes {
				if cfg.DefaultThemeSlug != "" && th == cfg.DefaultThemeSlug {
					continue
				}
				fmt.Printf("→ Resolving theme %s...\n", th)
				art, err := deps.Resolver.ResolvePackage(ctx, packages.PackageRef{Type: packages.PackageTypeTheme, Slug: th}, stageDir)
				if err != nil {
					return fmt.Errorf("failed to resolve theme %q: %w", th, err)
				}
				themeArts = append(themeArts, *art)
			}
		}

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

		var currentStep string
		progress := func(step, detail string) {
			currentStep = detail
			fmt.Printf("→ %s\n", detail)
		}

		result, createErr := creator.Create(ctx, req, progress)

		if createErr != nil {
			_ = os.RemoveAll(stageDir)
			if create.IsCollisionError(createErr) {
				fmt.Printf("\nCollision: %v\nPlease choose a different website name or slug.\n\n", createErr)
				continue
			}
			return fmt.Errorf("creation failed at step %q: %w", currentStep, createErr)
		}
		_ = os.RemoveAll(stageDir)
		fmt.Println("\n=== Website Provisioned Successfully! ===")
		fmt.Printf("Path: %s\n", result.WebsitePath)
		fmt.Printf("URL:  %s\n", result.WebsiteURL)
		fmt.Printf("Active Theme: %s\n", result.ActiveTheme)
		if result.DefaultThemeSkipped {
			fmt.Printf("Notice: Default theme (%s) skipped — Package API disabled.\n", cfg.DefaultThemeSlug)
		}
		if len(result.PackageStatuses) > 0 {
			fmt.Println("\nPackage Installation Statuses:")
			for _, ps := range result.PackageStatuses {
				if ps.Status == "stale" {
					fmt.Printf("  - %s %s (v%s): stale fallback (%s)\n", ps.Type, ps.Slug, ps.Version, ps.Reason)
				} else {
					fmt.Printf("  - %s %s (v%s): %s\n", ps.Type, ps.Slug, ps.Version, ps.Status)
				}
			}
		}
		if len(result.InstalledPlugins) > 0 {
			fmt.Printf("Plugins: %s\n", strings.Join(result.InstalledPlugins, ", "))
		}
		if len(result.InstalledThemes) > 0 {
			fmt.Printf("Themes:  %s\n", strings.Join(result.InstalledThemes, ", "))
		}
		if len(result.FailedTweaks) > 0 {
			fmt.Println("\nTweak Warnings:")
			for _, twErr := range result.FailedTweaks {
				fmt.Printf("  - %s\n", twErr)
			}
		}
		if len(result.SkippedTweaks) > 0 {
			fmt.Println("\nSkipped Tweaks:")
			for _, sk := range result.SkippedTweaks {
				fmt.Printf("  - %s\n", sk)
			}
		}
		fmt.Println()
		return nil
	}
}

func RunDefaultCreateFlow(ctx context.Context, cfg *config.Config) error {
	var pkgResolver PackageResolver
	if strings.TrimSpace(cfg.PackagesAPIURL) != "" {
		pkgCache, err := packages.NewCacheWithContext(ctx)
		if err != nil {
			return fmt.Errorf("failed to initialize package cache: %w", err)
		}
		pkgResolver = packages.NewResolver(cfg, pkgCache)
	}
	return RunCreateFlowWithDeps(ctx, cfg, CreateFlowDependencies{
		Resolver: pkgResolver,
	})
}

// Run is the main application loop using context.Background.
func (a *App) Run() error {
	return a.RunWithContext(context.Background())
}

// RunWithContext runs the main application loop with the supplied context for cancellation and signal handling.
// DeleteFlowDependencies abstracts dependencies for the deletion flow.
type DeleteFlowDependencies struct {
	WPClient deprovision.WPClient
	Select   func(candidates []deprovision.Candidate) ([]deprovision.Candidate, error)
	Confirm  func(selected []deprovision.Candidate) (bool, error)
}

// RunDefaultDeleteFlow runs the standard de-provisioning workflow.
func RunDefaultDeleteFlow(ctx context.Context, cfg *config.Config) error {
	cli := wpcli.NewClient()
	return RunDeleteFlowWithDeps(ctx, cfg, DeleteFlowDependencies{
		WPClient: cli,
		Select:   tui.PromptDeleteSelection,
		Confirm:  tui.PromptDeleteConfirmMulti,
	})
}

// RunDeleteFlowWithDeps executes the de-provisioning workflow using injected dependencies.
func RunDeleteFlowWithDeps(ctx context.Context, cfg *config.Config, deps DeleteFlowDependencies) error {
	candidates, err := deprovision.DiscoverCandidates(ctx, cfg.WebsitesPath, cfg.DeleteExcludes, deps.WPClient)
	if err != nil {
		return fmt.Errorf("failed to discover websites: %w", err)
	}

	if len(candidates) == 0 {
		fmt.Printf("No websites found in %s\n", cfg.WebsitesPath)
		return nil
	}

	selectFn := deps.Select
	if selectFn == nil {
		selectFn = tui.PromptDeleteSelection
	}

	selected, err := selectFn(candidates)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		fmt.Println("No websites selected.")
		return nil
	}

	confirmFn := deps.Confirm
	if confirmFn == nil {
		confirmFn = tui.PromptDeleteConfirmMulti
	}

	confirmed, err := confirmFn(selected)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Deletion cancelled.")
		return nil
	}

	results := deprovision.Deprovision(ctx, selected, deps.WPClient, deprovision.DeprovisionOptions{
		UsedHerd:    cfg.UsedHerd,
		Concurrency: 4,
	})

	tui.PrintDeleteSummary(results)
	return nil
}
func (a *App) RunWithContext(ctx context.Context) error {
	cfg, err := a.InitConfig(true)
	if err != nil {
		return err
	}
	a.config = cfg

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		action, err := a.menuFn()
		if err != nil {
			return fmt.Errorf("menu error: %w", err)
		}

		switch action {
		case "exit":
			fmt.Println("Goodbye!")
			return nil
		case "create":
			if err := a.createFn(ctx, a.config); err != nil {
				fmt.Printf("Error creating website: %v\n", err)
			}
		case "delete":
			if err := a.deleteFn(ctx, a.config); err != nil {
				fmt.Printf("Error deleting website: %v\n", err)
			}
		default:
			fmt.Printf("Option %q is coming soon.\n", action)
		}
	}
}
