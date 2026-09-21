# 01: Publish Validated Cross-Platform GitHub Releases

**What to build:** Replace the unrelated plugin release automation with one complete WPTUI release path. A pushed matching SemVer tag must validate application metadata, pass the race-enabled Go test suite, cross-compile the supported macOS and Windows executables, package and checksum them, and publish a correctly classified GitHub Release.

**Blocked by:** None (can start immediately)
**Status:** resolved

**Parent specification:** Tag-Triggered GitHub Releases

- [x] Add a JSON application metadata contract with product name `wptui`, initial version `0.1.0`, and public author identity `HungNth` at `https://github.com/HungNth`, without an email address.
- [x] Trigger release automation only for pushed tags beginning with `v`, retain an explicit tag-reference guard, and reject malformed SemVer before tests or builds begin.
- [x] Accept stable and prerelease SemVer values and require the tag without its leading `v` to match the manifest version exactly.
- [x] Grant the workflow token explicit `contents: write` permission required to create or update a GitHub Release and upload assets.
- [x] Resolve the Go toolchain from the module declaration and run the complete test suite with the race detector before compiling release binaries.
- [x] Cross-compile the existing WPTUI CLI entrypoint on one Ubuntu runner with CGO disabled for `darwin/amd64`, `darwin/arm64`, and `windows/amd64`.
- [x] Produce ordinary executables named `wptui` for macOS and `wptui.exe` for Windows; remove all unrelated plugin IDs, plugin manifest parsing, shared-library flags, headers, DLL/dylib extensions, and package names.
- [x] Package each executable alone in a ZIP named `wptui_<version>_<goos>_<goarch>.zip` and generate one SHA-256 checksum file covering exactly the three ZIP assets.
- [x] Keep build, packaging, and publication in one runner workspace without intermediate artifact upload/download jobs.
- [x] Publish only after validation, tests, all builds, all archives, and checksums succeed; attach the three ZIPs and checksum file and generate GitHub release notes.
- [x] Publish stable versions as normal releases and versions with a prerelease component as GitHub prereleases.
- [x] Verify observable behavior through the release workflow seam: matching stable and prerelease versions are accepted; malformed, mismatched, and non-tag inputs fail before compilation; all target binaries compile; archive contents and names are correct; and every checksum validates.
- [x] Run the race-enabled test suite and reproduce all three cross-build/package outputs without committing, pushing a tag, or publishing a real release.

## Demo Path

After implementation is approved and committed by the maintainer, update the application metadata version, push the matching SemVer tag, and observe a GitHub Release containing the three platform ZIPs, the checksum file, generated notes, and the correct stable or prerelease classification.
