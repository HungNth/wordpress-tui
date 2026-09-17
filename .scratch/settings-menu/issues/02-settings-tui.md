# Title: Settings TUI sub-menu and main menu enablement
Status: resolved
Labels: ready-for-agent

## Parent spec
.scratch/settings-menu/spec.md

## Required behavior
1. Enable `settings` item in `internal/tui/menu.go`:
   - Set `Disabled: false` and description "Manage application settings and view cache".
2. Implement `PromptSettingsAction() (SettingsAction, error)` in `internal/tui/settings_wizard.go`:
   - Action 1: `Open config.json in VS Code`
   - Action 2: `Open Cache Directory`
   - Action 3: `← Back to Main Menu`
3. Print success and error messages adhering to theme colors (green check, red error).

## Acceptance criteria
- [ ] Settings item appears enabled in main menu form.
- [ ] Sub-menu navigation with Up/Down, Enter, Esc works seamlessly.
- [ ] Unit tests in `internal/tui/settings_test.go`.
