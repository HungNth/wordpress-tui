package restore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"wptui/internal/backup"
	"wptui/internal/config"
	"wptui/internal/core"
	"wptui/internal/packages"
	"wptui/internal/siteconfig"
	"wptui/internal/wpcli"
)

type Strategy string

const (
	StrategyFullZIP Strategy = "full_zip"
	StrategyAI1WM   Strategy = "ai1wm"
)

type DBConnection = wpcli.DBConnection
type ConfigCreateOptions = wpcli.ConfigCreateOptions

type ProgressFunc func(step, description string)

type WPClient interface {
	siteconfig.WPClient
	DBCreate(ctx context.Context, dir string) (bool, error)
	DBDrop(ctx context.Context, dir string) error
	DBImport(ctx context.Context, dir, sqlFile string) error
	SearchReplace(ctx context.Context, dir, search, replace string) error
	OptionGet(ctx context.Context, dir, key string) (string, error)
	AI1WMRestore(ctx context.Context, dir, archiveFileName string) error
	CoreInstall(ctx context.Context, dir, url, title, adminUser, adminPass, adminEmail string) error
	HerdSecure(ctx context.Context, dir, slug string) error
	HerdUnsecure(ctx context.Context, dir, slug string) error
	IsSiteSecured(ctx context.Context, slug string) (bool, error)
	ConfigCreateWithOptions(ctx context.Context, dir, dbName string, conn DBConnection, opts ConfigCreateOptions) error
}

type AdminCredentialsUpdater interface {
	UpdateAdminCredentials(ctx context.Context, siteDir, newUsername, newPassword, newEmail string) error
}

type defaultAdminUpdater struct {
	client WPClient
}

func (u *defaultAdminUpdater) UpdateAdminCredentials(ctx context.Context, siteDir, newUsername, newPassword, newEmail string) error {
	if u.client == nil {
		return errors.New("cannot update admin credentials: nil WPClient")
	}

	admins, err := siteconfig.DiscoverAdministrators(ctx, siteDir, u.client)
	if err != nil {
		return fmt.Errorf("failed to discover site administrators: %w", err)
	}
	if len(admins) == 0 {
		return errors.New("no administrator user found to update")
	}
	targetAdmin := admins[0]

	dbCfg, err := siteconfig.ExtractDBConfig(ctx, siteDir, u.client)
	if err != nil {
		return fmt.Errorf("failed to extract database config: %w", err)
	}

	input := siteconfig.AdminInput{
		UserID:      targetAdmin.ID,
		NewUsername: newUsername,
		NewPassword: newPassword,
		NewEmail:    newEmail,
	}

	return siteconfig.UpdateAdminCredentials(ctx, siteDir, dbCfg, input, u.client, nil, nil)
}

type ExtensionInstaller interface {
	EnsureExtension(ctx context.Context, siteDir string) error
}

type defaultExtensionInstaller struct {
	cfg      *config.Config
	wpClient WPClient
}

func (e *defaultExtensionInstaller) EnsureExtension(ctx context.Context, siteDir string) error {
	if strings.TrimSpace(e.cfg.PackagesAPIURL) == "" {
		return errors.New("cannot install AI1WM extension: packages_api_url is not configured in config.json")
	}
	pkgCache, err := packages.NewCacheWithContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize package cache: %w", err)
	}
	resolver := packages.NewResolver(e.cfg, pkgCache)
	return backup.EnsureAI1WMExtension(ctx, siteDir, resolver, e.wpClient, nil)
}

type CoreExtractor interface {
	ExtractCore(ctx context.Context, destDir string) error
}

type defaultCoreExtractor struct{}

func (c *defaultCoreExtractor) ExtractCore(ctx context.Context, destDir string) error {
	coreCache, err := core.NewCache()
	if err != nil {
		return fmt.Errorf("failed to initialize core cache: %w", err)
	}
	coreResolver := core.NewResolver(coreCache)
	archivePath, _, err := coreResolver.Resolve(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve core archive: %w", err)
	}
	return core.ExtractCoreArchive(archivePath, destDir)
}

type Request struct {
	Strategy    Strategy
	ArchivePath string
	WebsiteName string
	WebsiteSlug string
	AdminUser   string
	AdminPass   string
	AdminEmail  string
}

