# CLI Version, Update Check, and Configuration Path Specification

Status: ready-for-agent

## Problem Statement

WPTUI currently launches its interactive terminal interface for every invocation. Users cannot ask an installed binary which WPTUI release it contains, determine whether a newer release is available, discover the active `config.json` location, or view command-line usage without entering the TUI.

The root application manifest is already the release metadata contract used to validate tags and name release assets, but its version is not available to the running binary. An installed binary also has no read-only Application Update Check. Users must manually inspect the repository and GitHub Releases, and they cannot reliably distinguish the running version from the version currently published in repository metadata.

These non-interactive operations must not initialize the TUI, run first-use configuration, read or create `config.json`, or silently install software. Their output and process status must remain predictable for both people and shell scripts.

## Solution

Add four mutually exclusive command-line actions to WPTUI:

- `-v` and `--version` print the embedded application version.
- `-u` and `--update` perform an Application Update Check against the application manifest on the repository's `main` branch.
- `-c` and `--config` print the absolute path at which WPTUI expects `config.json`.
- `-h` and `--help` print concise command-line usage.

The root application manifest remains the single metadata source and is embedded into every binary at compile time. The Application Update Check compares that embedded version with the remote manifest using full Semantic Versioning precedence, including prereleases. When a newer version exists, WPTUI prints the exact GitHub Release URL for that version. It never downloads or installs the release.

No arguments preserve the current behavior and open the TUI. Unknown flags, positional arguments, and combinations of multiple action flags are rejected before application initialization.

## User Stories

1. As a WPTUI user, I want to print the version of my installed binary, so that I know which release I am running.
2. As a terminal user, I want both `--version` and `-v`, so that I can use a descriptive or concise form.
3. As a shell-script author, I want version output on standard output, so that I can capture it without parsing error output.
4. As a WPTUI user, I want version output to identify both the application and release, so that the value is unambiguous when copied into a report.
5. As a release consumer, I want the binary version to come from the same application manifest used by release automation, so that published metadata and runtime identity do not drift.
6. As a developer building WPTUI locally, I want ordinary Go and Make builds to embed the manifest version without special linker flags, so that local and release builds use the same metadata path.
7. As a WPTUI user, I want to check for a newer release without opening the TUI, so that the check is fast and scriptable.
8. As a WPTUI user, I want both `--update` and `-u`, so that the Application Update Check has descriptive and concise forms.
9. As a security-conscious user, I want the Application Update Check to be read-only, so that checking does not replace or mutate my installed executable.
10. As a user with limited bandwidth, I want an update check to fetch only release metadata, so that it does not download a release archive.
11. As a WPTUI user, I want the remote version read from the repository's application manifest, so that the check follows the project's declared release metadata.
12. As a WPTUI user, I want a newer version reported with both my current version and the available version, so that the comparison is clear.
13. As a WPTUI user, I want a direct link to the exact GitHub Release when a newer version exists, so that I can review its release notes and assets.
14. As a WPTUI user, I want the release link to include the version reported by the remote manifest, so that it cannot silently lead to a different release.
15. As a WPTUI user, I want a clear message when my binary is already current, so that a successful check is distinguishable from missing output.
16. As a developer running a binary newer than the remote manifest, I want it treated as already up to date, so that the tool does not recommend a downgrade.
17. As a prerelease tester, I want prerelease identifiers compared according to Semantic Versioning, so that preview releases are ordered correctly.
18. As a stable-release user, I want a SemVer-greater prerelease in the manifest reported as an available update, so that the manifest remains the sole update authority.
19. As a release maintainer, I want build metadata ignored for SemVer precedence, so that versions differing only in build metadata are not treated as upgrades.
20. As a WPTUI user, I want malformed remote versions rejected, so that invalid metadata cannot produce a misleading update result.
21. As a WPTUI user, I want a remote manifest for another product rejected, so that WPTUI never reports an unrelated release.
22. As a WPTUI user, I want HTTP failures reported as errors, so that network failure is not mistaken for being up to date.
23. As a WPTUI user, I want an update request to time out promptly, so that an unavailable network does not leave the command hanging indefinitely.
24. As a WPTUI user, I want invalid JSON and incomplete manifests reported as errors, so that corrupt metadata cannot produce a false result.
25. As a terminal user, I want operational update-check failures on standard error, so that valid command output remains separate from diagnostics.
26. As an automation author, I want a successful update result and a successful up-to-date result both to exit with status zero, so that either is a valid completed check.
27. As an automation author, I want network and manifest failures to exit with status one, so that operational failures are detectable.
28. As a WPTUI user, I want to print the expected configuration path without opening Settings, so that I can inspect or edit the file with another tool.
29. As a terminal user, I want both `--config` and `-c`, so that configuration-path discovery has descriptive and concise forms.
30. As a shell-script author, I want configuration-path output to contain only the absolute path, so that it can be passed directly to another command.
31. As a new WPTUI user, I want the expected configuration path printed even before `config.json` exists, so that I know where first-run configuration will be stored.
32. As a new WPTUI user, I want configuration-path discovery not to create `config.json`, so that an informational command has no filesystem mutation.
33. As a WPTUI user, I want a failure to determine my home directory reported clearly, so that an empty or fabricated path is never printed.
34. As a WPTUI user, I want `--help` and `-h` to list every supported action flag, so that command-line capabilities are discoverable.
35. As a shell user, I want help output on standard output with a successful exit status, so that requesting usage is not treated as an error.
36. As a WPTUI user, I want unknown flags rejected, so that typos do not unexpectedly open the interactive application.
37. As a WPTUI user, I want positional arguments rejected, so that unsupported input is never silently ignored.
38. As a WPTUI user, I want combinations of multiple action flags rejected, so that command precedence is never ambiguous.
39. As an automation author, I want invalid command syntax to exit with status two, so that usage errors are distinguishable from operational failures.
40. As a terminal user, I want usage errors printed on standard error with concise usage guidance, so that I can correct the command quickly.
41. As an existing WPTUI user, I want invoking `wptui` without arguments to continue opening the TUI, so that current interactive behavior remains unchanged.
42. As an existing WPTUI user, I want informational action flags to bypass first-run configuration, so that checking metadata never starts a wizard.
43. As an existing WPTUI user, I want informational action flags not to read `config.json`, so that a malformed or unavailable configuration does not block them.
44. As a terminal user, I want command-line output in English and without ANSI color, so that it remains readable in colorless terminals and scripts.
45. As a maintainer, I want one command-level behavioral seam for all new CLI actions, so that tests cover user-visible behavior without duplicating parser and update-check implementation details.
46. As a maintainer, I want deterministic HTTP fixtures for update checks, so that permanent tests never depend on GitHub or the public internet.
47. As a maintainer, I want documentation to explain the new flags and clarify that `--update` does not install anything, so that users do not infer self-update behavior.
48. As a release maintainer, I accept a short interval in which the manifest on `main` names a release whose GitHub Release page is not yet published, so that runtime checking does not require another release asset or API request.

