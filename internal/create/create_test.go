package create_test

import (
	"context"
	"errors"
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
type mockRunner struct {
	mu           sync.Mutex
	calls        []string
	failOnSubstr string
	securedSites string
	activeTheme  string
}

func (m *mockRunner) Run(ctx context.Context, dir string, name string, args []string, stdin string) (string, string, error) {
	full := name + " " + strings.Join(args, " ")
	m.mu.Lock()
	m.calls = append(m.calls, full)
	m.mu.Unlock()
	if m.failOnSubstr != "" && strings.Contains(full, m.failOnSubstr) {
		return "", "mock error for " + full, errors.New("simulated error")
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
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")
	if err := os.MkdirAll(cfg.WebsitesPath, 0755); err != nil {
		t.Fatal(err)
	}
	runner := &mockRunner{}
	wpClient := wpcli.NewClientWithRunner(runner)

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

	// Verify herd_secure is the final progress step
	if len(progressSteps) > 0 && progressSteps[len(progressSteps)-1] != "herd_secure" {
		t.Errorf("expected herd_secure to be final progress step, got %s", progressSteps[len(progressSteps)-1])
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
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")
	runner := &mockRunner{}
	wpClient := wpcli.NewClientWithRunner(runner)

	creator := newTestCreator(cfg, wpClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	pluginZip := filepath.Join(tempDir, "plugin.zip")
	_ = os.WriteFile(pluginZip, []byte("fake zip"), 0600)

	defaultThemeZip := filepath.Join(tempDir, "theme.zip")
	_ = os.WriteFile(defaultThemeZip, []byte("fake zip"), 0600)

	req := create.Request{
		WebsiteName:   "Pack Store",
		WebsiteSlug:   "pack-store",
		AdminUsername: "admin",
		AdminPassword: "password",
		AdminEmail:    "admin@demo.test",
		DefaultTheme: &packages.Artifact{
			Ref:     packages.PackageRef{Type: "theme", Slug: "flatsome"},
			Version: "1.0.0",
			Path:    defaultThemeZip,
		},
		PluginArtifacts: []packages.Artifact{
			{
				Ref:     packages.PackageRef{Type: "plugin", Slug: "acf-pro"},
				Version: "1.0.0",
				Path:    pluginZip,
			},
		},
		ThemeArtifacts: []packages.Artifact{
			{
				Ref:     packages.PackageRef{Type: "theme", Slug: "flatsome"}, // duplicate of default theme!
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
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	runner := &mockRunner{
		failOnSubstr: "wp rewrite structure",
	}
	wpClient := wpcli.NewClientWithRunner(runner)

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
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	runner := &mockRunner{
		failOnSubstr: "wp core install",
	}
	wpClient := wpcli.NewClientWithRunner(runner)

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
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	runner := &mockRunner{
		failOnSubstr: "wp db create",
	}
	wpClient := wpcli.NewClientWithRunner(runner)

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
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	runner := &mockRunner{}
	wpClient := wpcli.NewClientWithRunner(runner)

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
	cfg := config.DefaultConfig(tempDir)
	cfg.UsedHerd = true
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	runner := &mockRunner{}
	wpClient := wpcli.NewClientWithRunner(runner)

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
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")
	collidedDir := filepath.Join(cfg.WebsitesPath, "existing-site")
	if err := os.MkdirAll(collidedDir, 0755); err != nil {
		t.Fatal(err)
	}

	runner := &mockRunner{}
	wpClient := wpcli.NewClientWithRunner(runner)

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
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	// Site was already secured prior to this run
	runner := &mockRunner{
		securedSites: "existing-secure.test",
		failOnSubstr: "wp plugin install", // fail later to trigger rollback
	}
	wpClient := wpcli.NewClientWithRunner(runner)

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

	// Ensure herd unsecure was NOT invoked because TLS was pre-existing
	for _, call := range runner.calls {
		if strings.Contains(call, "herd unsecure") {
			t.Errorf("pre-existing TLS was improperly unsecured during rollback: %s", call)
		}
	}
}

func TestCreator_HerdProbeFailureHalts(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	// Probe failure on herd secured
	runner := &mockRunner{
		failOnSubstr: "herd secured",
	}
	wpClient := wpcli.NewClientWithRunner(runner)

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
	if !strings.Contains(err.Error(), "failed to verify herd TLS status") {
		t.Errorf("expected probe failure error message, got %v", err)
	}
}

func TestCreator_RollbackOnCoreExtractionFailure(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	runner := &mockRunner{}
	wpClient := wpcli.NewClientWithRunner(runner)

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

