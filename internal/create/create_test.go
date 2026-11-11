package create_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"wptui/internal/config"
	"wptui/internal/create"
	"wptui/internal/packages"
	"wptui/internal/wpcli"
)

type mockFlockSite struct {
	slug    string
	path    string
	secured bool
}

type mockRunner struct {
	mu               sync.Mutex
	calls            []string
	failOnSubstr     string
	failOnSubstrings []string
	securedSites     string
	activeTheme      string
	flockSites       map[string]*mockFlockSite
}

func (m *mockRunner) Run(ctx context.Context, dir string, name string, args []string, stdin string) (string, string, error) {
	full := name + " " + strings.Join(args, " ")
	m.mu.Lock()
	m.calls = append(m.calls, full)
	if m.flockSites == nil {
		m.flockSites = make(map[string]*mockFlockSite)
	}
	m.mu.Unlock()

	if m.failOnSubstr != "" && strings.Contains(full, m.failOnSubstr) {
		return "", "mock error for " + full, errors.New("simulated error")
	}
	for _, sub := range m.failOnSubstrings {
		if strings.Contains(full, sub) {
			return "", "mock error for " + full, errors.New("simulated error for " + sub)
		}
	}

	if name == "flock" && len(args) > 0 {
		m.mu.Lock()
		defer m.mu.Unlock()
		switch args[0] {
		case "sites":
			out := "SITE   PHP   URL   ROOT   PATH\n"
			for _, s := range m.flockSites {
				scheme := "http"
				if s.secured {
					scheme = "https"
				}
				siteURL := fmt.Sprintf("%s://%s.test", scheme, s.slug)
				out += fmt.Sprintf("%s   8.3   %s   /   %s\n", s.slug, siteURL, s.path)
			}
			return out, "", nil
		case "link":
			if len(args) > 1 {
				m.flockSites[args[1]] = &mockFlockSite{slug: args[1], path: dir, secured: false}
			}
			return "Linked", "", nil
		case "secure":
			if len(args) > 1 {
				if s, ok := m.flockSites[args[1]]; ok {
					s.secured = true
				}
			}
			return "Secured", "", nil
		case "unsecure":
			if len(args) > 1 {
				if s, ok := m.flockSites[args[1]]; ok {
					s.secured = false
				}
			}
			return "Unsecured", "", nil
		case "unlink":
			if len(args) > 1 {
				delete(m.flockSites, args[1])
			}
			return "Unlinked", "", nil
		}
	}

	if strings.Contains(full, "herd secured") {
		return m.securedSites + "\n", "", nil
	}
	if strings.Contains(full, "theme list") {
		if m.activeTheme != "" {
			return m.activeTheme + "\n", "", nil
		}
		return "flatsome\n", "", nil
	}

	return "Success: Database created.", "", nil
}
func (m *mockRunner) LookPath(file string) (string, error) {
	return "/bin/" + file, nil
}

type mockCoreResolver struct {
	path    string
	version string
	err     error
}

func (m *mockCoreResolver) Resolve(ctx context.Context) (string, string, error) {
	if m.err != nil {
		return "", "", m.err
	}
	return m.path, m.version, nil
}

func mockCoreExtractor(archivePath, destDir string) error {
	_ = os.MkdirAll(filepath.Join(destDir, "wp-content", "themes"), 0755)
	_ = os.MkdirAll(filepath.Join(destDir, "wp-content", "plugins"), 0755)
	_ = os.WriteFile(filepath.Join(destDir, "wp-load.php"), []byte("<?php // core"), 0600)
	_ = os.WriteFile(filepath.Join(destDir, "index.php"), []byte("<?php // core"), 0600)
	return nil
}

func newTestConfig(tempDir string) *config.Config {
	cfg := config.DefaultConfig(tempDir)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "herd"
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")
	return cfg
}

func newTestCreator(cfg *config.Config, client *wpcli.Client, checkDB create.DBExistsFunc, opts ...create.CreatorOption) *create.Creator {
	defaultOpts := []create.CreatorOption{
		create.WithCoreResolver(&mockCoreResolver{path: "/test/core.zip", version: "7.1"}),
		create.WithCoreExtractor(mockCoreExtractor),
	}
	combined := append(defaultOpts, opts...)
	return create.NewCreator(cfg, client, checkDB, combined...)
}

