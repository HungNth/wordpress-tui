# WPTUI v1: Local WordPress Website Provisioning

Status: ready-for-agent

## Problem Statement

Developers repeatedly perform the same error-prone work when creating local WordPress Websites: preparing directories and databases, downloading WordPress, creating administrator accounts, applying preferred WordPress settings, installing licensed plugins and themes, configuring a local `.test` hostname, and recovering from partial failures. Premium Package downloads are also wasteful and unreliable when the same archive is fetched for every Website or a network failure interrupts installation.

The user needs one terminal application, invoked as `wptui`, that stores local-development defaults once and then provisions a complete Website consistently on Windows and macOS. Provisioning must not leave partial Websites, overwrite existing resources, leak credentials, repeatedly download unchanged Packages, or reuse corrupt cached archives.

## Solution

Build a Go terminal application called `wptui`. On first launch it creates a user config through an interactive wizard. Later launches load and validate that config, then present a menu containing `create`, `config`, `delete`, `backup`, `restore`, and `settings`. In v1, only `create` is enabled.

The create flow gathers a Website Name, a separately confirmed Website Slug, optional administrator overrides, tweak preferences, and Package selections. It performs all non-mutating validation and resolves every selected Package into a verified local archive before mutating Website resources. It then downloads WordPress core, generates `wp-config.php`, creates the database, runs `wp core install`, configures Herd TLS when applicable, applies selected tweaks, and installs selected plugins and themes.

Critical Provisioning is atomic. Failure or cancellation in core, database, Herd, Package, or theme steps removes only the Website resources created by that run. Individual WordPress tweak failures are non-fatal: WPTUI reports them, continues remaining eligible tweaks, and keeps the Website. Verified Package Cache artifacts are shared prerequisites and remain available for future runs; incomplete downloads are always removed.

The Package Cache keeps one verified archive per Package type and slug in the operating system's native user cache. Every Package installation checks current metadata. An exact-version cache hit is reused; a changed version is downloaded and atomically replaces the previous version; a valid stale cache may be used only for transient network or server failures.

## User Stories

