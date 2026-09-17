package deprovision_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"wptui/internal/deprovision"
)

type mockWPClient struct {
	mu             sync.Mutex
	configGetFn    func(ctx context.Context, dir, key string) (string, error)
	dbDropFn       func(ctx context.Context, dir string) error
	herdUnsecureFn func(ctx context.Context, dir, slug string) error
}

func (m *mockWPClient) ConfigGet(ctx context.Context, dir, key string) (string, error) {
	m.mu.Lock()
	fn := m.configGetFn
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, dir, key)
	}
	return "", nil
}

func (m *mockWPClient) DBDrop(ctx context.Context, dir string) error {
	m.mu.Lock()
	fn := m.dbDropFn
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, dir)
	}
	return nil
}

func (m *mockWPClient) HerdUnsecure(ctx context.Context, dir, slug string) error {
	m.mu.Lock()
	fn := m.herdUnsecureFn
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, dir, slug)
	}
	return nil
}
func TestDiscoverCandidates(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	// 1. Regular WP site with wp-config.php
	siteA := filepath.Join(tempDir, "site-a")
	if err := os.MkdirAll(siteA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(siteA, "wp-config.php"), []byte("<?php"), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Empty directory (no wp-config.php)
	siteB := filepath.Join(tempDir, "empty-site")
	if err := os.MkdirAll(siteB, 0755); err != nil {
		t.Fatal(err)
	}

	// 3. Hidden directory (.git)
	hiddenDir := filepath.Join(tempDir, ".git")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 4. Excluded directory (backups)
	backupsDir := filepath.Join(tempDir, "backups")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		t.Fatal(err)
	}

	candidates, err := deprovision.DiscoverCandidates(ctx, tempDir, []string{"backups"})
	if err != nil {
		t.Fatalf("DiscoverCandidates failed: %v", err)
	}

	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates (site-a, empty-site), got %d: %+v", len(candidates), candidates)
	}

	for _, c := range candidates {
		if c.Slug == "site-a" {
			if !c.HasWPConfig || c.DetectedDB != "" {
				t.Errorf("site-a expected HasWPConfig=true and initial DetectedDB='', got %+v", c)
			}
		} else if c.Slug == "empty-site" {
			if c.HasWPConfig || c.DetectedDB != "" {
				t.Errorf("empty-site expected HasWPConfig=false, DetectedDB='', got %+v", c)
			}
		} else {
			t.Errorf("unexpected candidate: %s", c.Slug)
		}
	}

	// Test lazy resolution
	client := &mockWPClient{
		configGetFn: func(ctx context.Context, dir, key string) (string, error) {
			if strings.Contains(dir, "site-a") && key == "DB_NAME" {
				return "site_a_db", nil
			}
			return "", errors.New("not found")
		},
	}
	for i := range candidates {
		if candidates[i].Slug == "site-a" {
			deprovision.ResolveCandidateDB(ctx, &candidates[i], client)
			if candidates[i].DetectedDB != "site_a_db" {
				t.Errorf("expected lazy resolved DB 'site_a_db', got %q", candidates[i].DetectedDB)
			}
		}
	}
}

func TestDeprovisionSingle_MandatoryDirectoryRemovalOnFailure(t *testing.T) {
	tempDir := t.TempDir()
	siteDir := filepath.Join(tempDir, "fail-test-site")
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		t.Fatal(err)
	}

	candidate := deprovision.Candidate{
		Slug:        "fail-test-site",
		Path:        siteDir,
		DetectedDB:  "fail_db",
		HasWPConfig: true,
	}

	// Simulate Herd and DB failure while DB_NAME still matches
	var dbDropCalled bool
	client := &mockWPClient{
		configGetFn: func(ctx context.Context, dir, key string) (string, error) {
			return "fail_db", nil
		},
		herdUnsecureFn: func(ctx context.Context, dir, slug string) error {
			return errors.New("simulated herd unsecure failure")
		},
		dbDropFn: func(ctx context.Context, dir string) error {
			dbDropCalled = true
			return errors.New("simulated db drop failure")
		},
	}
	res := deprovision.DeprovisionSingle(context.Background(), candidate, client, true)

	if !dbDropCalled {
		t.Errorf("expected dbDrop to be called when DB_NAME matches")
	}
	// Herd and DB should have errors
	if res.HerdErr == nil {
		t.Errorf("expected HerdErr to be set")
	}
	if res.DBErr == nil {
		t.Errorf("expected DBErr to be set")
	}

	// Directory removal MUST succeed regardless
	if res.DirErr != nil {
		t.Errorf("expected DirErr to be nil, got: %v", res.DirErr)
	}
	if !res.DirDone {
		t.Errorf("expected DirDone to be true")
	}

	// Verify directory is deleted from filesystem
	if _, err := os.Stat(siteDir); !os.IsNotExist(err) {
		t.Errorf("expected directory %s to be deleted, but it still exists", siteDir)
	}
}

