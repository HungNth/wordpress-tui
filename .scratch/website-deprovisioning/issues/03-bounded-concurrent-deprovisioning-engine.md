# 03: Bounded Concurrent De-provisioning Engine

Status: ready-for-agent
Blocked by: 01-core-discovery-and-single-site-deprovisioning.md
Parent spec: ../spec.md

## What to build

Implement the bounded concurrent de-provisioning execution engine:
1. Concurrent execution bounded to a maximum of 4 workers (`min(4, count)`) to prevent overloading system I/O or MySQL connection limits.
2. Execution lifecycle per website: unsecure Herd TLS (when enabled, best-effort), drop the database (when accurately identified), and mandatorily delete the directory regardless of prior resource errors.
3. Isolated error handling ensuring a failure on one website never halts other queued website operations.
4. Structured per-resource outcome mapping for Herd, Database, and Directory.

## Acceptance criteria

- [ ] Concurrency is bounded to a maximum of 4 concurrent workers.
- [ ] Partial failures on one website do not abort or disrupt other running deletions.
- [ ] Context cancellation terminates queued operations cleanly.
- [ ] Unit tests verify bounded concurrency under race detector, failure isolation, and result mapping.

## Testing seam

- `internal/deprovision/deprovision_test.go`: test concurrent de-provisioning of multiple sites under `-race`.

## Demo path

Run de-provisioning across 5 test directories concurrently; observe that max concurrency is bounded to 4 and all sites produce distinct per-resource results.
