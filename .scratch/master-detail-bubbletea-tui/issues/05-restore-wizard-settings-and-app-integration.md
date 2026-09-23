# 05: Restore Wizard, Application Settings, and App Integration

**Parent specification:** [Master-Detail Bubble Tea TUI](../spec.md)

**Design contract:** [WPTUI Terminal Design](../../../DESIGN.md)

**What to build:** The Restore Wizard and Application Settings views in the Content Pane, and the end-to-end integration of the Bubble Tea Master-Detail architecture into `internal/app` and `cmd/main.go`. Replaces the obsolete standalone Huh main menu loop. Connects all 5 top-level sidebar routes (`Websites`, `Create`, `Restore`, `Settings`, `Exit`) to their respective operational flows and progress monitors.

**Blocked by:** 01, 02, 03, 04.

**Status:** resolved

## Acceptance criteria

- [x] Selecting `Restore` in Sidebar renders the native Restore Wizard in the Content Pane:
  - Step 1: Format selection (Full ZIP or AI1WM).
  - Step 2: Archive path selection (browsing backup directory or custom path).
  - Step 3: Target parameters (Website Slug, Admin credentials).
  - Submitting launches restoration monitored by the In-Pane Progress Monitor.
- [x] Selecting `Settings` in Sidebar renders Application Settings in the Content Pane:
  - Option to edit `config.json` via configured code editor (`launcher.OpenEditor`).
  - Option to open system cache directory via file manager (`launcher.OpenFolder`).
  - Option to reload configuration without restarting WPTUI.
  - Returns concise status messages upon action completion.
- [x] `internal/app/app.go` and `cmd/main.go` initialize and run the Bubble Tea Master-Detail root model as the primary application interface, removing the obsolete sequential Huh menu loop.
- [x] First-run setup integration: Missing `config.json` launches the setup wizard in alt-screen mode, saving valid configuration before entering the Master-Detail application.
- [x] All 5 sidebar sections (`Websites`, `Create`, `Restore`, `Settings`, `Exit`) function end to end.
- [x] Clean exit on `q` or `Ctrl+C` cleanly tears down alt-screen buffer and returns user to their terminal prompt.
- [x] All existing automated tests and new unit/integration tests pass.

## Testing seam

End-to-end integration tests in `internal/app/app_test.go` and `internal/tui`. Test by launching the integrated App model with mock fixtures, exercising navigation across all 5 sidebar destinations, running restore and settings flows, and asserting clean termination.

## Demo path

1. Launch `wptui`. Verify immediate presentation of the Master-Detail dashboard.
2. Navigate between `Websites`, `Create`, `Restore`, and `Settings` using arrow keys.
3. In `Settings`, select `Open Cache Folder`. Verify external launcher executes and status message appears.
4. In `Restore`, choose format, select an archive, and run restoration with progress feedback.
5. In `Websites`, select a website and run `Backup`, observing real-time stepper and logs.
6. Press `q` from Sidebar and verify clean exit.

## Scope boundary

Delivers the final Restore and Settings content views and wires the complete application, retiring obsolete main menu loops.
