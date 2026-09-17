# 01: Focus and Selection Option Highlighting

Status: resolved
Blocked by: None (can start immediately)
Parent spec: ../spec.md

## What to build

Configure the global WPTUI Huh theme so that navigating options with Up/Down arrows highlights the active cursor option in bold bright cyan (`#00FFFF`), and toggling selections in multi-select forms highlights checked items (`[x]`) with a bold green indicator and distinct text styling.

## Acceptance criteria

- [x] Active option currently focused under the cursor renders in bold bright cyan (`#00FFFF`).
- [x] Selector indicator (`> `) in Select and MultiSelect fields renders in bold bright cyan.
- [x] Checked prefix (`[x]`) and selected option labels render in bold vibrant green (`#04B575`).
- [x] Unfocused and unselected options remain in standard muted contrast.
- [x] Unit tests verify that `AppTheme()` configures the expected colors for option focus and selection states.

## Testing seam

- `internal/tui/theme_test.go`: assert `AppTheme()` field styles for Option, SelectSelector, MultiSelectSelector, SelectedPrefix, and SelectedOption.

## Demo path


## Answer

Configured `AppTheme()` in `internal/tui/theme.go`:
1. Active cursor indicator `SelectSelector` and `MultiSelectSelector` styled in bold bright cyan (`#00FFFF`).
2. Active focused option line `Focused.Option` styled in bold bright cyan (`#00FFFF`).
3. Checked items `SelectedPrefix` (`[x]`) and `SelectedOption` styled in bold vibrant green (`#04B575`).
4. Added comprehensive assertions in `internal/tui/theme_test.go` (`TestAppTheme_OptionAndSelectionColors`).
Run `wptui`, navigate the Main Menu with Up/Down arrows to observe the bold cyan highlight on active options, and navigate to package selection to observe bold green highlights on checked items.
