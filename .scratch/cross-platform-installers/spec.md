# Specification: Cross-Platform WPTUI Installers and README

Status: ready-for-agent

## Problem Statement

WPTUI publishes ZIP archives for Intel macOS, Apple Silicon macOS, and 64-bit Windows, but users must currently discover the correct asset, download it, verify it, extract it, choose an installation directory, and configure PATH manually. The repository also has no README that explains what WPTUI does, which external tools it requires, how to install it, how first-run configuration works, or how contributors can build and test it.

This manual process is error-prone. Users can select the wrong architecture, install a corrupted or incomplete download, overwrite a working executable before a replacement is verified, or place the executable somewhere their shell cannot find. Apple Silicon users running a translated shell can also be misidentified as Intel when architecture detection relies only on `uname`.

## Solution

Provide a POSIX shell installer for macOS and a Windows PowerShell installer. Each installer resolves the latest stable GitHub Release through its published checksum manifest, selects the supported asset for the host architecture, downloads into temporary staging, verifies SHA-256 using operating-system-native tools, extracts the expected executable, and replaces an existing user-local installation only after every validation succeeds.

The macOS installer supports Intel and Apple Silicon, including native Apple Silicon detection when invoked under Rosetta. The Windows installer supports AMD64 and rejects unsupported Windows ARM64 hosts rather than silently relying on emulation. Both installers manage user-level PATH configuration without administrator privileges and remain safe to run repeatedly.

Create a concise English README that documents WPTUI's implemented workflows, runtime prerequisites, installer commands, first-run configuration, security limitations, and development commands.

## User Stories

1. As a macOS user, I want to install WPTUI with one shell command, so that I do not have to navigate GitHub Release assets manually.
2. As a Windows user, I want to install WPTUI with one PowerShell command, so that I can start the application without building it from source.
3. As an Intel Mac user, I want the installer to select the `darwin/amd64` release asset, so that the downloaded executable matches my hardware.
4. As an Apple Silicon user, I want the installer to select the `darwin/arm64` release asset, so that WPTUI runs natively.
5. As an Apple Silicon user running a translated shell, I want the installer to detect Rosetta translation and still select the native ARM64 executable, so that I do not receive an avoidable Intel build.
6. As a user on an unsupported macOS architecture, I want installation to stop with a clear error, so that an incompatible binary is not installed.
7. As a 64-bit Windows user, I want the installer to select the `windows/amd64` release asset, so that WPTUI runs on my supported system.
8. As a Windows ARM64 user, I want installation to stop with a clear unsupported-architecture message, so that the installer does not silently depend on x64 emulation.
9. As a user, I want installers to use the latest stable GitHub Release, so that prerelease builds are never installed unintentionally.
10. As a user, I want release resolution to avoid GitHub API authentication and rate-limit dependencies, so that installation remains reliable without a token.
11. As a user, I want the installer to identify the correct asset from the release checksum manifest, so that asset naming stays aligned with release automation.
12. As a security-conscious user, I want every downloaded ZIP verified against its published SHA-256 checksum, so that corrupted downloads are rejected.
13. As a macOS user, I want checksum verification to use the native `shasum` utility, so that installation does not require GNU coreutils.
14. As a Windows user, I want checksum verification to use native PowerShell hashing, so that installation does not require third-party tools.
15. As a user, I want a checksum mismatch to abort before touching my installed executable, so that a working installation is preserved.
16. As a user upgrading WPTUI, I want the existing executable replaced only after download, checksum, extraction, and executable validation succeed, so that failed upgrades do not break the current installation.
17. As a user rerunning the installer, I want it to update WPTUI without an interactive confirmation prompt, so that upgrades remain scriptable.
18. As a user, I want temporary downloads and extraction directories cleaned after success or failure, so that installer artifacts do not accumulate.
19. As a user, I want network, missing-asset, malformed-checksum, extraction, permission, and replacement failures reported clearly, so that I can diagnose installation problems.
20. As a macOS user, I want WPTUI installed under my home directory by default, so that installation does not require `sudo`.
21. As a Windows user, I want WPTUI installed under my local application-data directory by default, so that installation does not require Administrator privileges.
22. As a macOS user with a custom layout, I want to override the installation directory through an environment variable, so that I can control where the executable is placed.
23. As a Windows user with a custom layout, I want to override the installation directory through a PowerShell parameter, so that I can control where the executable is placed.
24. As a zsh user, I want the macOS installer to add the default installation directory to my login PATH when missing, so that new terminals can invoke `wptui` directly.
25. As a bash user, I want the macOS installer to add the default installation directory to the appropriate profile when missing, so that new shells can invoke `wptui` directly.
26. As a user of another shell, I want the installer to print the required PATH export instead of modifying an unknown profile, so that my shell configuration is not guessed incorrectly.
27. As a user rerunning the installer, I want PATH changes to remain idempotent, so that duplicate profile entries are not created.
28. As a Windows user, I want the installation directory added to User PATH and the active PowerShell session when missing, so that `wptui` becomes available without machine-wide configuration.
29. As a user, I want the installer to report the installed executable path and any terminal-restart action, so that I know how to launch WPTUI.
30. As a WPTUI user, I want runtime prerequisites documented separately from installation, so that the installer does not unexpectedly install PHP, WP-CLI, MySQL, or Laravel Herd.
31. As a prospective user, I want a concise overview of WPTUI's implemented Website workflows, so that I can decide whether the tool fits my local WordPress work.
32. As a new user, I want the README to explain first-run configuration and where configuration is stored, so that I understand what happens when WPTUI starts.
33. As a new user, I want copy-paste installation commands for macOS and Windows, so that installation is fast and predictable.
34. As a security-conscious user, I want the README to recommend reviewing downloaded scripts before execution, so that piping a remote script is an informed choice.
35. As a user, I want the README to disclose that release binaries are checksum-verified but not currently code-signed or notarized, so that I understand the integrity and trust guarantees.
36. As a contributor, I want the README to list current build, run, test, race-test, vet, format, and clean commands, so that local development follows repository conventions.
37. As a contributor, I want README terminology to match the project's domain glossary, so that Websites, Provisioning, De-provisioning, Backup, and Restoration are described consistently.