1. As a WordPress developer, I want to launch the application with `wptui`, so that local Website management has one memorable entry point.
2. As a first-time user, I want WPTUI to detect that no config exists, so that setup begins automatically.
3. As a first-time user, I want WPTUI to create its config under my home `.config/wptui` directory, so that configuration is predictable across Windows and macOS.
4. As a user, I want password fields to be masked during setup, so that nearby observers cannot read my credentials.
5. As a user, I want optional config inputs to use documented defaults, so that initial setup is quick.
6. As a user, I want invalid config JSON or invalid fields to stop startup with a precise error, so that WPTUI never silently overwrites my settings.
7. As a user, I want plaintext secrets redacted from output and errors, so that terminal history does not expose credentials.
8. As a user, I want to see all planned menu options, so that the intended product scope is visible.
9. As a v1 user, I want unfinished menu options visibly disabled, so that I cannot enter incomplete workflows.
10. As a developer, I want to enter a human-readable Website Name, so that WordPress receives an appropriate title.
11. As a developer, I want WPTUI to suggest a Website Slug from the Website Name, so that common names require little typing.
12. As a developer, I want to edit and explicitly confirm the Website Slug, so that directory, database, and hostname identifiers are predictable.
13. As a developer, I want invalid, empty, overlong, or Windows-reserved Website Slugs rejected before mutation, so that provisioning does not fail midway.
14. As a developer, I want WPTUI to check both the destination directory and database before mutation, so that existing Websites are never reused or overwritten.
15. As a developer, I want collisions to return me to the Website identity step, so that I can choose a different Slug safely.
16. As a developer, I want administrator fields prefilled from config, so that repeated local Websites use consistent credentials.
17. As a developer, I want to override the administrator username, password, and email per Website, so that exceptional Websites remain possible.
18. As a Herd user, I want Websites created under the configured websites path and secured with Herd, so that they are available at `https://<slug>.test`.
19. As a non-Herd user, I want WPTUI to work with my preconfigured wildcard local stack, so that Websites are available at `http://<slug>.test`.
20. As a non-Herd user, I want WPTUI to state the wildcard-stack prerequisite during setup, so that `.test` routing is not silently assumed.
21. As a user, I want WPTUI to verify required executables before mutation, so that missing PHP, WP-CLI, MySQL, or Herd fails early.
22. As a user, I want WordPress core installed at the latest stable version, so that new Websites begin current.
23. As a user, I want WordPress installed with locale `en_US` and email notification skipped, so that base installation is deterministic for local development.
24. As a user, I want database creation to succeed before `wp core install` runs, so that WordPress is never installed against a missing database.
25. As a user, I want database-creation failure to leave no Website behind, so that I do not need to clean up a broken directory manually.
26. As a user, I want DB and administrator passwords excluded from process arguments and logs where WP-CLI supports secure stdin prompting, so that local process inspection does not reveal secrets.
27. As a user, I want to opt into the configured WordPress tweaks, so that preferred local defaults can be applied consistently.
28. As a user, I want to skip every tweak with one choice, so that I can create a clean WordPress installation.
29. As a user, I want independent WordPress tweaks to run concurrently without racing on shared files or dependent language operations, so that setup is faster without corrupting the Website.
30. As a user, I want each failed tweak reported while remaining eligible tweaks continue, so that a low-impact customization failure does not remove an otherwise usable Website.
31. As a user, I want to decide whether to install plugins, so that Package integration remains optional per Website.
32. As a user, I want configured default plugins shown without being preselected, so that installation remains an explicit choice.
33. As a user, I want to select multiple plugins, so that a Website can be prepared in one run.
34. As a user, I want to search the Package Catalog by name or slug, so that Packages outside the configured defaults can be selected.
35. As a user, I want generic catalog entries excluded from plugin and theme pickers, so that only installable WordPress Packages appear.
36. As a user, I want selected plugins installed and activated, so that they are immediately usable.
37. As a user, I want the configured default theme installed and activated when Package integration is enabled, so that the Website starts with the preferred theme.
38. As a user, I want additional selected themes installed but not activated, so that only one theme is active.
39. As a user, I want the bundled WordPress theme retained when Package integration is disabled, so that Website creation can still succeed.
40. As a user, I want the completion summary to report that the default theme was skipped when Package integration is disabled, so that the result is not misleading.
41. As a user, I want Package features disabled when no Package API base URL is configured, so that cache contents do not bypass the configured integration state.
42. As a user, I want API requests to omit `license_key` entirely when the configured key is empty, so that unauthenticated API access uses the correct contract.
43. As a user, I want Package Catalog results loaded once and filtered locally, so that selection is responsive.
44. As a user, I want the Huh search workflow to return me to Package selection after each filtered multi-select, so that I can combine defaults and search results.
45. As a user, I want selected Packages deduplicated by type and slug, so that one archive is installed only once.
46. As a user, I want Package artifacts resolved before Website resources are created, so that most network failures occur before rollback is needed.
47. As a user, I want unchanged Package versions reused from cache, so that repeated Website creation avoids repeated downloads.
48. As a user, I want WPTUI to check authoritative metadata before every cache reuse, so that newer Package versions are detected.
49. As a user, I want version equality determined by exact string comparison, so that vendor-specific version formats are not misinterpreted as semantic versions.
50. As a user, I want a changed metadata version downloaded and stored, so that future Websites use the latest Package.
51. As a user, I want only the latest cached version retained for each Package identity, so that cache growth is bounded by the selected Package set.
52. As a user, I want a valid older cached Package used when metadata or download fails transiently, so that temporary network failures do not block local work.
53. As a user, I want stale fallback reported with the actual cached version and reason, so that I know the Website may not use the latest Package.
54. As a user, I want authentication, contract, validation, and security failures to stop installation instead of using stale cache, so that configuration errors and unsafe responses are not hidden.
55. As a user, I want malformed cache manifests quarantined and replaced with a new empty manifest, so that disposable cache metadata does not block WPTUI permanently.
56. As a user, I want corrupt or modified cached archives rejected, so that WP-CLI is not repeatedly given a known-bad file.
57. As a user, I want cached archives checked for regular-file status, expected size, locally recorded SHA-256, and readable ZIP structure, so that local corruption is detected before installation.
58. As a user, I want WPTUI to make clear that the locally recorded SHA-256 detects later local changes but does not authenticate the publisher, so that cache integrity is not confused with supply-chain authenticity.
59. As a user, I want Package downloads capped at 1 GiB, so that a bad response cannot consume unbounded disk space.
60. As a user, I want dynamic package downloads capped at 1 GiB and verified by ZIP structure and Content-Length, so that vendor server-side license stamping does not fail valid package installations.
61. As a user, I want signed `download_url` values used exactly as returned by metadata, so that `license_key`, expiry, and signature parameters remain valid.
62. As a user, I want WPTUI to avoid stripping, appending, or separately authenticating signed download URLs, so that it does not invalidate vendor signatures.
63. As a user, I want signed URL query strings redacted from output, so that API keys and signatures are not leaked.
64. As a user, I want download redirects limited and validated at every hop, so that signed downloads cannot redirect indefinitely or into unsafe network targets.
65. As a user, I want loopback, private, link-local, unspecified, and metadata-service addresses blocked for Package downloads, so that Package metadata cannot trigger local-network SSRF.
66. As a user, I want incomplete download files removed after success, failure, or cancellation, so that cache storage contains no reusable partial archive.
67. As a user, I want a newly downloaded archive fully validated before the cache manifest changes, so that an existing valid version remains recoverable until replacement succeeds.
68. As a user, I want the previous archive deleted only after the new archive and manifest commit successfully, so that cache replacement cannot destroy the last valid Package.
69. As a user, I want completed verified cache artifacts retained if later Website Provisioning fails, so that future runs benefit from completed downloads.
70. As a user, I want concurrent WPTUI processes to serialize cache mutation, so that `data.json` cannot lose or interleave updates.
71. As a user, I want Package type and slug validated before building cache paths, so that remote metadata cannot escape the cache root.
72. As a user, I want manifest paths revalidated when read, so that edited or corrupt cache metadata cannot reference arbitrary files.
73. As a user, I want progress shown as named steps instead of raw command output, so that provisioning remains readable.
74. As a user, I want sanitized command stderr shown when a step fails, so that I can diagnose the cause without exposing secrets.
75. As a user, I want cancellation to terminate active work and run ownership-aware cleanup, so that interrupting WPTUI does not leave partial Websites.
76. As a user, I want a completion summary containing the Website path, URL, installed Packages, skipped default theme, and stale-cache use, so that I can verify the final state.