## Implementation Decisions

1. **Command action contract**
   - Support exactly four command-line actions: version, update check, configuration-path discovery, and help.
   - Each action has one short and one long spelling: `-v`/`--version`, `-u`/`--update`, `-c`/`--config`, and `-h`/`--help`.
   - The actions are mutually exclusive. Any invocation containing more than one action is invalid, including combinations involving help.
   - Reject every unknown flag and every positional argument.
   - No arguments retain the existing interactive launch behavior.

2. **Early command dispatch**
   - Parse and dispatch command-line actions before creating the application, configuring signal-aware TUI execution, loading configuration, or invoking a first-run wizard.
   - Informational actions must not read, create, or modify `config.json`.
   - Do not perform an Application Update Check implicitly during normal TUI startup.

3. **Application manifest ownership**
   - Keep the root JSON application manifest as the single source of the product name, release version, and public author identity.
   - Embed that manifest into the binary at compile time using Go's embedding facilities rather than release-only linker flags or generated fallback values.
   - Ordinary Go builds, local Make builds, tests, and release builds must all consume the same manifest contract.
   - The embedded product name must be `wptui`, and its version must be valid Semantic Versioning without a leading `v`.
   - A malformed embedded manifest is an operational error for actions that require application metadata; do not substitute `dev`, `unknown`, or another fabricated version.

4. **Version output**
   - Version output is exactly the product name followed by the release with a display-only leading `v`, for example `wptui v0.2.1`.
   - Write version output to standard output and return exit status zero.
   - Version reporting performs no network or configuration access.

5. **Application Update Check source**
   - Fetch the remote application manifest from the raw `main` branch of the canonical `HungNth/wordpress-tui` GitHub repository.
   - Use a five-second request timeout.
   - Perform one metadata request only. Do not query the GitHub Releases API and do not probe the generated release link.
   - Validate a successful HTTP response, bounded JSON input, the expected product name, and a valid SemVer version before comparison.
   - Treat non-success HTTP status, timeout, transport failure, malformed JSON, unexpected product identity, missing required metadata, and invalid version as operational errors.