## Implementation Decisions

- **Installer Layout and Compatibility**:
  - Add one installer directory containing a macOS POSIX shell installer and a Windows PowerShell installer.
  - The macOS installer must run under POSIX `sh`; it must not require Bash-specific syntax.
  - The Windows installer must support Windows PowerShell 5.1 and later; it must not require PowerShell 7.
  - Neither installer requires Git, Go, `jq`, or an external package manager.

- **Release Discovery Contract**:
  - Use the repository's GitHub `releases/latest/download` endpoint to fetch the checksum manifest for the latest stable release.
  - Do not use the GitHub Releases API or require an access token.
  - Select the asset filename from the checksum manifest rather than reconstructing an unverified version string.
  - Expect the current release contract: ZIP assets named by product, version, operating system, and architecture, plus one checksum manifest.
  - Treat a missing checksum entry, duplicate matching entry, malformed digest, missing asset, or unexpected filename as a hard failure.

- **macOS Architecture Contract**:
  - Map Intel hardware to the Darwin AMD64 asset and Apple Silicon hardware to the Darwin ARM64 asset.
  - Detect Rosetta translation with the native `sysctl.proc_translated` signal in addition to `uname`; translated Apple Silicon must select ARM64.
  - Reject every other reported architecture with a clear error.

- **Windows Architecture Contract**:
  - Use runtime architecture information available in Windows PowerShell/.NET rather than parsing localized command output.
  - Support only AMD64 hosts because the release workflow publishes no Windows ARM64 asset.
  - Reject ARM64 and other architectures rather than downloading the AMD64 build for emulation.

