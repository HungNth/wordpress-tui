# 02: Unified Website Name and Slug input in create wizard

Status: ready-for-agent
Blocked by: None (can start immediately)
Parent spec: ../spec.md

## What to build

Unify the `Website Name` and `Website Slug` inputs into a single interactive form view instead of two separate steps. If the user leaves `Website Slug` blank, it is automatically derived from `Website Name`. If a custom slug is entered, it is validated for syntax and availability immediately.

## Acceptance criteria

- [ ] `Website Name` and `Website Slug` are presented together in the initial group of the create form.
- [ ] Leaving `Website Slug` blank is permitted; on submission it automatically derives from `Website Name` via `create.Slugify(inputs.WebsiteName)`.
- [ ] If a slug is entered explicitly, it is validated for syntax and directory/database availability.
- [ ] When the slug derives from name, availability is verified before advancing to administrator credentials.
- [ ] Unit tests in `internal/tui` verify unified form behavior, auto-derivation when blank, and explicit slug overriding.

## Testing seam

- `internal/tui`: Unit tests for `BuildCreateForm` confirming empty slug derivation and explicit slug validation.

## Demo path

Run `wptui`, choose Create, and observe `Website Name` and `Website Slug` on the same screen; leave slug blank and observe it automatically use the slugified name.
