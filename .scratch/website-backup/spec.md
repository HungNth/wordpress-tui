# Website Backup Specification

## Summary
Interactive backup workflow accessible from the WPTUI Main Menu under the `backup` option ("Backup website"). Allows users to select an existing website from `websites_path` and choose between two distinct backup strategies:
1. **Full Source Code & Database Archive**: Exports the database to `<slug>.sql`, packages the entire source tree into a `.zip` archive while respecting `backup_excludes`, moves the archive to `backup_path` under `full_<website-slug>_YYYY-MM-DD_HH-mm-ss.zip`, and cleans up the temporary database dump.
2. **All-in-One WP Migration (AI1WM)**: Ensures `all-in-one-wp-migration-unlimited-extension` is installed and up-to-date, executes `wp ai1wm backup --exclude-cache --exclude-files=...`, relocates the resulting `.wpress` artifact to `backup_path` under `ai1wm_<website-slug>_YYYY-MM-DD_HH-mm-ss.wpress`, and leaves the plugin intact on the site.

## Requirements

### 1. Configuration & Setup Wizard (`config.json`)
- Add field `backup_path` (string) to `config.Config`:
  ```json
  "backup_path": "F:\\laravel-herd\\wordpress\\backups"
  ```
- In `internal/tui/wizard.go`:
  - Prompt user for `backup_path` during the first-run wizard.
  - If left blank, automatically populate default: `filepath.Join(inputs.WebsitesPath, "backups")`.
- In `internal/config/config.go`:
  - Validate `backup_path` if present.

### 2. Main Menu Enablement & Navigation
- In `internal/tui/menu.go`:
  - Enable `backup` option: `{Key: "backup", Title: "Backup", Description: "Backup an existing WordPress website", Disabled: false}`.
  - Update coming-soon footer description to: `"Select an available action below.\nComing soon: Restore"`.
- In `internal/app/app.go`:
  - Route `case "backup":` to `a.backupFn(ctx, a.config)`.
- Interactive selection:
  - List candidate websites using `deprovision.DiscoverCandidates`.
  - Prompt user to select exactly 1 website (or choose `← Back to Main Menu` / Esc).
  - Once selected, prompt user to choose backup strategy:
    1. `1. Full source code & database (.zip)`
    2. `2. All-in-One WP Migration (.wpress)`
    3. `← Back`

### 3. Strategy 1: Full Source & Database Backup
- Execution steps:
  1. Print progress: `[1/4] Exporting database to <slug>.sql...`
     - Run `wp db export <slug>.sql` in website directory.
  2. Print progress: `[2/4] Creating zip archive in temporary workspace...`
     - Create `.zip` archive inside an isolated temporary folder outside `siteDir` (`os.MkdirTemp("", "wptui-backup-*")`), ensuring the archive never includes itself.
     - Normalize all zip entry paths to clean relative paths using forward slashes (`/`), stripping Windows backslashes and drive letters for cross-platform portability.
     - Exclude:
       - Patterns matching `config.BackupExcludes`.
       - Files matching `*.zip`, `*.wpress`.
       - The `backup_path` directory itself if located inside the website tree.
  3. Print progress: `[3/4] Moving backup archive to storage...`
     - Ensure `backup_path` exists (`os.MkdirAll`).
     - Relocate the temporary zip archive to `backup_path` under `full_<website-slug>_YYYY-MM-DD_HH-mm-ss.zip`, using `os.Rename` on the same volume and a copy-to-temp-then-rename fallback across volumes.
  4. Print progress: `[4/4] Cleaning up temporary database dump...`
     - **Strict Failure Invariant**: Delete temporary `<slug>.sql` file from the website directory **strictly after step 3 succeeds**. If zip creation, archive finalization, relocation, or dump deletion fails for any reason, retain `<slug>.sql` on disk and return an explicit error.
  5. Print summary:
     - File path, size in MB/GB, elapsed time in seconds.
### 4. Strategy 2: All-in-One WP Migration Backup
- Execution steps:
  1. Print progress: `[1/3] Verifying All-in-One WP Migration Unlimited extension...`
     - Check and install `all-in-one-wp-migration-unlimited-extension` using existing version-aware resolver (`siteconfig.InstallPackages`).
  2. Print progress: `[2/3] Generating AI1WM backup archive...`
     - Format excludes: comma-separated string from `config.BackupExcludes` (e.g. `--exclude-files=.idea,.vscode,node_modules,__MACOSX`).
     - Run WP-CLI command:
       `wp ai1wm backup --exclude-cache --exclude-files=<excludes>`
     - Parse stdout output to locate `Backup location: <source_wpress_path>`.
  3. Print progress: `[3/3] Relocating backup archive to storage...`
     - Ensure `backup_path` exists.
     - Move source `.wpress` file to `filepath.Join(backup_path, "ai1wm_<website-slug>_YYYY-MM-DD_HH-mm-ss.wpress")`.
  4. Print summary:
     - File path, size, elapsed time. Plugin remains installed on the site.

## Architectural Boundaries
- Backup business logic isolated in a dedicated internal package: `internal/backup`.
- Interactive TUI forms and progress reporting in `internal/tui/backup_wizard.go`.
- Orchestration and flow dispatch in `internal/app/backup_flow.go`.

## Testing Seams
- Unit tests for Strategy 1 zip archiving, excludes filtering, and DB cleanup in `internal/backup/full_backup_test.go`.
- Unit tests for Strategy 2 command formatting and output parsing in `internal/backup/ai1wm_backup_test.go`.
- Unit tests for TUI backup forms in `internal/tui/backup_test.go`.
- Orchestrator integration test in `internal/app/backup_flow_test.go`.