- **Native Tooling Contract**:
  - On macOS, use platform-native utilities: `curl` for HTTPS downloads, `shasum -a 256` for hashing, and the system archive extractor for ZIP files.
  - On Windows, use `Invoke-WebRequest`, `Get-FileHash`, and `Expand-Archive` or their Windows PowerShell 5.1-compatible equivalents.
  - Check required installer tools before download and produce a specific missing-tool error.
  - Do not assume GNU `sha256sum`, Bash, `jq`, or Unix tools on Windows.

- **Verification and Replacement Lifecycle**:
  - Create isolated temporary staging and register cleanup for success, failure, and interruption where the platform permits.
  - Download the checksum manifest and chosen ZIP before modifying the existing installation.
  - Parse exactly one SHA-256 digest for the selected asset and compare it case-insensitively with the locally computed digest.
  - Extract the archive into staging and require exactly the expected executable name; reject missing or ambiguous executable contents.
  - Ensure the macOS executable has execute permission before installation.
  - Stage the verified executable in the destination filesystem, then replace the existing executable as the final mutation.
  - If Windows cannot replace an executable because WPTUI is running, fail clearly and instruct the user to close it; do not delete the working binary first.

- **Installation Locations**:
  - macOS defaults to the current user's `.local/bin` directory and installs the executable as `wptui`.
  - macOS accepts an environment-variable override for the installation directory.
  - Windows defaults to a `wptui` directory under the current user's local application-data programs directory and installs `wptui.exe`.
  - Windows accepts an optional installation-directory parameter.
  - Installation is user-scoped and must not invoke `sudo`, request elevation, or write to system-wide application directories.

- **PATH Management**:
  - Modify PATH only when the installation directory is absent.
  - On macOS using the default install directory, append one idempotent PATH export to the zsh login profile or bash profile according to the configured shell.
  - For unrecognized shells, leave profile files untouched and print the required export command.
  - A piped POSIX installer cannot mutate its parent shell, so it must clearly tell users to open a new terminal or apply the printed export command.
  - On Windows, add the directory to User PATH without duplicating equivalent entries and update the active PowerShell process PATH.
  - Custom installation directories receive the same PATH treatment, using the actual resolved directory.

- **Upgrade and Error Behavior**:
  - Re-running an installer upgrades the existing user installation without prompting.
  - Network, HTTP, manifest, checksum, archive, permission, PATH, or replacement failures return a non-zero status and a concrete error message.
  - A failed operation preserves the previously installed executable.
  - Successful output identifies the installed path and any shell restart needed.

- **Runtime Dependency Boundary**:
  - Installers install only WPTUI and do not install or configure PHP, WP-CLI, MySQL, Laravel Herd, VS Code, or a Package API.
  - Runtime dependency validation remains application behavior.
  - README documentation identifies PHP, WP-CLI, and MySQL as runtime prerequisites; Laravel Herd is required only when its integration is enabled, and VS Code's `code` command is required only for the relevant Settings action.

- **README Contract**:
  - Create a concise English README using GitHub Flavored Markdown and GitHub admonitions where useful.
  - Do not add a logo because the repository has no verified image asset.
  - Include project overview, implemented workflows, supported operating systems, prerequisites, macOS and Windows installation, first run/configuration, development commands, and bounded security notes.
  - Use the agreed macOS `curl` installer command and the agreed Windows command that launches Windows PowerShell with `ExecutionPolicy Bypass`, downloads the raw installer, and executes it.
  - Add a short admonition recommending script review before piping remote content to a shell.
  - Disclose that checksums are verified but binaries are not currently Apple/Windows code-signed or notarized.
  - Describe configuration through the first-run wizard and its user-home configuration location; do not duplicate the complete JSON schema.
  - Use canonical glossary terms and describe only behavior evidenced in the current implementation.

- **Documentation and Domain Scope**:
  - No domain glossary changes are required because installation and distribution are not WordPress TUI domain concepts.
  - No ADR is required because the installer layout and GitHub Release coupling are explicit, localized, and inexpensive to revise.

## Testing Decisions