type Result struct {
	WebsiteName string
	WebsiteSlug string
	SitePath    string
	SiteURL     string
	Database    string
	AdminUser   string
	TLSError    string
}

type Restorer struct {
	cfg          *config.Config
	wpClient     WPClient
	adminUpdater AdminCredentialsUpdater
	extInstaller ExtensionInstaller
	coreExtract  CoreExtractor
	promptDumpFn DumpPromptFunc
}

func NewRestorer(cfg *config.Config, client WPClient) *Restorer {
	return &Restorer{
		cfg:          cfg,
		wpClient:     client,
		adminUpdater: &defaultAdminUpdater{client: client},
		extInstaller: &defaultExtensionInstaller{cfg: cfg, wpClient: client},
		coreExtract:  &defaultCoreExtractor{},
	}
}

func (r *Restorer) SetAdminUpdater(updater AdminCredentialsUpdater) {
	r.adminUpdater = updater
}

func (r *Restorer) SetExtensionInstaller(installer ExtensionInstaller) {
	r.extInstaller = installer
}

func (r *Restorer) SetCoreExtractor(extractor CoreExtractor) {
	r.coreExtract = extractor
}

func (r *Restorer) SetPromptDumpFunc(fn DumpPromptFunc) {
	r.promptDumpFn = fn
}

type restoreTracker struct {
	siteDir    string
	siteSlug   string
	createdDir bool
	createdDB  bool
	createdTLS bool
	wpClient   WPClient
}

func (t *restoreTracker) Rollback(ctx context.Context) {
	if t.createdTLS && t.wpClient != nil && t.siteSlug != "" && t.siteDir != "" {
		_ = t.wpClient.HerdUnsecure(ctx, t.siteDir, t.siteSlug)
	}
	if t.createdDB && t.wpClient != nil && t.siteDir != "" {
		_ = t.wpClient.DBDrop(ctx, t.siteDir)
	}
	if t.createdDir && t.siteDir != "" {
		_ = os.RemoveAll(t.siteDir)
	}
}

func (r *Restorer) Restore(ctx context.Context, req Request, onProgress ProgressFunc) (*Result, error) {
	if onProgress == nil {
		onProgress = func(step, description string) {}
	}

	switch req.Strategy {
	case StrategyFullZIP:
		return r.restoreFullZIP(ctx, req, onProgress)
	case StrategyAI1WM:
		return r.restoreAI1WM(ctx, req, onProgress)
	default:
		return nil, fmt.Errorf("unsupported restore strategy: %s", req.Strategy)
	}
}

