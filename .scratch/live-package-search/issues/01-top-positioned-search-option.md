# 01: Top-Positioned Search Option

Status: resolved
Blocked by: None (can start immediately)
Parent spec: ../spec.md

## What to build

Position the catalog search option as the very first item in the package MultiSelect choices list whenever catalog items are available, ensuring developers immediately see the search capability without having to scroll to the bottom of default packages.

## Acceptance criteria

- [x] When catalog items are available, the search trigger option is prepended at index 0 of the choices list.
- [x] Default package options follow immediately after the search option in their original order.
- [x] When catalog items are not available, the search option is not added and default package options remain unmodified.
- [x] Selecting both the search option and one or more default packages properly extracts both the requested default slugs and the search trigger.
- [x] Unit tests verify option positioning at index 0 and clean choice extraction.

## Testing seam

- `internal/tui/packages_picker_test.go`: test `BuildPackageOptions` and `ExtractSelectedPackages`.

## Demo path


## Answer

Implemented top-positioned search option in `internal/tui/packages_picker.go`:
1. Updated `BuildPackageOptions` to prepend `🔍 Type to search catalog...` at index 0 whenever `hasCatalog = true`.
2. Maintained original ordering of all default package options immediately following the search option.
3. Kept default options unchanged when `hasCatalog = false`.
4. Updated and verified unit tests in `internal/tui/packages_picker_test.go` (`TestInlineSearchOptionAndExtraction`).
Run `wptui`, choose `Create`, progress to package selection, and observe that `🔍 Type to search catalog...` appears as the very first option above all default packages.
