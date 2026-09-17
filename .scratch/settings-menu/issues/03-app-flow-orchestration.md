# Title: Settings app flow orchestration and hot-reload verification
Status: resolved
Labels: ready-for-agent

## Parent spec
.scratch/settings-menu/spec.md

## Required behavior
1. Add `settingsFn func(context.Context, *config.Config, func(*config.Config)) error` to `App` struct in `internal/app/app.go` and route `case "settings":` passing `func(newCfg *config.Config) { a.config = newCfg }`.
2. Implement `RunDefaultSettingsFlow` and `RunSettingsFlowWithDeps` in `internal/app/settings_flow.go`:
   - Sub-menu loop:
     - Action 1 (VS Code): invoke launcher `OpenInVSCode` -> on exit 0, call `config.Load(cfgPath)` -> if valid, invoke `onReload(newCfg)` to update `App.config`; if invalid, log error and preserve current `App.config`.
     - Action 2: Open cache dir strictly on Windows/macOS.
     - Action 3: Return to main menu.
   - If `code` CLI is not found or fails, display hard error without crashing.
3. Write end-to-end integration tests verifying flow orchestration, reload updating `App.config`, and invalid JSON retention in `internal/app/settings_flow_test.go`.

## Acceptance criteria
- [x] Selecting `settings` in main menu executes settings flow.
- [x] After `code --wait` returns cleanly, configuration is reloaded and validated, updating `App.config` via callback.
- [x] Invalid JSON preserves previous `App.config` in memory.
- [x] Hard failure displayed when `code` CLI is missing or exits non-zero without triggering reload.
- [x] Full test suite passes under `-race` with 0 warnings from `go vet`.
