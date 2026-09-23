# Root Delete Navigation and Batch De-provisioning Specification

Status: resolved

## Problem Statement

Currently in WPTUI:
1. De-provisioning websites is nested under the `Websites Hub` (`SectionWebsites`) single-site actions menu, without a top-level option at the application root navigation.
2. In the `Websites Hub`, selecting multiple websites and pressing `Enter` opens the single-candidate action menu for the highlighted item instead of initiating batch deletion, forcing awkward workflows.
3. Users selecting multiple sites want a single unified confirmation screen listing all selected websites, followed by concurrent de-provisioning (`opts.Concurrency: 4`), rather than multiple individual confirmations.

## Solution

1. **Sidebar Navigation**:
   - Add `Delete` as a top-level navigation destination in the persistent left Sidebar (`Websites`, `Create`, `Delete`, `Restore`, `Settings`, `Exit`).
   - Define `SectionDelete` in `Section` enum in `internal/tui/app_model.go`.

2. **Dedicated Batch De-provisioning View (`DeleteModel`)**:
   - When the user focuses `Delete`, the Content Pane displays a dedicated batch de-provisioning view:
     - Discovered candidates list with checkboxes `[ ]`/`[x]`, slugs, URLs, and directory paths.
     - Interactive controls: `Up`/`Down`/`j`/`k` to navigate, `Space` to toggle individual item, `a` to toggle select/deselect all.
     - Pressing `Enter` with 1 or more candidates selected switches immediately to the single confirmation screen. If none are selected, a helpful warning hint is displayed.
     - Pressing `Esc` or `Shift+Tab` returns focus to the Sidebar Navigation.
   - **Single Confirmation Screen**:
     - Explicit warning title: `⚠ Delete Selected Websites?`.
     - Displays formatted list of all selected targets (slug, directory path, database).
     - Selection choice toggle: `[ Yes, Delete All ]` vs `[ No, Cancel ]`, strictly defaulting to `[ No, Cancel ]`.
     - Keys: `Left`/`Right`/`Tab` to toggle choice; `Esc` to cancel back to list; `Enter` to confirm.
     - On `[ Yes, Delete All ]`, invokes the delete execution handler with all selected candidates.

3. **Websites Hub Integration**:
   - In `WebsitesHubList`:
     - Pressing `a` toggles select all / deselect all across all discovered candidates.
     - When `SelectedCount() > 1`, pressing `Enter` directly triggers `WebsitesHubBatchDeleteConfirm` (single confirmation screen for all checked websites).
     - When `SelectedCount() <= 1`, pressing `Enter` maintains existing behavior (opens single-candidate `WebsitesHubActions`).
     - Fast hotkey `d` remains active to trigger batch deletion when items are selected.

4. **Concurrent Execution Engine**:
   - In `runDeleteExecution`:
     - Resolve candidate database names concurrently using a bounded worker pool (up to 4 workers).
     - Execute de-provisioning via `deprovision.Deprovision` with `Concurrency: 4`.
     - Stream live log lines to the In-Pane Progress Monitor.
     - Upon completion, refresh candidate inventories in both `WebsitesHub` and `DeleteModel`, clearing selection maps.

## User Stories

1. As a local WordPress developer, I want to see a dedicated "Delete" option in the main sidebar menu, so that I can de-provision websites without searching through submenus.
2. As a local WordPress developer, I want to batch select multiple websites using `Space` and select/deselect all with `a`, so that I can quickly mark test sites for cleanup.
3. As a local WordPress developer, I want exactly one confirmation screen listing all selected websites, so that I don't have to confirm each site individually.
4. As a local WordPress developer, I want the confirmation prompt to default to `[ No, Cancel ]`, so that accidental keystrokes never cause data loss.
5. As a local WordPress developer, I want batch deletions to execute concurrently, so that cleaning up multiple sites is fast and efficient.
6. As a local WordPress developer, I want `Enter` in the Websites Hub to automatically trigger batch deletion if more than one site is checked, so that multi-selection behaves intuitively.

## Implementation Decisions

1. **Architecture and Package Seams**:
   - `internal/tui/app_model.go`:
     - Add `SectionDelete` to `Section` enum and update `DefaultSidebarItems`.
     - Instantiate and manage `deleteModel *DeleteModel`.
     - Update keyboard routing for `FocusSidebar` and `FocusContent`.
     - Update completion lifecycle to refresh both `websitesHub` and `deleteModel`.
   - `internal/tui/delete_model.go`:
     - Encapsulate the dedicated batch de-provisioning view model.
     - State machine: `DeleteViewStateList` and `DeleteViewStateConfirm`.
     - Support `Space` (toggle item), `a` (toggle all), `Enter` (go to confirm), `Esc` (return to sidebar).
     - Provide `OnDelete func(cands []deprovision.Candidate) tea.Cmd`.
   - `internal/tui/websites_hub.go`:
     - Add `a` hotkey in `WebsitesHubList` to select/deselect all candidates.
     - In `Enter` key handler, check `SelectedCount() > 1`: if true, switch to `WebsitesHubBatchDeleteConfirm`.
   - `internal/app/app_executors.go`:
     - In `runDeleteExecution`, resolve databases concurrently using bounded goroutines before delegating to `deprovision.Deprovision(..., Concurrency: 4)`.
