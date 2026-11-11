# 05: Composed terminal verification and documentation cutover

**Parent specification:** `../spec.md`

**What to build:** Complete the cutover by proving the selected Local Development Environment and Package Catalog behavior together on the actual terminal surface, aligning current user documentation, and removing remaining obsolete Herd-only and static-Package guidance.

**Blocked by:** 02: Flock Website lifecycle; 03: Laragon Website lifecycle; 04: API-only Package selection and Active Theme.

**Status:** completed

## Acceptance criteria

- [x] Composed scenarios cover first-run selection and one successful create/open/delete lifecycle for Herd, Flock, and Laragon using isolated command and filesystem fixtures.
- [x] Composed failure scenarios cover critical registration/reload rollback, Provisioning TLS rollback, Restoration TLS warning with HTTP reconciliation, best-effort cleanup, absent catalog, failed catalog, no Active Theme, and explicit Active Theme.
- [x] Actual terminal verification exercises environment selection, Back/Cancel, Package controls shown and hidden, Active Theme choice, environment-neutral progress/results, 80x24 handling, and colorless readability.
- [x] Verification records real Herd/Flock/Laragon executables or external Package services that could not be exercised rather than claiming simulated tests as real integration.
- [x] Current README and terminal design documentation describe the three environments, platform availability, runtime prerequisites, setup fields, URL policies, Laragon Auto SSL assumption, Package availability, and no-Active-Theme consequence accurately.
- [x] Current documentation and examples no longer present Herd as the default or describe static Package names/default themes as configuration.
- [x] The domain glossary and ADR-0017 remain aligned with implemented behavior; superseded Herd-only ADR decisions are not treated as current guidance.
- [x] Shared source, fixtures, output labels, and active documentation contain no obsolete `used_herd` contract or configuration-provided Package catalogs.
- [x] The narrow affected test commands pass once after the actual terminal smoke check; no unrelated broad suite or formatter is added to this ticket.

## Testing seam

Use the composed application seam for behavioral proof and the actual TUI for visual/interaction proof. Permanent tests assert observable state, command contracts, rollback safety, URL outcomes, and available actions rather than wiring or source text.

## Demo path

Start from no configuration, walk each platform-appropriate environment setup and lifecycle with isolated fixtures, compare Package-disabled and Package-enabled sessions, exercise one accepted warning path, and inspect the final reports plus updated user documentation.

## Verification

- Final `go test ./...` passed across all packages. `go build` for the actual WPTUI command succeeded. Implementation-workflow final suite includes unchanged-package regression coverage; changed-package checks were also exercised during integration.
- Actual Windows PTY: required setup sentinel rejection, explicit Flock selection and path/command suggestions, clean setup cancellation, API-enabled and API-disabled Website actions, separate plugin/theme searches, optional Active Theme, Back navigation, fixture-driven stepper/log/result output, corrected keyboard help, NO_COLOR readability, and the 79x23 minimum-size notice.
- Terminal lifecycle progress was driven by an isolated fixture harness, not real Website provisioning. The harness and the read-only Flock probe were deleted afterward.
- Installed Flock status/TLS parsing was exercised read-only. Real Herd/Flock create/secure/unlink/delete, Laragon reload/Auto SSL, macOS terminal behavior, and a live external Package Service were not exercised. Destructive behavior was verified only with isolated runners and temporary filesystem fixtures.
- Standards/spec reviews found and prompted fixes for rollback ordering, swallowed cleanup errors, redundant TLS mutations, setup validation fallbacks, progress ordering, and contextual keyboard help. Changes remain uncommitted.
