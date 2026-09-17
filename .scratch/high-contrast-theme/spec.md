# High-Contrast Dynamic UI Theme and Summary Visual System Specification

Status: ready-for-agent

## Problem Statement

When using WPTUI in terminal environments, interactive Select and MultiSelect options currently render in plain monochrome text. When navigating up and down using arrow keys across options (such as the Main Menu or Package selection lists), the active cursor item does not highlight prominently, creating ambiguity about which option is currently focused. Furthermore, selected packages in multi-select lists (`[x]`) are visually indistinguishable in color from unselected items, and the post-run terminal summary reports (for website provisioning and de-provisioning) output unstyled plain text, making it difficult for developers to quickly scan paths, URLs, warnings, and errors.

## Solution

WPTUI implements a cohesive, high-contrast visual theme palette across all interactive forms, custom Bubbletea models, and terminal summary reports:
1. Active option under cursor highlight: In all Select and MultiSelect forms, the focused item renders in high-visibility bright cyan (`#00FFFF`) and bold font, and the selector indicator (`> `) is bold cyan.
2. Selected package highlight: Checked packages (`[x]`) in multi-select lists and in the live search model render with vibrant green (`#04B575` / `#00FF00`) checkmarks and highlighted option labels, making chosen packages immediately distinguishable from the cursor position.
3. Color-coded summary reports: Post-operation completion reports (both Provisioning and De-provisioning) render structured visual feedback with bold green headers/statuses, bright cyan paths and `.test` URLs, yellow warnings and stale fallbacks, and bold red error notifications.

## User Stories

1. As a local WordPress developer, I want the active menu option to highlight in bright cyan when I press Up/Down arrows, so that I can instantly see which action is focused.
2. As a local WordPress developer, I want options that are not focused to remain muted, so that high visual contrast guides my attention to the active cursor.
3. As a local WordPress developer, I want packages I select with the Space key (`[x]`) to render with a bold green indicator and distinct text styling, so that I know exactly which items are currently in my basket.
4. As a local WordPress developer, I want the green checked styling to persist even when I navigate the cursor to other options, so that my selections remain visually obvious.
5. As a local WordPress developer, I want the live catalog search model to mirror this green checked indicator for selected packages, so that visual conventions are 100% consistent across all screens.
6. As a local WordPress developer, I want the provisioning completion report to display a bold green success banner, so that I immediately know site creation succeeded.
7. As a local WordPress developer, I want site file paths and `.test` web URLs in the summary report to render in bright cyan, so that I can easily spot and click/copy them.
8. As a local WordPress developer, I want package installation statuses in the summary report to be color-coded (green for cached, cyan for downloaded, yellow for stale fallback), so that I understand package resolution performance at a glance.
9. As a local WordPress developer, I want tweak warnings and errors to display in bold red/yellow text, so that potential configuration issues are impossible to miss.
10. As a local WordPress developer, I want the de-provisioning summary report to highlight deleted resources in green and errors in red, so that I have clear verification of the deletion outcome.

## Implementation Decisions

1. **Theme System Enhancements in `internal/tui/theme.go`**:
   - Configure `theme.Focused.SelectSelector` and `theme.Focused.MultiSelectSelector` with cyan foreground (`#00FFFF`) and bold text.
   - Configure `theme.Focused.Option` with cyan foreground and bold text for the currently focused option row.
   - Configure `theme.Focused.SelectedPrefix` with vibrant green foreground (`#04B575`) and bold text.
   - Configure `theme.Focused.SelectedOption` with green foreground and bold text for checked options in MultiSelect forms.
   - Ensure `CustomTheme()` propagates these styles across all Huh forms.

2. **Live Search View Cohesion in `internal/tui/live_search_model.go`**:
   - In `View()`, style checked items `[x]` with green text (`#04B575` / bold), while keeping the focused item line in bold cyan.

3. **Color-Coded Summary Formatters**:
   - Create Lipgloss helper styles for terminal summary reporting:
     - `SuccessStyle`: Bold green (`#04B575`) for successful titles and clean statuses.
     - `HighlightStyle`: Bright cyan (`#00FFFF`) for paths, `.test` URLs, and slugs.
     - `WarningStyle`: Bold yellow (`#FFA500`) for stale fallbacks and skipped tweaks.
     - `ErrorStyle`: Bold red (`#FF4444`) for errors and failed steps.
   - Apply these styles to `PrintDeleteSummary` and `PrintDeleteResult` in `internal/tui/delete_wizard.go`.
   - Apply these styles to the website provisioning summary in `internal/app/app.go`.

## Testing Decisions

Testing is structured across the following seams:
- **Theme Palette Unit Tests (`internal/tui/theme_test.go`)**:
  - Assert `AppTheme().Focused.Option.GetForeground()` matches bright cyan (`#00FFFF`).
  - Assert `AppTheme().Focused.SelectSelector.GetForeground()` matches bright cyan (`#00FFFF`).
  - Assert `AppTheme().Focused.SelectedPrefix.GetForeground()` matches vibrant green (`#04B575`).
  - Assert `AppTheme().Focused.SelectedOption.GetForeground()` matches vibrant green (`#04B575`).
- **Live Search View Test (`internal/tui/live_search_test.go`)**:
  - Verify that `LiveSearchModel.ViewString()` renders checked items with the styled green indicator.
- **Summary Styling Smoke Tests (`internal/tui` & `internal/app`)**:
  - Verify that summary functions render ANSI styled output containing expected color sequences for success, warning, and error states.

## Out of Scope

- Adding custom user-configurable theme color pickers to `config.json`.
- Modifying non-terminal GUI or web dashboards.
- Overriding terminal emulator default background colors.

## Further Notes

- ADR `0010-high-contrast-ui-palette-and-summary-system.md` records the architectural decisions for this visual system.
- Domain term `Visual Theme Palette` is recorded in `CONTEXT.md`.
