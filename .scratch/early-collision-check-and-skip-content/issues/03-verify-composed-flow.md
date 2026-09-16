# 03: Verify composed create flow with lean download and early collision check

**What to build:**
Verify the end-to-end composed `create` flow incorporating both `--skip-content` core downloads and early interactive collision checks, proving that existing sites are rejected at the slug prompt and new sites are provisioned without bundled themes or plugins.

**Blocked by:** 01 (Skip content during WordPress core download), 02 (Early interactive Website collision check in TUI wizard)

**Status:** ready-for-agent

## Acceptance criteria

- [ ] Interactive create flow rejects colliding slugs at the slug input step without prompting for packages or tweaks.
- [ ] End-to-end website creation downloads WordPress core with `--skip-content`.
- [ ] Explicitly selected packages and default themes install and activate properly on top of the lean core.
- [ ] E2E smoke tests and race detector tests pass across all packages with 0 race warnings.

## Testing seam

- `internal/app/e2e_smoke_test.go`: full flow test simulating user inputs and verifying early collision handling as well as clean `--skip-content` provisioning.

## Demo path

Run `go test -v -race ./...` and observe that all composed provisioning and collision tests pass.
