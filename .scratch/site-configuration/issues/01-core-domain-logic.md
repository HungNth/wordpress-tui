# Title: Core domain logic and database/WP-CLI execution in internal/siteconfig
Status: needs-info
Labels: needs-info

## Parent spec
.scratch/site-configuration/spec.md

## Required behavior
Implement business logic in package `internal/siteconfig`:
1. Extract DB credentials directly from the target Website's `wp-config.php` via WP-CLI:
   - `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_HOST`, and `table_prefix` using `--type=variable` for the prefix.
   - Keep secrets in memory only; redact them from logs and returned errors.
2. Discover Administrator User:
   - Query `wp user list --role=administrator --fields=ID,user_login,user_email --format=json`.
   - Return all administrators for TUI selection with the first administrator preselected.
   - Return a clear error if the Website has no administrator; never substitute a non-administrator user.
3. Administrative Credential Reconciler:
   - Resolve blank username, password, and email inputs from `config.json` defaults before execution.
   - Validate `table_prefix` as a safe SQL identifier fragment.
   - Update username through `database/sql` with `go-sql-driver/mysql` and a prepared statement:
     `UPDATE {validated_prefix}users SET user_login = ?, user_nicename = ? WHERE ID = ?;`
   - Update password through `wp user update <id> --prompt=user_pass`, passing the password via stdin.
   - Update email through `wp user update <id> --user_email=<new_email>` and `wp option update admin_email <new_email>`.
4. Apply Tweaks runner:
   - Reuses `wpcli` ConfigSet, RewriteStructure, OptionUpdate, LanguageCore.
5. Install Packages runner:
   - Plugin: `wp plugin install <pathOrSlug> --activate`.
   - Theme: `wp theme install <pathOrSlug>` (add `--activate` only if requested).

## Acceptance criteria
- [ ] Database values use prepared-statement parameters; the validated table prefix is the only interpolated SQL identifier fragment.
- [ ] DB credentials and passwords never appear in arguments, logs, summaries, or errors.
- [ ] Administrator discovery preserves the distinction between an administrator and the first arbitrary user.
- [ ] Email update changes both the user record and site-wide `admin_email` option.
- [ ] Unit tests cover credential redaction, prefix rejection, prepared-statement arguments, and WP-CLI calls.

## Testing seam
Unit test suite in `internal/siteconfig/siteconfig_test.go` with mock `wpcli.Runner`.
