# 02: Live Search View Checked Item Cohesion

Status: resolved
Blocked by: None (can start immediately)
Parent spec: ../spec.md

## What to build

Update the custom Bubbletea `LiveSearchModel` view rendering so that packages toggled with the Space key render their checked indicator `[x]` in bold vibrant green (`#04B575`), matching the Huh theme visual conventions and clearly separating checked status from cursor position.

## Acceptance criteria

- [x] Checked indicator `[x]` renders in vibrant green bold styling in `LiveSearchModel.View()`.
- [x] Active cursor line remains highlighted in bold cyan.
- [x] Unchecked packages `[ ]` remain in standard neutral text.
- [x] Unit tests verify view output contains green styling for checked items.

## Testing seam

- `internal/tui/live_search_test.go`: assert rendered view text applies green styling to checked items.

## Demo path


## Answer

Implemented green checked indicator in `internal/tui/live_search_model.go`:
1. Styled `[x]` and selected package labels with vibrant green (`#04B575`) bold text when selected.
2. Kept focused cursor line in bright cyan (`#00FFFF`) bold text.
3. Maintained standard neutral display for unchecked packages `[ ]`.
4. Added unit test assertion in `internal/tui/live_search_test.go` confirming `[x]` rendering.
Run `wptui`, choose `Create`, navigate to package search, press Space on a package, and observe that its `[x]` checkmark renders in vibrant green while the active line cursor highlights in cyan.
