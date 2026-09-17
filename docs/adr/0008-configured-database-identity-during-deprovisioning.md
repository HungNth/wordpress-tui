---
status: accepted
---

# Website De-provisioning via Configured Database Identity and Bounded Concurrency

## Context

Users need to delete one or more Websites directly from WPTUI. De-provisioning is a destructive operation touching three distinct resources: Herd TLS certificates, the MySQL database, and the Website filesystem directory.

Assuming database names equal Website Slugs is unsafe because existing installations or custom configs may use arbitrary database names. Dropping databases by guessed names risks catastrophic data loss. Furthermore, users require that selected directories are removed even if prior resource steps fail, while preserving directory exclusions and avoiding unbounded goroutine concurrency.

## Decision

1. **Menu Position**: `Delete` remains option 3 in the Main Menu (positioned after `Create` and `Config`).
2. **Website Discovery and Exclusions**:
   - `delete_excludes` is a distinct field (`DeleteExcludes []string` with JSON tag `delete_excludes`) in `Config`, completely separate from `backup_excludes` and `wp_content_copy_excludes`.
   - `DefaultConfig` initializes `DeleteExcludes` to `[]string{"backups"}`.
   - When loading an existing `config.json` via `config.Load(path)`, only if the `delete_excludes` field is completely absent (`DeleteExcludes == nil`), WPTUI normalizes `DeleteExcludes` to `[]string{"backups"}` and saves the file once (`config.Save(path, cfg)`). If `delete_excludes` is present (including an explicitly empty list `[]`), WPTUI preserves the user's configuration as-is without re-saving.
   - Users can subsequently edit or extend `delete_excludes` directly in `config.json`.
3. **Accurate Database Resolution**:
   - For eligible directories containing `wp-config.php`, query `wp config get DB_NAME` via WP-CLI to discover the exact database name for the confirmation prompt.
   - If `wp-config.php` is missing or `DB_NAME` cannot be determined, the database is marked `unknown — not deleted`. The database is never guessed from the Website Slug.
4. **Safety Confirmation**:
   - Present a clear summary table of selected items (Slug/Directory name, Directory path, Detected DB name) with a prominent warning.
   - Require explicit user confirmation (`huh.Confirm`) defaulting to `No`.
5. **Execution Lifecycle and Bounded Concurrency**:
   - Multi-Website de-provisioning executes via a bounded worker pool of `min(4, count)` goroutines (matching existing repository concurrency defaults).
   - Step lifecycle per Website:
     1. `herd unsecure <slug>` (if `used_herd = true`, best-effort).
     2. `wp db drop --yes` in the Website directory (only if `DB_NAME` was confirmed).
     3. `os.RemoveAll(siteDir)` is mandatory and always executed, regardless of Herd or database failures.
   - A failure on one Website never aborts others.
   - The final report gives distinct, per-resource outcomes for each Website:
     - `Herd`: `unsecured`, `not applicable`, or error detail.
     - `Database`: `deleted`, `unknown/skipped`, or error detail.
     - `Directory`: `deleted` or error detail.
