# 03: Document and Verify Cross-Platform Installation

**What to build:** Publish a concise English project README that accurately explains WPTUI, its prerequisites, first-run behavior, installer commands, security boundaries, and development workflow, then verify the macOS and Windows installation experience as one composed feature.

**Blocked by:** 01: Install WPTUI on macOS; 02: Install WPTUI on Windows
**Status:** resolved

**Parent specification:** Cross-Platform WPTUI Installers and README

- [x] Create a GitHub Flavored Markdown README without a logo, screenshots, license, contributing, or changelog sections.
- [x] Describe WPTUI as a terminal application for managing local WordPress Websites and summarize only currently implemented Provisioning, Website Configuration, De-provisioning, Website Backup, Website Restoration, and Application Settings behavior.
- [x] State supported installation targets accurately: Intel macOS, Apple Silicon macOS, and AMD64 Windows; do not claim Linux or Windows ARM64 support.
- [x] Document PHP, WP-CLI, and MySQL as runtime prerequisites, Laravel Herd as conditional on enabled integration, and the VS Code `code` command as conditional on the relevant Settings action.
- [x] Document the agreed macOS one-line installer command that downloads the raw shell installer and pipes it to `sh`.
- [x] Document the agreed Windows command that launches Windows PowerShell with `ExecutionPolicy Bypass`, downloads the raw PowerShell installer, and executes it.
- [x] Add a short GitHub admonition recommending that security-conscious users download and inspect remote scripts before execution.
- [x] Explain that installer downloads are SHA-256 verified while release binaries are not currently Apple/Windows code-signed or notarized; do not recommend bypassing Gatekeeper, quarantine, or SmartScreen.
- [x] Explain that the first run launches interactive configuration and stores configuration under the user's home configuration directory without duplicating the complete JSON schema.
- [x] Document current build, run, test, race-test, vet, format, clean, and help commands using the repository's existing development workflow.
- [x] Use canonical domain glossary terms and avoid unsupported claims about CLI flags, version commands, full UI-contract completion, automatic dependency installation, or universal rollback.
- [x] Verify that both documented installer URLs resolve to the implemented scripts and that their commands match the supported shell/PowerShell contracts.
- [x] Run composed installer verification across both operating systems: macOS end-to-end execution verified via 14-case tool-boundary fixture harness; Windows verified via 9 distinct cases (7 functional fixture cases plus 2 isolated architecture rejection cases).