## Implementation Decisions

- Support Windows and macOS in v1. Linux support is not part of this specification.
- Use `charm.land/huh/v2` in standalone mode for menu choices, inputs, password fields, validation, confirmation, multi-select, accessible mode, and progress indication.
- Implement Package Catalog search as a multi-step Huh form loop: enter query, filter in memory, multi-select results, then return to the selection menu.
- Keep the top-level application module responsible only for config bootstrap, dependency composition, menu routing, and presenting disabled options.
- Keep the terminal-interface module responsible for collecting input and rendering progress; it must not execute WP-CLI, database, Herd, or HTTP operations.
- Keep the create module as the reusable Provisioning orchestrator. Its interface accepts a fully resolved request and reports progress and a result independently of Huh.
- Keep the WP-CLI module responsible for command construction, secret-safe stdin use, process execution, sanitized output, and mapping configured tweaks to supported WP-CLI operations.
- Replace the remote-only Package API module with a deeper Packages module that owns the Package Catalog, metadata lookup, Package Cache, signed downloads, locks, validation, and artifact resolution.
- The Packages module exposes a catalog operation and a bulk resolve operation. The create module receives verified local artifacts and does not coordinate API/cache details.
- Do not create empty modules for `config`, `delete`, `backup`, `restore`, or `settings` menu options. Add each option module only when its behavior is implemented.
- Store config at `~/.config/wptui/config.json` on both supported operating systems.
- The first-run wizard collects `used_herd`, `websites_path`, Package API base URL and key, default administrator fields, and database fields.
- Default `used_herd` to true.
- Default `websites_path` to `~/Herd` when Herd is enabled and `~/Sites` when Herd is disabled.
- Default Package API base URL and key to empty. An empty base URL disables Package selection, metadata lookup, downloads, cache reuse, and default-theme installation.
- Default administrator values to username `admin`, password `admin`, and email `admin@admin.com`.
- Default database values to host `localhost`, port `3306`, username `root`, empty password, and empty socket.
- When a database socket is configured, honor it as the connection endpoint; otherwise use configured host and port.
- Store secrets as plaintext because the accepted config contract requires it, but mask secret input, restrict local file permissions where the operating system allows it, and redact secrets from all output.
- Reject invalid JSON, missing required values, invalid types, unsupported tweak types, and internally inconsistent config. Never overwrite an invalid existing config automatically.
- Seed `default_theme_slug` as `flatsome`.
- Seed theme choices for Flatsome (`flatsome`), Bricks (`bricks`), Etch Theme (`etch-theme`), Woodmart (`woodmart`), Avada (`Avada`), and Jannah (`jannah`).
- Seed plugin choices for Advanced Custom Fields PRO (`advanced-custom-fields-pro`), All-in-One WP Migration Unlimited Extension (`all-in-one-wp-migration-unlimited-extension`), Rank Math SEO PRO (`seo-by-rank-math-pro`), UpdraftPlus (`updraftplus`), WP Mail SMTP Pro (`wp-mail-smtp-pro`), Admin and Site Enhancements Pro (`admin-site-enhancements-pro`), WP Rocket (`wp-rocket`), Perfmatters (`perfmatters`), Duplicator Pro (`duplicator-pro`), FluentCart Pro (`fluent-cart-pro`), Etch (`etch`), and Automatic.css (`automaticcss-plugin`).
- Seed config tweaks for `WP_DEBUG=true`, `WP_DEBUG_LOG=true`, `WP_DEBUG_DISPLAY=true`, `WP_MEMORY_LIMIT='256M'`, `AUTOSAVE_INTERVAL=600`, `WP_POST_REVISIONS=5`, and `EMPTY_TRASH_DAYS=21`, preserving each configured raw/string mode.
- Seed the permalink structure as `/%category%/%postname%/`.
- Seed option updates for timezone `Asia/Ho_Chi_Minh`, time format `H:i`, date format `d/m/Y`, all configured image dimensions and crop values set to `0`, comment moderation enabled, pingback disabled, ping status closed, posts per page `30`, posts per RSS `210`, RSS excerpts enabled, and default avatar `identicon`.
- Seed Vietnamese core language installation and activation as the final configured tweaks.
- Seed backup excludes for `.idea`, `.vscode`, `node_modules`, and `__MACOSX`.
- Seed `wp-content` copy excludes for `.idea`, `.vscode`, `__MACOSX`, `node_modules`, and `cache`.
- Treat Website Name as the WordPress title only.
- Always show a separately editable Website Slug prefilled from the Website Name.
- Validate Website Slug as a 1–63 character ASCII param-case label containing letters, digits, and internal hyphens; reject leading/trailing hyphens and Windows-reserved names.
- Use Website Slug as the directory name, database name, and `.test` hostname.
- Check destination-directory and database collisions before creating either resource. A collision returns to Website identity input and never reuses or deletes the existing resource.
- Require `php`, `wp`, and `mysql` in `PATH`. Require `herd` when Herd mode is enabled.
- In Herd mode, trust the configured `websites_path` directly without calling `herd paths` or `herd park`. Do not call `herd link`. Secure the Website with `herd secure` as the final provisioning step and use `https://<slug>.test`.
- In non-Herd mode, require an external wildcard stack that serves children of `websites_path` at `.test` hostnames. Use `http://<slug>.test`; WPTUI does not configure that stack.
- Install latest stable WordPress core with base locale `en_US`, Website Name as title, the resolved administrator values, and email notification disabled.
- Keep `--skip-check` exclusively on `wp config create`, because the database does not yet exist at config-generation time. Never pass it to `wp core install`.
- Use this database/install ordering: create the temporary Website directory, download core, create `wp-config.php` with `--skip-check`, run `wp db create`, and only after database success run `wp core install`.
- Pass supported WP-CLI secrets through stdin prompting instead of command arguments. Never render secret-bearing command lines.
- If database creation fails, do not run `wp core install`, Herd, tweaks, or Package installation. Remove the temporary directory and config, and drop the database only if post-failure ownership checks show that this run created it.
- Apply WordPress tweaks as best-effort operations with a conflict-key scheduler and at most four active commands. Conflict keys use exact resources plus barriers: `*` conflicts with every key, and a namespace wildcard such as `db:*` conflicts with every key under that namespace. Assign config-set `{*}` because it mutates `wp-config.php` while every WP-CLI command reads that file; rewrite-structure `{db:*}`; option-update `{db:option:<configured-key>}`; core-language install `{fs:language:<locale>}`; and core-language activation `{db:option:WPLANG}` plus an explicit dependency on the matching install. Two tweaks conflict when either set contains `*`, their exact keys intersect, or one wildcard prefix matches a key in the other set. A tweak may start only when it conflicts with no running tweak and all dependencies succeeded. Record deterministic results in configured order regardless of start or completion order. A tweak failure does not trigger rollback or block conflict-free tweaks; a failed prerequisite causes only its dependent tweak to be skipped. User cancellation remains fatal and follows critical rollback policy.
- Normalize the configured Package API base URL to a trailing slash and resolve the relative path `packages` against it, preserving any configured path prefix such as `/api/v1/`; never resolve an absolute `/packages` reference. Add URL-encoded `license_key` only when the key is non-empty.
- The Package Catalog response contains the complete catalog in one response in v1; pagination is out of scope.
- Filter catalog search case-insensitively by Package name and slug. Exclude generic entries and filter plugin/theme pickers by exact Package type.
- Resolve metadata with the relative path `package/{escaped-slug}/metadata` against the same normalized trailing-slash base URL, preserving its configured path prefix. Path-escape the slug and query-encode the optional key instead of concatenating untrusted values.
- Validate metadata name-independent identity fields: type must be plugin or theme, slug must match the requested Package, version must be non-empty, and download URL must be a valid HTTPS URL. If size is present, it must be numeric and not exceed 1 GiB.
- Use the signed `download_url` exactly as metadata returns it. Do not strip or append `license_key`, expiry, signature, headers, or other authentication data. Do not store the signed URL in cache metadata.
- Redact the entire signed URL query from progress, logs, and errors.
- Install and activate selected plugins.
- Install and activate the configured default theme when Package integration is enabled.
- Install additional selected themes without activating them.
- When Package integration is disabled, retain the bundled WordPress theme and explicitly report that the configured default theme was skipped.
- Locate the Package Cache under the operating system's native user cache directory in a WPTUI Packages subdirectory.
- Protect cache mutation with one cross-process global file lock using `github.com/gofrs/flock`. Lock acquisition must observe cancellation through the active context.
- Store a manifest with schema version and one Cache Entry per Package identity. Each entry stores type, slug, exact version, cache-root-relative file path, byte size, locally computed SHA-256, and UTC RFC 3339 download time.
- Treat Package identity as `(type, slug)`. Keep only the latest cached version for that identity.
- Use exact version-string equality. Metadata is authoritative latest; do not semantically order vendor versions.
- Validate remote type and slug before deriving any path. Type is limited to plugin or theme. Slug must be a single safe path segment with no separators, control characters, `.` or `..`. After joining, confirm the result remains below the cache root.
- Revalidate every manifest `file_path` as relative and below the cache root. Reject symlinks and non-regular files.
- Validate a cache hit by file existence, regular-file status, manifest size, locally recorded SHA-256, and readable ZIP central directory. Cache matching is based on package identity and version equality; static metadata size matching is omitted because vendor distribution dynamically stamps license keys.
- Treat the local SHA-256 as local corruption/tamper detection only. It is not publisher authentication because the metadata contract does not provide a trusted checksum.
- If a cached artifact fails validation, remove its entry and managed file, then attempt a fresh download. Never use a corrupt archive as stale fallback.
- If the manifest cannot be parsed, rename it to a timestamped corrupt-manifest name, begin with an empty manifest, and leave unrecognized archives untouched.
- Parse metadata `size` as an advisory value when present; reject non-numeric values or sizes greater than the 1-GiB cap.
- The temporary file must never grow beyond the accepted 1-GiB artifact cap. When `Content-Length` is present, enforce that received bytes match `Content-Length`; do not enforce static metadata size against the stream, as server-side license injection alters package byte length.
- Accept at most five redirects. At each hop, resolve the new request solely from the server's `Location`; never copy the source query, API key, authorization data, or other credentials, and suppress `Referer` so a different origin cannot receive the signed source URL.
- For the initial URL and every redirect, require HTTPS, resolve every address for the hostname, reject the target if any resolved address is disallowed, and bind the actual connection to a validated resolved address while preserving the original hostname for TLS. The HTTP transport must not perform a second unvalidated DNS resolution.
- Block loopback, private, link-local, unspecified, and cloud/local metadata-service addresses for initial and redirected download targets.
- Stream downloads into a cache-local partial file, compute SHA-256 during the stream, enforce the 1-GiB ceiling, verify `Content-Length` completion when declared, and verify ZIP structure before publishing the archive.
- Remove partial files on success, error, cancellation, and later stale-part cleanup.
- Publish a new archive with an atomic rename, then atomically replace the manifest, then delete the previous version. Until both archive and manifest publication succeed, preserve the prior valid Cache Entry and artifact.
- Completed verified cache artifacts are not Website-owned resources. They survive later Provisioning rollback.
- Use stale cache only for transient failures: DNS/connection interruption, timeout, HTTP 408, HTTP 429, HTTP 5xx, or interrupted download.
- Do not use stale cache for authentication/authorization errors, other contract-level 4xx responses, invalid metadata, identity mismatch, archive integrity failure, or security-policy rejection.
- Resolve every selected Package into a verified local artifact before creating the Website directory or database.
- Provisioning tracks ownership of the Website directory, database, and Herd TLS state. Rollback affects only resources created by the current run.
- If Herd TLS was newly created by the run, rollback unsecures it. Pre-existing external state must not be removed.
- Any failure in core install, Package installation, theme installation, or Herd security, and any user cancellation after Website mutation begins, triggers ownership-aware rollback. An individual tweak failure is non-fatal and never triggers rollback.
- Normal progress output shows named steps. Concurrent tweak progress and results are rendered deterministically in configured order rather than process-completion order. Raw successful command output is suppressed; on error, show sanitized stderr and the failed step.
- Completion reports Website path and URL, installed plugins and themes, active theme, skipped default-theme state, cache-hit/download/stale status, stale versions used, failed tweaks, and dependency-skipped tweaks.

