package packages_test

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"wptui/internal/config"
	"wptui/internal/packages"
)

func TestResolver_ResolvePackage_CacheHitOnSecondCall(t *testing.T) {
	zipData := createValidZipBytes()

	var downloadHits int32
	var downloadServer *httptest.Server
	downloadServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&downloadHits, 1)
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipData)
	}))
	defer downloadServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta := packages.Metadata{
			Name:        "Test Plugin",
			Version:     "1.0.0",
			Slug:        "test-plugin",
			Type:        "plugin",
			Size:        fmt.Sprintf("%d", len(zipData)),
			DownloadURL: downloadServer.URL + "/download.zip?license_key=xyz",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(meta)
	}))
	defer apiServer.Close()

	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.PackagesAPIURL = apiServer.URL

	cache, err := packages.NewCache(filepath.Join(tempDir, "cache"))
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

	resolver := packages.NewResolver(cfg, cache, opts)
	stageDir := filepath.Join(tempDir, "stage")

	ref := packages.PackageRef{Type: "plugin", Slug: "test-plugin"}

	// First resolution: downloads and populates cache
	art1, err := resolver.ResolvePackage(context.Background(), ref, stageDir)
	if err != nil {
		t.Fatalf("first ResolvePackage failed: %v", err)
	}
	if art1.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", art1.Version)
	}
	if atomic.LoadInt32(&downloadHits) != 1 {
		t.Errorf("expected 1 download hit, got %d", atomic.LoadInt32(&downloadHits))
	}

	// Second resolution: must hit cache and NOT hit download server!
	art2, err := resolver.ResolvePackage(context.Background(), ref, stageDir)
	if err != nil {
		t.Fatalf("second ResolvePackage failed: %v", err)
	}
	if art2.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", art2.Version)
	}
	if atomic.LoadInt32(&downloadHits) != 1 {
		t.Errorf("expected download hits to remain 1 on cache hit, got %d", atomic.LoadInt32(&downloadHits))
	}

	if _, err := os.Stat(art2.Path); os.IsNotExist(err) {
		t.Errorf("cached artifact file does not exist at %s", art2.Path)
	}
}

func TestResolver_StagedArtifactIsImmuneToSubsequentCacheReplacement(t *testing.T) {
	zipData := createValidZipBytes()

	var downloadServer *httptest.Server
	downloadServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipData)
	}))
	defer downloadServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta := packages.Metadata{
			Name:        "Immutable Plugin",
			Version:     "1.0.0",
			Slug:        "immutable-plugin",
			Type:        "plugin",
			Size:        fmt.Sprintf("%d", len(zipData)),
			DownloadURL: downloadServer.URL + "/download.zip",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(meta)
	}))
	defer apiServer.Close()

	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.PackagesAPIURL = apiServer.URL

	cache, err := packages.NewCache(filepath.Join(tempDir, "cache"))
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

	resolver := packages.NewResolver(cfg, cache, opts)
	stageDir := filepath.Join(tempDir, "stage")
	ref := packages.PackageRef{Type: "plugin", Slug: "immutable-plugin"}

	// Resolve package (populates cache and stages copy into stageDir)
	art, err := resolver.ResolvePackage(context.Background(), ref, stageDir)
	if err != nil {
		t.Fatalf("ResolvePackage failed: %v", err)
	}

	// Now simulate a concurrent process upgrading the cache to v2.0.0, deleting v1.0.0 in cache
	stageUpgrade := filepath.Join(tempDir, "v2.zip")
	_ = os.WriteFile(stageUpgrade, zipData, 0600)
	_, err = cache.Put(context.Background(), ref, "2.0.0", stageUpgrade)
	if err != nil {
		t.Fatalf("cache.Put upgrade failed: %v", err)
	}

	// The staged artifact handed to Create must STILL exist and remain readable!
	if _, err := os.Stat(art.Path); os.IsNotExist(err) {
		t.Errorf("staged artifact at %s was deleted when cache replaced version!", art.Path)
	}
}