func TestCreator_SuccessFlow(t *testing.T) {
	tempDir := t.TempDir()
	cfg := newTestConfig(tempDir)
	if err := os.MkdirAll(cfg.WebsitesPath, 0755); err != nil {
		t.Fatal(err)
	}
	runner := &mockRunner{}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)
	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil // no DB collision
	})

	req := create.Request{
		WebsiteName:   "Demo Store",
		WebsiteSlug:   "demo-store",
		AdminUsername: "admin",
		AdminPassword: "adminpassword",
		AdminEmail:    "admin@demo.test",
	}

	var progressSteps []string
	progress := func(step, detail string) {
		progressSteps = append(progressSteps, step)
	}

	ctx := context.Background()
	res, err := creator.Create(ctx, req, progress)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	expectedURL := "https://demo-store.test" // Herd is true by default
	if res.WebsiteURL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, res.WebsiteURL)
	}

	expectedPath := filepath.Join(cfg.WebsitesPath, "demo-store")
	if res.WebsitePath != expectedPath {
		t.Errorf("expected Path %s, got %s", expectedPath, res.WebsitePath)
	}

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected website directory to exist at %s", expectedPath)
	}

	// Verify environment_secure is the final progress step
	if len(progressSteps) > 0 && progressSteps[len(progressSteps)-1] != "environment_secure" {
		t.Errorf("expected environment_secure to be final progress step, got %s", progressSteps[len(progressSteps)-1])
	}

	// Verify progress steps include core_resolve and core_extract
	var foundResolve, foundExtract bool
	for _, s := range progressSteps {
		if s == "core_resolve" {
			foundResolve = true
		}
		if s == "core_extract" {
			foundExtract = true
		}
	}
	if !foundResolve || !foundExtract {
		t.Errorf("expected core_resolve and core_extract progress steps, got: %v", progressSteps)
	}

	// Verify wp core download was NOT called
	for _, call := range runner.calls {
		if strings.Contains(call, "core download") {
			t.Errorf("expected no wp core download to be called, got call: %s", call)
		}
	}

	// Verify standard WordPress directories scaffolded
	themesDir := filepath.Join(expectedPath, "wp-content", "themes")
	pluginsDir := filepath.Join(expectedPath, "wp-content", "plugins")
	if _, err := os.Stat(themesDir); err != nil {
		t.Errorf("expected themes directory %s to exist: %v", themesDir, err)
	}
	if _, err := os.Stat(pluginsDir); err != nil {
		t.Errorf("expected plugins directory %s to exist: %v", pluginsDir, err)
	}
}