6. **Version precedence**
   - Compare the embedded and remote versions using full Semantic Versioning precedence.
   - Support stable versions, prerelease identifiers, and build metadata according to SemVer rules.
   - A remote prerelease is reportable whenever it has greater SemVer precedence than the embedded version; there is no separate stable/prerelease channel.
   - Ignore build metadata for precedence.
   - Remote versions equal to or lower than the embedded version produce the up-to-date result; never recommend a downgrade.
   - Do not reuse a numeric-only WordPress Package version comparator because it does not implement the application release contract.

7. **Update-check output**
   - When a newer release exists, print `Update available: v<current> -> v<remote>` followed by `Release: <url>` on the next line.
   - Construct the release URL as the canonical repository's GitHub Release tag URL using `v` plus the validated remote manifest version.
   - When no newer release exists, print `wptui v<current> is already up to date.`
   - Both valid outcomes write to standard output and return exit status zero.
   - Operational failures write a concise `Error: <cause>` diagnostic to standard error and return exit status one.

8. **Release-publication timing**
   - Continue using the raw branch manifest rather than publishing another manifest as a GitHub Release asset.
   - Accept that an update may be reported briefly before its GitHub Release page exists when the manifest is updated ahead of tag publication.
   - The release process should minimize this interval by creating the matching tag immediately after the manifest change is available.
   - Do not add a second network request to verify the release page.

9. **Configuration-path output**
   - Reuse the existing configuration-path contract based on the current user's home directory and the `.config/wptui/config.json` relative location.
   - Print the absolute path alone, followed by a newline, to standard output.
   - Return exit status zero whether or not the file or its parent directory currently exists.
   - Do not create directories or a configuration file.
   - Failure to determine the user home directory is an operational error on standard error with exit status one.

10. **Help and syntax errors**
    - Help output lists usage and all four short/long action pairs, writes to standard output, and returns exit status zero.
    - Syntax failures identify the invalid input, include concise usage guidance, write to standard error, and return exit status two.
    - Do not silently choose precedence when multiple actions are present.

11. **Output contract**
    - All new command-line text is English plain text.
    - Do not emit ANSI styling, progress animation, terminal control sequences, or interactive prompts.
    - Keep output stable and concise enough for direct human use and simple shell composition; structured JSON output is not part of this feature.

12. **Module boundaries**
    - Keep the process entrypoint as a thin adapter around one command-level runner that accepts arguments and output streams and returns a process exit code.
    - Isolate application-manifest parsing, embedded metadata access, remote manifest retrieval, and SemVer comparison from TUI application initialization.
    - Provide only the dependency seams required for deterministic command-level tests: HTTP behavior, remote manifest endpoint, user-home resolution, local manifest input, output streams, and interactive-app launch boundary where necessary.
    - Do not introduce a general command framework or third-party CLI framework for four fixed flags.
    - Prefer a maintained SemVer implementation if it prevents a hand-written partial parser; do not reuse the WordPress Package comparator.

13. **Documentation**
    - Document all four action flags in the README.
    - Explicitly describe `--update` as an Application Update Check that prints a link but does not download or install a release.
    - Document the output purpose of `--config` and preserve the canonical `config.json` locations already described for supported platforms.
    - Use the glossary term **Application Update Check** and avoid describing this feature as self-update or automatic update.

14. **Release automation**
    - Preserve the existing tag-triggered release workflow, supported platform matrix, archive naming, checksum publication, and manifest/tag validation.
    - No new release asset or GitHub API permission is required.
    - Runtime version embedding must work with the workflow's existing ordinary Go build command without requiring a separate metadata-generation step.

## Testing Decisions

1. **Primary behavioral seam**
   - Use one command-level runner immediately below the process entrypoint as the primary permanent test seam.
   - Tests supply arguments, standard output and standard error buffers, deterministic application metadata, an HTTP client/server, remote manifest endpoint, and a temporary user home as needed.
   - The runner returns the intended process exit code, allowing tests to verify complete observable behavior without calling `os.Exit`.
   - Keep the production entrypoint as a thin adapter; do not permanently test trivial forwarding from the entrypoint to the runner.

2. **What makes a good test**
   - Assert user-visible output stream, message content, exit status, network behavior, and absence of unintended initialization or mutation.
   - Test complete command outcomes rather than private parser branches, embedded source text, struct field copies, or helper call counts that are not user-observable.
   - Use table-driven cases where the same CLI contract applies across aliases or error variants.
   - Keep fixtures deterministic and isolated. Permanent tests must never call the public GitHub endpoint.

