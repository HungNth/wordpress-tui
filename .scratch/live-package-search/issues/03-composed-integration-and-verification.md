# 03: End-to-End Composed Integration and Verification

Status: ready-for-agent
Blocked by: 02-selection-accumulating-multi-query-search-flow.md
Parent spec: ../spec.md

## What to build

Integrate the head search option and multi-query selection accumulator into the end-to-end website creation and provisioning workflow, verifying that plugins and themes selected via the new search experience are correctly staged, downloaded, and installed without regression.

## Acceptance criteria

- [ ] End-to-end creation smoke tests verify that both default packages and accumulated searched packages are correctly installed.
- [ ] Theme selection and plugin selection flows both support the head search trigger and selection accumulator.
- [ ] Entire test suite passes cleanly under `-race` with 0 data races.
- [ ] `go vet ./...` reports 0 warnings.

## Testing seam

- `internal/app/e2e_smoke_test.go`: full end-to-end website provisioning smoke test verifying plugin and theme search installation.

## Demo path

Run `wptui`, create a site, select a default plugin and search-accumulate 2 extra plugins, select a searched theme, and observe complete successful provisioning.
