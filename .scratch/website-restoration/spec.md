# Specification: Dual-Engine Website Restoration (Full ZIP and AI1WM)

Status: ready-for-agent

## Problem Statement

Users who create backups using WordPress TUI or possess existing WordPress site archives currently have no automated mechanism to restore or migrate them into a working local development environment. Today, the restore option in the main menu is marked as coming soon and is disabled. Manually extracting archives, creating databases, configuring site parameters, importing SQL dumps, updating domain URLs via search-replace, resetting administrative credentials, and configuring local Herd hostnames is tedious, error-prone, and risks leaving broken or partially configured websites on disk.

## Solution

Enable the restore action in the WordPress TUI main menu, delivering a complete, interactive Website Restoration workflow that supports two distinct restoration engines:

1. **Full ZIP Restore Engine**: Unpacks a zip archive into temporary staging, locates the unique WordPress Root, relocates files to the target website directory, creates a clean configuration file with the source site's accurately discovered table prefix (failing fast if undetermined), imports the identified database dump, runs an automated search-replace to update old URLs to the new local development domain, updates administrator credentials to user-supplied values, configures Herd TLS, and removes the selected database dump file strictly upon full success.
2. **AI1WM Restore Engine**: Provisions a clean foundation site using cached WordPress core files, verifies and activates the All-in-One WP Migration unlimited extension, stages the migration archive into the plugin backup location, executes the plugin restoration command, cleans up the staged migration copy to preserve disk space, updates administrator credentials to user-supplied values strictly after database replacement, and configures Herd TLS.

Both strategies enforce strict fail-fast validation against existing directory or database collisions, provide an intuitive file picker with a custom path entry option, and enforce atomic rollback on critical failures while warning gracefully on Herd TLS errors.

## User Stories

