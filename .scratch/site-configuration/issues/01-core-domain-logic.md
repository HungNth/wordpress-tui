# Title: Core domain logic and database/WP-CLI execution in internal/siteconfig
Status: ready-for-agent
Labels: ready-for-agent

## Parent spec
.scratch/site-configuration/spec.md

## Required behavior
Implement business logic in package `internal/siteconfig`:
1. Extract DB credentials directly from target website `wp-config.php` via WP-CLI:
   - `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `table_prefix`.
2. Discover Administrator User:
   - Query `wp user list --role=administrator --fields=ID,user_login,user_email --format=json`.
   - Fallback to first user in `wp user list --fields=ID,user_login,user_email --format=json` if no administrator role found.
3. Administrative Credential Reconciler:
   - If username changed: Open MySQL connection via `database/sql` using `go-sql-driver/mysql`, execute prepared statement:
     `UPDATE {prefix}users SET user_login = ?, user_nicename = ? WHERE ID = ?;`
   - If password changed: Execute `wp user update <id> --prompt=user_pass` passing password via stdin.
   - If email changed: Execute `wp user update <id> --user_email=<new_email>` AND `wp option update admin_email <new_email>`.
4. Apply Tweaks runner:
   - Reuses `wpcli` ConfigSet, RewriteStructure, OptionUpdate, LanguageCore.
5. Install Packages runner:
   - Plugin: `wp plugin install <pathOrSlug> --activate`.
   - Theme: `wp theme install <pathOrSlug>` (add `--activate` only if requested).

## Acceptance criteria
- [ ] Database updates use real prepared statements via `database/sql` (no string formatting/injection).
- [ ] Stdin password injection avoids CLI argument leaks.
- [ ] Email update updates both user record and site-wide `admin_email` option.
- [ ] Isolated unit tests with mock WP-CLI runner and mock/test DB in `internal/siteconfig/siteconfig_test.go`.

## Testing seam
Unit test suite in `internal/siteconfig/siteconfig_test.go` with mock `wpcli.Runner`.
