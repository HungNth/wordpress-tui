package packages_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wptui/internal/packages"
)

func TestCache_StoreAndExactHit(t *testing.T) {
	tempCache := t.TempDir()
	zipData := createValidZipBytes()

	cache, err := packages.NewCache(tempCache)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	ref := packages.PackageRef{Type: "plugin", Slug: "sample-plugin"}
	entry, found, err := cache.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("expected cache miss initially")
	}

	// Write temp file to stage
	tmpFile := filepath.Join(tempCache, "stage.zip")
	if err := os.WriteFile(tmpFile, zipData, 0600); err != nil {
		t.Fatal(err)
	}

	// Put into cache
	storedEntry, err := cache.Put(context.Background(), ref, "1.0.0", tmpFile)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if storedEntry.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", storedEntry.Version)
	}

	// Read back from cache
	entry, found, err = cache.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get after Put failed: %v", err)
	}
	if !found {
		t.Fatalf("expected cache hit")
	}
	if entry.Version != "1.0.0" || entry.Size != int64(len(zipData)) {
		t.Errorf("entry mismatch: %+v", entry)
	}

	// Verify file is within cache
	absPath := filepath.Join(tempCache, entry.FilePath)
	if _, err := os.Stat(absPath); err != nil {
		t.Errorf("cached file does not exist at %s", absPath)
	}
}

func TestCache_PathTraversalProtectionDoesNotDeleteVictim(t *testing.T) {
	tempParent := t.TempDir()
	victimFile := filepath.Join(tempParent, "victim.txt")
	if err := os.WriteFile(victimFile, []byte("precious data"), 0600); err != nil {
		t.Fatal(err)
	}

	cacheDir := filepath.Join(tempParent, "cache")
	cache, err := packages.NewCache(cacheDir)
	if err != nil {
		t.Fatal(err)
	}

	// Inject a malicious entry pointing outside cache root
	manifestPath := filepath.Join(cacheDir, "data.json")
	maliciousManifest := packages.Manifest{
		SchemaVersion: 1,
		Packages: []packages.CacheEntry{
			{
				Type:     "plugin",
				Slug:     "evil-plugin",
				Version:  "1.0.0",
				FilePath: "../../victim.txt",
				Size:     13,
				SHA256:   "fake_hash",
			},
		},
	}
	data, _ := json.Marshal(maliciousManifest)
	_ = os.WriteFile(manifestPath, data, 0600)

	// Attempt to Get the malicious entry
	ref := packages.PackageRef{Type: "plugin", Slug: "evil-plugin"}
	_, found, err := cache.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if found {
		t.Errorf("expected malicious entry to not be found")
	}

	// Critical check: victimFile must NOT have been deleted!
	if _, err := os.Stat(victimFile); os.IsNotExist(err) {
		t.Fatalf("CRITICAL SECURITY BUG: victim file was deleted during cache cleanup!")
	}
}

func TestCache_CorruptManifestQuarantine(t *testing.T) {
	tempCache := t.TempDir()
	manifestPath := filepath.Join(tempCache, "data.json")

	if err := os.WriteFile(manifestPath, []byte("broken json"), 0600); err != nil {
		t.Fatal(err)
	}

	cache, err := packages.NewCache(tempCache)
	if err != nil {
		t.Fatalf("NewCache should recover from corrupt manifest: %v", err)
	}

	ref := packages.PackageRef{Type: "plugin", Slug: "any-plugin"}
	_, found, err := cache.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("Get should succeed on new empty manifest: %v", err)
	}
	if found {
		t.Errorf("expected cache miss on recovered empty manifest")
	}

	// Verify quarantine file exists
	entries, _ := os.ReadDir(tempCache)
	foundQuarantine := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "data.json.corrupt-") {
			foundQuarantine = true
			break
		}
	}
	if !foundQuarantine {
		t.Errorf("expected corrupt manifest to be renamed to quarantine file")
	}
}

func TestCache_AtomicVersionReplacementDeletesOld(t *testing.T) {
	tempCache := t.TempDir()
	zipData1 := createValidZipBytes()
	zipData2 := createValidZipBytes()

	cache, err := packages.NewCache(tempCache)
	if err != nil {
		t.Fatal(err)
	}

	ref := packages.PackageRef{Type: "plugin", Slug: "upgrading-plugin"}

	stage1 := filepath.Join(tempCache, "v1.zip")
	_ = os.WriteFile(stage1, zipData1, 0600)
	entry1, err := cache.Put(context.Background(), ref, "1.0.0", stage1)
	if err != nil {
		t.Fatal(err)
	}
	oldFile := filepath.Join(tempCache, entry1.FilePath)
	if _, err := os.Stat(oldFile); err != nil {
		t.Fatalf("v1 file does not exist: %v", err)
	}

	// Upgrade to v2.0.0
	stage2 := filepath.Join(tempCache, "v2.zip")
	_ = os.WriteFile(stage2, zipData2, 0600)
	entry2, err := cache.Put(context.Background(), ref, "2.0.0", stage2)
	if err != nil {
		t.Fatal(err)
	}

	newFile := filepath.Join(tempCache, entry2.FilePath)
	if _, err := os.Stat(newFile); err != nil {
		t.Fatalf("v2 file does not exist: %v", err)
	}

	// Old file should be deleted
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("expected old version file %s to be deleted after atomic replacement", oldFile)
	}
}

func TestCache_ConcurrentLockSafety(t *testing.T) {
	tempCache := t.TempDir()
	zipData := createValidZipBytes()

	cache1, _ := packages.NewCache(tempCache)
	cache2, _ := packages.NewCache(tempCache)

	ref1 := packages.PackageRef{Type: "plugin", Slug: "p1"}
	ref2 := packages.PackageRef{Type: "plugin", Slug: "p2"}

	stage1 := filepath.Join(tempCache, "s1.zip")
	stage2 := filepath.Join(tempCache, "s2.zip")
	_ = os.WriteFile(stage1, zipData, 0600)
	_ = os.WriteFile(stage2, zipData, 0600)

	done := make(chan bool)
	go func() {
		_, _ = cache1.Put(context.Background(), ref1, "1.0.0", stage1)
		done <- true
	}()
	go func() {
		_, _ = cache2.Put(context.Background(), ref2, "1.0.0", stage2)
		done <- true
	}()

	<-done
	<-done

	// Verify both entries exist in manifest
	finalCache, _ := packages.NewCache(tempCache)
	_, found1, _ := finalCache.Get(context.Background(), ref1)
	_, found2, _ := finalCache.Get(context.Background(), ref2)

	if !found1 || !found2 {
		t.Errorf("expected both concurrent puts to be recorded: p1=%v, p2=%v", found1, found2)
	}
}
