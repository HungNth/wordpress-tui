# 07: Fall back to a stale Package Cache during transient failures

Status: resolved
Blocked by: 06
Parent spec: ../spec.md

## What to build

Keep local Website creation moving when Package metadata or download infrastructure is temporarily unavailable by using the latest locally verified older archive, while refusing fallback for permanent, unsafe, or malformed responses.

## Acceptance criteria

- [x] A valid cached artifact may be used after DNS/connection interruption, timeout, HTTP 408, HTTP 429, HTTP 5xx, or an interrupted download.
- [x] If metadata identifies a newer version but downloading it fails transiently, the previous valid cached version may be used.
- [x] Authentication/authorization errors, other contract-level 4xx responses, invalid metadata, identity mismatch, size/integrity failure, and security-policy rejection never use stale fallback.
- [x] A corrupt or missing cached artifact cannot be used as fallback.
- [x] Stale fallback is available only when Package integration is configured; an empty Package API base URL still disables Package features.
- [x] Progress and completion results identify the actual stale version and the transient reason without exposing signed URL queries or credentials.
- [x] Partial failed downloads are removed while the previous valid cache artifact remains intact.
- [x] Tests cover every accepted transient class, representative non-transient refusals, a known-newer download failure, and summary reporting.

## Testing seam

Package resolver fallback decision logic: test with simulated network errors and verify that valid cache artifacts are selected only under allowed transient conditions.

## Demo path

Populate cache with a package, simulate server timeout during metadata/download on a subsequent Website creation, observe warning indicating stale cache fallback, and verify successful provisioning.

## Answer

Implemented in `internal/packages/errors.go`, `internal/packages/resolver.go`, `internal/create/create.go`, and `internal/app/app.go`:
- Built `IsTransientError` classifier detecting timeouts, connection resets, EOF/unexpected EOF, and HTTP statuses 408, 429, 500, 502, 503, 504.
- Defined `MetadataContractError` wrapping JSON decode, empty body, truncated response, and schema mismatch errors so invalid metadata is strictly non-transient and never triggers stale fallback.
- Fallback logic in `Resolver.ResolvePackage`: on transient metadata or download failures, if an uncorrupted verified cache entry exists, returns it with `IsStale = true` and `StaleReason` without deleting the valid archive.
- Non-transient errors (401, 403, 404, invalid metadata JSON, SSRF rejection) strictly halt and reject fallback.
- Summary in `app.go` and `Create` result records and reports `StalePackages` (with type, slug, version, and sanitized reason) to the user.
- Verified with unit tests in `internal/packages/fallback_test.go`.