## Testing Decisions

- Test observable behavior and resource state, not source text, private helper calls, Huh rendering details, or exact command-string formatting.
- The highest primary seam is the create use case: invoke the create orchestrator with a resolved request, scripted progress sink, temporary filesystem, and controlled adapters. Assert the resulting Website state, owned-resource cleanup, and user-visible result.
- The Packages resolver is the one necessary secondary seam because cache, redirect, SSRF, signed URL, and failure-class behavior is independently substantial and impractical to prove only through the full create flow.
- Test config bootstrap through its public load-or-create seam: missing config creates a validated config, valid config loads unchanged, and invalid config fails without overwrite.
- Use temporary home/config/cache directories so tests are deterministic and never touch the developer's real config or Package Cache.
- Test base URLs with path prefixes and trailing-slash variations to prove Catalog and metadata resolution preserve prefixes such as `/api/v1/`.
- Use local HTTP test servers for Package Catalog, metadata, signed downloads, redirects, transient failures, invalid responses, exact-size streams, oversized streams, and interrupted transfers.
- Use controlled DNS and dial adapters at the Packages seam to prove every resolved address is checked, mixed public/private multi-address hosts are rejected, the connection is pinned to a validated address, DNS rebinding cannot trigger a second resolution, and the policy repeats at every redirect hop.
- Test exact signed URL preservation on the initial request. For redirects, assert that only `Location` supplies the next query, source credentials are not copied, `Referer` is absent, and exposed errors and progress omit query secrets.
- Test cache behavior for exact-version hits, version replacement, stale fallback, non-transient failure refusal, invalid ZIP, size mismatch, SHA mismatch, malformed manifest quarantine, old-artifact retention during failed replacement, and partial-file cleanup.
- Test concurrent resolver calls against the same cache to prove that the manifest remains valid and contains every successful mutation.
- Test create behavior for path collision, database collision, missing executable, database-create failure, core-install failure, plugin/theme failure, Herd-secure failure, and cancellation at representative stages.
- In every rollback test, assert that pre-existing directory/database/TLS/cache resources remain untouched and only current-run Website resources are removed.
- Test the database/install ordering as an observable contract: core install must never be attempted when database creation fails.
- Test Herd and non-Herd URL outcomes and confirm Herd unlinking is never used.
- Test tweak opt-out, the four-command bound, exact and wildcard conflict matching, `*` config-set versus every other command, `db:*` rewrite versus every option/language database mutation, parallel option updates with distinct keys, overlap serialization for duplicate option keys, language-file conflicts, language install-before-activate dependency, deterministic configured-order reporting, continue-on-error behavior, dependency skips, and the invariant that individual tweak failures retain the Website without rollback.
- Test plugin activation, default-theme activation, additional-theme non-activation, and default-theme skip reporting.
- Test Package API disabled behavior even when a populated Package Cache exists; the cache must not bypass integration state.
- Exercise one runnable smoke scenario through the application composition with scripted UI answers, fake executables, a local Package server, and temporary directories. The proof is a complete create result and final filesystem/database/TLS adapter state, not merely passing unit tests.
- The repository currently has no implementation or prior test suite to copy. New tests must establish the above public seams without introducing test-only production abstractions beyond the approved adapters.

