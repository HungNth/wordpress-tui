# 02: Interactive Selection and Safety Confirmation Dialog

Status: resolved
Blocked by: 01-core-discovery-and-single-site-deprovisioning.md
Parent spec: ../spec.md

## What to build

Implement the interactive multi-site selection and safety preview dialog:
1. Interactive MultiSelect list presenting all discovered candidate websites for user selection.
2. Formatted summary preview table showing selected directory names, full file paths, and detected database names.
3. High-visibility warning text reminding the user of the permanent nature of the deletion.
4. Explicit confirmation prompt defaulting to No to guard against accidental destructive actions.

## Acceptance criteria

- [x] MultiSelect interface allows user to pick one or more websites from the candidate list or cancel.
- [x] Preview displays directory, filesystem path, and detected database for all selected websites.
- [x] Confirmation defaults to No and cancels execution cleanly if declined.
- [x] Unit tests verify multi-select option construction, preview rendering, and rejection handling.
## Testing seam

- `internal/tui/delete_wizard_test.go`: test form construction, empty candidates handling, and confirm default values.

## Demo path

Launch delete wizard with scripted inputs, select a candidate, view the warning table, and confirm or reject deletion.

## Answer

Implemented interactive multi-site selection and safety preview:
1. `BuildDeleteSelectionForm` and `PromptDeleteSelection` allowing multi-selection of candidates.
2. `BuildDeleteConfirmMultiForm` rendering formatted warning table with directory name, path, and detected DB.
3. `PromptDeleteConfirmMulti` with explicit confirmation prompt defaulting to `false`.
4. Unit tests in `internal/tui/delete_wizard_test.go` verifying form construction and default values.
