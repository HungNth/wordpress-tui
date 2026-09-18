# 0015: Dual-Engine Website Restoration and Staged Reconstitution

- **Status**: proposed
- **Date**: 2026-09-18

## Context

WordPress TUI supports dual-strategy backups (ADR 0014: Full ZIP and All-in-One WP Migration `.wpress`). The `restore` action in the main menu is currently marked coming soon. Reconstituting a WordPress website from an archive introduces critical questions around collision handling, archive discovery, database configuration reconciliation, and credential mutation timing.

## Decision

1. **Non-Colliding Restoration Invariant**:
   - Website Restoration strictly provisions a fresh, non-colliding Website.
   - If the user-supplied Website Slug collides with an existing directory in `websites_path` or database in MySQL, the operation fails fast during preflight validation, prompting for an alternate name. Overwriting existing sites is prohibited.

2. **Archive Discovery & Selection UX**:
   - The TUI presents an interactive file selection list displaying valid backup archives (`.zip` and `.wpress`) found in `backup_path`.
   - The first option at the top of the list is always `[Enter custom path...]`, enabling the user to paste an absolute file path from any disk location.

3. **WordPress Root Discovery in ZIP Archives**:
   - ZIP archives are extracted into an isolated temporary staging directory outside `websites_path`.
   - The staging tree is scanned for the unique directory containing `wp-content/`, `wp-includes/`, and `wp-settings.php`.
   - If zero or more than one candidate directory is discovered, the restore operation fails fast, rejecting ambiguous archive hierarchies.

4. **Database Dump Selection & Successful-Restore Cleanup**:
   - If the archive filename matches `full_<source-slug>_YYYY-MM-DD_HH-mm-ss.zip`, the dump is deterministically identified as `<wordpress-root>/<source-slug>.sql`.
   - For non-standard archive names:
     - Exactly one `.sql` file: automatically selected as the database dump.
     - Multiple `.sql` files: the user is prompted interactively to choose the dump file to import.
     - Zero `.sql` files: the operation fails fast with an explicit missing-database error.
   - Upon successful restore completion, only the designated database dump is removed from the target Website directory; other source `.sql` files are preserved.

5. **Configuration Reconciliation & URL Migration (Full ZIP)**:
   - The `$table_prefix` is parsed from the archive's `wp-config.php` (or SQL dump).
   - A fresh `wp-config.php` is generated using WP-CLI `config create --force --skip-check` with current configuration database parameters and the discovered table prefix.
   - The database dump is imported via `wp db import`.
   - The prior site URL is discovered automatically via `wp option get siteurl` and replaced with the new local hostname via `wp search-replace <old_url> <new_url> --all-tables-with-prefix --precise`.

6. **Post-Restore Administration & Staged Archive Cleanup (AI1WM)**:
   - A clean temporary site foundation is provisioned and `all-in-one-wp-migration-unlimited-extension` is verified/installed.
   - The `.wpress` artifact is staged into `wp-content/ai1wm-backups/` and restored via `wp ai1wm restore <filename> --yes`.
   - The staged `.wpress` copy inside `wp-content/ai1wm-backups/` is strictly deleted upon completion to conserve local disk space, retaining the original backup artifact.
   - Administrative credentials (user login, password, email) are updated strictly **after** `wp ai1wm restore` completes.

7. **Resilience & Rollback Policy**:
   - Any critical failure during extraction, database creation, database import, search-replace, or administrative credential update triggers a full rollback: unsecuring any newly created Herd hostname, dropping the newly created database, removing the newly created Website directory, and cleaning staging artifacts. The source backup artifact is strictly preserved.
   - If `herd secure` fails, the restore process logs a warning and proceeds without rolling back the restored Website.

- Prevents accidental data loss or corruption by enforcing non-colliding target directories and databases.
- Supports both standardized WPTUI backup artifacts and generic third-party WordPress zip archives.
- Preserves predictable disk cleanup guarantees.
