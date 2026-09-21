# 02: Install WPTUI on Windows

**What to build:** Provide a user-local Windows PowerShell installer that safely installs or upgrades the latest stable WPTUI AMD64 release, configures User PATH, and rejects unsupported Windows architectures.

**Blocked by:** None (can start immediately)
**Status:** resolved

**Parent specification:** Cross-Platform WPTUI Installers and README

- [x] Implement the installer for Windows PowerShell 5.1 and later without requiring PowerShell 7, Git, Go, `jq`, Unix tools, or a package manager.
- [x] Use `Invoke-WebRequest` with `-UseBasicParsing` so Windows PowerShell 5.1 does not depend on the legacy Internet Explorer parser.
- [x] Resolve the latest stable release through the GitHub `releases/latest/download` checksum manifest and require exactly one well-formed entry for the Windows AMD64 asset.
- [x] Detect operating-system architecture correctly under both 64-bit and 32-bit PowerShell processes by using runtime OS architecture information via CIM `Win32_Processor.Architecture` with `PROCESSOR_ARCHITEW6432`/environment fallback where necessary.
- [x] Support only AMD64 Windows hosts and reject ARM64 or other architectures with a clear error rather than silently relying on emulation.
- [x] Use native PowerShell capabilities (.NET Cryptography and System.IO.Compression) for HTTPS download, SHA-256 hashing, and ZIP extraction.
- [x] Download and extract in isolated temporary staging, remove staging after success or failure, and reject missing or ambiguous `wptui.exe` archive contents.
- [x] Compare the downloaded archive against the published SHA-256 digest case-insensitively and preserve an existing installation on missing, malformed, duplicate, or mismatched checksum data.
- [x] Install under the current user's local application-data programs directory by default without elevation, while supporting the agreed installation-directory parameter.
- [x] Stage the verified executable in the destination filesystem and replace the existing executable only after every validation succeeds.
- [x] If replacement fails because WPTUI is running, preserve the existing executable and report that the user must close WPTUI before retrying.
- [x] Add the resolved installation directory to User PATH and the active PowerShell process PATH without creating duplicate equivalent entries.
- [x] Report the installed executable path and whether a new terminal is needed.
- [x] Verify observable behavior on native Windows: AST parsing passes with zero errors; functional suite verified successful install/upgrade to custom directory, SHA-256 checksum mismatch failure and existing binary preservation, malformed digest rejection, multi-member archive rejection, wrong member rejection, non-matching asset rejection, User PATH idempotency, active session PATH handling, temporary file cleanup, and architecture rejection (CIM Win32_Processor Architecture = 12 correctly rejected as Windows ARM64; Architecture = 0 correctly rejected as unsupported architecture).
