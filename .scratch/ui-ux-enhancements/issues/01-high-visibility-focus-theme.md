# 01: High-visibility focus indicator theme

Status: resolved
Blocked by: None (can start immediately)
Parent spec: ../spec.md

## What to build

Configure a dedicated WPTUI theme where the active/focused field's vertical border indicator uses high-visibility bright cyan (`#00FFFF`), replacing the muted gray line and making the active input immediately recognizable across all terminal environments.

## Acceptance criteria
- [x] `tui.AppTheme()` returns a `*huh.Styles` theme where `Focused.Base` border foreground is set to bright cyan (`#00FFFF`).
- [x] All forms in `internal/tui` (wizard, main menu, create wizard, packages picker) apply `WithTheme(CustomTheme())`.
- [x] Unit test in `internal/tui` verifies that `AppTheme()` configures the expected border color.

## Testing seam

- `internal/tui/theme_test.go`: assert `AppTheme().Focused.Base.GetBorderForeground()` equals cyan.

## Demo path

Run `wptui` and observe that the vertical line indicator on active input fields renders in bright cyan.

## Answer

Implemented `AppTheme()` and `CustomTheme()` in `internal/tui/theme.go`:
- `Focused.Base` has its left border enabled with thick border and foreground color set to `#00FFFF` (bright cyan).
- `Focused.Title` styled with `#00FFFF` and bold.
- `CustomTheme()` implements `huh.Theme` and is applied via `.WithTheme(CustomTheme())` across all forms in `wizard.go`, `menu.go`, `create_wizard.go`, and `packages_picker.go`.
- Added `TestAppTheme_FocusedBorderCyan` in `internal/tui/theme_test.go` verifying RGBA color match and border state.
