package app_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"wptui/internal/app"
	"wptui/internal/config"
	"wptui/internal/packages"
	"wptui/internal/tui"
)

type e2eMockRunner struct {
	mu           sync.Mutex
	calls        []string
	stdin        []string
	failOnSubstr string
	parkedPath   string
}

func (m *e2eMockRunner) Run(ctx context.Context, dir string, name string, args []string, stdin string) (string, string, error) {
	call := name + " " + strings.Join(args, " ")
	m.mu.Lock()
	m.calls = append(m.calls, call)
	m.stdin = append(m.stdin, stdin)
	m.mu.Unlock()
	if m.failOnSubstr != "" && strings.Contains(call, m.failOnSubstr) {
		return "", "mock failure on: " + call, errors.New("simulated critical failure")
	}
	if strings.Contains(call, "core download") {
		// Materialize core base files
		_ = os.MkdirAll(filepath.Join(dir, "wp-content"), 0755)
		_ = os.WriteFile(filepath.Join(dir, "wp-load.php"), []byte("<?php // wp core"), 0600)
		// If --skip-content is absent, simulate standard bundle containing twentytwentyfour & hello.php
		if !strings.Contains(call, "--skip-content") {
			_ = os.MkdirAll(filepath.Join(dir, "wp-content", "themes", "twentytwentyfour"), 0755)
			_ = os.MkdirAll(filepath.Join(dir, "wp-content", "plugins", "akismet"), 0755)
			_ = os.WriteFile(filepath.Join(dir, "wp-content", "plugins", "akismet", "akismet.php"), []byte("<?php // akismet"), 0600)
			_ = os.WriteFile(filepath.Join(dir, "wp-content", "plugins", "hello.php"), []byte("<?php // hello"), 0600)
		}
	}

	if strings.Contains(call, "herd paths") || strings.Contains(call, "herd parked") {
		return m.parkedPath + "\n", "", nil
	}

	if strings.Contains(call, "SHOW DATABASES;") {
		return "information_schema\nmysql\nperformance_schema\n", "", nil
	}

	return "Success: Database created.", "", nil
}

func (m *e2eMockRunner) LookPath(file string) (string, error) {
	return "/usr/bin/" + file, nil
}

func createTestZip() []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	f, _ := w.Create("index.php")
	_, _ = f.Write([]byte("<?php // composition test package"))
	_ = w.Close()
	return buf.Bytes()
}

