# 03: Inline catalog search in package picker

Status: resolved
Blocked by: None (can start immediately)
Parent spec: ../spec.md

## What to build

Integrate package catalog search directly into the package MultiSelect choices as a terminal option (`[🔍 Type to search catalog...]`), eliminating the separate confirmation prompt (`Search catalog? [Yes/No]`).

## Acceptance criteria

- [x] When catalog items are available, a terminal choice `🔍 Type to search catalog...` (value `__search__`) is appended to the default MultiSelect options.
- [x] If `__search__` is not selected, package selection completes immediately without any extra confirmation prompt.
- [x] If `__search__` is selected, `__search__` is excluded from selected packages and the catalog query input is displayed immediately.
- [x] Search results also offer a terminal option to search again if more packages are desired.
- [x] Unit tests in `internal/tui` verify that `BuildPackageOptions` appends `__search__` last without disturbing default options order, and `ExtractSelectedPackages` extracts `wantsSearch` cleanly.

## Testing seam

- `internal/tui/packages_picker_test.go`: test `BuildPackageOptions` and `ExtractSelectedPackages` with and without `__search__` option.

## Demo path

Run `wptui`, choose Create, proceed to package selection, select default plugins without selecting search, and observe flow completes immediately without asking `Search catalog?`.

## Answer

Implemented inline catalog search in `internal/tui/packages_picker.go`:
- `BuildPackageOptions` appends `🔍 Type to search catalog...` (value `__search__`) when catalog items are present.
- `ExtractSelectedPackages` strips `__search__` from chosen packages and detects whether the search prompt should run.
- Eliminated separate `Search catalog?` confirmation form.
- Added unit tests in `internal/tui/packages_picker_test.go`.