func (r *Restorer) restoreFullZIP(ctx context.Context, req Request, progress ProgressFunc) (*Result, error) {
	targetDir := filepath.Join(r.cfg.WebsitesPath, req.WebsiteSlug)
	if _, err := os.Stat(targetDir); err == nil {
		return nil, fmt.Errorf("directory already exists: %s", targetDir)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("cannot inspect target directory %s: %w", targetDir, err)
	}

	// 1. Prepare isolated staging directory
	stagingDir, err := os.MkdirTemp("", "wptui-restore-stage-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create staging directory: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	// 2. Extract ZIP into staging
	progress("extract_staging", "Extracting backup archive...")
	if err := ExtractZipArchive(req.ArchivePath, stagingDir); err != nil {
		return nil, fmt.Errorf("archive extraction failed: %w", err)
	}

	// 3. Find unique WordPress Root
	progress("find_root", "Locating WordPress root in archive...")
	wpRoot, err := FindWordPressRoot(stagingDir)
	if err != nil {
		return nil, err
	}

	// 4. Resolve SQL Dump
	progress("resolve_dump", "Identifying database dump...")
	sqlDumpPath, err := ResolveSQLDump(req.ArchivePath, wpRoot, r.promptDumpFn)
	if err != nil {
		return nil, err
	}
	sqlDumpName := filepath.Base(sqlDumpPath)

	// 5. Extract table prefix
	progress("extract_prefix", "Determining table prefix...")
	tablePrefix, err := ExtractTablePrefix(wpRoot, sqlDumpPath)
	if err != nil {
		return nil, err
	}

	tracker := &restoreTracker{
		siteDir:  targetDir,
		siteSlug: req.WebsiteSlug,
		wpClient: r.wpClient,
	}
	success := false
	defer func() {
		if !success {
			tracker.Rollback(context.Background())
		}
	}()

	// 6. Relocate WordPress root to target directory
	progress("relocate_files", "Relocating website files...")
	if err := os.MkdirAll(r.cfg.WebsitesPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to prepare websites directory: %w", err)
	}
	if err := os.Mkdir(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}
	tracker.createdDir = true

	if err := copyDirectory(wpRoot, targetDir); err != nil {
		return nil, fmt.Errorf("failed to relocate files to destination: %w", err)
	}
	// 7. Generate fresh wp-config.php with discovered prefix
	progress("generate_config", "Configuring wp-config.php...")
	dbConn := DBConnection{
		Host:   r.cfg.DatabaseHost,
		Port:   r.cfg.DatabasePort,
		User:   r.cfg.DBUsername,
		Pass:   r.cfg.DBPassword,
		Socket: r.cfg.DBSocket,
	}

	opts := ConfigCreateOptions{
		SkipCheck: true,
		Force:     true,
		DBPrefix:  tablePrefix,
	}
	if err := r.wpClient.ConfigCreateWithOptions(ctx, targetDir, req.WebsiteSlug, dbConn, opts); err != nil {
		return nil, fmt.Errorf("failed to create wp-config.php: %w", err)
	}

	// 8. Create database and import dump
	progress("create_db", "Creating database...")
	created, err := r.wpClient.DBCreate(ctx, targetDir)
	if created {
		tracker.createdDB = true
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	progress("import_db", "Importing database dump...")
	targetSQLPath := filepath.Join(targetDir, sqlDumpName)
	// Pass relative or forward-slash normalized path to avoid Windows backslash escape issues in MySQL CLI
	if err := r.wpClient.DBImport(ctx, targetDir, sqlDumpName); err != nil {
		return nil, fmt.Errorf("failed to import database: %w", err)
	}

	// 9. Discover prior siteurl and execute search-replace
	progress("search_replace", "Updating site URLs...")
	oldURL, err := r.wpClient.OptionGet(ctx, targetDir, "siteurl")
	if err != nil {
		return nil, fmt.Errorf("failed to discover prior siteurl: %w", err)
	}
	oldURL = strings.TrimSpace(oldURL)
	if oldURL == "" {
		return nil, errors.New("prior siteurl is empty in imported database")
	}
	var newURL string
	if r.cfg.UsedHerd {
		newURL = "https://" + req.WebsiteSlug + ".test"
	} else {
		newURL = "http://" + req.WebsiteSlug + ".test"
	}

	if oldURL != "" && oldURL != newURL {
		if err := r.wpClient.SearchReplace(ctx, targetDir, oldURL, newURL); err != nil {
			return nil, fmt.Errorf("failed to replace site URL: %w", err)
		}
	}

	// 10. Update Admin credentials if requested
	adminUser := req.AdminUser
	if adminUser == "" {
		adminUser = r.cfg.DefaultAdminUsername
	}
	adminPass := req.AdminPass
	if adminPass == "" {
		adminPass = r.cfg.DefaultAdminPassword
	}
	adminEmail := req.AdminEmail
	if adminEmail == "" {
		adminEmail = r.cfg.DefaultAdminEmail
	}

	progress("update_admin", "Synchronizing administrator credentials...")
	if err := r.adminUpdater.UpdateAdminCredentials(ctx, targetDir, adminUser, adminPass, adminEmail); err != nil {
		return nil, fmt.Errorf("failed to update admin credentials: %w", err)
	}
	// 11. Configure Herd TLS
	var tlsWarning string
	if r.cfg.UsedHerd {
		alreadySecured, _ := r.wpClient.IsSiteSecured(ctx, req.WebsiteSlug)
		progress("secure_herd", "Securing domain with Herd TLS...")
		if err := r.wpClient.HerdSecure(ctx, targetDir, req.WebsiteSlug); err != nil {
			tlsWarning = fmt.Sprintf("Herd TLS warning: %v", err)
			httpURL := "http://" + req.WebsiteSlug + ".test"
			if srErr := r.wpClient.SearchReplace(ctx, targetDir, newURL, httpURL); srErr != nil {
				tlsWarning = fmt.Sprintf("Herd TLS warning: %v (failed to revert URL to HTTP: %v)", err, srErr)
			} else {
				newURL = httpURL
			}
		} else if !alreadySecured {
			tracker.createdTLS = true
		}
	}
	// 12. Strictly upon successful completion of entire lifecycle: remove selected SQL dump
	if err := os.Remove(targetSQLPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to clean up imported SQL dump %s: %w", targetSQLPath, err)
	}

	success = true
	return &Result{
		WebsiteName: req.WebsiteName,
		WebsiteSlug: req.WebsiteSlug,
		SitePath:    targetDir,
		SiteURL:     newURL,
		Database:    req.WebsiteSlug,
		AdminUser:   adminUser,
		TLSError:    tlsWarning,
	}, nil
}

func (r *Restorer) restoreAI1WM(ctx context.Context, req Request, progress ProgressFunc) (*Result, error) {
	targetDir := filepath.Join(r.cfg.WebsitesPath, req.WebsiteSlug)
	if _, err := os.Stat(targetDir); err == nil {
		return nil, fmt.Errorf("directory already exists: %s", targetDir)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("cannot inspect target directory %s: %w", targetDir, err)
	}

	tracker := &restoreTracker{
		siteDir:  targetDir,
		siteSlug: req.WebsiteSlug,
		wpClient: r.wpClient,
	}
	success := false
	defer func() {
		if !success {
			tracker.Rollback(context.Background())
		}
	}()

	// 1. Prepare websites directory and target site directory
	progress("create_directory", "Preparing website directory...")
	if err := os.MkdirAll(r.cfg.WebsitesPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to prepare websites directory: %w", err)
	}
	if err := os.Mkdir(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create website directory: %w", err)
	}
	tracker.createdDir = true
	// 2. Extract minimal WordPress core foundation
	progress("extract_core", "Extracting WordPress core foundation...")
	if err := r.coreExtract.ExtractCore(ctx, targetDir); err != nil {
		return nil, fmt.Errorf("failed to extract core foundation: %w", err)
	}

	// 3. Configure wp-config.php
	progress("create_config", "Configuring foundation wp-config.php...")
	dbConn := DBConnection{
		Host:   r.cfg.DatabaseHost,
		Port:   r.cfg.DatabasePort,
		User:   r.cfg.DBUsername,
		Pass:   r.cfg.DBPassword,
		Socket: r.cfg.DBSocket,
	}
	opts := ConfigCreateOptions{
		SkipCheck: true,
		Force:     true,
	}
	if err := r.wpClient.ConfigCreateWithOptions(ctx, targetDir, req.WebsiteSlug, dbConn, opts); err != nil {
		return nil, fmt.Errorf("failed to create foundation wp-config.php: %w", err)
	}

	// 4. Create database
	progress("create_db", "Creating database...")
	created, err := r.wpClient.DBCreate(ctx, targetDir)
	if created {
		tracker.createdDB = true
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// 5. Core install minimal foundation
	progress("core_install", "Installing foundation WordPress...")
	var targetURL string
	if r.cfg.UsedHerd {
		targetURL = "https://" + req.WebsiteSlug + ".test"
	} else {
		targetURL = "http://" + req.WebsiteSlug + ".test"
	}
	if err := r.wpClient.CoreInstall(ctx, targetDir, targetURL, "Temporary", "temp", "temp", "temp@example.com"); err != nil {
		return nil, fmt.Errorf("failed to install foundation core: %w", err)
	}

	// 6. Ensure AI1WM Unlimited extension is installed and active
	progress("install_extension", "Ensuring All-in-One WP Migration Unlimited extension is active...")
	if err := r.extInstaller.EnsureExtension(ctx, targetDir); err != nil {
		return nil, fmt.Errorf("failed to install AI1WM extension: %w", err)
	}

	// 7. Stage .wpress file into wp-content/ai1wm-backups/
	progress("stage_archive", "Staging migration archive...")
	ai1wmBackupsDir := filepath.Join(targetDir, "wp-content", "ai1wm-backups")
	if err := os.MkdirAll(ai1wmBackupsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create ai1wm-backups directory: %w", err)
	}

	archiveFileName := filepath.Base(req.ArchivePath)
	stagedArchive := filepath.Join(ai1wmBackupsDir, archiveFileName)

	info, err := os.Stat(req.ArchivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source archive: %w", err)
	}

	if err := copyFile(req.ArchivePath, stagedArchive, info.Mode()); err != nil {
		return nil, fmt.Errorf("failed to stage migration archive: %w", err)
	}

	// 8. Execute AI1WM Restore command
	progress("ai1wm_restore", "Restoring website via All-in-One WP Migration...")
	if err := r.wpClient.AI1WMRestore(ctx, targetDir, archiveFileName); err != nil {
		return nil, fmt.Errorf("AI1WM restore command failed: %w", err)
	}

	// 9. Remove staged .wpress copy from wp-content/ai1wm-backups/
	if err := os.Remove(stagedArchive); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to clean up staged migration archive %s: %w", stagedArchive, err)
	}

	// 10. Discover prior siteurl and execute search-replace to reconcile target URL
	progress("search_replace", "Updating site URLs...")
	oldURL, err := r.wpClient.OptionGet(ctx, targetDir, "siteurl")
	if err != nil {
		return nil, fmt.Errorf("failed to discover prior siteurl: %w", err)
	}
	oldURL = strings.TrimSpace(oldURL)
	if oldURL == "" {
		return nil, errors.New("prior siteurl is empty in imported database")
	}
	newURL := targetURL
	if oldURL != "" && oldURL != newURL {
		if err := r.wpClient.SearchReplace(ctx, targetDir, oldURL, newURL); err != nil {
			return nil, fmt.Errorf("failed to replace site URL: %w", err)
		}
	}

	// 11. Update Admin credentials strictly AFTER AI1WM restore completes
	adminUser := req.AdminUser
	if adminUser == "" {
		adminUser = r.cfg.DefaultAdminUsername
	}
	adminPass := req.AdminPass
	if adminPass == "" {
		adminPass = r.cfg.DefaultAdminPassword
	}
	adminEmail := req.AdminEmail
	if adminEmail == "" {
		adminEmail = r.cfg.DefaultAdminEmail
	}

	progress("update_admin", "Synchronizing administrator credentials...")
	if err := r.adminUpdater.UpdateAdminCredentials(ctx, targetDir, adminUser, adminPass, adminEmail); err != nil {
		return nil, fmt.Errorf("failed to update admin credentials: %w", err)
	}

	// 12. Configure Herd TLS
	var tlsWarning string
	if r.cfg.UsedHerd {
		alreadySecured, _ := r.wpClient.IsSiteSecured(ctx, req.WebsiteSlug)
		progress("secure_herd", "Securing domain with Herd TLS...")
		if err := r.wpClient.HerdSecure(ctx, targetDir, req.WebsiteSlug); err != nil {
			tlsWarning = fmt.Sprintf("Herd TLS warning: %v", err)
			httpURL := "http://" + req.WebsiteSlug + ".test"
			if srErr := r.wpClient.SearchReplace(ctx, targetDir, newURL, httpURL); srErr != nil {
				tlsWarning = fmt.Sprintf("Herd TLS warning: %v (failed to revert URL to HTTP: %v)", err, srErr)
			} else {
				newURL = httpURL
			}
		} else if !alreadySecured {
			tracker.createdTLS = true
		}
	}
	success = true
	return &Result{
		WebsiteName: req.WebsiteName,
		WebsiteSlug: req.WebsiteSlug,
		SitePath:    targetDir,
		SiteURL:     newURL,
		Database:    req.WebsiteSlug,
		AdminUser:   adminUser,
		TLSError:    tlsWarning,
	}, nil
}

func copyDirectory(src, dst string) error {
	cleanSrc := filepath.Clean(src)
	cleanDst := filepath.Clean(dst)

	return filepath.Walk(cleanSrc, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(cleanSrc, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(cleanDst, info.Mode())
		}

		target := filepath.Join(cleanDst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// Ensure target file is writable (at least 0644) so WP-CLI or subsequent tools can rewrite/overwrite it
	targetMode := mode | 0644
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, targetMode)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
