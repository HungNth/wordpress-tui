# 01: Tracer-Bullet End-to-End Single Site De-provisioning Flow

Status: resolved
Blocked by: None (can start immediately)
Parent spec: ../spec.md

## What to build

Implement the complete end-to-end single-website deletion vertical slice:
1. Inspect candidate website directory to extract configured database name accurately without guessing from slug.
2. Discover eligible first-level directories in the websites storage location, filtering out hidden folders, symlinks/junctions, and excluded paths.
3. If no websites exist, report that no websites were found and return to the main menu without error.
4. Present an interactive confirmation prompt defaulting to No before destructive actions.
5. Execute de-provisioning: unsecure Herd TLS (when Herd is enabled, best-effort), drop the database (when accurately identified), and mandatorily delete the website directory even if previous resource steps fail.
6. Display a per-resource summary of outcomes upon completion.

## Acceptance criteria

- [x] WP-CLI inspection extracts the configured database name from candidate directory or leaves it empty when missing/unparseable.
- [x] Discovery returns eligible first-level directories, filtering out hidden folders, symlinks, and excluded paths.
- [x] When no candidate directories exist, outputs message indicating no websites found and returns to main menu without error.
- [x] Main menu item Delete is enabled and dispatches to the de-provisioning workflow.
- [x] Confirmation prompt defaults to No and cancels gracefully when declined.
- [x] De-provisioning unsecures Herd, drops database, and mandatorily deletes the website directory even when prior steps fail.
- [x] End-to-end integration test exercises deletion flow from main menu through directory deletion.

## Testing seam

- `internal/app/e2e_smoke_test.go`: test end-to-end deletion flow from Main Menu.
- `internal/deprovision/deprovision_test.go`: test discovery and single-site de-provisioning.

## Demo path

Run `wptui`, choose `Delete`, confirm deletion of a test website, and observe that its directory and database are removed and the per-resource summary is displayed.

## Answer

Implemented end-to-end single-website de-provisioning:
1. Added `wpcli.Client.ConfigGet` executing `wp config get <key>` to accurately extract `DB_NAME`.
2. Created `internal/deprovision` with `DiscoverCandidates` filtering hidden directories, symlinks/junctions, and `delete_excludes`.
3. Added `DeprovisionSingle` with mandatory directory removal even if Herd unsecure or DB drop fails.
4. Enabled `delete` in Main Menu and wired `RunDeleteFlowWithDeps` in `internal/app/app.go`.
5. Verified with unit tests in `internal/wpcli`, `internal/deprovision`, `internal/tui`, and `internal/app/app_test.go`.