1. As a WordPress developer, I want to select "Restore" from the main menu, so that I can recreate a functional WordPress website from an existing archive.
2. As a WordPress developer, I want to choose between "Full ZIP Restore" and "AI1WM Restore", so that I can restore websites regardless of the backup format used.
3. As a developer browsing backups, I want to see a list of backup archives currently available in my configured backup location, so that I can quickly select recent backups without typing file paths.
4. As a developer with backups in arbitrary folders, I want to see a custom path option at the top of the archive picker, so that I can provide a path to any archive file on disk, with automatic rejection of missing, unreadable, non-regular, or unsupported files.
5. As a developer restoring a website, I want to input a Website Name and have it automatically converted to a standardized Website Slug, so that directory paths and local hostnames remain consistent.
6. As a developer, I want the restoration wizard to validate that the target Website Slug does not collide with an existing folder or database, so that I never accidentally overwrite pre-existing sites.
7. As a developer, I want optional administrator username, password, and email input fields that fall back to default configuration values when left blank, so that I can access the restored site with known credentials.
8. As a developer restoring from a ZIP archive, I want the system to unpack the archive in an isolated temporary location outside the target web root, so that failed extractions do not pollute my project folders.
9. As a developer restoring from a ZIP archive, I want the system to locate the unique WordPress Root directory containing core installation markers, so that nested archive folder hierarchies are automatically resolved.
10. As a developer restoring from an ambiguous ZIP archive, I want the system to abort with an explicit error if zero or multiple WordPress Root candidates are detected, so that I do not accidentally restore staging subdirectories or invalid archives.
11. As a developer restoring a standard WPTUI ZIP backup, I want the system to deterministically identify the primary database dump matching the source slug, so that restoration is completely automated without manual prompts.
12. As a developer restoring an external ZIP archive with multiple SQL files, I want to interactively select which SQL file represents the primary database dump, so that complex archives can be restored correctly.
13. As a developer restoring an external ZIP archive without any SQL files, I want the system to abort with a clear error message, so that I am informed of missing database assets before files are moved.
14. As a developer restoring from a ZIP archive, I want the original table prefix to be extracted from the archive configuration or SQL dump and used when generating a fresh configuration, so that imported database tables remain properly bound.
15. As a developer restoring from a ZIP archive whose table prefix cannot be determined, I want the system to fail fast with an explicit error, so that I am not left with a site pointing to non-existent database tables.
16. As a developer restoring from a ZIP archive, I want a fresh site configuration generated with current configuration database credentials, so that obsolete database hosts, users, or salts from the backup source are replaced.
17. As a developer restoring from a ZIP archive, I want the selected database dump to be imported cleanly, so that all posts, terms, and options are restored.
18. As a developer restoring from a ZIP archive, I want the prior site URL to be automatically retrieved from the imported database and replaced with the new development domain, followed by administrative credential synchronization, so that site links and admin credentials work immediately.
19. As a developer restoring from a ZIP archive, I want the temporary database dump file to be removed strictly upon successful completion of the entire restore sequence, so that manual recovery is not hindered if an intermediate step fails.
20. As a developer restoring from an AI1WM migration file, I want the system to provision a minimal foundation site using cached WordPress core files, so that migration CLI commands have a valid WordPress installation to execute against.
21. As a developer restoring from an AI1WM archive, I want the system to ensure the All-in-One WP Migration unlimited extension is installed and activated on the foundation site, so that the restoration CLI command is available.
22. As a developer restoring from an AI1WM archive, I want the migration archive copied into the plugin backup location and restored via plugin CLI, so that native plugin restoration restores all content and database tables.
23. As a developer restoring from an AI1WM archive, I want the staged copy inside the plugin backup location removed after a successful restore, so that duplicate copies do not consume unnecessary disk space.
24. As a developer restoring an AI1WM archive, I want administrator credentials updated strictly after the plugin restore finishes, so that user credentials overwrite the accounts imported from the archive.
25. As a developer using Laravel Herd, I want the restored website domain to be automatically secured with TLS, so that local HTTPS works out of the box.
26. As a developer using Laravel Herd, I want any error during TLS securing to be reported as a warning while allowing the restored website to succeed, so that minor certificate issues do not destroy a fully restored site.
27. As a developer encountering a critical restore failure, I want the system to roll back by dropping the created database and deleting the created website directory, so that partial or corrupted installations do not remain.
28. As a developer experiencing a failed restore, I want my source backup file to remain completely untouched and preserved, so that I never lose backup data due to a restore failure.
29. As a terminal user, I want clear, real-time progress indicators during extraction, database import, search-replace, and credential synchronization, so that I understand the system's active operations.
30. As a terminal user, I want a high-contrast completion summary displaying the website name, directory path, local URL, database name, and admin username, so that I can immediately access and test the restored site.

## Implementation Decisions

- **Restoration Subsystem Architecture**:
  - Build a dedicated restoration module responsible for orchestrating website reconstitution, root discovery, dump selection, table prefix resolution, and engine dispatch.
  - Expose a high-level restorer component that coordinates file operations, command execution, and administrative credential updates.
  - Enable the restore menu item in the terminal interface, removing the coming soon placeholder.
  - Integrate a dedicated restoration execution flow into the top-level application runner with options injection and error propagation.

- **Archive Selection and Path Validation Contract**:
  - The archive picker scans the configured backup location for regular files with supported extensions (`.zip` and `.wpress`).
  - The custom path entry option strictly validates that the specified path exists, is a regular file (not a directory), is readable, and ends with a supported extension (`.zip` or `.wpress`), returning explicit validation errors on invalid inputs.

- **Full ZIP Execution Sequence**:
  - Reconstitutes the website via an ordered lifecycle: unpack staging → unique root discovery → table prefix resolution → target directory relocation → configuration file generation → database creation and dump import → automated URL search-replace → administrative credential reconciliation → local TLS configuration → strictly successful-only dump file removal.

