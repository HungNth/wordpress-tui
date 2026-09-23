# 03: Concurrent Batch De-provisioning Engine and Channel Wiring

Status: resolved
Blocked by: 01-sidebar-delete-navigation-and-batch-view.md, 02-websites-hub-batch-selection-and-enter-shortcut.md
Parent spec: ../spec.md

## What to build

Implement concurrent execution of candidate database resolution and wire the complete lifecycle into the Bubble Tea runtime:
1. In `internal/app/app_executors.go` (`runDeleteExecution`):
   - Refactor candidate DB resolution to run concurrently across candidates using bounded concurrency (worker pool up to 4 goroutines).
   - Ensure `deprovision.Deprovision` runs concurrently with `opts.Concurrency: 4` across all selected candidates.
   - Stream individual site status and log messages to `ch <- tui.LogLineMsg(...)` as each site completes or encounters errors.
2. In `internal/tui/app_model.go`:
   - When de-provisioning completes in `ProgressMonitor`, refresh candidate inventories in both `m.websitesHub` and `m.deleteModel`, clearing any cached selection sets.
   - When `activeSec == SectionDelete`, return focus to `Delete` list (or sidebar) cleanly upon progress completion.

## Acceptance criteria

- [x] `runDeleteExecution` resolves databases concurrently before de-provisioning.
- [x] Multiple websites are deleted concurrently using bounded concurrency (capped at 4).
- [x] In-Pane Progress Monitor displays real-time log output and final completion status.
- [x] Post-deletion refresh updates candidate lists in both `WebsitesHub` and `DeleteModel`.
- [x] Unit/integration tests in `internal/app/app_executors_test.go` or `internal/deprovision/` verify concurrent execution without race conditions.

## Testing seam

- `internal/app/delete_execution_test.go`: test `runDeleteExecution` and verify concurrent execution of multiple candidates.
- `internal/tui/app_model_test.go`: test post-execution refresh and cleanup for `DeleteModel`.

## Demo path

Select 3 websites, confirm batch delete, observe concurrent progress logs in Progress Monitor, and verify all 3 websites are removed from the list when returning.

## Answer

Implemented concurrent batch de-provisioning engine and lifecycle wiring:
1. Added concurrent candidate database resolution in `internal/app/app_executors.go` (`runDeleteExecution`) with bounded worker pool (up to 4 goroutines).
2. Added `OnProgress` streaming callback to `deprovision.DeprovisionOptions` in `internal/deprovision/deprovision.go` to pipe real-time per-site status into `ch <- tui.LogLineMsg(...)`.
3. Wired post-execution refresh in `internal/tui/app_model.go` to reload candidates and clear selection in both `WebsitesHub` and `DeleteModel`.
4. Verified with automated test suites in `internal/deprovision/deprovision_test.go` and `internal/app/delete_execution_test.go`. All tests pass cleanly.
