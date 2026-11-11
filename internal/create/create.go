package create

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"wptui/internal/config"
	"wptui/internal/core"
	"wptui/internal/localenv"
	"wptui/internal/packages"
	"wptui/internal/wpcli"
)

type Request struct {
	WebsiteName     string
	WebsiteSlug     string
	AdminUsername   string
	AdminPassword   string
	AdminEmail      string
	ApplyTweaks     bool
	ActiveThemeSlug string
	PluginArtifacts []packages.Artifact
	ThemeArtifacts  []packages.Artifact
}

type PackageStatusInfo struct {
	Type    packages.PackageType
	Slug    string
	Version string
	Status  string // "cached", "downloaded", "stale"
	Reason  string // set when Status is "stale"
}

type Result struct {
	WebsitePath      string
	WebsiteURL       string
	InstalledPlugins []string
	InstalledThemes  []string
	ActiveTheme      string
	PackageStatuses  []PackageStatusInfo
	FailedTweaks     []string
	SkippedTweaks    []string
}

type ProgressFunc func(step, detail string)

type CollisionError struct {
	Resource string
	Path     string
}

func (e *CollisionError) Error() string {
	return fmt.Sprintf("%s collision detected: %s already exists", e.Resource, e.Path)
}

func IsCollisionError(err error) bool {
	var collErr *CollisionError
	return errors.As(err, &collErr)
}

type DBExistsFunc func(ctx context.Context, dbName string) (bool, error)
type CoreResolver interface {
	Resolve(ctx context.Context) (archivePath string, version string, err error)
}
type CoreExtractor func(archivePath, destDir string) error

type CreatorOption func(*Creator)

func WithCoreResolver(r CoreResolver) CreatorOption {
	return func(c *Creator) {
		c.coreResolver = r
	}
}

func WithCoreExtractor(e CoreExtractor) CreatorOption {
	return func(c *Creator) {
		c.coreExtractor = e
	}
}

type Creator struct {
	cfg           *config.Config
	wpClient      *wpcli.Client
	checkDBExist  DBExistsFunc
	coreResolver  CoreResolver
	coreExtractor CoreExtractor
}

