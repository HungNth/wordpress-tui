# Application Settings Specification

## Summary
Interactive system-level maintenance and settings workflow accessible from the WPTUI Main Menu under the `settings` option ("Application settings"). It allows users to:
1. Open `config.json` directly in VS Code using `code --wait <path>`, safely blocking until editing finishes and immediately hot-reloading & validating the configuration into `App.config` via a callback.
2. Open the WPTUI cache directory (`filepath.Join(os.UserCacheDir(), "wptui")`) using the native OS file manager on Windows (`explorer.exe`) or macOS (`open`).
3. Return cleanly to the Main Menu via `← Back to Main Menu` or Esc.

## Requirements

### 1. Main Menu Enablement & Sub-Menu Loop
- Enable the `settings` option in `internal/tui/menu.go` (`Disabled: false`, updated description).
- Wire `case "settings":` in `internal/app/app.go` to invoke `RunSettingsFlow`.
- Display an interactive Huh single-select form with:
  - `1. Open config.json in VS Code`
  - `2. Open Cache Directory`
  - `← Back to Main Menu`
- Stay in the Settings sub-menu after each action so users can perform multiple actions, exiting to the main menu only when choosing `← Back` or pressing Esc.

### 2. Action 1: Open `config.json` in VS Code
- Locate `config.json` from the active application config path.
- Check if `code` CLI binary exists in PATH using `exec.LookPath("code")`.
  - If missing: display an explicit error `VS Code ('code' CLI) not found in PATH` and return to the Settings menu. Do NOT attempt silent fallback editors.
- Launch `code --wait <config_path>`:
  - Blocks execution until the user closes the VS Code tab or window.
- After process completion:
  - Reload and validate configuration from disk using `config.Load(cfgPath)`.
  - If valid: update `App.config` via `onReload(newCfg)` callback and display `[✓] Configuration reloaded successfully.`
  - If invalid JSON / validation error: display `[✗] Error reloading configuration: <err>` while preserving the previous valid in-memory config to prevent crashes.

### 3. Action 2: Open Cache Directory (Windows & macOS)
- Resolve cache directory:
  ```go
  userCache, err := os.UserCacheDir()
  cacheDir := filepath.Join(userCache, "wptui")
  ```
- Check if `cacheDir` exists. If not, report `Cache directory does not exist yet`. Do NOT create empty directories preemptively.
- Launch the platform-specific native file manager:
  - Windows (`windows`): `exec.Command("explorer.exe", cacheDir).Start()`
  - macOS (`darwin`): `exec.Command("open", cacheDir).Start()`
  - Other operating systems: report unsupported operating system error.
- Display `[✓] Opened cache directory: <path>` in green.

## Architectural Boundaries
- Platform launching logic isolated in `internal/launcher`.
- TUI interactive form in `internal/tui/settings_wizard.go`.
- Flow orchestrator in `internal/app/settings_flow.go`.

## Testing Seams
- Unit tests for launcher command builders and OS detection with mock runners in `internal/launcher/launcher_test.go`.
- Unit tests for TUI settings choices in `internal/tui/settings_test.go`.
- Flow orchestration test verifying reload updates `App.config` in `internal/app/settings_flow_test.go`.