## Out of Scope

- Implementing the `config`, `delete`, `backup`, `restore`, or `settings` menu options.
- Editing an existing config through the TUI after first-run creation.
- Linux support.
- Installing PHP, WP-CLI, MySQL, Herd, Valet, Laragon, or another local web server.
- Configuring DNS, hosts files, certificates, or web-server routing for non-Herd wildcard stacks.
- Non-interactive command flags or automation mode for create.
- WordPress multisite.
- Selecting a WordPress core version other than latest stable.
- Selecting the base WordPress locale per Website.
- Package Catalog pagination or server-side search.
- Package installation when the Package API base URL is empty, including cache-only installation.
- Retaining multiple cached versions, manual Package downgrade, or Package rollback selection.
- A cache-management, clear-cache, or cache-inspection UI.
- Publisher authenticity verification unless the Package API later provides a trusted checksum or signature separate from the signed download URL.
- Configurable Package download-size limits; v1 uses the accepted fixed 1-GiB artifact cap.
- Parallel Package downloads; v1 may resolve selected Packages sequentially under the global cache lock.
- Backup and restore behavior represented by the existing exclude lists.

## Further Notes

- The accepted domain language is Website, Website Name, Website Slug, Package, Package Catalog, Package Cache, Cache Entry, and Provisioning.
- The accepted architectural decisions are Huh standalone for the terminal interface, atomic critical Website Provisioning with best-effort WordPress tweaks, and a verified single-version Package Cache with transient stale fallback.
- The Package metadata fixture represents `size` as a string and supplies `download_url`; the implementation must validate this runtime contract rather than relying on HTTP headers alone.
- The Package API is trusted to supply installable PHP/theme code, but signed download URLs still receive HTTPS, redirect, IP-range, size, path, and log-redaction protections.
- `--skip-check` belongs only to `wp config create`. Database creation must complete successfully before `wp core install` is attempted.
