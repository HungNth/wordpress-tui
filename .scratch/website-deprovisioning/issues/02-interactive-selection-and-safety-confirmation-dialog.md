# 02: Interactive Selection and Safety Confirmation Dialog

Status: ready-for-agent
Blocked by: 01-core-discovery-and-single-site-deprovisioning.md
Parent spec: ../spec.md

## What to build

Implement the interactive multi-site selection and safety preview dialog:
1. Interactive MultiSelect list presenting all discovered candidate websites for user selection.
2. Formatted summary preview table showing selected directory names, full file paths, and detected database names.
3. High-visibility warning text reminding the user of the permanent nature of the deletion.
4. Explicit confirmation prompt defaulting to No to guard against accidental destructive actions.

## Acceptance criteria

- [ ] MultiSelect interface allows user to pick one or more websites from the candidate list or cancel.
- [ ] Preview displays directory, filesystem path, and detected database for all selected websites.
- [ ] Confirmation defaults to No and cancels execution cleanly if declined.
- [ ] Unit tests verify multi-select option construction, preview rendering, and rejection handling.
## Testing seam

- `internal/tui/delete_wizard_test.go`: test form construction, empty candidates handling, and confirm default values.

## Demo path

Launch delete wizard with scripted inputs, select a candidate, view the warning table, and confirm or reject deletion.
