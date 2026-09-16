# 01: Core Cache Management and Manifest Storage

**What to build:** An OS-native storage manager for WordPress Core Archives located in `wptui/core/`, protected by a cross-process file lock (`data.lock`) and tracked by a `data.json` manifest recording the cached version, archive filename, file size, SHA-256 checksum, and download timestamp. It verifies archive integrity, checks whether a cached version matches, and purges older version archives upon committing a new release.

**Blocked by:** None (can start immediately)

**Status:** resolved

- [x] Core cache directory defaults to OS-native user cache path under `wptui/core` and supports custom directory override
- [x] Manifest `data.json` tracks `version`, `file_path`, `size`, `sha256`, and `downloaded_at`
- [x] File operations on manifest and archives are synchronized across processes using `data.lock`
- [x] Corrupt or invalid manifest files are safely quarantined and recreated
- [x] Cached archive files are validated for existence, regular file mode, non-symlink, and size/checksum match
- [x] Committing a new core archive removes previous version archive files from disk
