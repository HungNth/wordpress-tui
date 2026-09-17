# 0013: External Tool Launcher and Settings Menu

## Status
Accepted

## Context
Users need to view and edit the application configuration (`config.json`) directly in an external editor (VS Code) and inspect cached assets (core archives, package downloads) in their native operating system file manager without manually locating hidden application directories.

Because `code <path>` returns immediately if an existing VS Code window is open, simply executing the command without waiting causes a race condition where any automatic configuration reload reads stale data while the user is still making edits. Furthermore, cross-platform file manager invocation differs between Windows (`explorer.exe`) and macOS (`open`).

## Decision
1. **Settings Menu Subsystem**: Implement a dedicated `settings` flow accessible from the WPTUI main menu with three sub-options:
   - `1. Open config.json in VS Code`
   - `2. Open Cache Directory`
   - `← Back to Main Menu`
2. **Hard Failure for VS Code Absence and Wait-Before-Reload**:
   - Check if `code` is available in PATH via `exec.LookPath("code")`. If not found or if invocation fails, trigger a hard failure (`VS Code ('code' CLI) not found in PATH`) without attempting any fallback editor or shell launcher.
   - Launch `code --wait <config_path>`, blocking execution until the user finishes editing and closes the editor window/tab.
   - Only upon successful zero-exit return from the editor, reload and validate `config.json` via `config.Load(cfgPath)`.
   - If validation or parsing fails, log an explicit warning and retain the existing in-memory `App.config` unchanged; only update `App.config` via callback when validation succeeds.
3. **Platform-Specific Directory Opening (Windows & macOS Only)**:
   - Resolve the root cache directory: `filepath.Join(os.UserCacheDir(), "wptui")`.
   - If the directory does not exist, report an explicit error (`Cache directory does not exist yet`) rather than creating it preemptively.
   - On Windows (`runtime.GOOS == "windows"`): launch `explorer.exe <dir>`.
   - On macOS (`runtime.GOOS == "darwin"`): launch `open <dir>`.
   - On other platforms: report an unsupported operating system error.
## Consequences
- Requires users to have the `code` CLI installed in PATH to use the direct editing action.
- Guarantees data consistency by reloading configuration strictly after editing finishes and propagating the new config to `App.config`.
- Avoids speculative directory creation while providing native file exploration on supported operating systems.
