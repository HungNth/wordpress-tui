# 02: End-to-End AI1WM Migration Restoration

**Parent specification:** `.scratch/website-restoration/spec.md`

**What to build:**
Deliver the complete, interactive All-in-One WP Migration (`.wpress`) Website Restoration capability through the terminal interface. Extend the restoration strategy selection to offer "AI1WM Restore", update the archive picker to discover and validate `.wpress` files, and wire the dedicated AI1WM restoration engine into the application workflow. The engine provisions a minimal foundation site using cached WordPress core files and database creation, verifies and activates the `all-in-one-wp-migration-unlimited-extension`, copies the `.wpress` file into the plugin backup location (`wp-content/ai1wm-backups/`), executes `wp ai1wm restore <filename> --yes`, removes the staged `.wpress` copy to preserve disk space, updates administrative credentials strictly after the database is replaced, and configures Herd TLS. Ensure atomic rollback cleans up temporary directories, foundation sites, and databases on failure while preserving the source `.wpress` file. Conclude with unified progress feedback and a color-coded completion summary.

**Testing seam:**
- Primary high-level seam: `Restorer.Restore(ctx, params)` configured with Strategy `StrategyAI1WM` using mock CLI execution, asserting minimal foundation provisioning, extension installation, archive staging into `wp-content/ai1wm-backups/`, plugin restore invocation, staged archive removal, post-restore credential update sequencing, and atomic rollback on simulated CLI failures.

**Demo path:**
1. Launch `wptui` and select `Restore` from the main menu.
2. Select strategy `AI1WM Restore`.
3. Select an existing backup `.wpress` file or choose `[Enter custom path...]` to provide an external `.wpress` archive.
4. Enter Website Name, confirm normalized slug, and enter optional admin credentials.
5. Observe progress steps for minimal foundation provisioning, AI1WM extension verification, archive staging, non-interactive restore execution, staged copy removal, and admin credential update.
6. Verify the color-coded completion summary with site directory, local URL, database name, and admin username.
7. Verify that the website is fully functional, administrative credentials match user input, and no duplicate `.wpress` file remains in `wp-content/ai1wm-backups/`.

**Blocked by:** 01: End-to-End Full ZIP Website Restoration.

**Status:** resolved

## Acceptance criteria

- [x] Restoration strategy selection presents "AI1WM Restore" alongside "Full ZIP Restore".
- [x] Archive picker lists valid `.wpress` files from the configured backup location with `[Enter custom path...]` as the first item.
- [x] Custom path input validates that the file exists, is a regular file, is readable, and ends with `.wpress`, showing clear errors on invalid inputs.
- [x] Reuses website name, slug normalization, non-collision checks, and administrator credential collection established in Ticket 01.
- [x] CLI execution interface provides the non-interactive restoration command (`wp ai1wm restore <filename> --yes`).
- [x] Provisions a minimal WordPress foundation site using cached core files, a new database, and basic configuration without executing full theme/plugin provisioning.
- [x] Verifies and activates `all-in-one-wp-migration-unlimited-extension` on the foundation site prior to executing restore commands.
- [x] Staging copies the `.wpress` file into `wp-content/ai1wm-backups/` inside the target website directory.
- [x] Executes the AI1WM restore command non-interactively, restoring site content and database tables.
- [x] The staged `.wpress` copy inside `wp-content/ai1wm-backups/` is strictly removed upon completion to avoid duplicate disk usage.
- [x] Administrator credentials (user login via direct database update, password and email via CLI) are synchronized strictly after the AI1WM restore command completes, ensuring imported accounts are reconciled.
- [x] Herd TLS is configured if enabled, treating any TLS failure as a non-fatal warning without rolling back the restored website.
- [x] Any critical failure during foundation provisioning, extension installation, archive staging, command execution, or credential reconciliation triggers atomic rollback (dropping the created database, deleting the target directory, and cleaning temporary staging) while preserving the source `.wpress` file.
- [x] Progress spinners and step descriptions display real-time feedback during AI1WM execution, concluding with the unified color-coded summary report.
