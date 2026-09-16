package packages_test

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"wptui/internal/config"
	"wptui/internal/packages"
)

func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		isTransient bool
	}{
		{"timeout error", context.DeadlineExceeded, true},
		{"http 500 server error", &packages.HTTPError{StatusCode: 500}, true},
		{"http 502 bad gateway", &packages.HTTPError{StatusCode: 502}, true},
		{"http 503 service unavailable", &packages.HTTPError{StatusCode: 503}, true},
		{"http 429 rate limited", &packages.HTTPError{StatusCode: 429}, true},
		{"http 408 request timeout", &packages.HTTPError{StatusCode: 408}, true},
		{"http 401 unauthorized", &packages.HTTPError{StatusCode: 401}, false},
		{"http 403 forbidden", &packages.HTTPError{StatusCode: 403}, false},
		{"http 404 not found", &packages.HTTPError{StatusCode: 404}, false},
		{"ssrf rejection", errors.New("SSRF protection: disallowed IP"), false},
		{"invalid zip", errors.New("not a valid ZIP archive"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := packages.IsTransientError(tt.err)
			if got != tt.isTransient {
				t.Errorf("IsTransientError(%v) = %v, want %v", tt.err, got, tt.isTransient)
			}
		})
	}
}

func TestResolver_FallbackToStaleCacheOnTransientMetadataFailure(t *testing.T) {
	tempDir := t.TempDir()
	zipData := createValidZipBytes()

	cacheDir := filepath.Join(tempDir, "cache")
	cache, err := packages.NewCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}

	ref := packages.PackageRef{Type: "plugin", Slug: "fallback-plugin"}

	// Seed cache with v1.0.0
	stageFile := filepath.Join(tempDir, "seed.zip")
	_ = os.WriteFile(stageFile, zipData, 0600)
	_, err = cache.Put(context.Background(), ref, "1.0.0", stageFile)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Simulate transient 503 Service Unavailable on metadata API
	server503 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server503.Close()

	cfg := config.DefaultConfig(tempDir)
	cfg.PackagesAPIURL = server503.URL

	resolver := packages.NewResolver(cfg, cache)
	stageDir := filepath.Join(tempDir, "stage")

	art, err := resolver.ResolvePackage(context.Background(), ref, stageDir)
	if err != nil {
		t.Fatalf("expected stale cache fallback on 503, got error: %v", err)
	}

	if !art.IsStale {
		t.Errorf("expected artifact to be marked IsStale = true")
	}
	if art.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", art.Version)
	}

	// 2. Non-transient 401 Unauthorized must NOT use stale fallback
	server401 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server401.Close()

	cfg.PackagesAPIURL = server401.URL
	resolver401 := packages.NewResolver(cfg, cache)

	_, err = resolver401.ResolvePackage(context.Background(), ref, stageDir)
	if err == nil {
		t.Fatal("expected error on 401 unauthorized, but stale cache was improperly used")
	}

	// 3. Malformed/truncated JSON metadata must NOT use stale fallback
	serverTruncated := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name": "broken", "slug":`)) // truncated JSON
	}))
	defer serverTruncated.Close()

	cfg.PackagesAPIURL = serverTruncated.URL
	resolverTruncated := packages.NewResolver(cfg, cache)

	_, err = resolverTruncated.ResolvePackage(context.Background(), ref, stageDir)
	if err == nil {
		t.Fatal("expected error on truncated JSON metadata, but stale cache was improperly used")
	}
}
func TestResolver_FallbackToStaleCacheOnNewVersionDownloadFailure(t *testing.T) {
	tempDir := t.TempDir()
	zipData := createValidZipBytes()

	cacheDir := filepath.Join(tempDir, "cache")
	cache, err := packages.NewCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}

	ref := packages.PackageRef{Type: "plugin", Slug: "known-plugin"}

	// Seed cache with v1.0.0
	stageFile := filepath.Join(tempDir, "seed.zip")
	_ = os.WriteFile(stageFile, zipData, 0600)
	_, err = cache.Put(context.Background(), ref, "1.0.0", stageFile)
	if err != nil {
		t.Fatal(err)
	}

	// Download server returns 502 Bad Gateway
	downloadServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer downloadServer.Close()

	// Metadata announces newer v2.0.0
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta := packages.Metadata{
			Name:        "Known Plugin",
			Version:     "2.0.0", // newer version!
			Slug:        "known-plugin",
			Type:        "plugin",
			Size:        "500",
			DownloadURL: downloadServer.URL + "/download.zip",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(meta)
	}))
	defer apiServer.Close()

	cfg := config.DefaultConfig(tempDir)
	cfg.PackagesAPIURL = apiServer.URL

	rootCAs := x509.NewCertPool()
	rootCAs.AddCert(downloadServer.Certificate())
	downloadIP := downloadServer.Listener.Addr().(*net.TCPAddr).IP

	opts := packages.DownloadOptions{
		AllowedIPs: []net.IP{downloadIP},
		RootCAs:    rootCAs,
	}

	resolver := packages.NewResolver(cfg, cache, opts)
	stageDir := filepath.Join(tempDir, "stage")

	art, err := resolver.ResolvePackage(context.Background(), ref, stageDir)
	if err != nil {
		t.Fatalf("expected stale fallback when v2 download fails with 502: %v", err)
	}

	if !art.IsStale {
		t.Errorf("expected IsStale to be true")
	}
	if art.Version != "1.0.0" {
		t.Errorf("expected fallback to keep v1.0.0, got %s", art.Version)
	}
}
