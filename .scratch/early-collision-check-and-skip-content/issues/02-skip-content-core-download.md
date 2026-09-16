# 02: Skip content during WordPress core download

Status: resolved
Blocked by: None (can start immediately)
Parent spec: ../spec.md

## What to build

When provisioning a new WordPress Website, WPTUI passes `--skip-content` to `wp core download` so that default bundled themes (such as Twenty Twenty-Four) and default plugins (Akismet, Hello Dolly) are omitted, providing a lean, faster installation with only explicitly chosen themes and plugins.

## Acceptance criteria

- [x] `wpcli.Client.CoreDownload` appends `--skip-content` to the `wp core download` invocation: `wp core download --skip-content` (with optional `--locale`).
- [x] All unit tests in `internal/wpcli` assert that `--skip-content` is present in the executed command.
- [x] All create orchestrator tests in `internal/create` assert that core download runs with `--skip-content`.
- [x] When a configured default theme exists in config but Package integration is unavailable, WPTUI reports `DefaultThemeSkipped: true`. If no default theme is configured, `DefaultThemeSkipped` remains `false`.
- [x] When no theme is installed via Package integration, no bundled theme is retained; active theme reporting accurately reflects WordPress state.
- [x] Existing tweak and package installation flows continue to execute without regressions.

## Testing seam

- `internal/wpcli`: `TestWPCLIClient_Flow` asserts `wp core download --skip-content --locale=en_US`.
- `internal/create`: `TestCreator_SuccessFlow` and `TestCreator_PackageInstallAndDeduplication` verify successful creation with `--skip-content` and accurate `DefaultThemeSkipped` reporting.
- `internal/app`: E2E smoke test verifying that website creation produces an installation with `--skip-content`.

## Demo path

Run `go test -v ./internal/wpcli ./internal/create` and observe that `CoreDownload` executes with `--skip-content` and theme reporting matches the semantic contract.

## Answer

Implemented `--skip-content` flag in `internal/wpcli/wpcli.go`:
- `wp core download` executes with `--skip-content` without positional URL.
- Updated unit assertions in `internal/wpcli/wpcli_test.go` and `internal/create/create_test.go` confirming the `--skip-content` argument.
- Preserved semantic `DefaultThemeSkipped` contract: `DefaultThemeSkipped` is set to `true` strictly when a configured default theme exists but Package integration is unavailable; otherwise it remains `false`.
- Tests pass across `wpcli` and `create`.