func TestCreator_PackageInstallAndDeduplication(t *testing.T) {
	tempDir := t.TempDir()
	cfg := newTestConfig(tempDir)
	runner := &mockRunner{}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	pluginZip := filepath.Join(tempDir, "plugin.zip")
	_ = os.WriteFile(pluginZip, []byte("fake zip"), 0600)

	defaultThemeZip := filepath.Join(tempDir, "theme.zip")
	_ = os.WriteFile(defaultThemeZip, []byte("fake zip"), 0600)

	req := create.Request{
		WebsiteName:     "Pack Store",
		WebsiteSlug:     "pack-store",
		AdminUsername:   "admin",
		AdminPassword:   "password",
		AdminEmail:      "admin@demo.test",
		ActiveThemeSlug: "flatsome",
		PluginArtifacts: []packages.Artifact{
			{
				Ref:     packages.PackageRef{Type: "plugin", Slug: "acf-pro"},
				Version: "1.0.0",
				Path:    pluginZip,
			},
		},
		ThemeArtifacts: []packages.Artifact{
			{
				Ref:     packages.PackageRef{Type: "theme", Slug: "flatsome"},
				Version: "1.0.0",
				Path:    defaultThemeZip,
			},
			{
				Ref:     packages.PackageRef{Type: "theme", Slug: "bricks"},
				Version: "1.0.0",
				Path:    defaultThemeZip,
			},
		},
	}

	res, err := creator.Create(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if len(res.InstalledPlugins) != 1 || res.InstalledPlugins[0] != "acf-pro" {
		t.Errorf("expected acf-pro to be installed, got %v", res.InstalledPlugins)
	}
	if res.ActiveTheme != "flatsome" {
		t.Errorf("expected active theme flatsome, got %s", res.ActiveTheme)
	}
	// Verify flatsome was not duplicated in installed themes
	flatsomeCount := 0
	for _, th := range res.InstalledThemes {
		if th == "flatsome" {
			flatsomeCount++
		}
	}
	if flatsomeCount != 1 {
		t.Errorf("expected flatsome to be deduplicated and installed once, got count %d in %v", flatsomeCount, res.InstalledThemes)
	}
}

func TestCreator_ApplyTweaksBestEffort(t *testing.T) {
	tempDir := t.TempDir()
	cfg := newTestConfig(tempDir)

	runner := &mockRunner{
		failOnSubstr: "wp rewrite structure",
	}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	req := create.Request{
		WebsiteName:   "Tweak Store",
		WebsiteSlug:   "tweak-store",
		AdminUsername: "admin",
		AdminPassword: "adminpassword",
		AdminEmail:    "admin@demo.test",
		ApplyTweaks:   true,
	}

	res, err := creator.Create(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("Create should not fail when non-critical tweak fails: %v", err)
	}

	// Website MUST be kept
	websitePath := filepath.Join(cfg.WebsitesPath, "tweak-store")
	if _, err := os.Stat(websitePath); os.IsNotExist(err) {
		t.Errorf("expected website directory to be kept despite tweak failure")
	}

	// Result must report failed tweak
	if len(res.FailedTweaks) == 0 {
		t.Errorf("expected FailedTweaks to contain rewrite failure")
	}
}

func TestCreator_RollbackOnCoreInstallFailure(t *testing.T) {
	tempDir := t.TempDir()
	cfg := newTestConfig(tempDir)

	runner := &mockRunner{
		failOnSubstr: "wp core install",
	}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	req := create.Request{
		WebsiteName:   "Fail Site",
		WebsiteSlug:   "fail-site",
		AdminUsername: "admin",
		AdminPassword: "adminpassword",
		AdminEmail:    "admin@demo.test",
	}

	ctx := context.Background()
	_, err := creator.Create(ctx, req, func(step, detail string) {})
	if err == nil {
		t.Fatal("expected error on core install failure, got nil")
	}

	// Verify wp db drop was invoked because DB was owned by this run
	dbDropCalled := false
	for _, call := range runner.calls {
		if strings.Contains(call, "wp db drop") {
			dbDropCalled = true
			break
		}
	}
	if !dbDropCalled {
		t.Errorf("expected wp db drop to be called during rollback")
	}

	websitePath := filepath.Join(cfg.WebsitesPath, "fail-site")
	if _, err := os.Stat(websitePath); !os.IsNotExist(err) {
		t.Errorf("expected website directory %s to be rolled back and removed", websitePath)
	}
}

func TestCreator_DBCreateFailureDoesNotDropExternalDB(t *testing.T) {
	tempDir := t.TempDir()
	cfg := newTestConfig(tempDir)

	runner := &mockRunner{
		failOnSubstr: "wp db create",
	}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	req := create.Request{
		WebsiteName:   "Early Fail",
		WebsiteSlug:   "early-fail",
		AdminUsername: "admin",
		AdminPassword: "adminpassword",
		AdminEmail:    "admin@demo.test",
	}

	ctx := context.Background()
	_, err := creator.Create(ctx, req, nil)
	if err == nil {
		t.Fatal("expected error on db create failure, got nil")
	}

	// Because DBCreate failed, createdDB is false. wp db drop should NOT be called.
	for _, call := range runner.calls {
		if strings.Contains(call, "wp db drop") {
			t.Errorf("wp db drop should NOT be called if DBCreate itself failed: %s", call)
		}
	}

	// Directory was created, so directory must be cleaned up
	siteDir := filepath.Join(cfg.WebsitesPath, "early-fail")
	if _, err := os.Stat(siteDir); !os.IsNotExist(err) {
		t.Errorf("directory should have been removed during rollback")
	}
}

func TestCreator_DBPreflightErrorHaltsBeforeMutation(t *testing.T) {
	tempDir := t.TempDir()
	cfg := newTestConfig(tempDir)

	runner := &mockRunner{}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	expectedErr := errors.New("connection refused to mysql")
	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, expectedErr
	})

	req := create.Request{
		WebsiteName:   "Preflight Fail",
		WebsiteSlug:   "preflight-fail",
		AdminUsername: "admin",
		AdminPassword: "adminpassword",
		AdminEmail:    "admin@demo.test",
	}

	_, err := creator.Create(context.Background(), req, nil)
	if err == nil {
		t.Fatal("expected error when DB preflight fails, got nil")
	}

	if !strings.Contains(err.Error(), "database preflight check failed") {
		t.Errorf("unexpected error message: %v", err)
	}

	// Make sure no directory was created
	siteDir := filepath.Join(cfg.WebsitesPath, "preflight-fail")
	if _, err := os.Stat(siteDir); !os.IsNotExist(err) {
		t.Errorf("directory should NOT have been created on preflight error")
	}

	// Make sure no mutation commands were executed
	for _, call := range runner.calls {
		if strings.Contains(call, "wp ") || strings.Contains(call, "create") {
			t.Errorf("no mutation commands should be executed if preflight fails, got %s", call)
		}
	}
}

