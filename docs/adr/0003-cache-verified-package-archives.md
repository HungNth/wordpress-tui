# Cache verified Package archives

WPTUI keeps one verified archive per Package type and slug in the OS-native user cache, indexed by an atomically written manifest and protected by a cross-process file lock. Each installation checks authoritative metadata first, reuses an exact-version cache hit, downloads and verifies a changed version, or falls back to a valid stale archive only for transient network/server failures; completed cache artifacts survive Website rollback, while partial downloads are removed on success, failure, and cancellation.
