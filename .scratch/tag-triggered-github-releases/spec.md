# Specification: Tag-Triggered GitHub Releases

Status: ready-for-agent

## Problem Statement

The maintainer cannot reliably publish installable WPTUI releases by pushing a version tag. The existing release workflow was copied from an unrelated shared-library plugin project: it expects a missing plugin manifest, builds the repository root as a C shared library, produces DLL and dylib assets under another product's name, and uses an invalid Go version selector. WPTUI is instead a normal Go command-line executable built through its existing CLI entrypoint and currently supports direct use on macOS and Windows.

The repository also lacks a single application metadata contract for the product name, release version, and public author identity. Without that contract, a mistyped or stale tag can publish assets under an unintended version.

## Solution

Add a JSON application manifest containing WPTUI's product name, current release version, and public author identity. Replace the copied plugin workflow with a tag-triggered GitHub Release workflow that validates the pushed SemVer tag against the manifest before doing expensive work, gates publication on the race-enabled Go test suite, cross-compiles ordinary WPTUI executables for the agreed macOS and Windows targets on one Ubuntu runner, packages each executable as a ZIP archive, generates SHA-256 checksums, and publishes the assets with generated GitHub release notes.

Stable SemVer tags publish normal releases. SemVer prerelease tags publish GitHub prereleases. Any invalid tag, manifest mismatch, failed test, failed build, packaging error, or checksum error prevents publication.

## User Stories

1. As the WPTUI maintainer, I want a GitHub Release created when I push a valid version tag, so that publishing a release requires one predictable action.
2. As the WPTUI maintainer, I want stable tags such as `v0.1.0` to publish normal releases, so that users can distinguish production releases from previews.
3. As the WPTUI maintainer, I want prerelease tags such as `v0.2.0-beta.1` to publish GitHub prereleases, so that preview builds are clearly marked.
4. As the WPTUI maintainer, I want malformed version tags rejected before tests or builds start, so that invalid release identifiers never produce assets.
5. As the WPTUI maintainer, I want the pushed tag version to match the application manifest exactly, so that repository metadata and published releases cannot silently diverge.
6. As the WPTUI maintainer, I want the first manifest version to be `0.1.0`, so that the repository starts with an explicit pre-1.0 release contract.
7. As the WPTUI maintainer, I want the manifest to identify the product as `wptui`, so that release assets use the actual CLI identity.
8. As the WPTUI maintainer, I want the manifest to record `HungNth` and `https://github.com/HungNth` as the public author identity, so that ownership is explicit without exposing an email address.
9. As the WPTUI maintainer, I want the release workflow to use the Go version declared by the module, so that CI and repository toolchain requirements remain synchronized.
10. As the WPTUI maintainer, I want the complete race-enabled Go test suite to pass before release builds begin, so that a failing application is not published.
11. As a macOS Intel user, I want a `darwin/amd64` WPTUI executable, so that I can run the application on an Intel Mac.
12. As an Apple Silicon user, I want a native `darwin/arm64` WPTUI executable, so that I can run the application without Intel emulation.
13. As a 64-bit Windows user, I want a `windows/amd64` WPTUI executable, so that I can run the application directly on supported Windows systems.
14. As a release consumer, I want each platform build packaged in a ZIP archive, so that extraction is consistent across supported systems.
15. As a release consumer, I want the macOS archives to contain an executable named `wptui`, so that the command name matches the project.
16. As a release consumer, I want the Windows archive to contain `wptui.exe`, so that Windows recognizes it as an executable.
17. As a release consumer, I want asset filenames to include the product, version, operating system, and architecture, so that I can choose the correct download without opening it.
18. As a release consumer, I want SHA-256 checksums for every ZIP archive, so that I can verify download integrity.
19. As a release consumer, I want generated GitHub release notes, so that I can see the changes without the maintainer manually duplicating commit history.
20. As the WPTUI maintainer, I want all three executables cross-compiled on one Ubuntu runner, so that release automation stays small and avoids unnecessary native runner cost.
21. As the WPTUI maintainer, I want CGO disabled for release builds, so that the pure-Go binaries do not acquire unintended native runtime dependencies.
22. As the WPTUI maintainer, I want validation to finish before any platform build starts, so that metadata errors fail quickly and cheaply.
23. As the WPTUI maintainer, I want publication to happen only after every platform archive and checksum has been produced successfully, so that users never see a knowingly incomplete release.
24. As the WPTUI maintainer, I want the workflow token to have the explicit repository-content permission needed to create releases and upload assets, so that publication succeeds without broader permissions.
25. As a contributor, I want all plugin-specific names, manifest assumptions, shared-library flags, and DLL/dylib packaging removed, so that the workflow reflects WPTUI rather than the source repository it was copied from.
26. As a contributor, I want the workflow to avoid intermediate artifact upload and download jobs, so that the implementation remains easy to understand and maintain.
27. As the WPTUI maintainer, I want clear workflow failures for tag, manifest, test, build, packaging, and publication errors, so that a failed release can be diagnosed from the Actions log.

## Implementation Decisions

- **Application Metadata Contract**:
  - Introduce a root-level JSON application manifest as the release metadata source.
  - The manifest contains the product name, version, and a structured author object.
  - Initial values are product name `wptui`, version `0.1.0`, author name `HungNth`, and author URL `https://github.com/HungNth`.
  - The author contract contains no email address.
  - The manifest version excludes the leading `v`; release tags include it.

