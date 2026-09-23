# 01: Sidebar Delete Navigation and Dedicated Batch De-provisioning View

Status: resolved
Blocked by: none
Parent spec: ../spec.md

## What to build

Implement the top-level "Delete" sidebar destination and its dedicated batch de-provisioning view:
1. Add `SectionDelete` to `Section` enum in `internal/tui/app_model.go` and insert `{Section: SectionDelete, Title: "Delete", Description: "Batch de-provision local websites"}` into `DefaultSidebarItems`.
2. Implement `DeleteModel` in `internal/tui/delete_model.go`:
   - Scans and lists candidate websites from `cfg.WebsitesPath` using `deprovision.DiscoverCandidates`.
   - Renders candidates with checkboxes (`[ ]` / `[x]`), slug, site URL, and path.
   - Supports keyboard interactions:
     - `Up` / `Down` / `k` / `j`: Move candidate cursor.
     - `Space`: Toggle selection of the highlighted website.
     - `a`: Toggle select all / deselect all candidates.
     - `Enter`: If at least 1 candidate is selected, transition to single confirmation view. If 0 selected, display warning notice.
     - `Esc` / `Shift+Tab` / `Left`: Return focus to Sidebar Navigation.
   - Renders single Confirmation Screen:
     - Warning header: `⚠ Delete Selected Websites?`.
     - Displays formatted list of all selected targets.
     - Confirmation toggle: `[ Yes, Delete All ]` vs `[ No, Cancel ]` (defaulting safely to `[ No, Cancel ]`).
     - `Left` / `Right` / `Tab` to switch choice; `Esc` to cancel back to list; `Enter` to confirm.
     - On confirmation, invokes `OnDelete` callback with `[]deprovision.Candidate`.
3. Integrate `DeleteModel` into `AppModel`:
   - Route `FocusContent` input events to `DeleteModel` when `activeSec == SectionDelete`.
   - Render `DeleteModel` in Content Pane when `activeSec == SectionDelete`.
   - Update footer keyboard help string for `SectionDelete`.

## Acceptance criteria

- [x] `DefaultSidebarItems` includes `Delete` between `Create` and `Restore`.
- [x] Focusing `Delete` renders the candidate website list in Content Pane with `[ ]` checkboxes.
- [x] `Space` toggles selection; `a` toggles select all / deselect all.
- [x] Pressing `Enter` with selected candidates displays the single confirmation screen.
- [x] Confirmation prompt defaults to `[ No, Cancel ]`. Confirming `[ Yes, Delete All ]` invokes `OnDelete` with all selected candidates.
- [x] Unit tests in `internal/tui/delete_model_test.go` and `internal/tui/app_model_test.go` verify navigation, selection toggles, confirmation default, and callback invocation.

## Testing seam

- `internal/tui/delete_model_test.go`: test candidate list rendering, selection toggle, select-all toggle, confirmation view, and choice toggle.
- `internal/tui/app_model_test.go`: test sidebar navigation index to `SectionDelete`, content pane routing, and footer help.

## Demo path

Navigate down the sidebar to "Delete", press `Enter` to focus content pane, press `a` to select all or `Space` to select websites, press `Enter` to see single confirmation dialog, toggle to `[ Yes, Delete All ]`, and press `Enter`.

## Answer

Implemented top-level `Delete` sidebar navigation and dedicated batch de-provisioning view:
1. Added `SectionDelete` enum and inserted `Delete` between `Create` and `Restore` in `DefaultSidebarItems` within `internal/tui/app_model.go`.
2. Created `internal/tui/delete_model.go` implementing `DeleteModel` with multi-select support (`Space`, `a`), single confirmation view defaulting to `[ No, Cancel ]`, and `OnDelete` callback.
3. Integrated `DeleteModel` into `AppModel` update, view, and execution cleanup lifecycles.
4. Added test suite in `internal/tui/delete_model_test.go` and updated `app_model_test.go` and `app_test.go`. All tests pass.
