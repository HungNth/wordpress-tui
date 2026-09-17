# 01: Top-Positioned Search Option

Status: ready-for-agent
Blocked by: None (can start immediately)
Parent spec: ../spec.md

## What to build

Position the catalog search option as the very first item in the package MultiSelect choices list whenever catalog items are available, ensuring developers immediately see the search capability without having to scroll to the bottom of default packages.

## Acceptance criteria

- [ ] When catalog items are available, the search trigger option is prepended at index 0 of the choices list.
- [ ] Default package options follow immediately after the search option in their original order.
- [ ] When catalog items are not available, the search option is not added and default package options remain unmodified.
- [ ] Selecting both the search option and one or more default packages properly extracts both the requested default slugs and the search trigger.
- [ ] Unit tests verify option positioning at index 0 and clean choice extraction.

## Testing seam

- `internal/tui/packages_picker_test.go`: test `BuildPackageOptions` and `ExtractSelectedPackages`.

## Demo path

Run `wptui`, choose `Create`, progress to package selection, and observe that `🔍 Type to search catalog...` appears as the very first option above all default packages.
