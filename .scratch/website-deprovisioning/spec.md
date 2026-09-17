# Website De-provisioning (Delete Website) Specification

Status: ready-for-agent

## Problem Statement

When local WordPress developers need to decommission local websites, they currently must perform multiple manual steps:
1. Manually determine which database corresponds to the website directory.
2. Manually drop the MySQL database.
3. Manually run Herd CLI to unsecure TLS certificates for `.test` domains.
4. Manually delete the website directory on disk.

Performing these steps manually is error-prone. Guessing database names based on folder names can destroy unrelated databases if the user customized their `DB_NAME`. In addition, failing to cleanly remove Herd TLS certificates leaves orphaned reverse proxy configurations. Users need a reliable, safe, and automated way within WPTUI to discover existing websites, select one or more targets, preview what will be removed, and concurrently de-provision them with clear safety guardrails.

## Solution

WPTUI introduces an interactive `Delete` workflow (Option 3 in the Main Menu) that:
1. Discovers candidate directories located inside `websites_path` using pure filesystem inspection (`os.ReadDir`). If `websites_path` does not exist or yields zero eligible directories, WPTUI outputs `No websites found in <websites_path>` and returns immediately to the Main Menu without entering selection or confirmation.
2. Filters out hidden directories (`.*`), symlinks and irregular entries identified via `DirEntry.Type()`, and excluded paths specified by `delete_excludes` (defaulting to `["backups"]`).
3. Presents an interactive multi-select picker allowing the user to select one or multiple targets for removal instantly without pre-fetching database names.
4. Accurately extracts the configured database name (`DB_NAME`) via WP-CLI inspection lazily, only for the selected websites, immediately prior to displaying the confirmation table.
5. Displays a high-visibility summary table detailing the selected directory name, exact directory path, and detected database name, followed by an explicit `huh.Confirm` dialog defaulting to `No`.
6. Concurrently de-provisions selected websites using a bounded worker pool (`min(4, count)` goroutines):
   - Unsecures Herd TLS (`herd unsecure <slug>`) if `used_herd = true` on a best-effort basis.
   - Drops the database (`wp db drop --yes`) only when accurately identified.
   - Mandatorily deletes the directory (`os.RemoveAll(siteDir)`), even if prior Herd or database steps encountered errors.
7. Produces a clear, per-resource structured summary table reporting individual outcomes for Herd, Database, and Directory across all selected websites.

## User Stories

1. As a local WordPress developer, I want to access the Delete option directly from the WPTUI main menu, so that I can decommission websites without leaving the terminal tool.
2. As a local WordPress developer, I want WPTUI to scan my `websites_path` directory, so that I don't have to manually type directory paths.
3. As a local WordPress developer, I want WPTUI to exclude hidden directories (`.git`, `.idea`, `.vscode`) from the deletion list, so that internal tool directories are never deleted by accident.
4. As a local WordPress developer, I want WPTUI to ignore symlinks and Windows junctions in `websites_path`, so that deleting a link never recursively deletes files outside my website directory.
5. As a local WordPress developer, I want WPTUI to exclude the `backups` directory by default via `delete_excludes`, so that my archive storage remains protected.
6. As a local WordPress developer, I want to customize `delete_excludes` in `config.json`, so that I can protect custom directories (e.g. shared assets or templates).
7. As a local WordPress developer, I want legacy `config.json` files missing `delete_excludes` to automatically receive `["backups"]` on load and persist to disk, so that my existing setup remains secure without manual edits.
8. As a local WordPress developer, I want an explicitly empty `delete_excludes: []` in my `config.json` to be preserved, so that WPTUI does not re-add `backups` if I deliberately chose to allow deleting it.
9. As a local WordPress developer, I want candidate directories without WordPress or with an empty directory to remain selectable, so that I can clean up aborted or corrupted site folders.
10. As a local WordPress developer, I want WPTUI to accurately inspect `wp-config.php` using WP-CLI to find the real `DB_NAME`, so that databases with custom names are correctly identified.
11. As a local WordPress developer, I want WPTUI to mark the database as `unknown — not deleted` when `wp-config.php` is missing or unparseable, so that WPTUI never guesses a database name from a folder slug.
12. As a local WordPress developer, I want to select multiple websites using an interactive multi-select checklist, so that I can clean up multiple test sites in a single batch.
13. As a local WordPress developer, I want a warning if no websites exist in `websites_path`, returning me to the main menu without throwing an error.
14. As a local WordPress developer, I want a clear preview table before deletion showing folder names, paths, and detected database names, so that I know exactly what is about to be destroyed.
15. As a local WordPress developer, I want an explicit confirmation prompt that defaults to `No`, so that accidental keystrokes do not delete my sites.
16. As a local WordPress developer, I want multi-site deletions to run concurrently using a bounded worker pool of up to 4 workers, so that batch deletions complete quickly without overwhelming disk I/O or MySQL connections.
17. As a local WordPress developer, I want Herd TLS certificates to be removed via `herd unsecure` when `used_herd` is enabled, so that my local development proxy remains tidy.
18. As a local WordPress developer, I want a failure in `herd unsecure` to be logged as a warning without blocking database or directory deletion, so that stale certs don't block cleanup.
19. As a local WordPress developer, I want the website database to be dropped cleanly using `wp db drop --yes`, so that MySQL storage is reclaimed.
20. As a local WordPress developer, I want a failure in dropping the database to be reported clearly without preventing the website directory from being deleted, fulfilling my instruction to delete the folder regardless.
21. As a local WordPress developer, I want website directory deletion (`os.RemoveAll`) to be mandatory and execute even if prior steps failed, so that leftover broken folders are guaranteed to be cleaned up.
22. As a local WordPress developer, I want errors on one website to remain isolated from other selected websites, so that one failure does not halt the rest of the batch.
23. As a local WordPress developer, I want a structured summary report upon completion detailing the exact outcome for Herd, Database, and Directory per website, so that I have full visibility into what succeeded and what failed.

