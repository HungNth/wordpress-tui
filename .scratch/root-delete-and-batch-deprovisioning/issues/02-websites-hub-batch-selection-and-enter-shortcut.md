# 02: Websites Hub Batch Selection and Enter Shortcut

Status: resolved
Blocked by: none
Parent spec: ../spec.md

## What to build

Enhance multi-site selection and enter shortcut handling in `WebsitesHubModel` (`internal/tui/websites_hub.go`):
1. In `WebsitesHubList` state, add key handling for `"a"` to toggle select all / deselect all:
   - If all candidates are currently selected, clear selection map.
   - If not all candidates are selected, select all candidates.
2. In `WebsitesHubList` state, enhance `"enter"` key handling:
   - If `m.SelectedCount() > 1`: transition directly to `WebsitesHubBatchDeleteConfirm` (single batch delete confirmation screen), resetting `deleteConfirmChoice = false`.
   - If `m.SelectedCount() <= 1`: maintain existing behavior (open `WebsitesHubActions` for single website details).
3. Ensure `WebsitesHubBatchDeleteConfirm` presents exactly 1 confirmation screen listing all selected sites and, when confirmed, calls `OnBatchDelete` with all selected candidates.

## Acceptance criteria

- [x] Pressing `a` in `WebsitesHubList` selects all candidates if not all are selected; deselects all if all are selected.
- [x] In `WebsitesHubList`, when 2 or more candidates are checked, pressing `Enter` directly enters `WebsitesHubBatchDeleteConfirm`.
- [x] When 0 or 1 candidate is checked, pressing `Enter` opens `WebsitesHubActions` as before.
- [x] Exactly 1 confirmation screen is presented before invoking `OnBatchDelete`.
- [x] Unit tests in `internal/tui/websites_hub_test.go` verify `a` key toggle, `Enter` branching on `SelectedCount() > 1`, and batch deletion callback execution.

## Testing seam

- `internal/tui/websites_hub_test.go`: test `a` key select-all/deselect-all logic, `Enter` key transition to `WebsitesHubBatchDeleteConfirm` when `SelectedCount() > 1`, and single confirmation execution.

## Demo path

Open Websites Hub, press `Space` on 2 websites, press `Enter` to see single batch delete confirmation screen immediately, without entering single-site details menu.

## Answer

Implemented Websites Hub select-all toggle and batch deletion `Enter` shortcut:
1. Added `a` key handling in `WebsitesHubList` to select all / deselect all candidates.
2. Updated `enter` key handling in `WebsitesHubList` to immediately transition to `WebsitesHubBatchDeleteConfirm` when `SelectedCount() > 1`, keeping single details when `SelectedCount() <= 1`.
3. Verified single confirmation execution invoking `OnBatchDelete` directly.
4. Added test `TestWebsitesHub_SelectAllAndEnterBatchDelete` in `internal/tui/websites_hub_test.go`. All tests pass.
