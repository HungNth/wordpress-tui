# 04: Verify composed UI/UX flow

Status: ready-for-agent
Blocked by: 01, 02, 03
Parent spec: ../spec.md

## What to build

Verify the composed interactive experience incorporating the bright cyan focus indicator, unified name and slug form, and inline catalog search across the complete end-to-end flow.

## Acceptance criteria

- [ ] Interactive form views display the bright cyan focus border.
- [ ] Unified form creates websites correctly with blank slug auto-derivation or custom slug input.
- [ ] Package selection smoothly flows directly from defaults or inline search without intermediate confirmation dialogs.
- [ ] All package unit tests and race detector tests pass with 0 race warnings.

## Testing seam

- `internal/app/e2e_smoke_test.go` and `internal/tui/*_test.go`.

## Demo path

Run `go test -v -race ./...` and observe that all tests pass.
