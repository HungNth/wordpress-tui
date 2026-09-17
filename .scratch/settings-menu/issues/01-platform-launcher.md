# Title: Platform-aware external launcher subsystem
Status: ready-for-agent
Labels: ready-for-agent

## Parent spec
.scratch/settings-menu/spec.md

## Required behavior
Implement a dedicated `internal/launcher` package providing cross-platform process execution:
1. `OpenInVSCode(ctx context.Context, filePath string, runner ProcessRunner) error`:
   - Checks `LookPath("code")`. Returns an error if not found.
   - Executes `code --wait <filePath>`, blocking until editor closes.
2. `OpenDirectory(ctx context.Context, dirPath string, runner ProcessRunner, goos string) error`:
   - Checks if `dirPath` exists. If not, returns error indicating it does not exist (does NOT create directories).
   - Dispatches based on supported OS:
     - Windows (`windows`): `explorer.exe <dirPath>`
     - macOS (`darwin`): `open <dirPath>`
     - Other OS: returns error indicating unsupported operating system.

## Acceptance criteria
- [ ] Explicit error when `code` is not in PATH.
- [ ] Blocks on `code --wait` to prevent reload races.
- [ ] Error if directory does not exist.
- [ ] Correctly executes explorer on Windows and open on macOS.
- [ ] Comprehensive unit tests with mock ProcessRunner in `internal/launcher/launcher_test.go`.
