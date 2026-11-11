# 02: Flock Website lifecycle

**Parent specification:** `../spec.md`

**What to build:** Add Flock as a complete Local Development Environment choice. A Flock user can configure WPTUI, Provision and restore linked HTTPS Websites, open them through the operating system, and De-provision them without stale Flock registrations.

**Blocked by:** 01: Selected environment schema and Herd baseline.

**Status:** completed

## Acceptance criteria

- [x] Windows and macOS setup offer Flock with editable `$HOME/Sites` and `flock` suggestions.
- [x] Dependency checks invoke the configured executable directly and report a missing Flock command before mutation.
- [x] Provisioning links the Website from its directory and then secures its slug; either failure is critical and triggers ownership-aware rollback.
- [x] Rollback unsecures and unlinks only Flock resources created by the failed run and preserves unrelated pre-existing resources.
- [x] Website Restoration treats link failure as critical, while secure failure retains the restored Website, reconciles its WordPress URL to HTTP, and reports a warning.
- [x] De-provisioning attempts unsecure and unlink on a best-effort basis without blocking verified database or mandatory directory removal.
- [x] Successful Flock operations report `https://<slug>.test`; a restoration with failed TLS reports its retained HTTP URL.
- [x] `Open in Browser` uses the operating-system launcher; no `flock open` command or substitute is introduced.
- [x] Progress and result output identify Flock operations without Herd wording and preserve partial-success detail.
- [x] Direct command execution uses fixed argument vectors and the correct Website working directory; no shell command parsing is introduced.

## Testing seam

Use the composed application seam with a runner spy to exercise successful Provisioning, restoration TLS warning, rollback ownership, and best-effort De-provisioning. Assert user-visible outcomes and command contracts, not adapter internals.

## Demo path

Select Flock during first-run setup, Provision a Website while observing link and secure progress, open its HTTPS URL through the system browser launcher, then De-provision it and observe unsecure/unlink, database, and directory outcomes.

## Verification

- `go test ./...` passed, including Flock path-collision protection, pre-existing-resource ownership, failed-link cleanup, surfaced rollback failures, and composed create/delete behavior.
- The installed Flock executable was exercised read-only through `sites`; the production TLS parser correctly identified an existing HTTPS Website. No real registration, certificate, database, or Website was changed.
