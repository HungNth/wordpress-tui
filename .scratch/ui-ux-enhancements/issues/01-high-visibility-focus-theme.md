# 01: High-visibility focus indicator theme

Status: ready-for-agent
Blocked by: None (can start immediately)
Parent spec: ../spec.md

## What to build

Configure a dedicated WPTUI theme where the active/focused field's vertical border indicator uses high-visibility bright cyan (`#00FFFF`), replacing the muted gray line and making the active input immediately recognizable across all terminal environments.

## Acceptance criteria

- [ ] `tui.AppTheme()` returns a `*huh.Styles` theme where `Focused.Base` border foreground is set to bright cyan (`#00FFFF`).
- [ ] All forms in `internal/tui` (wizard, main menu, create wizard, packages picker) apply `WithTheme(AppTheme())`.
- [ ] Unit test in `internal/tui` verifies that `AppTheme()` configures the expected border color.

## Testing seam

- `internal/tui/theme_test.go`: assert `AppTheme().Focused.Base.GetBorderForeground()` equals cyan.

## Demo path

Run `wptui` and observe that the vertical line indicator on active input fields renders in bright cyan.
