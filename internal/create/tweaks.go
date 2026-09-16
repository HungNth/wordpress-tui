package create

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"wptui/internal/config"
)

type TweakStatus string

const (
	TweakStatusSuccess TweakStatus = "success"
	TweakStatusFailed  TweakStatus = "failed"
	TweakStatusSkipped TweakStatus = "skipped"
)

type TweakResult struct {
	Index  int
	Tweak  config.WPTweak
	Status TweakStatus
	Error  string
}

// KeysConflict determines whether two conflict keys overlap according to wildcard and barrier rules.
func KeysConflict(k1, k2 string) bool {
	if k1 == "*" || k2 == "*" {
		return true
	}
	if strings.HasSuffix(k1, ":*") {
		prefix := strings.TrimSuffix(k1, "*")
		if strings.HasPrefix(k2, prefix) {
			return true
		}
	}
	if strings.HasSuffix(k2, ":*") {
		prefix := strings.TrimSuffix(k2, "*")
		if strings.HasPrefix(k1, prefix) {
			return true
		}
	}
	return k1 == k2
}

// HasConflictWithAny checks if a set of keys conflicts with any active key.
func HasConflictWithAny(keys []string, activeKeys []string) bool {
	for _, k := range keys {
		for _, active := range activeKeys {
			if KeysConflict(k, active) {
				return true
			}
		}
	}
	return false
}

// DeriveConflictKeys returns the exact and wildcard resource keys for a tweak using TweakType constants.
func DeriveConflictKeys(tw config.WPTweak) []string {
	switch tw.Type {
	case config.TweakTypeConfigSet:
		return []string{"*"}
	case config.TweakTypeRewriteStructure:
		return []string{"db:*"}
	case config.TweakTypeOptionUpdate:
		return []string{"db:option:" + tw.Key}
	case config.TweakTypeLanguageCore:
		if tw.Key == "install" {
			return []string{"fs:language:" + tw.Value}
		}
		return []string{"db:option:WPLANG"}
	default:
		return []string{"*"}
	}
}

type tweakTask struct {
	index        int
	tweak        config.WPTweak
	conflictKeys []string
	prereqIndex  int
	status       TweakStatus
	errMessage   string
	started      bool
	done         bool
}

type TweakExecutor func(ctx context.Context, index int, tw config.WPTweak) error

// RunTweaks executes tweaks using the conflict-key scheduler with at most 4 concurrent commands.
func RunTweaks(ctx context.Context, tweaks []config.WPTweak, executor TweakExecutor, progress ProgressFunc) ([]TweakResult, error) {
	if len(tweaks) == 0 {
		return nil, nil
	}

	// 2-pass dependency mapping
	langInstallMap := make(map[string]int)
	for i, tw := range tweaks {
		if tw.Type == config.TweakTypeLanguageCore && tw.Key == "install" {
			langInstallMap[tw.Value] = i
		}
	}

	completedCount := 0
	tasks := make([]*tweakTask, len(tweaks))
	for i, tw := range tweaks {
		prereq := -1
		if tw.Type == config.TweakTypeLanguageCore && tw.Key == "activate" {
			if installIdx, ok := langInstallMap[tw.Value]; ok {
				prereq = installIdx
			} else {
				prereq = -2 // Missing prerequisite in config
			}
		}

		t := &tweakTask{
			index:        i,
			tweak:        tw,
			conflictKeys: DeriveConflictKeys(tw),
			prereqIndex:  prereq,
			status:       TweakStatusSuccess,
		}

		if prereq == -2 {
			t.started = true
			t.done = true
			t.status = TweakStatusSkipped
			t.errMessage = fmt.Sprintf("skipped because matching language install for %q was not configured", tw.Value)
			completedCount++
		}

		tasks[i] = t
	}

	tweakCtx, cancelTweaks := context.WithCancel(ctx)
	defer cancelTweaks()

	var wg sync.WaitGroup
	var mu sync.Mutex
	cond := sync.NewCond(&mu)

	activeWorkers := 0
	maxWorkers := 4
	var activeKeys []string
	nextReportIndex := 0

	reportCompletedLocked := func() {
		for nextReportIndex < len(tasks) && tasks[nextReportIndex].done {
			t := tasks[nextReportIndex]
			if progress != nil {
				if t.status == TweakStatusFailed {
					progress("tweak", fmt.Sprintf("Applied tweak: %s %s (failed: %s)", t.tweak.Type, t.tweak.Key, t.errMessage))
				} else if t.status == TweakStatusSkipped {
					progress("tweak", fmt.Sprintf("Applied tweak: %s %s (skipped: %s)", t.tweak.Type, t.tweak.Key, t.errMessage))
				} else {
					progress("tweak", fmt.Sprintf("Applied tweak: %s %s", t.tweak.Type, t.tweak.Key))
				}
			}
			nextReportIndex++
		}
	}

	for {
		select {
		case <-ctx.Done():
			cancelTweaks()
			wg.Wait()
			return nil, ctx.Err()
		default:
		}

		mu.Lock()
		reportCompletedLocked()

		if completedCount == len(tasks) {
			mu.Unlock()
			break
		}

		startedAny := false
		for _, task := range tasks {
			if task.started || task.done {
				continue
			}

			// Check prerequisite
			if task.prereqIndex >= 0 {
				prereqTask := tasks[task.prereqIndex]
				if !prereqTask.done {
					continue // Prereq not finished yet
				}
				if prereqTask.status != TweakStatusSuccess {
					task.started = true
					task.done = true
					task.status = TweakStatusSkipped
					task.errMessage = fmt.Sprintf("skipped because prerequisite task %d (%s) failed", task.prereqIndex, prereqTask.tweak.Key)
					completedCount++
					startedAny = true
					reportCompletedLocked()
					continue
				}
			}

			// Check worker pool and conflict keys
			if activeWorkers < maxWorkers && !HasConflictWithAny(task.conflictKeys, activeKeys) {
				task.started = true
				activeWorkers++
				activeKeys = append(activeKeys, task.conflictKeys...)
				startedAny = true
				wg.Add(1)

				go func(t *tweakTask) {
					defer wg.Done()
					execErr := executor(tweakCtx, t.index, t.tweak)

					mu.Lock()
					defer mu.Unlock()

					t.done = true
					activeWorkers--
					completedCount++

					if execErr != nil {
						t.status = TweakStatusFailed
						t.errMessage = execErr.Error()
					} else {
						t.status = TweakStatusSuccess
					}

					// Remove task's conflict keys from activeKeys
					var remainingKeys []string
					for _, ak := range activeKeys {
						matched := false
						for _, tk := range t.conflictKeys {
							if ak == tk {
								matched = true
								break
							}
						}
						if !matched {
							remainingKeys = append(remainingKeys, ak)
						}
					}
					activeKeys = remainingKeys

					reportCompletedLocked()
					cond.Broadcast()
				}(task)
			}
		}

		if !startedAny && activeWorkers > 0 {
			cond.Wait()
		}
		mu.Unlock()
	}

	wg.Wait()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	results := make([]TweakResult, len(tasks))
	for i, t := range tasks {
		results[i] = TweakResult{
			Index:  t.index,
			Tweak:  t.tweak,
			Status: t.status,
			Error:  t.errMessage,
		}
	}

	return results, nil
}
