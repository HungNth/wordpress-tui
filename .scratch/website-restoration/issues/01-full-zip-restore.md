# 01: End-to-End Full ZIP Website Restoration

**Parent specification:** `.scratch/website-restoration/spec.md`

**What to build:**
Deliver the complete, interactive Full ZIP Website Restoration capability through the terminal interface. Enable the "Restore" action in the main menu, prompt the user with an archive picker that lists `.zip` files from the backup location with `[Enter custom path...]` at the top, validate that the selected archive is a readable regular zip file, and gather target website name, slug, and administrative credential overrides. Perform fail-fast collision checks against existing directories and databases. During execution, unpack the archive in temporary staging, locate the unique WordPress Root, resolve the primary database dump (matching the standard slug convention or prompting when multiple dumps exist), extract the table prefix (failing fast if undetermined), relocate files to the destination website directory, generate a clean configuration file, import the database dump, execute an automated search-replace to update old URLs to the new local domain, reconcile administrator credentials, configure Herd TLS (logging non-fatal warnings on error), delete the temporary database dump strictly upon full success, and execute atomic rollback on intermediate failures. Conclude with real-time step progress feedback and a color-coded completion summary.

**Testing seam:**
- Primary high-level seam: `Restorer.Restore(ctx, params)` with mock CLI execution, verifying the full end-to-end lifecycle, file transitions, atomic rollback on database/CLI failure, and non-fatal Herd TLS warning handling.
- Domain-pure seams: `FindWordPressRoot` (unique root discovery, zero/multiple candidates fail-fast), `ResolveSQLDump` (standard slug convention, single dump auto-select, multiple dumps interactive prompt, zero dumps fail-fast), and `ExtractTablePrefix` (table prefix extraction from configuration or SQL dump, undetermined prefix fail-fast).

**Demo path:**
1. Launch `wptui` and select `Restore` from the main menu.
2. Select strategy `Full ZIP Restore`.
3. Select an existing backup zip or choose `[Enter custom path...]` to provide an external zip archive.
4. Enter Website Name (e.g. `Restored Site`), confirm normalized slug, and optionally provide or default admin credentials.
5. Observe progress steps for staging, root discovery, configuration generation, database import, URL search-replace, and admin reconciliation.
6. Verify the color-coded completion summary with site directory, local URL, database name, and admin username.
7. Verify that the target website directory is created, database is populated, and temporary SQL dump is cleaned up.

**Blocked by:** None (can start immediately).

**Status:** resolved

## Acceptance criteria

- [x] Main menu displays an enabled "Restore" option and dispatches to the restoration workflow.
- [x] Archive picker lists valid `.zip` files from the configured backup location with `[Enter custom path...]` as the first item.
- [x] Custom path input validates that the file exists, is a regular file, is readable, and ends with `.zip`, showing clear errors on invalid inputs.
- [x] Website Name input automatically normalizes to an ASCII param-case Website Slug.
- [x] Preflight validation performs fail-fast rejection if the Website Slug matches an existing folder in the websites path or an existing database in MySQL.
- [x] Optional administrative username, password, and email inputs fall back to default configuration values when left blank.
- [x] ZIP extraction occurs in an isolated temporary staging location outside the target web root.
- [x] Root discovery strictly identifies the single directory containing `wp-content/`, `wp-includes/`, and `wp-settings.php`, failing fast if zero or multiple candidates are found.
- [x] Database dump resolution selects `<slug>.sql` for standard WPTUI archives, auto-selects if exactly one `.sql` file exists, prompts interactively if multiple `.sql` files exist, and fails fast if zero exist.
- [x] Table prefix is extracted from the archive configuration or SQL dump; if undetermined, the operation fails fast with an explicit error instead of assuming a default.
- [x] CLI execution interface provides database import (`wp db import`) and precise domain search-replace (`wp search-replace`).
- [x] Files are relocated to the destination directory, a fresh configuration file is generated with current database parameters and the discovered table prefix, and the database dump is imported.
- [x] Prior site URL is discovered automatically and replaced with the new `.test` domain across all prefixed tables.
- [x] Administrator credentials are synchronized using direct database login update and CLI credential updates.
- [x] Herd TLS is configured if enabled; any TLS error is captured as a non-fatal warning without aborting or rolling back the restored website.
- [x] The imported database dump file is deleted from the target website directory strictly upon successful completion of the full restore sequence.
- [x] Any critical failure during staging, root discovery, dump resolution, database import, search-replace, or credential update triggers atomic rollback (dropping the created database, deleting the target directory, and cleaning temporary staging) while strictly preserving the source backup file.
- [x] Real-time progress spinners and step descriptions display during execution, concluding with a color-coded summary report displaying site details.
