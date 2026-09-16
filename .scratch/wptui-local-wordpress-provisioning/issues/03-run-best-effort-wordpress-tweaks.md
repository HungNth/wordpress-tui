# 03: Run WordPress tweaks concurrently and safely

Status: resolved
Blocked by: 02
Parent spec: ../spec.md

## What to build

Let a user opt into configured WordPress tweaks and receive a usable Website even when individual low-impact tweaks fail. Independent tweaks run concurrently through an explicit, testable conflict-key scheduler; conflicts and dependencies are serialized safely.

## Acceptance criteria

- [x] The create flow offers one opt-in choice for all configured tweaks and skips every tweak when declined.
- [x] The scheduler runs at most four tweak commands concurrently.
- [x] Conflict matching supports exact keys, a global `*` barrier, and namespace wildcards such as `db:*`.
- [x] Config-set tweaks use the global barrier and therefore run sequentially and exclusively from every other WP-CLI tweak command.
- [x] Rewrite-structure tweaks use the database wildcard and conflict with every database-mutating tweak.
- [x] Option-update tweaks use an option-specific database key; distinct option keys may run concurrently, while duplicate writes to one key remain serialized in configured order.
- [x] Core-language install uses a locale-specific file key; activation uses the `WPLANG` option key and runs only after the matching install succeeds.
- [x] A failed tweak is captured with sanitized error details, does not rollback the Website, and does not prevent conflict-free tweaks from running.
- [x] A failed prerequisite skips only its dependent tweak and records the cause.
- [x] Progress and final results are rendered deterministically in configured order regardless of start or completion order.
- [x] User cancellation remains fatal and follows the critical ownership-aware rollback policy.
- [x] Tests cover wildcard/exact conflict detection, rewrite-versus-option exclusion, config-set-versus-any exclusion, distinct-key parallelism, overlap serialization, dependency skips, continue-on-error, and Website retention after tweak failure.

## Testing seam

Tweak scheduler engine and WP-CLI tweak execution seam: test with simulated commands observing concurrency limits, conflict blocks, dependency chains, and error resilience.

## Demo path

Create a Website with tweaks opted in, observe progress displaying concurrent steps and deterministic results, and verify that a failing non-critical tweak is reported in the final summary while the Website is kept.

## Answer

Implemented in `internal/create/tweaks.go` and `internal/wpcli/wpcli.go`:
- Built conflict-key scheduler with global barrier `*`, namespace wildcard `db:*`, `option:<key>`, and file locks `fs:language:<locale>`.
- 2-pass dependency mapping for language install/activate ensuring activation always waits for matching install (even out-of-order) and marks activation skipped if install was never configured.
- Bounded concurrency with at most 4 parallel workers.
- Non-fatal error handling: failed tweaks are captured in `Result.FailedTweaks`, dependent tweaks are marked skipped, while conflict-free tweaks proceed and the Website is preserved.
- Full cancellation safety: worker goroutines are cleanly waited via `sync.WaitGroup` before `RunTweaks` returns on context cancellation, ensuring no background operations race against rollback.
- Unit and integration tests in `internal/create/tweaks_test.go` and `internal/create/create_test.go`.
