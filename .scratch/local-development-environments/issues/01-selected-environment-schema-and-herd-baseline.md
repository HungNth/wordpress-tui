# 01: Selected environment schema and Herd baseline

**Parent specification:** `../spec.md`

**What to build:** Replace the Herd boolean with an explicit Local Development Environment configuration and migrate the currently supported Herd behavior through one centralized environment boundary. A fresh user can explicitly choose Herd during setup, then Provision, restore, open, and De-provision Websites with the same safety guarantees under the new schema.

**Blocked by:** None (can start immediately).

**Status:** completed

## Acceptance criteria

- [x] Configuration requires exact lower-case `app_name` and non-empty `app_command` values, accepts Herd, and serializes only the new schema.
- [x] Configuration containing the removed `used_herd`, static theme/plugin arrays, or default-theme field is rejected with a clear clean-cutover error; no migration, alias, fallback, or deprecated field remains.
- [x] First-run setup requires an explicit Local Development Environment choice and supplies editable Herd Websites-directory and executable-command suggestions without silently selecting Herd.
- [x] PHP, WP-CLI, MySQL, and the configured Herd executable are checked before mutation.
- [x] Shared workflows use one environment-selected boundary for dependency checks, Website Registration status, TLS, cleanup, and URL policy rather than Herd booleans distributed through business logic.
- [x] Herd Provisioning secures the Website as a critical final operation and rolls back only resources owned by that run on failure.
- [x] Herd Website Restoration preserves its existing TLS-warning behavior and reconciles the retained Website to HTTP when securing fails.
- [x] Herd De-provisioning attempts unsecure on a best-effort basis and still performs existing database verification and mandatory directory removal.
- [x] Browser opening continues through the operating-system launcher rather than an environment command.
- [x] Progress, results, and destructive summaries use environment-neutral resource labels while retaining useful Herd detail.
- [x] Obsolete boolean APIs, branches, fixtures, and shared Herd-only names are removed rather than wrapped or deprecated.

## Testing seam

Use composed application workflow tests with an injected command runner, temporary configuration, and isolated Website fixtures. Add narrow configuration and first-run model tests only for schema rejection, explicit selection, and serialization behavior.

## Demo path

Start without a config, explicitly select Herd, save the generated schema, Provision a Website through the TUI, open it through the system browser launcher, and De-provision it while observing separate environment, database, and directory outcomes.

## Verification

- `go test ./...` passed, including configuration rejection without mutation and composed environment create/delete workflows.
- Actual setup terminal rejected the empty environment sentinel, displayed all three Windows choices, and advanced after an explicit Flock selection. Cancellation left user configuration untouched.
- Real Herd Website mutation was not exercised; lifecycle command contracts were verified with isolated runners.