3. **Command syntax cases**
   - Verify every short and long action spelling.
   - Verify no arguments preserve the interactive launch path at the command boundary without duplicating TUI behavior already covered elsewhere.
   - Verify unknown flags, positional arguments, and every representative multi-action combination fail on standard error with exit status two and do not launch the application.
   - Verify help succeeds on standard output and lists every supported action.

4. **Version cases**
   - Verify version output format, standard-output placement, and zero exit status using deterministic embedded metadata.
   - Verify malformed or wrong-product embedded metadata fails rather than emitting a fabricated version.
   - Do not pin a permanent test to the repository's current release number merely to detect future manifest bumps.

5. **Application Update Check cases**
   - Use an in-process HTTP test server, following existing resolver test prior art, to serve remote manifests and controlled failures.
   - Cover greater stable version, equal version, lower remote version, greater prerelease, stable-versus-prerelease precedence, and build-metadata equality.
   - Verify newer output contains the current version, remote version, and exact version-specific GitHub Release link.
   - Verify equal and lower remote versions produce the up-to-date message and never a downgrade recommendation.
   - Cover malformed JSON, missing required metadata, wrong product name, invalid SemVer, non-success HTTP status, transport failure, and timeout.
   - Verify successful results use standard output/status zero and operational failures use standard error/status one.
   - Verify one update-check invocation performs one manifest request and never downloads an archive or probes the release URL.

6. **Configuration-path cases**
   - Use a temporary home directory to verify the absolute expected path and zero exit status when `config.json` is absent.
   - Verify the command creates no directory or file.
   - Verify home-directory resolution failure produces standard error and status one.

7. **Isolation from interactive startup**
   - Verify each informational action completes without invoking the interactive application boundary or first-run configuration boundary.
   - Verify invalid syntax also fails before those boundaries.
   - Do not duplicate existing TUI model or application workflow tests.

8. **Implementation smoke proof**
   - During implementation, build or run the real command once to demonstrate `--version`, `--config`, `--help`, and representative invalid syntax on the actual executable surface.
   - Exercise `--update` against a deterministic local HTTP fixture through the command-level seam rather than the public network.
   - The real-binary invocation is verification evidence, not a permanent subprocess test suite.

9. **Prior art**
   - Reuse the repository's existing `httptest.Server` approach for deterministic remote metadata behavior.
   - Reuse its established options/dependency-injection style for replacing external boundaries in composed tests.
   - Preserve the release workflow's existing SemVer manifest contract rather than inventing a second version grammar.

## Out of Scope

- Downloading, extracting, verifying, or installing a WPTUI release from the running application.
- Replacing the current executable or invoking the existing installer scripts.
- Automatic, background, periodic, or startup update checks.
- Update notifications inside the TUI, Application Settings, Sidebar Navigation, or Content Pane.
- A stable-versus-prerelease update channel selector.
- A flag that suppresses prerelease results.
- JSON, YAML, or other structured command output.
- Configurable update endpoints, mirrors, proxies, or repository settings.
- Retries, caching, offline fallback, or stale update metadata.
- GitHub Releases API integration or authentication tokens.
- Verification that the generated GitHub Release page exists before reporting an update.
- Publishing `manifest.json` as a GitHub Release asset.
- Changes to release targets, archive names, checksums, installer behavior, signing, or notarization.
- Commit hashes, build timestamps, dirty-tree markers, or additional build provenance in version output.
- A general subcommand hierarchy or adoption of a third-party CLI framework.
- Aliases other than the four agreed short/long action pairs.
- Backward-compatibility aliases for unsupported flags or positional syntax.
- Changing the existing configuration location or schema.
- Creating `config.json` or its parent directories from `--config`.
- Changing interactive TUI behavior when WPTUI is invoked without arguments.

## Further Notes

- The application manifest currently declares version `0.2.1`; the implementation must not hardcode that value because routine release bumps must flow through embedding automatically.
- The release workflow already validates that a pushed `v<version>` tag exactly matches the application manifest and supports SemVer prereleases. Runtime comparison must honor the same release model.
- The existing numeric WordPress Package version comparator is intentionally not authoritative for WPTUI application releases because it does not implement full SemVer prerelease and build-metadata rules.
- The canonical Application Update Check terminology is recorded in the project glossary. No ADR is required because the feature is localized, inexpensive to change, and follows the existing manifest and release contracts.
- The selected test strategy is one command-level runner seam rather than subprocess compilation tests or multiple lower-level permanent suites.
