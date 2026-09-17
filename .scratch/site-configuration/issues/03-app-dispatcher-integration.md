# Title: App dispatcher integration for config menu
Status: ready-for-agent
Labels: ready-for-agent

## Parent spec
.scratch/site-configuration/spec.md

## Required behavior
Integrate `config` action into `internal/app/app.go`:
1. Add `configFn func(context.Context, *config.Config) error` to `App` struct with sensible defaults.
2. In `RunWithContext`, route case `"config":` to `a.configFn(ctx, a.config)`.
3. Implement `RunDefaultConfigFlow` and `RunConfigFlowWithDeps`:
   - Discover websites via `deprovision.DiscoverCandidates`.
   - Loop: Select site -> Sub-menu -> Execute selected action -> Show summary -> Continue on site / Pick another site / Exit to main menu.
   - Resolve packages from API/cache using existing `PackageResolver`.
   - Re-use `tui.SelectPackagesFlow` with live catalog search.

## Acceptance criteria
- [ ] Selecting `config` in main menu executes the interactive website configuration flow.
- [ ] Clean error handling when no websites are found or when actions fail.
- [ ] Unit tests for `App` menu routing with mocked `configFn` in `internal/app/app_test.go`.

## Testing seam
`internal/app/app_test.go` and `internal/app/config_flow_test.go`.
