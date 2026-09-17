# 0011: Direct Database Administrative Credential Mutation and Site Configuration

## Status
Accepted

## Context
WordPress core and WP-CLI deliberately restrict modifying `user_login` through standard CLI commands (`wp user update` accepts `--user_email` and `--user_pass`, but rejects changing `user_login`). Furthermore, configuring an existing Website post-provisioning requires modifying credentials, running WordPress tweaks, and adding packages without rebuilding the site.

Connecting to MySQL via CLI pipes or string interpolation risks injection vulnerabilities and argument leaks. Using standard prepared statements over a dedicated database connection solves this securely.

## Decision
1. **Isolated Package**: Implement post-provisioning configuration in a dedicated internal package (`internal/siteconfig`), separating it from global application settings (`internal/config`).
2. **Site-First Navigation**: The user selects a single Website first (`internal/deprovision.DiscoverCandidates`), then accesses an action sub-menu (`1. wp_tweaks`, `2. Change admin`, `3. Plugins`, `4. Themes`, `< Back`), maintaining the active website context across multiple configuration actions.
3. **Prepared Statement Mutation**: When reconciling administrative credentials, extract the actual database name, user, password, and host directly from the chosen Website's `wp-config.php` via WP-CLI (`wp config get ...`), and mutate `user_login` and `user_nicename` directly via `database/sql` using `go-sql-driver/mysql` prepared statements (`UPDATE {prefix}users SET user_login = ?, user_nicename = ? WHERE ID = ?`).
4. **Synchronized Email Updates**: When the administrator's email is updated, update both the admin user record (`wp user update <id> --user_email=...`) and the site option (`wp option update admin_email ...`).
5. **Safe Theme Activation**: When installing themes in configuration mode, do not activate immediately unless explicitly requested by the user, preserving the running site's active appearance.

## Consequences
- Requires `github.com/go-sql-driver/mysql` dependency for type-safe, injection-proof database updates.
- Protects existing websites from inadvertent visual breaks or credential desynchronization.
