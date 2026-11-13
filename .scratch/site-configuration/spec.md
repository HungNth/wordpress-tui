# Website Configuration Specification

## Summary
Post-provisioning configuration workflows for existing local WordPress **Websites** managed by `wptui` inside the fullscreen Bubble Tea Master-Detail interface (`Websites Hub`). Users select an existing Website from `websites_path` and run configuration actions with real-time progress in the **In-Pane Progress Monitor**:
1. **Config (`Apply wp_tweaks`)**: Apply `wp_tweaks` from `config.json`.
2. **Change admin credentials**: Update administrator username via direct MySQL prepared statement, password and email via WP-CLI, synchronized with site `admin_email`.
3. **Install Plugins**: Search the Package Catalog and install/upgrade plugins with automatic activation.
4. **Install Theme**: Search the Package Catalog and install/upgrade themes with optional user-confirmed activation.
5. **Toggle Debug Mode**: Inspect `WP_DEBUG` and immediately switch Website Debug Mode between enabled (`WP_DEBUG=true`, `WP_DEBUG_LOG=true`, `WP_DEBUG_DISPLAY=false`) and disabled (`false`, `false`, `false`).
## Requirements

### 1. Website Hub Navigation & Action Dispatch
- Discovers existing Websites under `websites_path` via `deprovision.DiscoverCandidates`.
- Selecting one Website opens `Website Details → Select Action`:
  - `Config` (always available)
  - `Install Plugins` (shown only when plugins exist in the loaded Package Catalog)
  - `Install Theme` (shown only when themes exist in the loaded Package Catalog)
  - `Change admin credentials` (always available)
  - `Toggle Debug Mode` (always available)
- Executing an action locks Sidebar navigation and transitions the Content Pane into the **In-Pane Progress Monitor** (`bubbles/viewport` logs + step progress + outcome report). Cancelling with `q`/`Ctrl+C` signals operation cancellation before returning to Websites Hub.

### 2. Action: Config (`Apply wp_tweaks`)
- Preflights Website directory existence and `wp-config.php` presence before executing tweaks.
- Reuses the existing tweak execution logic (`wpcli.ConfigSet`, `wpcli.RewriteStructure`, `wpcli.OptionUpdate`, `wpcli.LanguageCore`) via `siteconfig.ApplyTweaks`.
- Iterates over all tweaks configured in `config.json`, emits real-time per-tweak progress/logs, continues past individual failures, and renders a completion report (`Website Configured Successfully` or `Configuration Completed with Warnings` with failure counts).

### 3. Action: Change Administrator Credentials
- Preflights Website directory existence and `wp-config.php` presence.
- Discovers administrators via `wp user list --role=administrator --fields=ID,user_login,user_email --format=json`, using the first administrator (`admins[0]`) and failing cleanly if no administrator exists without substituting non-administrators.
- Prompts inside the Websites Hub form for replacement Username, Password (masked), and Email initialized from `config.json` defaults.
- Extracts database credentials from `wp-config.php` through WP-CLI without printing, logging, or returning secrets:
  - `wp config get DB_NAME`
  - `wp config get DB_USER`
  - `wp config get DB_PASSWORD`
  - `wp config get DB_HOST`
  - `wp config get table_prefix --type=variable`
- Validates `table_prefix` as an SQL identifier fragment and executes a prepared statement via `database/sql` + `github.com/go-sql-driver/mysql`:
  `UPDATE {validated_prefix}users SET user_login = ?, user_nicename = ? WHERE ID = ?;`
- Updates password via `wp user update <id> --prompt=user_pass` (stdin), updates email via `wp user update <id> --user_email=<new_email>`, and synchronizes `wp option update admin_email <new_email>`.
- Renders the completion or failure report in the In-Pane Progress Monitor without leaking secrets.

### 4. Actions: Install Plugins / Install Theme (Version-Aware)
- Hidden when the Package Catalog is absent or has no entries for that Package type.
- Reuses `LiveSearchModel` with multi-query selection accumulation inside Websites Hub; theme installation offers optional single-theme activation selection (`None` supported).
- Preflights Website directory and `wp-config.php`, resolves Package archives via `packages.PackageResolver`, and executes `siteconfig.InstallPackages` (checking `wp <type> get <slug> --field=version`, installing or `--force` upgrading only when target version is newer or unparseable, skipping equal/older installed versions).
- Reports per-package results and warnings in the In-Pane Progress Monitor.


### 5. Action: Toggle Debug Mode
- Always listed in single-Website `Website Details → Select Action` (never a batch action or shortcut).
- Preflights Website directory existence and `wp-config.php` presence before mutating.
- Normalizes `WP_DEBUG=true`/`1` as enabled and `false`/`0`/missing as disabled; stops before any write on non-boolean values or real read failures.
- Attempts all three `WP_DEBUG`, `WP_DEBUG_LOG`, `WP_DEBUG_DISPLAY` writes in deterministic order with best-effort continuation and no rollback.
- Supports `q`/`Ctrl+C` cancellation between WP-CLI writes and reports Website, Website Directory, target state, and per-constant failures in the In-Pane Progress Monitor.
## Architectural Boundaries
- Business logic isolated in `internal/siteconfig`.
- Websites Hub views, forms, and In-Pane Progress Monitor in `internal/tui`.
- Bubble Tea action wiring and asynchronous executors in `internal/app/app_executors.go`.
- WP-CLI and database operations in `internal/wpcli` and `internal/siteconfig`.

## Testing Seams
- Unit tests with mock runners for WP-CLI and database execution in `internal/siteconfig/siteconfig_test.go`.
- Websites Hub and progress monitor tests in `internal/tui`.
- Executor preflight, progress, and wiring tests in `internal/app`.