func TestCreator_HerdPathTrustSucceedsWithoutParkedInspection(t *testing.T) {
	tempDir := t.TempDir()
	cfg := newTestConfig(tempDir)

	runner := &mockRunner{}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	req := create.Request{
		WebsiteName: "Trusted Site",
		WebsiteSlug: "trusted-site",
	}

	res, err := creator.Create(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("expected create to succeed trusting websites_path, got error: %v", err)
	}
	if res.WebsiteURL != "https://trusted-site.test" {
		t.Errorf("expected https://trusted-site.test, got %s", res.WebsiteURL)
	}

	// Ensure herd paths was NEVER called
	for _, call := range runner.calls {
		if strings.Contains(call, "herd paths") || strings.Contains(call, "herd parked") {
			t.Errorf("unexpected herd paths inspection call: %s", call)
		}
	}
}

func TestCreator_CollisionPreflight(t *testing.T) {
	tempDir := t.TempDir()
	cfg := newTestConfig(tempDir)
	collidedDir := filepath.Join(cfg.WebsitesPath, "existing-site")
	if err := os.MkdirAll(collidedDir, 0755); err != nil {
		t.Fatal(err)
	}

	runner := &mockRunner{}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	req := create.Request{
		WebsiteName:   "Existing Site",
		WebsiteSlug:   "existing-site",
		AdminUsername: "admin",
		AdminPassword: "adminpassword",
		AdminEmail:    "admin@demo.test",
	}

	_, err := creator.Create(context.Background(), req, func(step, detail string) {})
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}

	if !create.IsCollisionError(err) {
		t.Errorf("expected collision error type, got %v", err)
	}

	// Make sure existing directory was NOT deleted
	if _, err := os.Stat(collidedDir); os.IsNotExist(err) {
		t.Errorf("existing collided directory was improperly deleted")
	}
}

func TestCreator_PreExistingTLSRollbackDoesNotUnsecure(t *testing.T) {
	tempDir := t.TempDir()
	cfg := newTestConfig(tempDir)

	// Site was already secured prior to this run
	runner := &mockRunner{
		securedSites: "existing-secure.test",
		failOnSubstr: "wp plugin install", // fail later to trigger rollback
	}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	dummyZip := filepath.Join(tempDir, "p.zip")
	_ = os.WriteFile(dummyZip, []byte("zip"), 0600)

	req := create.Request{
		WebsiteName:   "Existing Secure",
		WebsiteSlug:   "existing-secure",
		AdminUsername: "admin",
		AdminPassword: "password",
		AdminEmail:    "admin@demo.test",
		PluginArtifacts: []packages.Artifact{
			{
				Ref:     packages.PackageRef{Type: "plugin", Slug: "failing-plugin"},
				Version: "1.0.0",
				Path:    dummyZip,
			},
		},
	}

	_, err := creator.Create(context.Background(), req, nil)
	if err == nil {
		t.Fatal("expected error on plugin install, got nil")
	}

	// Ensure neither herd secure nor herd unsecure was invoked because TLS was pre-existing
	for _, call := range runner.calls {
		if strings.Contains(call, "herd secure") {
			t.Errorf("pre-existing TLS was needlessly secured: %s", call)
		}
		if strings.Contains(call, "herd unsecure") {
			t.Errorf("pre-existing TLS was improperly unsecured during rollback: %s", call)
		}
	}
}

