# 04: Install one Package from signed metadata

Status: resolved
Blocked by: 02
Parent spec: ../spec.md

## What to build

Let a user select one configured plugin or the configured default theme and receive a Website with that Package installed from a securely resolved, signed metadata download. Package resolution completes before Website mutation, and Package failure remains a critical rollback condition.

## Acceptance criteria

- [x] Package API base URLs preserve configured path prefixes for metadata requests, including trailing-slash variations.
- [x] Optional license keys are query-encoded when present and omitted entirely when empty.
- [x] Metadata type, slug, version, decimal-string size, and HTTPS download URL are validated before download.
- [x] The signed download URL is requested exactly as returned; WPTUI neither strips nor appends query/authentication data and never stores the signed URL.
- [x] Signed URL queries are redacted from progress and errors.
- [x] Downloads enforce the accepted 1-GiB artifact cap, exact metadata size, optional Content-Length agreement, readable ZIP structure, and cleanup of every partial file.
- [x] Redirects are capped at five, use only the server Location for the next URL, do not copy source credentials, and suppress Referer.
- [x] Every resolved address at the initial URL and each redirect is checked; mixed public/private results are rejected, and the connection is pinned to a validated address without a second DNS resolution.
- [x] Remote type and slug cannot escape the managed download location through separators, control characters, dot segments, or unsafe joins.
- [x] The verified local artifact is ready before Website directory or database mutation begins.
- [x] A selected plugin is installed and activated; the configured default theme is installed and activated when Package integration is enabled.
- [x] Download, validation, install, and cancellation failures produce sanitized output, clean temporary files, and trigger critical Website rollback after mutation begins.
- [x] Resolver tests cover prefixed base URLs, exact signed queries, redirect credential isolation, DNS rebinding/multi-address rejection, size limits, invalid ZIPs, and interrupted transfers.

## Testing seam

Package resolver and download security seam: test against local HTTP servers with controlled DNS/dial adapters verifying SSRF checks, redirect hygiene, size bounds, and rollback triggers.

## Demo path

Create a Website selecting one configured plugin, observe metadata resolution and download before filesystem mutation, and verify the plugin is installed and activated via WP-CLI.

## Answer

Implemented `internal/packages`:
- Path-preserving metadata URL builder resolving relative references (`package/{slug}/metadata`) against trailing-slash normalized base URLs without stripping `/api/v1/`.
- License key encoding: attached as query parameter when provided, omitted completely when empty.
- Comprehensive metadata validation: type match, slug match, non-empty version, positive byte size <= 1 GiB, HTTPS download URL.
- Strict SSRF protection: DNS resolution iterates all addresses, rejects loopback, private RFC1918, IPv6 ULA `fc00::/7`, link-local, unspecified, and cloud metadata (`169.254.169.254`) with zero bypass switch in production. Connections are pinned to the validated address.
- 5-redirect limit with HTTPS enforcement, `Referer` suppression, and query credential isolation.
- Pre-mutation resolution in `Creator.Create`: all packages are resolved into staging artifacts before directory or database creation begins; failure halts without leaving partial Websites.
- Plugin installation and activation via `wp plugin install <zip> --activate`; default theme installed and activated via `wp theme install <zip> --activate`.
- Validated with unit tests in `internal/packages/download_test.go`, `internal/packages/resolver_test.go`, and `internal/create/create_test.go`.
