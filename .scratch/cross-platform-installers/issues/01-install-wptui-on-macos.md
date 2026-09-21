# 01: Install WPTUI on macOS

**What to build:** Provide a user-local POSIX shell installer that safely installs or upgrades the latest stable WPTUI release on Intel and Apple Silicon macOS, including native Apple Silicon selection when invoked through Rosetta.

**Blocked by:** None (can start immediately)
**Status:** resolved

**Parent specification:** Cross-Platform WPTUI Installers and README

- [x] Implement the installer using POSIX `sh` only; do not require Bash, Git, Go, `jq`, GNU coreutils, or a package manager.
- [x] Resolve the latest stable release through the GitHub `releases/latest/download` checksum manifest and require exactly one well-formed entry for the selected Darwin asset.
- [x] Select AMD64 on Intel hardware and ARM64 on native Apple Silicon or Rosetta-translated Apple Silicon, using both `uname` and `sysctl.proc_translated`; reject unsupported architectures clearly.
- [x] Check for required native tools before downloading and use `curl`, the system ZIP extractor, and `shasum -a 256`.
- [x] Download and extract in isolated temporary staging, clean staging on success, failure, and interruption, and reject missing or ambiguous executable archive contents.
- [x] Compare the downloaded archive against the published SHA-256 digest and abort without modifying an existing installation on missing, malformed, duplicate, or mismatched checksum data.
- [x] Ensure the verified macOS executable has execute permission, stage it in the destination filesystem, and replace the existing executable only as the final successful mutation.
- [x] Install for the current user under `.local/bin` by default without `sudo`, while supporting the agreed installation-directory environment override.
- [x] Add the resolved installation directory to PATH idempotently: zsh uses its login profile, bash uses its profile, and unknown shells receive an export instruction without profile mutation.
- [x] Report the installed executable path and clearly state whether the user must open a new terminal or apply an export command.
- [x] Verify observable behavior with controlled release fixtures: mocked Intel, mocked native-ARM64, and mocked Rosetta selection branches; non-Darwin rejection; unsupported architecture rejection; successful install/upgrade; executable mode; checksum failures; malformed manifests; multi-member and wrong-name archives; preserved existing binary; custom install directory; bash profile hierarchy and idempotence; unknown shell non-mutation; and temporary-file cleanup. (All 14 tests passed in a tool-boundary mock harness; native macOS kernel execution remains subject to physical Darwin host).
