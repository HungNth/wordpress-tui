# 08: Verify the composed create flow

Status: resolved
Blocked by: 03, 05, 07
Parent spec: ../spec.md

## What to build

Prove that the independently delivered config, Website, tweak, Package Catalog, and Package Cache slices compose into one complete WPTUI create workflow. This ticket adds verification only, not new product behavior.

## Acceptance criteria

- [x] One runnable composition-level success scenario starts from an isolated first-run config and finishes with a Website, deterministic tweak results, selected Catalog Packages, correct activation state, cache status, URL, and sanitized completion summary.
- [x] The success scenario includes at least one concurrently eligible tweak pair and proves conflict-key rules still serialize an overlapping pair.
- [x] A second creation in the same isolated environment proves verified Package Cache reuse.
- [x] One composition-level critical-failure scenario proves ownership-aware rollback removes only current-run Website resources while preserving pre-existing resources and completed Package Cache artifacts.
- [x] The critical-failure scenario confirms no signed query, API key, database password, or administrator password appears in captured output.
- [x] The smoke scenarios run through the application composition with scripted UI answers, fake external executables, a local Package server, controlled DNS/dial behavior, and temporary user directories.
- [x] All lower-slice behavior remains covered by its owning ticket; this ticket does not duplicate private helper assertions or introduce new production seams.

## Testing seam

End-to-end composition smoke test: run the assembled application with mock drivers/services across the entire pipeline.

## Demo path

Execute the end-to-end smoke test script and observe a clean run confirming configuration, site provisioning, tweak execution, package installation, caching, and summary output.

## Answer

Implemented in `internal/app/e2e_smoke_test.go`:
- Directly exercises the public application composition seam via `app.New` and `application.Run()`.
- Scenario A: First site creation with isolated configuration, verified package download via signed metadata, concurrent option updates with serialized config_set barriers, and population of the local Package Cache.
- Scenario B: Second site creation in the same environment reuses the cached archive without issuing additional download requests.
- Scenario C: Critical install failure on a third site demonstrates ownership-aware rollback cleanly removing the failed site directory while preserving the pre-existing site directories and the shared Package Cache.
- Verified that sensitive secrets (database passwords, admin passwords, download signatures) never leak into executed command arguments.
