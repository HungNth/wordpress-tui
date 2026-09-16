# 01: Skip content during WordPress core download

**What to build:**
When provisioning a new WordPress Website, WPTUI passes `--skip-content` to `wp core download` so that default bundled themes (such as Twenty Twenty-Four) and default plugins (Akismet, Hello Dolly) are omitted, providing a lean, faster installation with only explicitly chosen themes and plugins.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

## Acceptance criteria

- [ ] `wpcli.Client.CoreDownload` appends `--skip-content` to the `wp core download` invocation: `wp core download https://wordpress.org/latest.zip --skip-content`.
- [ ] All unit tests in `internal/wpcli` assert that `--skip-content` is present in the executed command.
- [ ] All create orchestrator tests in `internal/create` assert that core download runs with `--skip-content`.
- [ ] When Package integration is disabled or no theme is selected, WPTUI reports `DefaultThemeSkipped: true` without attempting to retain or activate a bundled theme.
- [ ] Existing tweak and package installation flows continue to execute without regressions.

## Testing seam

- `internal/wpcli`: `TestWPCLIClient_Flow` asserts `wp core download https://wordpress.org/latest.zip --skip-content`.
- `internal/create`: `TestCreator_SuccessFlow` and `TestCreator_PackageInstallAndDeduplication` verify successful creation with `--skip-content`.

## Demo path

Run `go test -v ./internal/wpcli ./internal/create` and observe that `CoreDownload` executes with `--skip-content`.