- **Highest Testing Seam**:
  - Treat each installer as a command-line product and test its externally observable behavior end to end against controlled release fixtures and temporary user directories.
  - Keep application Go internals outside the installer test seam; the installers consume published artifacts and do not change runtime application behavior.

- **Good Test Characteristics**:
  - Exercise platform/architecture selection, remote filename selection, checksum enforcement, extraction, replacement, cleanup, PATH behavior, and exit status through script invocation.
  - Assert installed files, preserved prior installations, profile/User PATH effects, and emitted user guidance rather than implementation functions or source text.
  - Keep fixtures deterministic and avoid publishing a real release, editing the developer's actual profiles, or changing the real User PATH.

- **macOS Installer Cases**:
  - POSIX shell syntax validation succeeds.
  - Intel detection selects the Darwin AMD64 fixture.
  - native Apple Silicon and Rosetta-translated detection select the Darwin ARM64 fixture.
  - unsupported architecture exits non-zero without creating an installation.
  - valid checksum installs an executable with execute permission.
  - checksum mismatch, missing manifest entry, malformed checksum, missing executable, or ambiguous archive content preserves any existing executable.
  - default and overridden installation directories work.
  - zsh and bash PATH entries are added once; repeated installation does not duplicate them; unknown shells receive instructions without profile mutation.

- **Windows Installer Cases**:
  - Windows PowerShell parser accepts the script under the 5.1 language contract.
  - AMD64 selects the Windows AMD64 fixture; ARM64 and unsupported architectures exit without installation.
  - valid checksum installs `wptui.exe`; checksum and archive failures preserve an existing executable.
  - default and parameter-overridden installation directories work.
  - User PATH is updated once and repeated installation does not create duplicates.
  - replacement failure caused by an in-use executable produces a clear close-WPTUI message while preserving the old executable.

- **README Verification**:
  - Confirm the documented installer commands resolve to the implemented scripts.
  - Confirm prerequisites, supported targets, first-run configuration, development commands, checksum behavior, and unsigned-binary warning match repository behavior.
  - Do not add source-text snapshot tests for README prose.

- **Prior Art**:
  - Follow the staged-download, checksum, replacement, architecture, locking, and PATH-safety patterns used by the current official OpenAI Codex installers, adapted to WPTUI's simpler GitHub Release contract and two supported operating systems.
  - Reuse the repository's existing release asset names and checksum manifest; do not introduce a second distribution naming convention.

## Out of Scope

- Linux installation or Linux support claims.
- Windows ARM64 binaries or automatic AMD64 emulation on Windows ARM64.
- Selecting or pinning a specific historical or prerelease version.
- System-wide installation, `sudo`, Administrator elevation, or machine-wide PATH changes.
- Homebrew, Scoop, Winget, Chocolatey, DMG, PKG, MSI, or other package-manager/installer formats.
- Installing or configuring PHP, WP-CLI, MySQL, Laravel Herd, VS Code, or Package API credentials.
- Apple code signing, Apple notarization, Windows Authenticode signing, or bypassing Gatekeeper, quarantine, or SmartScreen.
- Automatic background updates, uninstall scripts, telemetry, or installer self-update behavior.
- Runtime application changes, configuration-schema changes, command-line flags, or a version command.
- Changes to the existing release workflow, target matrix, or release asset naming contract.
- Logo, screenshot, license, contributing, or changelog sections in the README.

## Further Notes

- Canonical repository: `HungNth/wordpress-tui`.
- Current release automation publishes Darwin AMD64, Darwin ARM64, and Windows AMD64 ZIPs plus a sorted SHA-256 checksum manifest.
- The installers become usable only after at least one stable GitHub Release exists with the expected assets.
- A checksum fetched from the same GitHub Release detects transfer corruption and inconsistent assets; it does not replace publisher code signing or independently establish publisher identity.
- The repository currently has no README or verified logo/image asset.
- The official current Codex installer sources were used as behavioral prior art, but WPTUI does not need Codex's hosted metadata service or its broader target matrix.
