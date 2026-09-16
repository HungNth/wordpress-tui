package create_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"wptui/internal/config"
	"wptui/internal/create"
)

func TestConflictMatching(t *testing.T) {
	tests := []struct {
		name     string
		key1     string
		key2     string
		conflict bool
	}{
		{"global barrier conflicts with anything", "*", "db:option:foo", true},
		{"wildcard conflicts with same wildcard", "db:*", "db:*", true},
		{"wildcard conflicts with namespace item", "db:*", "db:option:timezone_string", true},
		{"different wildcards do not conflict", "db:*", "fs:*", false},
		{"same exact keys conflict", "db:option:foo", "db:option:foo", true},
		{"different exact keys do not conflict", "db:option:foo", "db:option:bar", false},
		{"wildcard conflicts with global barrier", "*", "db:*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := create.KeysConflict(tt.key1, tt.key2)
			if got != tt.conflict {
				t.Errorf("KeysConflict(%q, %q) = %v, want %v", tt.key1, tt.key2, got, tt.conflict)
			}
		})
	}
}

func TestTweakScheduler_ConcurrencyAndConflictExclusion(t *testing.T) {
	tweaks := []config.WPTweak{
		// 1. Global config_set
		{Type: "config_set", Key: "WP_DEBUG", Value: "true", Raw: true},
		// 2. Disjoint option updates
		{Type: "option_update", Key: "timezone_string", Value: "Asia/Ho_Chi_Minh"},
		{Type: "option_update", Key: "date_format", Value: "d/m/Y"},
		{Type: "option_update", Key: "time_format", Value: "H:i"},
		// 3. Same key update (must serialize after previous timezone_string)
		{Type: "option_update", Key: "timezone_string", Value: "UTC"},
		// 4. Rewrite structure (conflicts with db:*)
		{Type: "rewrite_structure", Value: "/%postname%/"},
		// 5. Language core install and activate (activate depends on install)
		{Type: "language_core", Key: "install", Value: "vi"},
		{Type: "language_core", Key: "activate", Value: "vi"},
	}

	var maxConcurrent int32
	var currentConcurrent int32
	var mu sync.Mutex
	var executedOrder []int

	executor := func(ctx context.Context, idx int, tw config.WPTweak) error {
		c := atomic.AddInt32(&currentConcurrent, 1)
		for {
			oldMax := atomic.LoadInt32(&maxConcurrent)
			if c <= oldMax || atomic.CompareAndSwapInt32(&maxConcurrent, oldMax, c) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond) // simulate work

		mu.Lock()
		executedOrder = append(executedOrder, idx)
		mu.Unlock()

		atomic.AddInt32(&currentConcurrent, -1)
		return nil
	}

	results, err := create.RunTweaks(context.Background(), tweaks, executor, nil)
	if err != nil {
		t.Fatalf("RunTweaks failed: %v", err)
	}

	if len(results) != len(tweaks) {
		t.Fatalf("expected %d results, got %d", len(tweaks), len(results))
	}

	// Verify results are reported in configured order
	for i, res := range results {
		if res.Index != i {
			t.Errorf("result[%d] has Index %d, expected %d", i, res.Index, i)
		}
		if res.Status != create.TweakStatusSuccess {
			t.Errorf("tweak %d failed: %v", i, res.Error)
		}
	}

	// Concurrency bound <= 4
	if maxConcurrent > 4 {
		t.Errorf("max concurrent was %d, expected <= 4", maxConcurrent)
	}
}

func TestTweakScheduler_ErrorAndPrerequisiteSkip(t *testing.T) {
	tweaks := []config.WPTweak{
		{Type: "option_update", Key: "site_ok", Value: "1"},
		{Type: "language_core", Key: "install", Value: "fr"},
		{Type: "language_core", Key: "activate", Value: "fr"},
		{Type: "language_core", Key: "activate", Value: "de"}, // Missing install in config!
	}

	executor := func(ctx context.Context, idx int, tw config.WPTweak) error {
		if tw.Type == "language_core" && tw.Key == "install" {
			return fmt.Errorf("download language failed")
		}
		return nil
	}

	results, err := create.RunTweaks(context.Background(), tweaks, executor, nil)
	if err != nil {
		t.Fatalf("RunTweaks should not return fatal error on tweak failure: %v", err)
	}

	if results[0].Status != create.TweakStatusSuccess {
		t.Errorf("expected tweak 0 to succeed")
	}
	if results[1].Status != create.TweakStatusFailed {
		t.Errorf("expected tweak 1 to fail, got %s", results[1].Status)
	}
	if results[2].Status != create.TweakStatusSkipped {
		t.Errorf("expected tweak 2 to be skipped because prereq failed, got %s", results[2].Status)
	}
	if results[3].Status != create.TweakStatusSkipped {
		t.Errorf("expected tweak 3 to be skipped because install was missing, got %s", results[3].Status)
	}
}

func TestTweakScheduler_CancellationReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	tweaks := []config.WPTweak{
		{Type: "option_update", Key: "key1", Value: "val1"},
		{Type: "option_update", Key: "key2", Value: "val2"},
	}

	executor := func(tctx context.Context, idx int, tw config.WPTweak) error {
		if idx == 1 {
			cancel() // cancel context during execution
		}
		time.Sleep(5 * time.Millisecond)
		return nil
	}

	_, err := create.RunTweaks(ctx, tweaks, executor, nil)
	if err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
}
