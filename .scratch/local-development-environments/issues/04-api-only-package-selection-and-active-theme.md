# 04: API-only Package selection and Active Theme

**Parent specification:** `../spec.md`

**What to build:** Make the configured Package Service the sole source of user-selectable plugins and themes. Package choices appear only from a successfully loaded Package Catalog, while Website Provisioning remains usable without Packages or an Active Theme.

**Blocked by:** 01: Selected environment schema and Herd baseline.

**Status:** completed

## Acceptance criteria

- [x] Static theme/plugin configuration types, seeded names, option builders, default-theme behavior, and their obsolete tests are removed completely.
- [x] A missing Package Service URL omits Package selection from Create and hides plugin/theme actions for existing Websites.
- [x] A failed, malformed, unauthorized, or empty catalog request produces one concise redacted warning and hides unavailable Package choices for the session without automatic retry.
- [x] Plugin and theme actions are gated independently by whether the loaded catalog contains the relevant Package type.
- [x] Website Provisioning with no available catalog completes without Packages or an Active Theme and reports that state without failing an active-theme lookup.
- [x] A usable catalog provides separate searchable plugin and theme selections while preserving choices across search queries.
- [x] Users may select multiple themes and optionally choose exactly one of those selected themes as the Active Theme; selection order never determines activation.
- [x] Existing-Website theme installation offers an explicit Active Theme choice instead of activating multiple themes in sequence.
- [x] Every selectable Package originates from the loaded catalog; resolution failure never falls back to installation by WordPress.org slug.
- [x] Verified Package Cache and established stale-cache behavior remain available after catalog-based selection.
- [x] Provisioning resolves every selected artifact before mutation, while existing-Website installation retains per-Package success/failure outcomes without source substitution.
- [x] Package choices, errors, progress, and summaries follow the terminal design's cancellation, warning, secret-redaction, and colorless-readability rules.

## Testing seam

Use composed application tests with deterministic HTTP catalog fixtures and injected Package resolvers. Add narrow TUI state tests for conditional actions, separate selections, accumulation across searches, cancellation, and optional Active Theme behavior.

## Demo path

Run once without a Package Service and create a Website with no Package step. Run again with a fixture catalog, select plugins and multiple themes through separate searches, choose one Active Theme, and observe deterministic installation. Exercise a catalog failure and confirm the warning plus hidden actions.

## Verification

- `go test ./...` passed, including absent-API composed workflows, catalog gating, explicit Active Theme selection, no-theme success, and existing Package Cache behavior.
- Actual fixture-backed terminal sessions showed separate Plugins and Themes searches, optional single Active Theme choices, hidden installation actions and a direct Create button without an API, and corrected contextual keyboard help.
- A live external Package Service was not contacted. Catalog requests and downloads were exercised with deterministic test fixtures.
