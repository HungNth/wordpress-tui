# Website Configuration Specification

## Summary
Interactive post-provisioning configuration workflow for existing WordPress websites managed by `wptui`. Located under the Main Menu option `config` ("Configure an existing website"), this feature allows users to select an existing website from `websites_path` and perform one or more configuration actions in a site-first loop:
1. Apply `wp_tweaks` from `config.json`.
2. Change administrator information (username via direct MySQL prepared statement, password & email via WP-CLI, synchronized with site `admin_email`).
3. Install plugins with catalog defaults, live search, and automatic activation.
4. Install themes with catalog defaults, live search, and optional user-confirmed activation.

## Requirements

### 1. Site-First Navigation & Sub-Menu Loop
- Invoked via Main Menu option `config` in `internal/app/app.go`.
- Scans `websites_path` using existing candidate discovery (`deprovision.DiscoverCandidates`).
- Allows selecting exactly **1 website** to configure, or choosing `< Back` / Esc to return to Main Menu.
- Once a website is selected, enters an interactive configuration sub-menu for that site:
  - `1. Apply wp_tweaks`
  - `2. Change admin info`
  - `3. Install plugins`
  - `4. Install themes`
  - `< Back` (returns to website selection).
- After an action completes, displays a color-coded execution summary. The user can perform another action on the same website, choose `< Back` to pick another website, or return to the main menu.

### 2. Action 1: Apply `wp_tweaks`
- Reuses the existing tweak execution logic (`wpcli.ConfigSet`, `wpcli.RewriteStructure`, `wpcli.OptionUpdate`, `wpcli.LanguageCore`).
- Shows real-time progress for each tweak being executed (e.g. `[1/15] Applying WP_DEBUG...`).
- Iterates over all tweaks specified in `config.json`.
- Displays status for each applied tweak in color (green: success, yellow/red: skipped/failed).
### 3. Action 2: Change Administrator Information
- Identifies the administrator user:
  - Runs `wp user list --role=administrator --fields=ID,user_login,user_email --format=json`.
  - If multiple administrators exist, displays them for selection with the first administrator preselected.
  - If no administrator exists, reports an error and returns to the Website Configuration menu; it does not silently select the first non-administrator user.
- Displays the selected administrator's ID, current username, and current email.
- Prompts for replacement values using defaults from `config.json`:
  - **New Username**: blank input uses `default_admin_username`.
  - **New Password**: blank input uses `default_admin_password`.
  - **New Email**: blank input uses `default_admin_email`.
- Execution steps:
  - Shows real-time progress for credential extraction and updates.
  - Extracts database credentials from the selected Website's `wp-config.php` through WP-CLI. Secrets may be held in memory for the database connection but MUST NOT be printed, logged, or included in errors:
    - `wp config get DB_NAME`
    - `wp config get DB_USER`
    - `wp config get DB_PASSWORD`
    - `wp config get DB_HOST`
    - `wp config get table_prefix --type=variable`
  - If **Username** changed:
    - Validates the table prefix before using it as an SQL identifier.
    - Connects through Go `database/sql` using `github.com/go-sql-driver/mysql`.
    - Executes the prepared statement:
      `UPDATE {validated_prefix}users SET user_login = ?, user_nicename = ? WHERE ID = ?;`
  - Updates the password with `wp user update <id> --prompt=user_pass`, passing the password via stdin.
  - Updates the email with `wp user update <id> --user_email=<new_email>`.
  - Synchronizes the Website's administrative email with `wp option update admin_email <new_email>`.
- Displays a color-coded execution status without secrets.

### 4. Action 3 & 4: Install Plugins / Themes (Version-Aware)
- Automatically fetches package catalog from API so that `Type to search plugin catalog` is always available at top of picker.
- Reuses `tui.SelectPackagesFlow` with `[x]` multi-query accumulator.
- Resolves package artifacts via `packages.PackageResolver`.
- For each package, shows active progress:
  - Queries installed version using `wp plugin get <slug> --field=version` (or `theme`).
  - If not installed: executes `wp <type> install <pathOrSlug> [--activate]`.
  - If already installed:
    - Compares installed version with target version:
      - `target > installed`: executes `wp <type> install <pathOrSlug> --force [--activate]`.
      - `target == installed`: skips installation, reporting `Already up to date (v...)`.
      - `target < installed`: skips installation, warning that current version is newer.
      - Unknown version format: defaults to `--force`.
- Displays complete color-coded installation summary.
## Architectural Boundaries
- Business logic isolated in `internal/siteconfig`.
- Form inputs and TUI views in `internal/tui`.
- WP-CLI and database operations in `internal/wpcli` and `internal/siteconfig`.
- Integrated into `internal/app/app.go` under action `config`.

## Testing Seams
- Unit tests with mock runners for WP-CLI and database execution in `internal/siteconfig/siteconfig_test.go`.
- Form validation and flow tests in `internal/tui`.
- End-to-end flow test in `internal/app/app_test.go` or `internal/app/config_flow_test.go`.
