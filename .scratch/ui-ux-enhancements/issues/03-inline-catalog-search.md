# 03: Inline catalog search in package picker

Status: ready-for-agent
Blocked by: None (can start immediately)
Parent spec: ../spec.md

## What to build

Integrate package catalog search directly into the package MultiSelect choices as a terminal option (`[🔍 Type to search catalog...]`), eliminating the separate confirmation prompt (`Search catalog? [Yes/No]`).

## Acceptance criteria

- [ ] When catalog items are available, a terminal choice `🔍 Type to search catalog...` (value `__search__`) is appended to the default MultiSelect options.
- [ ] If `__search__` is not selected, package selection completes immediately without any extra confirmation prompt.
- [ ] If `__search__` is selected, `__search__` is excluded from selected packages and the catalog query input is displayed immediately.
- [ ] Search results also offer a terminal option to search again if more packages are desired.
- [ ] Unit tests in `internal/tui` verify that `SelectPackagesFlow` completes cleanly without search when `__search__` is unselected, and triggers search when selected.

## Testing seam

- `internal/tui/packages_picker_test.go`: test `SelectPackagesFlow` with and without `__search__` option.

## Demo path

Run `wptui`, choose Create, proceed to package selection, select default plugins without selecting search, and observe flow completes immediately without asking `Search catalog?`.
