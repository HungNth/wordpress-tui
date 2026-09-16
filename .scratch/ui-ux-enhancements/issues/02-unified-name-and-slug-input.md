# 02: Unified Website Name and Slug input in create wizard

Status: resolved
Blocked by: None (can start immediately)
Parent spec: ../spec.md

## What to build

Unify the `Website Name` and `Website Slug` inputs into a single interactive form view instead of two separate steps. If the user leaves `Website Slug` blank, it is automatically derived from `Website Name`. If a custom slug is entered, it is validated for syntax and availability immediately.

## Acceptance criteria

- [x] `Website Name` and `Website Slug` are presented together in the initial group of the create form.
- [x] Leaving `Website Slug` blank is permitted; on submission it automatically derives from `Website Name` via `create.Slugify(inputs.WebsiteName)`.
- [x] If a slug is entered explicitly, it is validated for syntax and directory/database availability.
- [x] When the slug derives from name, availability is verified before advancing to administrator credentials.
- [x] Unit tests in `internal/tui` verify unified form behavior, auto-derivation when blank, and explicit slug overriding.

## Testing seam

- `internal/tui`: Unit tests for `BuildCreateForm` confirming empty slug derivation and explicit slug validation.

## Demo path

Run `wptui`, choose Create, and observe `Website Name` and `Website Slug` on the same screen; leave slug blank and observe it automatically use the slugified name.

## Answer

Unified `Website Name` and `Website Slug` in `internal/tui/create_wizard.go`:
- Both fields now appear together in the first form group.
- Blank slug input is supported: the validator derives `create.Slugify(inputs.WebsiteName)` and validates availability against directory and database collisions; when form is completed, blank slug automatically populates with the derived slug.
- Explicit slug input validates syntax and collision availability immediately.
- Added unit tests in `internal/tui/wizard_test.go`.
