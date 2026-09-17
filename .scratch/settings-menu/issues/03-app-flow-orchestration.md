# Title: Settings app flow orchestration and hot-reload verification
Status: ready-for-agent
Labels: ready-for-agent

## Parent spec
.scratch/settings-menu/spec.md

## Required behavior
1. Add `settingsFn func(context.Context, *config.Config, func(*config.Config)) error` to `App` struct in `internal/app/app.go` and route `case "settings":` passing `func(newCfg *config.Config) { a.config = newCfg }`.
2. Implement `RunDefaultSettingsFlow` and `RunSettingsFlowWithDeps` in `internal/app/settings_flow.go`:
   - Sub-menu loop: Action 1 (VS Code) -> `config.Load` -> invoke reload callback to update `App.config`; Action 2 -> Open cache dir on Windows/macOS; Action 3 -> return to main menu.
   - Gracefully report malformed JSON on reload without crashing and without updating `App.config`.
3. Write end-to-end integration tests verifying flow orchestration, reload updating `App.config`, and invalid JSON retention in `internal/app/settings_flow_test.go`.

## Acceptance criteria
- [ ] Selecting `settings` in main menu executes settings flow.
- [ ] After `code --wait` returns, configuration is reloaded and validated, updating `App.config` via callback.
- [ ] Invalid JSON preserves previous `App.config` in memory.
- [ ] Full test suite passes under `-race` with 0 warnings from `go vet`.
