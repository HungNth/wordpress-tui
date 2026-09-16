# 05: Select multiple Packages from the Package Catalog

Status: resolved
Blocked by: 04
Parent spec: ../spec.md

## What to build

Let a user combine configured default Packages with additional Catalog search results, install multiple plugins and themes in one create flow, and receive accurate activation and skip reporting.

## Acceptance criteria

- [x] The create flow independently asks whether to install plugins and additional themes.
- [x] Configured plugin and theme choices are displayed without preselecting every item.
- [x] The Package Catalog endpoint preserves the configured base-path prefix and returns the complete Catalog in one request.
- [x] Huh provides a repeatable search loop: enter a query, filter results in memory, multi-select matches, and return to the selection menu.
- [x] Search is case-insensitive across Package name and slug; generic entries are hidden; plugin and theme pickers enforce exact Package type.
- [x] Configured and searched selections are deduplicated by Package type and slug.
- [x] Every selected Package resolves through the signed-metadata behavior from Ticket 04 before Website mutation.
- [x] Selected plugins are installed and activated; the configured default theme is installed and activated; additional themes are installed without activation.
- [x] When the Package API base URL is empty, Package selection and cache use are disabled, the bundled WordPress theme remains active, and the summary explicitly reports the skipped configured default theme.
- [x] Progress and errors remain sanitized, and a critical Package install failure triggers ownership-aware rollback.
- [x] A runnable demo proves combined default/search selection, deduplication, activation policy, and the API-disabled outcome.

## Testing seam

Catalog client and multi-package selection form model: test catalog filtering, type segregation, selection deduplication, and bulk resolution.

## Demo path

Trigger creation with plugin and theme choices, use the search loop to pick an additional plugin, verify both default and searched items install, and observe correct activation states in the summary.

## Answer

Implemented `internal/packages/catalog.go`, `internal/tui/packages_picker.go`, and updated `internal/create/create.go` and `internal/app/app.go`:
- Catalog endpoint preserves URL base prefix (`packages`) and filters out `generic` items.
- In-memory case-insensitive search by package name and slug for exact matching types.
- Huh interactive search loop allowing repeated search, multi-selection, and automatic deduplication by slug and type.
- Clean integration in `Create`: pre-mutation batch resolution, plugin activation, default theme activation, additional theme installation without activation, and explicit summary notice when default theme is skipped because Package API is disabled.
- Verified with unit tests in `internal/packages/catalog_test.go` and `internal/tui/packages_picker_test.go`.
