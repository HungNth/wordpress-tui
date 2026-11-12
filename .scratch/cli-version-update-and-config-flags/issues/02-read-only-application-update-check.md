# 02: Read-Only Application Update Check

**Parent specification:** [CLI Version, Update Check, and Configuration Path Specification](../spec.md)

**What to build:** Add `-u` and `--update` as a complete Application Update Check. WPTUI compares its embedded release identity with the application manifest on the canonical repository's `main` branch, reports whether a newer stable or prerelease version exists, and prints the exact GitHub Release link without downloading or installing anything.

**Blocked by:** 01: Embedded Release Identity and Local CLI Actions.

**Status:** resolved

- [x] `wptui --update` and `wptui -u` run through the established command-level runner without initializing the TUI, first-run configuration, or `config.json` loading.
- [x] The command fetches the raw application manifest from the canonical `HungNth/wordpress-tui` repository's `main` branch using one request with a five-second timeout.
- [x] A valid remote manifest requires a successful HTTP response, bounded valid JSON, the expected `wptui` product identity, and a valid Semantic Versioning release without a leading `v`.
- [x] Embedded and remote releases are compared with full Semantic Versioning precedence, including prerelease identifiers and build metadata rules; the numeric-only WordPress Package comparator is not reused.
- [x] A SemVer-greater stable or prerelease remote version prints `Update available: v<current> -> v<remote>` and the exact `https://github.com/HungNth/wordpress-tui/releases/tag/v<remote>` link to standard output, then exits with status zero.
- [x] An equal or lower remote version prints `wptui v<current> is already up to date.` to standard output, exits with status zero, and never recommends a downgrade.
- [x] Build metadata alone does not make a release newer.
- [x] Non-success HTTP status, timeout, transport failure, malformed JSON, missing metadata, wrong product identity, and invalid SemVer each produce a concise `Error: <cause>` diagnostic on standard error and exit with status one.
- [x] The Application Update Check performs no archive download, release-page probe, executable replacement, installer invocation, configuration mutation, retry, cache fallback, or background scheduling.
- [x] The check accepts the agreed publication window in which the manifest can name a release before its GitHub Release page exists; it does not issue a second request to verify the generated link.
- [x] Deterministic command-level tests use an in-process HTTP fixture and cover newer stable, equal, lower remote, greater prerelease, stable/prerelease ordering, build-metadata equality, exact release-link output, every manifest/HTTP failure class, timeout, output streams, exit statuses, and isolation from interactive startup.
- [x] Permanent tests never call GitHub or another public network service and assert externally visible outcomes rather than private helper structure.
- [x] The README documents both update aliases and explicitly defines the operation as an Application Update Check that only reports availability and a release link; it does not install a release.
- [x] Existing release targets, asset names, checksum publication, installer behavior, and tag/manifest validation remain unchanged.
