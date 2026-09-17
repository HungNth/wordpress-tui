# 03: End-to-End Composed Integration and Verification

Status: resolved
Blocked by: 02-selection-accumulating-multi-query-search-flow.md
Parent spec: ../spec.md

## What to build

Integrate the head search option and multi-query selection accumulator into the end-to-end website creation and provisioning workflow, verifying that plugins and themes selected via the new search experience are correctly staged, downloaded, and installed without regression.

## Acceptance criteria

- [x] End-to-end creation smoke tests verify that both default packages and accumulated searched packages are correctly installed.
- [x] Theme selection and plugin selection flows both support the head search trigger and selection accumulator.
- [x] Entire test suite passes cleanly under `-race` with 0 data races.
- [x] `go vet ./...` reports 0 warnings.

## Testing seam

- `internal/app/e2e_smoke_test.go`: full end-to-end website provisioning smoke test verifying plugin and theme search installation.

## Demo path


## Answer

Composed and verified end-to-end live package search:
1. Both plugin and theme selection seamlessly benefit from top-positioned search and multi-query selection accumulation through `SelectPackagesFlow`.
2. All packages in the repository pass `go test -v -race ./...` with 0 data race warnings.
3. Static analysis with `go vet ./...` passes with 0 warnings.
Run `wptui`, create a site, select a default plugin and search-accumulate 2 extra plugins, select a searched theme, and observe complete successful provisioning.