## Implementation Decisions

1. **Architecture and Package Seams**:
   - `internal/deprovision`: Houses domain logic for website discovery, candidate inspection, and the de-provisioning execution engine.
     - `DiscoverCandidates(ctx context.Context, websitesPath string, deleteExcludes []string) ([]Candidate, error)`: Scans directory instantly via filesystem inspection, filtering exclusions and symlinks/junctions without invoking slow WP-CLI processes.
     - `ResolveCandidateDB(ctx context.Context, c *Candidate, client WPClient)`: Lazily queries `wp config get DB_NAME` only for candidate websites selected by the user, immediately prior to confirmation preview.
     - `Deprovision(ctx context.Context, candidates []Candidate, client WPClient, opts DeprovisionOptions) []Result`: Coordinates bounded worker pool (`min(4, len(candidates))`) executing Herd unsecure, database drop, and directory removal.
   - `internal/tui` (interactive UI):
     - `menu.go`: Enable `delete` menu option (un-disable item 3 in Main Menu).
     - `delete_wizard.go`: Interactive forms using Huh for selecting candidate websites (displaying clean slug labels), lazily previewing details with the summary warning table, and running the confirmation prompt.
   - `internal/app`:
     - Wire `DeleteFn` dependency into `app.App` and execute `internal/deprovision` when option `delete` is selected in `RunWithContext`.

2. **Accurate Lazy Database Resolution via WP-CLI**:
   - In `internal/wpcli/wpcli.go`, `ConfigGet(ctx context.Context, dir string, key string) (string, error)` invokes `wp config get <key>` with the target site directory as its working directory.
   - Use `ResolveCandidateDB` to inspect candidate databases only after the user submits their selection from the multi-select list.
   - If `wp-config.php` does not exist in the candidate folder or `ConfigGet` fails, `DetectedDB` is left empty (`""`).
   - Database deletion calls existing `Client.DBDrop(ctx, dir)`. If `DetectedDB` is empty, database drop is skipped and marked `unknown/skipped`.

3. **Symlink and Irregular Entry Protection**:
   - During `DiscoverCandidates`, check `entry.Type()&os.ModeSymlink != 0 || entry.Type()&os.ModeIrregular != 0` directly from `os.ReadDir` entries to skip symlinks and recognized irregular reparse points.
4. **Bounded Concurrency Engine**:
   - The worker pool is bounded by a semaphore channel `chan struct{}` of capacity `min(4, len(selected))`.
   - Results are collected into an indexed slice corresponding to the original candidate selection order.

5. **Mandatory Directory Removal**:
   - The lifecycle per website inside the worker is:
     1. Unsecure Herd: `cli.HerdUnsecure(ctx, dir, slug)` (only if `used_herd = true`).
     2. Drop DB: re-verify `DB_NAME` via `cli.ConfigGet` against the confirmed preview value before dropping; if changed or unreadable, record an error and skip `cli.DBDrop(ctx, dir)` to prevent accidental data loss. Otherwise, execute `cli.DBDrop(ctx, dir)`.
     3. Remove directory: `os.RemoveAll(dir)` — executed regardless of errors in steps 1 and 2.
   - Each step records its own status (`unsecured`, `not applicable`, `failed: <err>`, `deleted`, `skipped`, etc.) in `deprovision.Result`.

## Testing Decisions

Testing focuses on the **highest-level seam possible** with minimal internal coupling:

1. **Primary Highest-Level Seam (`internal/app` / `app.App`)**:
   - Wire an injectable `DeleteFlowDependencies` into `app.App` (matching the existing precedent of `CreateFlowDependencies`).
   - Tests exercise the complete user workflow end-to-end:
     - Empty discovery: verifies `No websites found in <websites_path>` and graceful return to Main Menu.
     - Main menu dispatch (`delete`).
     - Website discovery and exclusion filtering (`delete_excludes`, hidden directories, symlinks/junctions).
     - WP-CLI database name inspection (`wp config get DB_NAME`).
     - User prompt interaction (scripted multi-select and confirmation dialog).
     - Bounded concurrent de-provisioning execution (Herd unsecure, database drop, and mandatory directory removal).
     - Structured summary report output.
   - Dependencies adapter: Mock `wpcli.Runner` (existing in `internal/wpcli`) capturing CLI commands and simulated filesystem directories in `t.TempDir()`.

2. **Secondary Core Seam (`internal/deprovision`)**:
   - Unit-level tests for edge cases: concurrent cancellation via context, corrupt/locked directories, and non-WordPress/empty folder cleanup.

3. **Already Implemented Prerequisite**:
   - `delete_excludes` configuration schema, normalization, and automatic migration for absent or `null` values are already implemented and tested in `internal/config` (commit `d8ea747`). All remaining tickets focus strictly on the interactive deletion flow.
## Out of Scope

- Deleting remote websites or websites hosted outside `websites_path`.
- Creating automated backups before deletion (backups will be handled under the dedicated `backup` option).
- Re-securing or restoring websites once deleted (restoration will be handled under `restore`).
- Interactive custom database name input during deletion.

## Further Notes

- The configuration migration for `delete_excludes` is already completed, tested, and persisted in `internal/config`.
- ADR `0008-configured-database-identity-during-deprovisioning.md` serves as the authoritative architectural record for this specification.
