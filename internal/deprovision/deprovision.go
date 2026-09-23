package deprovision

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Candidate represents an eligible first-level directory for de-provisioning.
type Candidate struct {
	Slug        string
	Path        string
	DetectedDB  string
	HasWPConfig bool
}

// Result records the outcome of de-provisioning a single Candidate.
type Result struct {
	Candidate Candidate
	HerdErr   error
	DBErr     error
	DirErr    error
	HerdDone  bool
	DBDone    bool
	DirDone   bool
}

// WPClient represents the interface required by de-provisioning.
type WPClient interface {
	ConfigGet(ctx context.Context, dir, key string) (string, error)
	DBDrop(ctx context.Context, dir string) error
	HerdUnsecure(ctx context.Context, dir, slug string) error
}

// DiscoverCandidates scans websitesPath, filtering out hidden directories,
// symlinks/junctions, and paths listed in deleteExcludes. It does not invoke WP-CLI,
// ensuring instant rendering of the selection menu.
func DiscoverCandidates(ctx context.Context, websitesPath string, deleteExcludes []string) ([]Candidate, error) {
	if _, err := os.Stat(websitesPath); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(websitesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read websites directory: %w", err)
	}

	excludeMap := make(map[string]bool)
	for _, excl := range deleteExcludes {
		excludeMap[strings.ToLower(strings.TrimSpace(excl))] = true
	}

	var candidates []Candidate

	for _, entry := range entries {
		name := entry.Name()
		// 1. Ignore hidden directories
		if strings.HasPrefix(name, ".") {
			continue
		}
		// 2. Ignore configured delete_excludes
		if excludeMap[strings.ToLower(name)] {
			continue
		}

		// 3. Ignore symlinks and irregular entries identified via DirEntry.Type()
		entryType := entry.Type()
		if entryType&os.ModeSymlink != 0 || entryType&os.ModeIrregular != 0 {
			continue
		}

		// 4. Must be a directory
		if !entry.IsDir() {
			continue
		}

		fullPath := filepath.Join(websitesPath, name)
		// Check if wp-config.php exists on disk without calling WP-CLI
		wpConfigPath := filepath.Join(fullPath, "wp-config.php")
		hasWPConfig := false
		if _, err := os.Stat(wpConfigPath); err == nil {
			hasWPConfig = true
		}

		candidates = append(candidates, Candidate{
			Slug:        name,
			Path:        fullPath,
			HasWPConfig: hasWPConfig,
		})
	}

	return candidates, nil
}

// ResolveCandidateDB lazily extracts DB_NAME via WP-CLI for selected candidates.
func ResolveCandidateDB(ctx context.Context, c *Candidate, client WPClient) {
	if !c.HasWPConfig || client == nil {
		return
	}
	dbName, err := client.ConfigGet(ctx, c.Path, "DB_NAME")
	if err == nil && strings.TrimSpace(dbName) != "" {
		c.DetectedDB = strings.TrimSpace(dbName)
	}
}

// DeprovisionOptions configures de-provisioning execution.
type DeprovisionOptions struct {
	UsedHerd    bool
	Concurrency int
	OnProgress  func(r Result)
}

// DeprovisionSingle deletes a single candidate site, executing:
// 1. Herd unsecure (best-effort, if used_herd)
// 2. Database drop (if detectedDB is present)
// 3. Mandatory directory removal (always executed, even if previous steps failed)
func DeprovisionSingle(ctx context.Context, c Candidate, client WPClient, usedHerd bool) Result {
	res := Result{Candidate: c}

	// 1. Herd Unsecure (best effort)
	if usedHerd && client != nil {
		if err := client.HerdUnsecure(ctx, c.Path, c.Slug); err != nil {
			res.HerdErr = err
		} else {
			res.HerdDone = true
		}
	}

	// 2. Drop Database (only if accurately identified and still matches confirmed preview)
	if c.DetectedDB != "" && client != nil {
		currentDB, err := client.ConfigGet(ctx, c.Path, "DB_NAME")
		if err != nil || currentDB != c.DetectedDB {
			res.DBErr = fmt.Errorf("database name changed or unreadable since confirmation (previewed %q, current %q); skipping drop", c.DetectedDB, currentDB)
		} else {
			if err := client.DBDrop(ctx, c.Path); err != nil {
				res.DBErr = err
			} else {
				res.DBDone = true
			}
		}
	}

	// 3. Mandatory Directory Removal (always runs)
	if err := os.RemoveAll(c.Path); err != nil {
		res.DirErr = err
	} else {
		res.DirDone = true
	}

	return res
}

// Deprovision executes de-provisioning across multiple candidates concurrently,
// bounded by opts.Concurrency (capped between 1 and 4, defaulting to min(4, len(candidates))).
func Deprovision(ctx context.Context, candidates []Candidate, client WPClient, opts DeprovisionOptions) []Result {
	n := len(candidates)
	if n == 0 {
		return nil
	}

	limit := opts.Concurrency
	if limit <= 0 {
		limit = 4
	}
	if limit > 4 {
		limit = 4
	}
	if limit > n {
		limit = n
	}

	results := make([]Result, n)
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup

	for i, c := range candidates {
		if err := ctx.Err(); err != nil {
			results[i] = Result{
				Candidate: c,
				DirErr:    err,
			}
			continue
		}

		select {
		case <-ctx.Done():
			// Mark remaining as cancelled via context error
			results[i] = Result{
				Candidate: c,
				DirErr:    ctx.Err(),
			}
			continue
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(idx int, cand Candidate) {
			defer func() {
				<-sem
				wg.Done()
			}()

			res := DeprovisionSingle(ctx, cand, client, opts.UsedHerd)
			results[idx] = res
			if opts.OnProgress != nil {
				opts.OnProgress(res)
			}
		}(i, c)
	}

	wg.Wait()
	return results
}
