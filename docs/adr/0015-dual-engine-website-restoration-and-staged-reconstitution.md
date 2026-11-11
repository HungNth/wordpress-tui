# 0015: Dual-Engine Website Restoration and Staged Reconstitution

- **Status**: proposed
- **Date**: 2026-09-18

## Context

The integrated Restore Wizard currently discovers archives only in `backup_path`, despite the need to restore downloaded or third-party archives from other locations. Reconstituting a WordPress Website from an archive also introduces critical questions around collision handling, archive discovery, database configuration reconciliation, and credential mutation timing.

## Decision

1. **Non-Colliding Restoration Invariant**:
   - Website Restoration strictly provisions a fresh, non-colliding Website.
   - If the user-supplied Website Slug collides with an existing directory in `websites_path` or database in MySQL, the operation fails fast during preflight validation, prompting for an alternate name. Overwriting existing sites is prohibited.

2. **Restore Archive Discovery & Selection UX**:
   - After the user chooses Full ZIP or AI1WM, the TUI presents valid Restore Archives found in `backup_path`. The first option is always `[Enter custom path...]`, including when the configured directory is missing, empty, or cannot be scanned; scan failures are reported without blocking custom input.
   - Full ZIP accepts `.zip`; AI1WM accepts `.wpress`. Extension matching is case-insensitive, and compatible third-party filenames do not need to follow WPTUI naming conventions.
   - A custom path must be absolute. Before use, surrounding whitespace and exactly one balanced pair of outer double quotes are removed, then the path is normalized using host path semantics. Unbalanced quotes are invalid.
   - On Enter, both custom paths and listed entries must identify an existing, readable regular file with the extension required by the chosen Restore Engine. Validation errors remain inline and prevent advancement. The same validation runs again before restoration progress starts; a failure returns to archive input while preserving target parameters.
   - Picker validation does not inspect archive contents; the selected Restore Engine retains responsibility for structural validation.
   - A Restore Archive outside `backup_path` is used in place without warning or copying, and the source is preserved. The target-parameter step displays its full normalized path.

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
   - Any critical failure during extraction, database creation, database import, search-replace, administrative credential update, or environment registration/reload (per [ADR-0017](0017-selected-local-development-environment-integration.md)) triggers a full rollback: cleaning newly created environment registrations or certificates (while protecting pre-existing registrations), dropping the newly created database, removing the newly created Website directory, and cleaning staging artifacts. The source backup artifact is strictly preserved.
   - If TLS setup (`secure`) fails during restoration under Herd or Flock, the restore process logs a warning, reconciles the restored database URL to HTTP (`http://<slug>.test`), and proceeds without rolling back the restored Website.

- Prevents accidental data loss or corruption by enforcing non-colliding target directories and databases.
- Supports both standardized WPTUI backup artifacts and generic third-party WordPress zip archives.
- Preserves predictable disk cleanup guarantees.
