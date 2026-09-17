# 01: Tracer-Bullet End-to-End Single Site De-provisioning Flow

Status: ready-for-agent
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

- [ ] WP-CLI inspection extracts the configured database name from candidate directory or leaves it empty when missing/unparseable.
- [ ] Discovery returns eligible first-level directories, filtering out hidden folders, symlinks, and excluded paths.
- [ ] When no candidate directories exist, outputs message indicating no websites found and returns to main menu without error.
- [ ] Main menu item Delete is enabled and dispatches to the de-provisioning workflow.
- [ ] Confirmation prompt defaults to No and cancels gracefully when declined.
- [ ] De-provisioning unsecures Herd, drops database, and mandatorily deletes the website directory even when prior steps fail.
- [ ] End-to-end integration test exercises deletion flow from main menu through directory deletion.

## Testing seam

- `internal/app/e2e_smoke_test.go`: test end-to-end deletion flow from Main Menu.
- `internal/deprovision/deprovision_test.go`: test discovery and single-site de-provisioning.

## Demo path

Run `wptui`, choose `Delete`, confirm deletion of a test website, and observe that its directory and database are removed and the per-resource summary is displayed.
