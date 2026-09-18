# 0014: Dual-Strategy Website Backup and Storage Layout

## Status
Accepted

## Context
Users need reliable, self-contained backups of their local WordPress websites for disaster recovery and cross-environment migration. Two primary strategies are required:
1. **Full Source Code & Database Archive**: Direct file archiving including an SQL database export, suitable for manual inspection and raw restoration.
2. **All-in-One WP Migration (AI1WM) Package**: A standardized `.wpress` archive created via WP-CLI with exclusion filtering, designed for seamless plugin-based restoration across differing operating systems.

Key challenges include:
- SQL database exports using raw host paths or CRLF line endings can introduce portability issues when imported across Windows and macOS environments.
- Temporary database dumps left inside public web roots pose serious data security risks.
- Archiving source trees recursively without strict exclusion boundaries risks nested backup bloat and infinite loops if the backup directory is located within the website path.

## Decision
1. **Configuration Property `backup_path`**:
   - Add `backup_path` to `config.json`.
   - During the first-run setup wizard, prompt the user for `backup_path`, defaulting to `filepath.Join(inputs.WebsitesPath, "backups")` if left blank.
2. **Standardized Backup Destination and Naming**:
   - All completed backup artifacts are relocated to `backup_path`.
   - Filenames strictly follow the pattern:
     - Strategy 1 (Full): `full_<website-slug>_YYYY-MM-DD_HH-mm-ss.zip`
     - Strategy 2 (AI1WM): `ai1wm_<website-slug>_YYYY-MM-DD_HH-mm-ss.wpress`
3. **Strategy 1: Full Source & Portable Database Dump**:
   - Dump the database using WP-CLI `wp db export <slug>.sql` directly in the website directory.
   - Create the initial `.zip` archive inside an isolated temporary directory outside `siteDir` (`os.MkdirTemp("", "wptui-full-backup-*")`), completely avoiding self-inclusion bugs if `backup_path` resides within `siteDir`.
   - When constructing the `.zip` archive, normalize all file entry paths as clean relative paths using forward slashes (`/`), removing any Windows backslashes or drive prefixes to guarantee seamless extraction portability across Windows and macOS.
   - Strictly exclude entries matching `config.BackupExcludes` and any prior backup archives (`*.zip`, `*.wpress`).
   - Atomically move the completed `.zip` archive from the temporary directory into `backup_path` under `full_<website-slug>_YYYY-MM-DD_HH-mm-ss.zip`.
   - **Strict Cleanup Invariant**: Delete the temporary `<slug>.sql` file **strictly after both archive creation AND relocation to `backup_path` succeed**. If zip creation, closing, or moving fails for any reason, retain `<slug>.sql` on disk so data is preserved for retry and manual recovery.
4. **Strategy 2: All-in-One WP Migration Unlimited Integration**:
   - Leverage the existing version-aware package installer to verify and install `all-in-one-wp-migration-unlimited-extension` (which automatically bundles/activates the core migration plugin).
   - Execute `wp ai1wm backup --exclude-cache --exclude-files=<excludes>`, where `<excludes>` is a comma-separated list formatted from `config.BackupExcludes`.
   - Parse the WP-CLI output line `Backup location: <path>` to identify the generated `.wpress` file, move it to `backup_path`, rename it to `ai1wm_<website-slug>_YYYY-MM-DD_HH-mm-ss.wpress`, and retain the plugin on the website for future operations.

## Consequences
- Guarantees clean web roots without abandoned SQL dump files.
- Prevents infinite recursion and nested backup bloat.
- Provides two distinct, dependable backup mechanisms tailored for developer flexibility and cross-platform WordPress migration.