- **Tag and Version Contract**:
  - Trigger on pushed tags beginning with `v`, then perform an explicit tag-reference guard and SemVer validation before tests or builds.
  - Accept stable SemVer and SemVer prerelease identifiers.
  - Require the tag without its leading `v` to equal the manifest version exactly, including any prerelease identifier.
  - Stable versions create normal GitHub releases; versions containing a prerelease component create GitHub prereleases.
  - Validation failure terminates the workflow before platform compilation begins.

- **Release Authorization**:
  - Grant the workflow token explicit `contents: write` permission at workflow or release-job scope.
  - Use that token only to create or update the GitHub Release and upload its assets.

- **Quality Gate**:
  - Resolve the Go toolchain from the module's declared Go version rather than duplicating a version string in workflow configuration.
  - Run the full Go test suite with the race detector after metadata validation and before release builds.
  - Any failed test prevents all release publication.

- **Build Strategy**:
  - Build the existing WPTUI CLI entrypoint used by the repository's local build rules; do not build the module root.
  - Produce ordinary executables rather than C shared libraries.
  - Cross-compile on one Ubuntu GitHub-hosted runner with CGO disabled.
  - Build exactly three targets: macOS Intel (`darwin/amd64`), macOS Apple Silicon (`darwin/arm64`), and Windows x64 (`windows/amd64`).
  - Name the executable `wptui` on macOS and `wptui.exe` on Windows.
  - Do not preserve plugin headers, plugin identifiers, plugin manifest parsing, shared-library extensions, compiler setup, or unrelated package names from the copied workflow.

- **Packaging Contract**:
  - Package each target independently as a ZIP containing its single platform executable.
  - Name assets using `wptui_<version>_<goos>_<goarch>.zip`, where version comes from the validated manifest and does not include a leading `v`.
  - Generate one checksum file containing SHA-256 entries for all ZIP assets.
  - Keep packaging and publication in the same job workspace; do not upload and redownload intermediate GitHub Actions artifacts.

- **GitHub Release Contract**:
  - Publish only after validation, tests, all three builds, all three archives, and the checksum file succeed.
  - Attach the three ZIP archives and checksum file to the GitHub Release associated with the pushed tag.
  - Generate release notes through GitHub.
  - Mark a release as prerelease when the validated SemVer contains a prerelease component.
  - Preserve GitHub's automatically generated source archives; no custom source bundle is required.

- **Documentation Scope**:
  - The specification is sufficient release-process documentation for this effort.
  - No domain glossary entry or ADR is required because release packaging is not WPTUI domain vocabulary and the chosen automation is inexpensive to change.

## Testing Decisions

- **Highest Testing Seam**:
  - Treat the release workflow as the single highest behavioral seam. It must demonstrate the complete sequence from tag and manifest validation through tests, cross-compilation, packaging, checksums, and publication readiness.
  - Avoid introducing Go interfaces, helper packages, or permanent unit tests solely to test workflow plumbing.

- **Good Test Characteristics**:
  - Verify observable release outcomes: accepted and rejected version inputs, successful platform compilation, exact archive contents, expected asset names, checksum validity, and prevention of publication after a failed prerequisite.
  - Do not assert YAML line arrangement, shell implementation details, action internals, or copied source text.

- **Validation Cases**:
  - A stable tag matching the manifest is accepted and classified as a normal release.
  - A prerelease tag matching the manifest is accepted and classified as a prerelease.
  - A malformed tag is rejected before tests and builds.
  - A valid SemVer tag that does not match the manifest is rejected before tests and builds.
  - A non-tag reference is rejected by the explicit release guard.

- **Build and Package Cases**:
  - The existing race-enabled test command succeeds before release builds.
  - All three agreed GOOS/GOARCH combinations compile from the existing CLI entrypoint with CGO disabled.
  - The two macOS ZIPs each contain one executable named `wptui`.
  - The Windows ZIP contains one executable named `wptui.exe`.
  - The checksum file contains one valid SHA-256 entry for each ZIP and no unrelated files.

- **Prior Art in the Codebase**:
  - The existing local build rules establish the CLI entrypoint, binary name, and race-enabled test command.
  - Existing GitHub workflow configuration establishes the tag-triggered release intent, generated release notes, checksum publication, and required repository-content permission, while its unrelated plugin build assumptions are replaced.

- **Verification Boundary**:
  - Implementation verification must run the race-enabled test suite and reproduce all three cross-compiles and packages without pushing a tag.
  - Actual GitHub Release creation is exercised only after the maintainer approves and pushes the first matching tag; the implementation agent must not create a commit, push a tag, or publish a release without explicit authorization.

## Out of Scope

- Linux binaries.
- Windows ARM64 binaries.
- A universal macOS binary combining Intel and Apple Silicon architectures.
- DMG, PKG, MSI, installer, Homebrew, Scoop, or other distribution formats.
- Apple code signing, Apple notarization, Windows Authenticode signing, or certificate-secret provisioning.
- Runtime version embedding, build commit embedding, an About screen, or a `--version` command.
- Automatic manifest version updates, automatic commits, automatic tag creation, or tag pushing.
- GoReleaser or another release framework.
- Separate changelog generation beyond GitHub-generated release notes.
- Manual workflow dispatch.
- Preserving backward compatibility with the copied plugin release contract.

## Further Notes

- The application currently has no release tags, so `0.1.0` is the agreed initial manifest version.
- WPTUI's platform-aware launcher explicitly supports macOS and Windows, matching the selected release targets.
- The release guard and explicit `contents: write` permission are required even though the workflow trigger is tag-scoped, keeping authorization and release intent visible in the workflow itself.
- This specification changes release automation and metadata only; it does not change terminal UI behavior or application runtime behavior.