func TestDeprovisionSingle_DBMismatchSkipsDrop(t *testing.T) {
	tempDir := t.TempDir()
	siteDir := filepath.Join(tempDir, "mismatch-site")
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		t.Fatal(err)
	}

	candidate := deprovision.Candidate{
		Slug:        "mismatch-site",
		Path:        siteDir,
		DetectedDB:  "previewed_db",
		HasWPConfig: true,
	}

	var dbDropCalled bool
	client := &mockWPClient{
		configGetFn: func(ctx context.Context, dir, key string) (string, error) {
			// Return changed database name
			return "different_live_db", nil
		},
		dbDropFn: func(ctx context.Context, dir string) error {
			dbDropCalled = true
			return nil
		},
	}

	res := deprovision.DeprovisionSingle(context.Background(), candidate, client, false)

	if dbDropCalled {
		t.Errorf("expected DBDrop to NOT be called when DB_NAME mismatches")
	}
	if res.DBErr == nil {
		t.Errorf("expected DBErr explaining DB name mismatch, got nil")
	}
	if !res.DirDone {
		t.Errorf("expected directory removal to succeed despite DB mismatch")
	}
	if _, err := os.Stat(siteDir); !os.IsNotExist(err) {
		t.Errorf("expected directory %s to be deleted", siteDir)
	}
}

func TestDiscoverCandidates_IgnoresSymlinks(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := t.TempDir()

	// Create symlink pointing outside
	symlinkPath := filepath.Join(tempDir, "symlink-site")
	if err := os.Symlink(targetDir, symlinkPath); err != nil {
		t.Skipf("skipping symlink test on platform without symlink support: %v", err)
	}
	candidates, err := deprovision.DiscoverCandidates(context.Background(), tempDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Errorf("expected symlink to be ignored, got %+v", candidates)
	}
}

func TestDeprovision_ConcurrentBounded(t *testing.T) {
	tempDir := t.TempDir()
	const totalSites = 8

	var candidates []deprovision.Candidate
	for i := 1; i <= totalSites; i++ {
		dirName := filepath.Join(tempDir, filepath.Base(tempDir)+"-site-"+string(rune('0'+i)))
		if err := os.MkdirAll(dirName, 0755); err != nil {
			t.Fatal(err)
		}
		candidates = append(candidates, deprovision.Candidate{
			Slug:        "site-" + string(rune('0'+i)),
			Path:        dirName,
			DetectedDB:  "db-" + string(rune('0'+i)),
			HasWPConfig: true,
		})
	}

	var currentWorkers int32
	var peakWorkers int32

	client := &mockWPClient{
		configGetFn: func(ctx context.Context, dir, key string) (string, error) {
			// Return db-<number> matching the path suffix
			lastChar := dir[len(dir)-1:]
			return "db-" + lastChar, nil
		},
		herdUnsecureFn: func(ctx context.Context, dir, slug string) error {
			cur := atomic.AddInt32(&currentWorkers, 1)
			for {
				peak := atomic.LoadInt32(&peakWorkers)
				if cur <= peak || atomic.CompareAndSwapInt32(&peakWorkers, peak, cur) {
					break
				}
			}
			time.Sleep(15 * time.Millisecond) // simulate work to allow overlap
			atomic.AddInt32(&currentWorkers, -1)

			// Inject partial failure on site-3
			if slug == "site-3" {
				return errors.New("simulated herd error on site-3")
			}
			return nil
		},
		dbDropFn: func(ctx context.Context, dir string) error {
			// Inject partial failure on site-5
			if strings.Contains(dir, "site-5") {
				return errors.New("simulated db error on site-5")
			}
			return nil
		},
	}

	results := deprovision.Deprovision(context.Background(), candidates, client, deprovision.DeprovisionOptions{
		UsedHerd:    true,
		Concurrency: 4,
	})

	if len(results) != totalSites {
		t.Fatalf("expected %d results, got %d", totalSites, len(results))
	}

	peak := atomic.LoadInt32(&peakWorkers)
	if peak > 4 {
		t.Errorf("expected peak concurrent workers <= 4, got %d", peak)
	}
	if peak < 2 {
		t.Errorf("expected concurrency to be exercised (peak >= 2), got %d", peak)
	}

	for i, res := range results {
		// Partial failure verification: site-3 has HerdErr, site-5 has DBErr
		if res.Candidate.Slug == "site-3" {
			if res.HerdErr == nil {
				t.Errorf("site-3 expected HerdErr to be set")
			}
		} else {
			if res.HerdErr != nil {
				t.Errorf("site %s unexpected HerdErr: %v", res.Candidate.Slug, res.HerdErr)
			}
		}

		if res.Candidate.Slug == "site-5" {
			if res.DBErr == nil {
				t.Errorf("site-5 expected DBErr to be set")
			}
		} else {
			if res.DBErr != nil {
				t.Errorf("site %s unexpected DBErr: %v", res.Candidate.Slug, res.DBErr)
			}
		}

		// Directory removal MUST succeed on all sites regardless of partial errors
		if !res.DirDone {
			t.Errorf("site %d directory not marked done: %+v", i+1, res)
		}
		if _, err := os.Stat(res.Candidate.Path); !os.IsNotExist(err) {
			t.Errorf("expected directory %s to be removed from disk", res.Candidate.Path)
		}
	}
}

func TestDeprovision_CancellationAbortsRemaining(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel context

	candidates := []deprovision.Candidate{
		{Slug: "site-cancel-1", Path: "/dummy/1"},
		{Slug: "site-cancel-2", Path: "/dummy/2"},
	}

	results := deprovision.Deprovision(ctx, candidates, nil, deprovision.DeprovisionOptions{})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, res := range results {
		if !errors.Is(res.DirErr, context.Canceled) {
			t.Errorf("expected context.Canceled error, got %v", res.DirErr)
		}
	}
}
