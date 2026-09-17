# 0011: Direct Database Administrative Credential Mutation and Site Configuration

## Status
Accepted

## Context
WordPress core and WP-CLI deliberately restrict modifying `user_login` through standard CLI commands (`wp user update` accepts `--user_email` and `--user_pass`, but rejects changing `user_login`). Furthermore, configuring an existing Website post-provisioning requires modifying credentials, running WordPress tweaks, and adding packages without rebuilding the site.

Connecting to MySQL via CLI pipes or string interpolation risks injection vulnerabilities and argument leaks. Using standard prepared statements over a dedicated database connection solves this securely.

## Decision
1. **Isolated Package**: Implement post-provisioning configuration in a dedicated internal package (`internal/siteconfig`), separating it from global application settings (`internal/config`).
2. **Site-First Navigation**: The user selects a single Website first (`internal/deprovision.DiscoverCandidates`), then accesses an action sub-menu (`1. wp_tweaks`, `2. Change admin`, `3. Plugins`, `4. Themes`, `< Back`), maintaining the active website context across multiple configuration actions.
3. **Administrator Selection**: Query users with the `administrator` role and preselect the first administrator. If several administrators exist, the user can select another. A non-administrator is never silently substituted when no administrator exists.
4. **Prepared Statement Mutation**: Extract the actual database name, user, password, host, and table prefix from the chosen Website's `wp-config.php` via WP-CLI. Keep credentials in memory only and redact them from logs and errors. Validate the table prefix as an SQL identifier fragment, then mutate `user_login` and `user_nicename` through `database/sql` with `go-sql-driver/mysql`; only values use prepared-statement parameters because SQL identifiers cannot be bound.
5. **Configured Replacement Defaults**: Blank username, password, and email inputs resolve to the corresponding defaults in `config.json`, as requested for the Website Configuration workflow.
6. **Synchronized Email Updates**: Update both the selected administrator's user email (`wp user update <id> --user_email=...`) and the Website option (`wp option update admin_email ...`).
7. **Safe Theme Activation**: Install selected themes without activation unless the user explicitly confirms activation.

## Consequences
- Implementation requires `github.com/go-sql-driver/mysql`; add it with the implementation rather than to the planning-only change.
- DB credentials and passwords remain process-memory secrets and never appear in command arguments, logs, summaries, or errors.
- Existing Websites retain their active theme unless the user explicitly changes it.