func TestTicket08_ComposedCreateFlowSmoke(t *testing.T) {
	tempHome := t.TempDir()
	zipBytes := createTestZip()

	// 1. Mock Download and API servers
	var downloadCount int32
	downloadServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&downloadCount, 1)
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipBytes)
	}))
	defer downloadServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := "test-plugin"
		typ := packages.PackageTypePlugin
		if strings.Contains(r.URL.Path, "flatsome") {
			slug = "flatsome"
			typ = packages.PackageTypeTheme
		}
		meta := packages.Metadata{
			Name:        "Test Package",
			Version:     "1.0.0",
			Slug:        slug,
			Type:        typ,
			Size:        fmt.Sprintf("%d", len(zipBytes)),
			DownloadURL: downloadServer.URL + "/download.zip?license_key=secret_key&signature=sig123",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(meta)
	}))
	defer apiServer.Close()

	// 2. Setup Config
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	cfg.PackagesAPIURL = apiServer.URL
	cfg.PackagesAPIKey = "secret_key"
	cfg.DefaultThemeSlug = "flatsome"
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}

	runner := &e2eMockRunner{
		parkedPath: cfg.WebsitesPath,
	}
	cacheDir := filepath.Join(tempHome, "cache")
	pkgCache, err := packages.NewCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}

	rootCAs := x509.NewCertPool()
	rootCAs.AddCert(downloadServer.Certificate())
	downloadIP := downloadServer.Listener.Addr().(*net.TCPAddr).IP

	opts := packages.DownloadOptions{
		AllowedIPs: []net.IP{downloadIP},
		RootCAs:    rootCAs,
	}

	resolver := packages.NewResolver(cfg, pkgCache, opts)

	// Scenario A: First site creation through app.New and application.Run()
	menuCallsA := 0
	scriptedInputsA := tui.CreateInputs{
		WebsiteName:   "Composition Site One",
		WebsiteSlug:   "site-one",
		AdminUsername: "admin",
		AdminPassword: "topsecret",
		AdminEmail:    "admin@site-one.test",
		ApplyTweaks:   true,
	}

	appOne := app.New(app.Options{
		HomeDir: tempHome,
		MenuFn: func() (string, error) {
			menuCallsA++
			if menuCallsA == 1 {
				return "create", nil
			}
			return "exit", nil
		},
		CreateFn: func(ctx context.Context, c *config.Config) error {
			return app.RunCreateFlowWithDeps(ctx, c, app.CreateFlowDependencies{
				Runner:   runner,
				Resolver: resolver,
				PromptCreate: func(cfg *config.Config, checker ...tui.SlugAvailabilityChecker) (*tui.CreateInputs, error) {
					return &scriptedInputsA, nil
				},
				PromptPackages: func(ctx context.Context, cfg *config.Config, catalog []packages.CatalogItem) ([]string, []string, error) {
					return []string{"test-plugin"}, nil, nil
				},
			})
		},
	})

	if err := appOne.Run(); err != nil {
		t.Fatalf("first site appOne.Run failed: %v", err)
	}

	siteOneDir := filepath.Join(cfg.WebsitesPath, "site-one")
	if _, err := os.Stat(siteOneDir); os.IsNotExist(err) {
		t.Errorf("expected site-one directory to exist at %s", siteOneDir)
	}

	// Verify clean lean core: no default bundled themes or plugins in wp-content
	bundledThemeDir := filepath.Join(siteOneDir, "wp-content", "themes", "twentytwentyfour")
	if _, err := os.Stat(bundledThemeDir); !os.IsNotExist(err) {
		t.Errorf("expected bundled theme twentytwentyfour to be absent under --skip-content, found at %s", bundledThemeDir)
	}
	helloDollyPlugin := filepath.Join(siteOneDir, "wp-content", "plugins", "hello.php")
	if _, err := os.Stat(helloDollyPlugin); !os.IsNotExist(err) {
		t.Errorf("expected bundled plugin hello.php to be absent under --skip-content, found at %s", helloDollyPlugin)
	}
	akismetPlugin := filepath.Join(siteOneDir, "wp-content", "plugins", "akismet")
	if _, err := os.Stat(akismetPlugin); !os.IsNotExist(err) {
		t.Errorf("expected bundled plugin akismet to be absent under --skip-content, found at %s", akismetPlugin)
	}
	if atomic.LoadInt32(&downloadCount) != 2 {
		t.Errorf("expected 2 download hits after site one, got %d", atomic.LoadInt32(&downloadCount))
	}

	// Scenario B: Second site creation through app.New and application.Run() reuses cache
	menuCallsB := 0
	scriptedInputsB := tui.CreateInputs{
		WebsiteName:   "Composition Site Two",
		WebsiteSlug:   "site-two",
		AdminUsername: "admin",
		AdminPassword: "topsecret",
		AdminEmail:    "admin@site-two.test",
		ApplyTweaks:   false,
	}

	appTwo := app.New(app.Options{
		HomeDir: tempHome,
		MenuFn: func() (string, error) {
			menuCallsB++
			if menuCallsB == 1 {
				return "create", nil
			}
			return "exit", nil
		},
		CreateFn: func(ctx context.Context, c *config.Config) error {
			return app.RunCreateFlowWithDeps(ctx, c, app.CreateFlowDependencies{
				Runner:   runner,
				Resolver: resolver,
				PromptCreate: func(cfg *config.Config, checker ...tui.SlugAvailabilityChecker) (*tui.CreateInputs, error) {
					return &scriptedInputsB, nil
				},
				PromptPackages: func(ctx context.Context, cfg *config.Config, catalog []packages.CatalogItem) ([]string, []string, error) {
					return []string{"test-plugin"}, nil, nil
				},
			})
		},
	})

	if err := appTwo.Run(); err != nil {
		t.Fatalf("second site appTwo.Run failed: %v", err)
	}

	siteTwoDir := filepath.Join(cfg.WebsitesPath, "site-two")
	if _, err := os.Stat(siteTwoDir); os.IsNotExist(err) {
		t.Errorf("expected site-two directory to exist at %s", siteTwoDir)
	}
	// Verify download count remained 1 because cache was reused!
	if atomic.LoadInt32(&downloadCount) != 2 {
		t.Errorf("expected download count to remain 2 on cache hit, got %d", atomic.LoadInt32(&downloadCount))
	}

	// Scenario C: Third site creation with critical failure triggers rollback through app
	menuCallsC := 0
	failingRunner := &e2eMockRunner{
		failOnSubstr: "wp core install",
		parkedPath:   cfg.WebsitesPath,
	}

	scriptedInputsC := tui.CreateInputs{
		WebsiteName:   "Failed Site Three",
		WebsiteSlug:   "site-three",
		AdminUsername: "admin",
		AdminPassword: "topsecret",
		AdminEmail:    "admin@site-three.test",
	}

	appThree := app.New(app.Options{
		HomeDir: tempHome,
		MenuFn: func() (string, error) {
			menuCallsC++
			if menuCallsC == 1 {
				return "create", nil
			}
			return "exit", nil
		},
		CreateFn: func(ctx context.Context, c *config.Config) error {
			return app.RunCreateFlowWithDeps(ctx, c, app.CreateFlowDependencies{
				Runner:   failingRunner,
				Resolver: resolver,
				PromptCreate: func(cfg *config.Config, checker ...tui.SlugAvailabilityChecker) (*tui.CreateInputs, error) {
					return &scriptedInputsC, nil
				},
				PromptPackages: func(ctx context.Context, cfg *config.Config, catalog []packages.CatalogItem) ([]string, []string, error) {
					return []string{"test-plugin"}, nil, nil
				},
			})
		},
	})

	_ = appThree.Run()

	// Verify site-three directory was removed by rollback
	siteThreeDir := filepath.Join(cfg.WebsitesPath, "site-three")
	if _, err := os.Stat(siteThreeDir); !os.IsNotExist(err) {
		t.Errorf("site-three directory was not rolled back")
	}

	// Verify site-one and site-two directories were NOT touched
	if _, err := os.Stat(siteOneDir); os.IsNotExist(err) {
		t.Errorf("site-one directory was improperly deleted!")
	}
	if _, err := os.Stat(siteTwoDir); os.IsNotExist(err) {
		t.Errorf("site-two directory was improperly deleted!")
	}

	// Verify package cache was NOT deleted
	cachedEntry, found, err := pkgCache.Get(context.Background(), packages.PackageRef{Type: "plugin", Slug: "test-plugin"})
	if err != nil || !found || cachedEntry == nil {
		t.Errorf("package cache was improperly deleted during site three rollback: found=%v, err=%v", found, err)
	}

	// Verify that secrets (passwords, license keys, signatures) did NOT leak into command arguments
	for _, call := range runner.calls {
		if strings.Contains(call, "topsecret") {
			t.Errorf("admin password leaked in command arguments: %s", call)
		}
		if strings.Contains(call, "signature=sig123") {
			t.Errorf("signed download signature leaked in command arguments: %s", call)
		}
	}
	for _, call := range failingRunner.calls {
		if strings.Contains(call, "topsecret") {
			t.Errorf("admin password leaked in failing runner command arguments: %s", call)
		}
	}
}