- **WordPress Root Discovery Contract**:
  - A discovery operation traverses the extracted directory tree in an isolated temporary location.
  - A directory qualifies as a WordPress root if and only if it directly contains the standard core directory and file markers: `wp-content`, `wp-includes`, and `wp-settings.php`.
  - The operation strictly enforces a unique root: finding zero qualifying directories or finding more than one qualifying directory triggers an immediate fail-fast error, preventing ambiguous or nested extractions.

- **Database Dump Resolution Contract**:
  - For archives matching the standard WPTUI full backup naming convention, the dump is deterministically identified using the source slug at the WordPress root.
  - For external archives, the root is scanned for SQL dump files:
    - If zero files exist, the operation halts immediately with a missing dump error.
    - If exactly one file exists, it is automatically selected.
    - If multiple files exist, an interactive selection callback prompts the user to choose the primary dump.

- **Table Prefix Resolution Contract**:
  - The table prefix is extracted by inspecting the archive configuration file for prefix declarations.
  - If unreadable or missing, table creation statements in the database dump are scanned for prefix patterns.
  - If the table prefix cannot be determined from either source, the operation fails fast with an explicit undetermined prefix error, refusing to assume a default prefix.

- **Command Capabilities Extension**:
  - Extend the WordPress CLI execution interface with support for database import, precise domain search-replace across all prefixed tables, and non-interactive AI1WM archive restoration.

- **AI1WM Staging and Execution Flow**:
  - Provision a lightweight foundation site using cached core files, a new database, and basic configuration.
  - Ensure the All-in-One WP Migration unlimited extension is present and active using the version-aware package installer.
  - Stage the migration file into the plugin's required backup storage directory, execute the restore command with confirmation flags, and remove the staged copy upon completion.
  - Reconcile administrative credentials via direct database login updates and CLI credential synchronization strictly after the plugin command completes.
  - Conditionally trigger TLS configuration if local development integration is enabled.

- **Error Handling and Atomic Rollback**:
  - Manage newly created resources using an ownership tracker.
  - Any critical failure before completion drops the created database, removes the created website directory, and cleans temporary staging locations.
  - The original backup artifact is never altered or deleted during normal operation or rollback.
  - A failure during TLS securing is treated as non-fatal: it logs a warning in the completion summary without rolling back the restored website.

## Testing Decisions

- **Good Test Characteristics**:
  - Tests exercise observable behavior through public package APIs and simulated command runners rather than asserting on private state or implementation wiring.
  - Tests verify concrete file transitions: staging cleanup, destination file relocation, configuration generation, and dump file removal.
  - Tests verify edge cases and failure paths: missing or multiple WordPress roots, missing SQL dumps, undetermined table prefixes, atomic rollback on command errors, and non-fatal TLS warnings.
- **Modules Under Test**:
  - The restoration subsystem:
    - High-level restoration seam: exercises complete Full ZIP and AI1WM execution flows, resource transitions, rollback on command errors, and TLS warning tolerance.
    - Pure domain seams: WordPress root discovery, SQL dump resolution, and table prefix extraction.
  - The CLI execution adapter: verifies parameter construction and error propagation for database import, search-replace, and AI1WM restore operations.
  - The terminal presentation layer: verifies archive selection listing order (custom path prompt first, followed by scanned backup files) and form validation.
  - The application orchestration layer: integration smoke test verifying restoration command dispatch and options propagation.
- **Prior Art in Codebase**:
  - Backup test suites: fixtures using temporary directories, mock command runners, and error injection.
  - Creation test suites: atomic rollback assertions and resource verification.
  - Site configuration test suites: administrative credential reconciliation tests.

## Out of Scope

- In-place restore or overwriting existing websites.
- Downloading backup archives directly from remote cloud storage URLs.
- Converting between backup formats.
- Restoring partial website components such as themes or database tables in isolation.

## Further Notes

- Adheres to ADR 0014 regarding backup naming and storage standards.
- Adheres to ADR 0015 regarding dual-engine restoration, non-colliding site creation, and staged reconstitution.
- Reuses visual theme styling and color-coded summary conventions from ADR 0010.