func TestCreator_HerdProbeFailureHalts(t *testing.T) {
	tempDir := t.TempDir()
	cfg := newTestConfig(tempDir)

	// Probe failure on herd secured
	runner := &mockRunner{
		failOnSubstr: "herd secured",
	}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	req := create.Request{
		WebsiteName:   "Probe Fail",
		WebsiteSlug:   "probe-fail",
		AdminUsername: "admin",
		AdminPassword: "password",
		AdminEmail:    "admin@demo.test",
	}

	_, err := creator.Create(context.Background(), req, nil)
	if err == nil {
		t.Fatal("expected error on herd probe failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to verify environment TLS status") {
		t.Errorf("expected probe failure error message, got %v", err)
	}
}

func TestCreator_RollbackOnCoreExtractionFailure(t *testing.T) {
	tempDir := t.TempDir()
	cfg := newTestConfig(tempDir)

	runner := &mockRunner{}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	failingExtractor := func(archivePath, destDir string) error {
		return errors.New("simulated extraction failure: corrupted zip entry")
	}

	creator := create.NewCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	}, create.WithCoreResolver(&mockCoreResolver{path: "/test.zip", version: "7.1"}), create.WithCoreExtractor(failingExtractor))

	req := create.Request{
		WebsiteName:   "Doomed Extract Site",
		WebsiteSlug:   "doomed-extract",
		AdminUsername: "admin",
		AdminPassword: "password",
		AdminEmail:    "admin@fail.test",
	}

	_, err := creator.Create(context.Background(), req, nil)
	if err == nil {
		t.Fatal("expected error on failing core extraction, got nil")
	}
	if !strings.Contains(err.Error(), "failed to extract WordPress core") {
		t.Errorf("expected extraction error, got: %v", err)
	}

	// Verify website directory was rolled back
	siteDir := filepath.Join(cfg.WebsitesPath, "doomed-extract")
	if _, err := os.Stat(siteDir); !os.IsNotExist(err) {
		t.Errorf("expected website directory to be rolled back on extraction failure, but it exists at %s", siteDir)
	}
}

func TestCreator_NoActiveTheme(t *testing.T) {
	tempDir := t.TempDir()
	cfg := newTestConfig(tempDir)
	runner := &mockRunner{}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	req := create.Request{
		WebsiteName: "No Theme Site",
		WebsiteSlug: "no-theme-site",
	}

	res, err := creator.Create(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("expected create to succeed without active theme, got error: %v", err)
	}
	if res.ActiveTheme != "" {
		t.Errorf("expected empty active theme, got %q", res.ActiveTheme)
	}
}

func TestCreator_FlockLifecycleAndRollback(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.AppName = config.AppNameFlock
	cfg.AppCommand = "flock"
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	runner := &mockRunner{
		failOnSubstr: "wp plugin install", // fail late to trigger rollback
	}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	dummyZip := filepath.Join(tempDir, "p.zip")
	_ = os.WriteFile(dummyZip, []byte("zip"), 0600)

	req := create.Request{
		WebsiteName: "Flock Test",
		WebsiteSlug: "flock-test",
		PluginArtifacts: []packages.Artifact{
			{
				Ref:     packages.PackageRef{Type: "plugin", Slug: "failing-plugin"},
				Version: "1.0.0",
				Path:    dummyZip,
			},
		},
	}

	_, err := creator.Create(context.Background(), req, nil)
	if err == nil {
		t.Fatal("expected create failure on plugin install, got nil")
	}

	// Verify link was executed, and rollback unlinked and removed directory
	var linkCalled, unlinkCalled bool
	for _, call := range runner.calls {
		if strings.Contains(call, "flock link flock-test") {
			linkCalled = true
		}
		if strings.Contains(call, "flock unlink flock-test") {
			unlinkCalled = true
		}
	}
	if !linkCalled {
		t.Errorf("expected flock link to be called")
	}
	if !unlinkCalled {
		t.Errorf("expected flock unlink to be called during rollback")
	}
}

