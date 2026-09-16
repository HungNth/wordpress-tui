package core_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"wptui/internal/core"
)

func TestResolver_OnlineDownloadAndCache(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := core.NewCache(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	downloadDir := t.TempDir()
	zipPath, expectedSHA, expectedSize := createSampleZip(t, downloadDir, "wordpress-7.1-no-content.zip")
	zipBytes, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}

	var downloadCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/version-check" {
			resp := core.CoreVersionCheckResponse{
				Offers: []core.CoreOffer{
					{
						Response: "upgrade",
						Current:  "7.1",
						Version:  "7.1",
						Packages: core.CorePackages{
							NoContent: "http://" + r.Host + "/download/wordpress-7.1-no-content.zip",
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/download/wordpress-7.1-no-content.zip" {
			atomic.AddInt32(&downloadCalls, 1)
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipBytes)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	resolver := core.NewResolver(cache,
		core.WithVersionCheckURL(server.URL+"/version-check"),
		core.WithHTTPClient(server.Client()),
		core.WithCheckTimeout(1*time.Second),
	)

	archivePath, version, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if version != "7.1" {
		t.Errorf("expected version 7.1, got %s", version)
	}
	if !filepath.IsAbs(archivePath) {
		t.Errorf("expected absolute path, got %s", archivePath)
	}

	fi, err := os.Stat(archivePath)
	if err != nil {
		t.Fatalf("failed to stat downloaded archive: %v", err)
	}
	if fi.Size() != expectedSize {
		t.Errorf("size mismatch: %d != %d", fi.Size(), expectedSize)
	}

	manifest, _, found, err := cache.Get(context.Background())
	if err != nil || !found {
		t.Fatalf("expected cache manifest to exist, found=%v, err=%v", found, err)
	}
	if manifest.SHA256 != expectedSHA {
		t.Errorf("manifest sha256 mismatch: %s != %s", manifest.SHA256, expectedSHA)
	}
	if atomic.LoadInt32(&downloadCalls) != 1 {
		t.Errorf("expected 1 download call, got %d", downloadCalls)
	}
}

func TestResolver_CacheHit(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := core.NewCache(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	srcDir := t.TempDir()
	sampleZip, _, _ := createSampleZip(t, srcDir, "seed.zip")
	if _, err := cache.Put(context.Background(), "7.1", sampleZip); err != nil {
		t.Fatal(err)
	}

	var downloadCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/version-check" {
			resp := core.CoreVersionCheckResponse{
				Offers: []core.CoreOffer{
					{
						Current: "7.1",
						Version: "7.1",
						Packages: core.CorePackages{
							NoContent: "http://" + r.Host + "/download",
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		atomic.AddInt32(&downloadCalls, 1)
		http.NotFound(w, r)
	}))
	defer server.Close()

	resolver := core.NewResolver(cache,
		core.WithVersionCheckURL(server.URL+"/version-check"),
		core.WithHTTPClient(server.Client()),
	)

	archivePath, version, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if version != "7.1" {
		t.Errorf("expected version 7.1, got %s", version)
	}
	if archivePath == "" {
		t.Errorf("expected non-empty archive path")
	}
	if atomic.LoadInt32(&downloadCalls) != 0 {
		t.Errorf("expected 0 download calls on cache hit, got %d", downloadCalls)
	}
}

func TestResolver_OfflineFallback(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := core.NewCache(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	srcDir := t.TempDir()
	sampleZip, _, _ := createSampleZip(t, srcDir, "seed.zip")
	if _, err := cache.Put(context.Background(), "7.0", sampleZip); err != nil {
		t.Fatal(err)
	}

	// Server is unstarted/invalid URL
	resolver := core.NewResolver(cache,
		core.WithVersionCheckURL("http://127.0.0.1:54321/offline"),
		core.WithCheckTimeout(100*time.Millisecond),
	)

	archivePath, version, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("expected offline fallback to succeed, got: %v", err)
	}
	if version != "7.0" {
		t.Errorf("expected fallback version 7.0, got %s", version)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("expected fallback archive on disk: %v", err)
	}
}

func TestResolver_OfflineNoCacheError(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := core.NewCache(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	resolver := core.NewResolver(cache,
		core.WithVersionCheckURL("http://127.0.0.1:54321/offline"),
		core.WithCheckTimeout(100*time.Millisecond),
	)

	_, _, err = resolver.Resolve(context.Background())
	if err == nil {
		t.Fatal("expected error when offline and no cache exists")
	}
	if !strings.Contains(err.Error(), "no cached core archive exists") {
		t.Errorf("expected error to explain missing cache and network requirement, got: %v", err)
	}
}
