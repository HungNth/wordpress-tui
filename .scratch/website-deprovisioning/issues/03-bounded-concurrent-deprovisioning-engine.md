# 03: Bounded Concurrent De-provisioning Engine

Status: resolved
Blocked by: 01-core-discovery-and-single-site-deprovisioning.md
Parent spec: ../spec.md

## What to build

Implement the bounded concurrent de-provisioning execution engine:
1. Concurrent execution bounded to a maximum of 4 workers (`min(4, count)`) to prevent overloading system I/O or MySQL connection limits.
2. Execution lifecycle per website: unsecure Herd TLS (when enabled, best-effort), drop the database (when accurately identified), and mandatorily delete the directory regardless of prior resource errors.
3. Isolated error handling ensuring a failure on one website never halts other queued website operations.
4. Structured per-resource outcome mapping for Herd, Database, and Directory.

## Acceptance criteria

- [x] Concurrency is bounded to a maximum of 4 concurrent workers.
- [x] Partial failures on one website do not abort or disrupt other running deletions.
- [x] Context cancellation terminates queued operations cleanly.
- [x] Unit tests verify bounded concurrency under race detector, failure isolation, and result mapping.

## Testing seam

- `internal/deprovision/deprovision_test.go`: test concurrent de-provisioning of multiple sites under `-race`.

## Demo path


## Answer
Implemented bounded concurrent de-provisioning:
1. `Deprovision` engine function in `internal/deprovision/deprovision.go` with semaphore channel capacity capped to max 4 workers (`min(4, len(candidates))`).
2. Isolated per-site lifecycle: Herd unsecure -> DB drop -> mandatory directory removal.
3. Context cancellation handling marking queued candidates aborted cleanly.
4. Unit tests in `internal/deprovision/deprovision_test.go` with race detector: `TestDeprovision_ConcurrentBounded` measuring peak active concurrency <= 4 with partial failures on site-3 and site-5, and `TestDeprovision_CancellationAbortsRemaining`.
