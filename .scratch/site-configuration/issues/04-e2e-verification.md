# Title: Composed end-to-end integration and verification
Status: ready-for-agent
Labels: ready-for-agent

## Parent spec
.scratch/site-configuration/spec.md

## Required behavior
Implement comprehensive end-to-end tests exercising all 4 actions in sequence on a mock website:
1. Apply tweaks and assert WP-CLI config/option updates.
2. Reconcile administrator credentials, asserting direct database update for `user_login` / `user_nicename` via `go-sql-driver/mysql`, password stdin prompt, and synchronized `admin_email` option.
3. Install plugins with mock catalog and assert `wp plugin install --activate`.
4. Install themes and assert `wp theme install`.
5. Verify exit and back navigation.

## Acceptance criteria
- [ ] Full end-to-end integration test runs without live network or live MySQL dependencies.
- [ ] `go test -v -race ./...` passes across all packages with 0 data races.
- [ ] `go vet ./...` reports 0 warnings.
- [ ] Git working tree is clean.

## Testing seam
Composed test in `internal/app/e2e_siteconfig_test.go`.
