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
	"wptui/internal/packages"
	"wptui/internal/wpcli"
)

type Request struct {
	WebsiteName         string
	WebsiteSlug         string
	AdminUsername       string
	AdminPassword       string
	AdminEmail          string
	ApplyTweaks         bool
	DefaultTheme        *packages.Artifact // nil if skipped or disabled
	DefaultThemeSkipped bool
	PluginArtifacts     []packages.Artifact
	ThemeArtifacts      []packages.Artifact
}

type PackageStatusInfo struct {
	Type    packages.PackageType
	Slug    string
	Version string
	Status  string // "cached", "downloaded", "stale"
	Reason  string // set when Status is "stale"
}

type Result struct {
	WebsitePath         string
	WebsiteURL          string
	InstalledPlugins    []string
	InstalledThemes     []string
	ActiveTheme         string
	DefaultThemeSkipped bool
	PackageStatuses     []PackageStatusInfo
	FailedTweaks        []string
	SkippedTweaks       []string
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
	createdTLS bool
	siteDir    string
	siteSlug   string
	wpClient   *wpcli.Client
}

func (o *ownershipTracker) Rollback(ctx context.Context) {
	if o.createdTLS {
		_ = o.wpClient.HerdUnsecure(ctx, o.siteDir, o.siteSlug)
	}
	if o.createdDB {
		_ = o.wpClient.DBDrop(ctx, o.siteDir)
	}
	if o.createdDir && o.siteDir != "" {
		_ = os.RemoveAll(o.siteDir)
	}
}

func (c *Creator) Create(ctx context.Context, req Request, progress ProgressFunc) (*Result, error) {
	if progress == nil {
		progress = func(step, detail string) {}
	}

	if strings.TrimSpace(req.WebsiteName) == "" {
		return nil, errors.New("website name cannot be empty")
	}

	if err := ValidateSlug(req.WebsiteSlug); err != nil {
		return nil, fmt.Errorf("invalid website slug: %w", err)
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
	if err := c.wpClient.CheckDependencies(c.cfg.UsedHerd); err != nil {
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

	owner := &ownershipTracker{
		siteDir:  websitePath,
		siteSlug: req.WebsiteSlug,
		wpClient: c.wpClient,
	}

	success := false
	defer func() {
		if !success {
			owner.Rollback(context.Background())
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
	var siteURL string
	if c.cfg.UsedHerd {
		siteURL = "https://" + req.WebsiteSlug + ".test"
	} else {
		siteURL = "http://" + req.WebsiteSlug + ".test"
	}

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

	if req.DefaultTheme != nil {
		recordStatus(req.DefaultTheme)
	}
	for i := range req.PluginArtifacts {
		recordStatus(&req.PluginArtifacts[i])
	}
	for i := range req.ThemeArtifacts {
		recordStatus(&req.ThemeArtifacts[i])
	}

	if req.DefaultTheme != nil {
		progress("theme_install", fmt.Sprintf("Installing default theme %s...", req.DefaultTheme.Ref.Slug))
		if err := c.wpClient.ThemeInstall(ctx, websitePath, req.DefaultTheme.Path, true); err != nil {
			return nil, fmt.Errorf("failed to install default theme: %w", err)
		}
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
	if req.DefaultTheme != nil {
		installedThemes = append(installedThemes, req.DefaultTheme.Ref.Slug)
	}
	for _, art := range req.ThemeArtifacts {
		// Deduplicate: avoid re-installing if already installed as default theme
		if req.DefaultTheme != nil && art.Ref.Slug == req.DefaultTheme.Ref.Slug {
			continue
		}
		progress("theme_install", fmt.Sprintf("Installing additional theme %s...", art.Ref.Slug))
		if err := c.wpClient.ThemeInstall(ctx, websitePath, art.Path, false); err != nil {
			return nil, fmt.Errorf("failed to install additional theme %s: %w", art.Ref.Slug, err)
		}
		installedThemes = append(installedThemes, art.Ref.Slug)
	}

	// Always query WordPress for the active theme after installation
	activeTheme, err := c.wpClient.ThemeGetActive(ctx, websitePath)
	if err != nil {
		return nil, fmt.Errorf("failed to query active WordPress theme: %w", err)
	}

	// Step 9: Secure with Herd if used_herd is true (final provisioning step)
	if c.cfg.UsedHerd {
		alreadySecured, err := c.wpClient.IsSiteSecured(ctx, req.WebsiteSlug)
		if err != nil {
			return nil, fmt.Errorf("failed to verify herd TLS status: %w", err)
		}
		progress("herd_secure", "Securing site with Herd TLS...")
		if err := c.wpClient.HerdSecure(ctx, websitePath, req.WebsiteSlug); err != nil {
			return nil, err
		}
		if !alreadySecured {
			owner.createdTLS = true
		}
	}

	success = true

	return &Result{
		WebsitePath:         websitePath,
		WebsiteURL:          siteURL,
		InstalledPlugins:    installedPlugins,
		InstalledThemes:     installedThemes,
		ActiveTheme:         activeTheme,
		DefaultThemeSkipped: req.DefaultThemeSkipped,
		PackageStatuses:     packageStatuses,
		FailedTweaks:        failedTweaks,
		SkippedTweaks:       skippedTweaks,
	}, nil
}
