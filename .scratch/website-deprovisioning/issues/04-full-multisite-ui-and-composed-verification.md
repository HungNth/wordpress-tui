# 04: Full Multi-Site Interactive UI and Composed Verification

Status: ready-for-agent
Blocked by: 02-interactive-selection-and-safety-confirmation-dialog.md, 03-bounded-concurrent-deprovisioning-engine.md
Parent spec: ../spec.md

## What to build

Compose the full multi-site interactive UI and concurrent engine into the main application workflow:
1. Connect candidate selection, preview, and confirmation dialogs into the main application menu dispatch.
2. Execute multi-site deletions through the concurrent engine.
3. Format and output a clear per-resource final summary report for all processed websites.
4. Verify full workflow integration under race-detector testing.

## Acceptance criteria

- [ ] End-to-end multi-site deletion workflow runs smoothly from the main menu.
- [ ] Concurrency remains bounded to max 4 workers without data races.
- [ ] Partial failures on one site are isolated and displayed accurately in the final summary report.
- [ ] Full test suite passes under race detector without warnings.
## Testing seam

- `internal/app/e2e_smoke_test.go`: composed multi-site interactive de-provisioning smoke test with race detector.

## Demo path

Run `wptui`, choose `Delete`, select 3 websites, confirm deletion, observe concurrent progress, and verify that all 3 directories and databases are removed with a detailed final report.
