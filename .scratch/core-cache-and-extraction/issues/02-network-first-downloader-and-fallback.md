# 02: Network-First Core Downloader and Offline Fallback

**What to build:** A network-first WordPress core downloader and version resolver that queries the official WordPress release API (`https://api.wordpress.org/core/version-check/1.7/`) with a fast 3-second timeout. When a new version is detected or cache is empty, it downloads the official `packages.no_content` archive into a `.part` temporary file using Go's HTTP client, validates ZIP archive structure and checksum, and commits it into the Core Cache. If the network times out or fails, it falls back to the existing cached Core Archive; if no cache exists and the network is unavailable, it returns an actionable error.

**Blocked by:** 01: Core Cache Management and Manifest Storage

**Status:** resolved

- [x] Queries `https://api.wordpress.org/core/version-check/1.7/` with a 3-second context timeout to extract latest version and `packages.no_content` download URL
- [x] Streams archive download directly into a temporary `.part` file using Go's standard HTTP client
- [x] Validates ZIP archive structure using `archive/zip` and calculates SHA-256 and byte size before committing
- [x] Commits verified archive to Core Cache and updates `data.json` atomically
- [x] Automatically falls back to valid cached Core Archive if network check fails or times out
- [x] Returns clear, actionable error if both network is unreachable and no valid cache exists