func TestCreator_LaragonLifecycleAndRollback(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.AppName = config.AppNameLaragon
	cfg.AppCommand = "C:\\laragon\\laragon.exe"
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	runner := &mockRunner{
		failOnSubstr: "wp core install", // fail after directory creation
	}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	req := create.Request{
		WebsiteName: "Laragon Test",
		WebsiteSlug: "laragon-test",
	}

	_, err := creator.Create(context.Background(), req, nil)
	if err == nil {
		t.Fatal("expected create failure on core install, got nil")
	}

	// Laragon reload should have been called twice: after dir creation, and again after rollback deletion
	reloadCount := 0
	for _, call := range runner.calls {
		if strings.Contains(call, "laragon.exe reload") {
			reloadCount++
		}
	}
	if reloadCount != 2 {
		t.Errorf("expected 2 laragon reload calls (creation and rollback), got %d (calls: %v)", reloadCount, runner.calls)
	}
}
func TestCreator_InvalidActiveThemeSlugRejected(t *testing.T) {
	tempDir := t.TempDir()
	cfg := newTestConfig(tempDir)
	runner := &mockRunner{}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	dummyZip := filepath.Join(tempDir, "theme.zip")
	_ = os.WriteFile(dummyZip, []byte("theme"), 0600)

	req := create.Request{
		WebsiteName:     "Invalid Theme Site",
		WebsiteSlug:     "invalid-theme-site",
		ActiveThemeSlug: "nonexistent-theme",
		ThemeArtifacts: []packages.Artifact{
			{
				Ref:     packages.PackageRef{Type: "theme", Slug: "selected-theme"},
				Version: "1.0.0",
				Path:    dummyZip,
			},
		},
	}

	_, err := creator.Create(context.Background(), req, nil)
	if err == nil {
		t.Fatal("expected error when ActiveThemeSlug is not in ThemeArtifacts, got nil")
	}
	if !strings.Contains(err.Error(), "active theme \"nonexistent-theme\" is not present in selected theme artifacts") {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify no directory was created
	siteDir := filepath.Join(cfg.WebsitesPath, "invalid-theme-site")
	if _, err := os.Stat(siteDir); !os.IsNotExist(err) {
		t.Errorf("no directory mutation should have occurred before preflight check")
	}
}
func TestCreator_RollbackUnlinkFailureExposedAndCleanupAttempted(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.AppName = config.AppNameFlock
	cfg.AppCommand = "flock"
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	runner := &mockRunner{
		failOnSubstrings: []string{"wp core install", "flock unlink"},
	}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	req := create.Request{
		WebsiteName: "Flock Rollback Fail",
		WebsiteSlug: "flock-rb-fail",
	}

	_, err := creator.Create(context.Background(), req, nil)
	if err == nil {
		t.Fatal("expected create error, got nil")
	}

	// Error must expose both the primary failure and the rollback cleanup failure
	errStr := err.Error()
	if !strings.Contains(errStr, "wp core install") {
		t.Errorf("expected error to contain primary install failure, got: %s", errStr)
	}
	if !strings.Contains(errStr, "environment unregister cleanup failed") {
		t.Errorf("expected error to contain rollback unregister failure, got: %s", errStr)
	}

	// DB drop must still have been attempted
	var dbDropCalled bool
	for _, call := range runner.calls {
		if strings.Contains(call, "wp db drop") {
			dbDropCalled = true
			break
		}
	}
	if !dbDropCalled {
		t.Errorf("expected wp db drop to be attempted despite unregister failure")
	}

	// Directory must still have been cleaned up
	siteDir := filepath.Join(cfg.WebsitesPath, "flock-rb-fail")
	if _, err := os.Stat(siteDir); !os.IsNotExist(err) {
		t.Errorf("expected directory to be deleted despite unregister failure, still exists at %s", siteDir)
	}
}

func TestCreator_InvalidConfigPreflightFailsBeforeMutation(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.AppName = config.AppNameHerd
	cfg.AppCommand = "" // invalid empty command
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	runner := &mockRunner{}
	wpClient := wpcli.NewClientWithRunnerForConfig(runner, cfg)

	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	req := create.Request{
		WebsiteName: "Invalid Config Site",
		WebsiteSlug: "invalid-cfg-site",
	}

	_, err := creator.Create(context.Background(), req, nil)
	if err == nil {
		t.Fatal("expected preflight error for invalid config, got nil")
	}

	// Verify no directory was created
	siteDir := filepath.Join(cfg.WebsitesPath, "invalid-cfg-site")
	if _, err := os.Stat(siteDir); !os.IsNotExist(err) {
		t.Errorf("no directory mutation should have occurred before preflight check")
	}

	// Verify no commands were run
	if len(runner.calls) != 0 {
		t.Errorf("expected 0 commands executed on preflight failure, got: %v", runner.calls)
	}
}