func NewCreator(cfg *config.Config, client *wpcli.Client, checkDB DBExistsFunc, opts ...CreatorOption) *Creator {
	if client == nil {
		client = wpcli.NewClient()
	}
	var resolver CoreResolver
	if cache, err := core.NewCache(); err == nil && cache != nil {
		resolver = core.NewResolver(cache)
	}
	c := &Creator{
		cfg:           cfg,
		wpClient:      client,
		checkDBExist:  checkDB,
		coreResolver:  resolver,
		coreExtractor: core.ExtractCoreArchive,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type ownershipTracker struct {
	createdDir bool
	createdDB  bool
	createdReg bool
	createdTLS bool
	siteDir    string
	siteSlug   string
	wpClient   *wpcli.Client
	isLaragon  bool
}

func (o *ownershipTracker) Rollback(ctx context.Context) error {
	var errs []error
	if o.createdTLS {
		if err := o.wpClient.EnvironmentUnsecure(ctx, o.siteDir, o.siteSlug); err != nil {
			errs = append(errs, fmt.Errorf("environment unsecure cleanup failed: %w", err))
		}
	}
	if o.createdReg && !o.isLaragon {
		if err := o.wpClient.EnvironmentUnregister(ctx, o.siteDir, o.siteSlug); err != nil {
			errs = append(errs, fmt.Errorf("environment unregister cleanup failed: %w", err))
		}
	}
	if o.createdDB {
		if err := o.wpClient.DBDrop(ctx, o.siteDir); err != nil {
			errs = append(errs, fmt.Errorf("database drop cleanup failed: %w", err))
		}
	}
	if o.createdDir && o.siteDir != "" {
		if err := os.RemoveAll(o.siteDir); err != nil {
			errs = append(errs, fmt.Errorf("directory removal cleanup failed: %w", err))
		}
	}
	if o.isLaragon && o.createdDir {
		if err := o.wpClient.EnvironmentUnregister(ctx, o.siteDir, o.siteSlug); err != nil {
			errs = append(errs, fmt.Errorf("laragon reload cleanup failed: %w", err))
		}
	}
	return errors.Join(errs...)
}

func (c *Creator) Create(ctx context.Context, req Request, progress ProgressFunc) (result *Result, err error) {
	if progress == nil {
		progress = func(step, detail string) {}
	}

	if strings.TrimSpace(req.WebsiteName) == "" {
		return nil, errors.New("website name cannot be empty")
	}

	if err := ValidateSlug(req.WebsiteSlug); err != nil {
		return nil, fmt.Errorf("invalid website slug: %w", err)
	}

	if req.ActiveThemeSlug != "" {
		found := false
		for _, art := range req.ThemeArtifacts {
			if art.Ref.Slug == req.ActiveThemeSlug {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("active theme %q is not present in selected theme artifacts", req.ActiveThemeSlug)
		}
	}

	adminUser := req.AdminUsername
	if strings.TrimSpace(adminUser) == "" {
		adminUser = c.cfg.DefaultAdminUsername
	}
	adminPass := req.AdminPassword
	if strings.TrimSpace(adminPass) == "" {
		adminPass = c.cfg.DefaultAdminPassword
	}
	adminEmail := req.AdminEmail
	if strings.TrimSpace(adminEmail) == "" {
		adminEmail = c.cfg.DefaultAdminEmail
	}

	websitePath := filepath.Join(c.cfg.WebsitesPath, req.WebsiteSlug)

	// Preflight 1: Dependencies in PATH
	if err := c.wpClient.CheckDependencies(); err != nil {
		return nil, fmt.Errorf("dependency preflight failed: %w", err)
	}
	// Preflight 2: Directory collision
	if fi, err := os.Stat(websitePath); err == nil && fi != nil {
		return nil, &CollisionError{Resource: "Directory", Path: websitePath}
	}

	// Preflight 3: Database collision
	if c.checkDBExist != nil {
		exists, err := c.checkDBExist(ctx, req.WebsiteSlug)
		if err != nil {
			return nil, fmt.Errorf("database preflight check failed: %w", err)
		}
		if exists {
			return nil, &CollisionError{Resource: "Database", Path: req.WebsiteSlug}
		}
	}

	isLaragon := c.cfg != nil && strings.EqualFold(c.cfg.AppName, config.AppNameLaragon)
	owner := &ownershipTracker{
		siteDir:   websitePath,
		siteSlug:  req.WebsiteSlug,
		wpClient:  c.wpClient,
		isLaragon: isLaragon,
	}

	success := false
	defer func() {
		if !success {
			if rbErr := owner.Rollback(context.Background()); rbErr != nil {
				err = errors.Join(err, rbErr)
			}
		}
	}()

	// Ensure websites root path exists
	if err := os.MkdirAll(c.cfg.WebsitesPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to prepare websites directory: %w", err)
	}

	// Step 1: Create Website leaf directory atomically
	progress("create_directory", "Creating website directory...")
	if err := os.Mkdir(websitePath, 0755); err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, &CollisionError{Resource: "Directory", Path: websitePath}
		}
		return nil, fmt.Errorf("failed to create website directory: %w", err)
	}
	owner.createdDir = true

	// Register environment (e.g. link for Flock, reload for Laragon)
	regDetail := "Registering environment..."
	if isLaragon {
		regDetail = "Reloading Laragon..."
	}
	progress("environment_register", regDetail)
	registered, err := c.wpClient.EnvironmentRegister(ctx, websitePath, req.WebsiteSlug)
	if registered {
		owner.createdReg = true
	}
	if err != nil {
		return nil, fmt.Errorf("environment registration failed: %w", err)
	}

	// Step 2: Resolve and extract WordPress core
	if c.coreResolver == nil {
		return nil, errors.New("core resolver is not configured")
	}
	if c.coreExtractor == nil {
		return nil, errors.New("core extractor is not configured")
	}

	progress("core_resolve", "Checking WordPress core...")
	archivePath, _, err := c.coreResolver.Resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve WordPress core: %w", err)
	}

	progress("core_extract", "Extracting WordPress core...")
	if err := c.coreExtractor(archivePath, websitePath); err != nil {
		return nil, fmt.Errorf("failed to extract WordPress core: %w", err)
	}

	// Step 3: Generate wp-config.php with skip-check
	progress("config_create", "Generating wp-config.php...")
	dbConn := wpcli.DBConnection{
		Host:   c.cfg.DatabaseHost,
		Port:   c.cfg.DatabasePort,
		User:   c.cfg.DBUsername,
		Pass:   c.cfg.DBPassword,
		Socket: c.cfg.DBSocket,
	}
	if err := c.wpClient.ConfigCreateWithOptions(
		ctx,
		websitePath,
		req.WebsiteSlug,
		dbConn,
		wpcli.ConfigCreateOptions{SkipCheck: true},
	); err != nil {
		return nil, err
	}

	// Step 4: Create database
	progress("db_create", "Creating database...")
	created, err := c.wpClient.DBCreate(ctx, websitePath)
	if created {
		owner.createdDB = true
	}
	if err != nil {
		return nil, err
	}
	// Step 5: Core install
	siteURL := localenv.SiteURL(c.cfg, req.WebsiteSlug)

	progress("core_install", "Running WordPress installation...")
	if err := c.wpClient.CoreInstall(
		ctx,
		websitePath,
		siteURL,
		req.WebsiteName,
		adminUser,
		adminPass,
		adminEmail,
	); err != nil {
		return nil, err
	}

	// Step 7: Apply tweaks if opted in
	var failedTweaks []string
	var skippedTweaks []string
	if req.ApplyTweaks && len(c.cfg.WPTweaks) > 0 {
		progress("tweaks", "Applying WordPress tweaks...")
		tweakExecutor := func(tctx context.Context, idx int, tw config.WPTweak) error {
			switch tw.Type {
			case config.TweakTypeConfigSet:
				return c.wpClient.ConfigSet(tctx, websitePath, tw.Key, tw.Value, tw.Raw)
			case config.TweakTypeRewriteStructure:
				return c.wpClient.RewriteStructure(tctx, websitePath, tw.Value)
			case config.TweakTypeOptionUpdate:
				return c.wpClient.OptionUpdate(tctx, websitePath, tw.Key, tw.Value)
			case config.TweakTypeLanguageCore:
				return c.wpClient.LanguageCore(tctx, websitePath, tw.Key, tw.Value)
			default:
				return fmt.Errorf("unsupported tweak type %s", tw.Type)
			}
		}

		tweakResults, err := RunTweaks(ctx, c.cfg.WPTweaks, tweakExecutor, progress)
		if err != nil {
			return nil, err
		}

		for _, tr := range tweakResults {
			if tr.Status == TweakStatusFailed {
				failedTweaks = append(failedTweaks, fmt.Sprintf("%s %s: %s", tr.Tweak.Type, tr.Tweak.Key, tr.Error))
			} else if tr.Status == TweakStatusSkipped {
				skippedTweaks = append(skippedTweaks, fmt.Sprintf("%s %s: %s", tr.Tweak.Type, tr.Tweak.Key, tr.Error))
			}
		}
	}

	// Step 8: Install verified local package artifacts
	var packageStatuses []PackageStatusInfo
	recordStatus := func(art *packages.Artifact) {
		if art == nil {
			return
		}
		st := "downloaded"
		if art.IsStale {
			st = "stale"
		} else if art.IsCached {
			st = "cached"
		}
		packageStatuses = append(packageStatuses, PackageStatusInfo{
			Type:    art.Ref.Type,
			Slug:    art.Ref.Slug,
			Version: art.Version,
			Status:  st,
			Reason:  art.StaleReason,
		})
	}

	for i := range req.PluginArtifacts {
		recordStatus(&req.PluginArtifacts[i])
	}
	for i := range req.ThemeArtifacts {
		recordStatus(&req.ThemeArtifacts[i])
	}

	var installedPlugins []string
	for _, art := range req.PluginArtifacts {
		progress("plugin_install", fmt.Sprintf("Installing plugin %s...", art.Ref.Slug))
		if err := c.wpClient.PluginInstall(ctx, websitePath, art.Path, true); err != nil {
			return nil, fmt.Errorf("failed to install plugin %s: %w", art.Ref.Slug, err)
		}
		installedPlugins = append(installedPlugins, art.Ref.Slug)
	}

	var installedThemes []string
	for _, art := range req.ThemeArtifacts {
		activate := (req.ActiveThemeSlug != "" && art.Ref.Slug == req.ActiveThemeSlug)
		progress("theme_install", fmt.Sprintf("Installing theme %s...", art.Ref.Slug))
		if err := c.wpClient.ThemeInstall(ctx, websitePath, art.Path, activate); err != nil {
			return nil, fmt.Errorf("failed to install theme %s: %w", art.Ref.Slug, err)
		}
		installedThemes = append(installedThemes, art.Ref.Slug)
	}

	var activeTheme string
	if req.ActiveThemeSlug != "" {
		act, err := c.wpClient.ThemeGetActive(ctx, websitePath)
		if err != nil {
			return nil, fmt.Errorf("failed to query active WordPress theme: %w", err)
		}
		activeTheme = act
	}

	// Step 9: Secure environment TLS if configured and not Laragon
	if c.cfg != nil && !isLaragon && c.cfg.AppName != "" {
		alreadySecured, err := c.wpClient.IsSiteSecured(ctx, req.WebsiteSlug)
		if err != nil {
			return nil, fmt.Errorf("failed to verify environment TLS status: %w", err)
		}
		if !alreadySecured {
			owner.createdTLS = true
			progress("environment_secure", "Securing site environment...")
			if err := c.wpClient.EnvironmentSecure(ctx, websitePath, req.WebsiteSlug); err != nil {
				return nil, err
			}
		}
	}

	success = true

	return &Result{
		WebsitePath:      websitePath,
		WebsiteURL:       siteURL,
		InstalledPlugins: installedPlugins,
		InstalledThemes:  installedThemes,
		ActiveTheme:      activeTheme,
		PackageStatuses:  packageStatuses,
		FailedTweaks:     failedTweaks,
		SkippedTweaks:    skippedTweaks,
	}, nil
}
