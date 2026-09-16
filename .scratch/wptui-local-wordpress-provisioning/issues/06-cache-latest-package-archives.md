# 06: Cache the latest verified Package archives

Status: resolved
Blocked by: 04
Parent spec: ../spec.md

## What to build

Avoid downloading an unchanged Package on every Website creation by keeping one verified archive per Package identity in the operating system's native user cache. A changed version replaces the old archive atomically without sacrificing the last valid cache entry.

## Acceptance criteria

- [x] Cache mutation is protected by one cancellation-aware cross-process file lock.
- [x] The manifest stores schema version plus type, slug, exact version, cache-relative path, size, locally computed SHA-256, and UTC download time for each Cache Entry.
- [x] Package identity is type plus slug, and version freshness uses exact string equality rather than semantic ordering.
- [x] A matching Cache Entry is reused only when its path remains inside the cache root, it is a regular non-symlink file, size and local SHA-256 match, and ZIP structure is readable.
- [x] The local SHA-256 is treated only as detection of later local changes, not as publisher authenticity.
- [x] Invalid entries and managed corrupt files are removed before attempting a fresh download; corrupt archives are never reused.
- [x] A malformed manifest is quarantined, a new empty manifest is created, and unrecognized archives are left untouched.
- [x] New downloads use a cache-local partial file, publish the verified archive atomically, commit the manifest atomically, and delete the prior version only after both publications succeed.
- [x] Failed replacement preserves the previous valid Cache Entry and artifact.
- [x] Success, failure, and cancellation remove partial files; completed verified cache artifacts survive later Website rollback.
- [x] Concurrent resolver tests prove no manifest updates are lost or interleaved.
- [x] A demo creates two Websites with the same Package version and proves the second run performs a verified cache hit without downloading the archive again.

## Testing seam

Package cache storage and manifest manager: test directly with temporary cache roots, verifying atomic updates, concurrency locks, corruption recovery, and integrity checks.

## Demo path

Run Website creation twice with the same plugin selected; observe the first run download and populate cache, and the second run report an immediate verified cache hit.

## Answer

Implemented in `internal/packages/cache.go` and `internal/packages/resolver.go`:
- OS-native user cache root at `os.UserCacheDir()/wptui/packages/` with cross-process `flock.Flock` on `data.lock`.
- JSON manifest tracking `type`, `slug`, `version`, relative `file_path`, `size`, `sha256`, and UTC `downloaded_at`.
- Strict integrity checks with `os.Lstat` across all intermediate directory components up to the cache root, disallowing symlinks, file size discrepancies, SHA-256 mismatches, or unreadable ZIP archives.
- Malformed manifest quarantine: corrupt manifests are moved to `data.json.corrupt-<timestamp>` and a fresh manifest initialized without deleting unknown artifacts.
- Atomic two-phase update: writes artifact to destination, updates manifest atomically via temp file + rename, and only deletes previous version files after commit.
- Path traversal protection: entries attempting to point outside cache root are discarded from manifest without touching the external victim files.
- Cache hit in `Resolver.ResolvePackage`: subsequent runs match exact version strings and return cached archives without issuing download requests.
- Validated with unit tests in `internal/packages/cache_test.go` and `internal/packages/resolver_test.go`.
