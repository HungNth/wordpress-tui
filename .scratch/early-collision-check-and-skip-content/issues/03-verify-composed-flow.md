# 03: Verify composed create flow with lean download and early collision check

Status: ready-for-agent
Blocked by: 01, 02
Parent spec: ../spec.md

## What to build

Verify the end-to-end composed `create` flow incorporating both early interactive collision checks and `--skip-content` core downloads, proving that colliding sites are rejected early at the slug prompt and new sites are provisioned cleanly without bundled themes or plugins.

## Acceptance criteria

- [ ] Interactive create flow rejects colliding slugs at the slug input step without prompting for packages or tweaks.
- [ ] End-to-end website creation downloads WordPress core with `--skip-content`.
- [ ] Completed website filesystem verification confirms that default bundled themes (`wp-content/themes/twentytwenty*`) and default plugins (`wp-content/plugins/hello.php`, `akismet`) are absent.
- [ ] Explicitly selected packages and default themes install and activate properly on top of the lean core.
- [ ] E2E smoke tests and race detector tests pass across all packages with 0 race warnings.

## Testing seam

- `internal/app/e2e_smoke_test.go`: full flow test simulating user inputs and verifying early collision handling as well as clean `--skip-content` provisioning with observable filesystem checks.

## Demo path

Run `go test -v -race ./...` and observe that all composed provisioning and collision tests pass.
