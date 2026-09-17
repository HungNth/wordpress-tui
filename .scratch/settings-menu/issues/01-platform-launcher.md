# Title: Platform-aware external launcher subsystem
Status: ready-for-agent
Labels: ready-for-agent

## Parent spec
.scratch/settings-menu/spec.md

## Required behavior
Implement a dedicated `internal/launcher` package providing cross-platform process execution:
1. `OpenInVSCode(ctx context.Context, filePath string, runner ProcessRunner) error`:
   - Checks `LookPath("code")`. If missing, returns hard failure error `VS Code ('code' CLI) not found in PATH` without fallback editors.
   - Executes `code --wait <filePath>`, blocking until editor closes.
2. `OpenDirectory(ctx context.Context, dirPath string, runner ProcessRunner, goos string) error`:
   - Checks if `dirPath` exists. If not, returns error indicating cache directory does not exist (does NOT create directories).
   - Dispatches based on supported OS:
     - Windows (`windows`): `explorer.exe <dirPath>`
     - macOS (`darwin`): `open <dirPath>`
     - Other OS: returns error indicating unsupported operating system.

## Acceptance criteria
- [ ] Explicit hard failure when `code` is not in PATH.
- [ ] Blocks on `code --wait` to prevent reload races.
- [ ] Error returned if directory does not exist.
- [ ] Dispatches strictly on Windows (`explorer.exe`) and macOS (`open`).
- [ ] Comprehensive unit tests with mock ProcessRunner in `internal/launcher/launcher_test.go`.
