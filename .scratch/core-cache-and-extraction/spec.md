# WordPress Core Cache and Native Extraction

Status: ready-for-agent

## Problem Statement

When creating a new WordPress Website in WPTUI:
1. Website Provisioning currently delegates WordPress core acquisition to WP-CLI executing `wp core download --skip-content`.
2. In restricted, high-latency, or international network environments (particularly on macOS), WP-CLI's internal PHP cURL download routinely stalls or times out after 10 minutes (`cURL error 28: Operation timed out after 600000 milliseconds`), failing Website creation and leaving no reusable artifacts on disk.
3. In contrast, modern HTTP clients and web browsers complete the download swiftly over standard connections.
4. WP-CLI's download command does not support resuming interrupted transfers, does not accept local zip files when `--skip-content` is passed, and re-downloads core files on every Website creation unless users manually discover and seed WP-CLI's internal cache folder.

## Solution

WPTUI introduces an OS-native **Core Cache** and Go-native core extraction pipeline:
1. **Network-First Version Check with Fast Fallback**: During Website Provisioning, WPTUI queries the official WordPress version-check API (`https://api.wordpress.org/core/version-check/1.7/`) with a short 3-second timeout to determine the latest release version and its official content-stripped archive URL (`packages.no_content`).
2. **Resilient Go HTTP Download & Verification**: If the Core Cache is empty or outdated, WPTUI streams the content-stripped core zip file using Go's HTTP client into a temporary `.part` file, verifies ZIP structure and computes SHA-256 and size, then atomically commits the archive into the Core Cache directory (`wptui/core/`). Older core archives are removed upon successful commit.
3. **Offline Stale Cache Fallback**: When the version-check API fails or times out, WPTUI immediately checks the Core Cache manifest (`data.json`) and disk. If a verified Core Archive exists, WPTUI proceeds using the cached archive. If no cached archive exists and the network is unavailable, provisioning terminates early with a clear actionable message.
4. **Native Safe Extraction**: WPTUI extracts the Core Archive directly into the Website directory using standard zip extraction, stripping the leading `wordpress/` directory prefix and preventing Zip Slip directory traversal vulnerabilities.
5. **Standard Directory Scaffolding**: After extracting the core files, WPTUI automatically scaffolds `wp-content/`, `wp-content/themes/`, and `wp-content/plugins/` with `0755` permissions, establishing a standard WordPress layout ready for subsequent configuration and package installation.
6. **Elimination of `wp core download`**: The fragile `wp core download` command is completely removed from Provisioning. Subsequent steps (`wp config create`, database setup, `wp core install`, and best-effort tweaks) remain intact.

## User Stories

1. As a developer creating a Website in an environment with high network latency, I want WPTUI to download WordPress core using Go's HTTP client, so that downloads succeed without PHP cURL timeouts.
2. As a developer, I want WPTUI to download the official `no_content` WordPress package, so that default bundled themes and plugins are never downloaded over the wire.
3. As a developer, I want WPTUI to cache verified Core Archives in an OS-native Core Cache directory (`wptui/core/`), so that subsequent Website creations complete in seconds without re-downloading core files.
4. As a developer, I want WPTUI to maintain a manifest (`data.json`) in the Core Cache containing the version, relative archive path, file size, SHA-256 hash, and download timestamp, so that cache state is auditable and validated.
5. As a developer, I want WPTUI to use a cross-process file lock (`data.lock`) around Core Cache operations, so that concurrent Website provisioning instances do not corrupt the manifest or archives.
6. As a developer, I want WPTUI to verify ZIP archive integrity and compute checksums before committing an archive to cache, so that truncated or corrupt downloads are never used.
7. As a developer working offline or with an unstable connection, I want WPTUI to use the cached Core Archive if the version-check API fails or times out, so that local development is uninterrupted.
8. As a developer, I want the version-check API query to timeout after 3 seconds, so that network lag does not stall Website creation when a valid local archive is already cached.
9. As a developer running WPTUI without internet and without a pre-existing Core Cache, I want WPTUI to halt immediately with an explanatory error, so that I understand an initial network connection is required.
10. As a developer, I want WPTUI to extract the Core Archive directly into the Website folder, so that creation is independent of WP-CLI's download command.
11. As a developer, I want archive extraction to strip the root `wordpress/` folder prefix, so that files like `index.php` and `wp-load.php` are placed directly in the Website root directory.
12. As a developer, I want archive extraction to validate destination paths against directory traversal (Zip Slip), so that corrupt or malicious archives cannot write outside the Website directory.
13. As a developer, I want WPTUI to create `wp-content/`, `wp-content/themes/`, and `wp-content/plugins/` directories during core extraction, so that the Website directory conforms to standard WordPress layout conventions.
14. As a developer, I want Website provisioning to roll back and clean up the Website directory if core extraction fails, so that partial or corrupted directories are not left behind.
15. As a developer, I want WPTUI to clean up older version archives when a new core version is successfully cached, so that disk space is not wasted.

## Implementation Decisions

1. **Core Package Architecture (`internal/core`)**:
   - Introduce a focused `internal/core` package responsible for:
     - Querying the official WordPress release API (`https://api.wordpress.org/core/version-check/1.7/`).
     - Managing the Core Cache directory (`wptui/core/`), its manifest (`data.json`), and file lock (`data.lock`).
     - Downloading the `packages.no_content` archive to a `.part` temporary file and verifying ZIP integrity.
     - Extracting the Core Archive into a destination directory with `wordpress/` prefix stripping and path validation.
     - Creating `wp-content/`, `wp-content/themes/`, and `wp-content/plugins/` subdirectories.

2. **Core Cache Manifest Schema**:
   The `data.json` file inside `wptui/core/` records the single active verified Core Archive:
   - `schema_version`: Integer tracking manifest schema format.
   - `version`: String release version (e.g. `"7.1"`).
   - `file_path`: Relative filename of the archive within the core cache directory (e.g. `"wordpress-7.1-no-content.zip"`).
   - `size`: Integer byte size of the archive.
   - `sha256`: Hexadecimal SHA-256 hash of the archive.
   - `downloaded_at`: RFC3339 UTC timestamp when the archive was verified and committed.

3. **Core Cache Storage Location**:
   - Located in the OS-native user cache directory under `wptui/core/` (`~/.cache/wptui/core` on Linux, `~/Library/Caches/wptui/core` on macOS, `%LocalAppData%\wptui\core` on Windows).
   - Completely decoupled from the Package Cache directory (`wptui/packages/`), ensuring independent lifecycles between core and packages.

4. **Network-First Resolution Flow**:
   - `ResolveCoreArchive(ctx)` coordinates the download and cache lookup:
     1. Creates a 3-second context for the version check request.
     2. If the API check succeeds and reports a version differing from `data.json` (or if cache is empty or archive file is missing):
        - Initiates download of `packages.no_content` URL.
        - Streams response into `<filename>.part`.
        - Verifies archive integrity with standard ZIP reader and calculates SHA-256.
        - Acquires `data.lock`, renames `.part` into final cache path, writes updated `data.json`, and deletes old archive files.
     3. If the API check fails, times out, or returns a version matching `data.json`:
        - Validates that the cached archive file exists on disk and matches the manifest size/hash.
        - If valid: returns the cached Core Archive path.
        - If missing or corrupt and network was unavailable: returns an actionable error stating an internet connection is required to seed the initial Core Cache.

5. **Direct Archive Extraction in Provisioning**:
   - In `internal/create/create.go`:
     - Replace the `wpClient.CoreDownload()` call with Core Archive resolution and extraction.
     - Core extraction reads the resolved zip archive and writes entries to `websitePath`, stripping `wordpress/` from each entry name and verifying the resolved destination is within `websitePath`.
     - Ensures `wp-content/`, `wp-content/themes/`, and `wp-content/plugins/` directories exist with `0755` permissions.
     - Progress callbacks report `"core_resolve"` ("Checking WordPress core...") and `"core_extract"` ("Extracting WordPress core...").
   - Existing atomic rollback via `ownershipTracker` automatically removes `websitePath` if extraction or subsequent steps fail.

6. **WP-CLI Client Clean Up**:
   - `wpClient.CoreDownload` is deprecated or removed from the active provisioning flow, eliminating runtime dependence on WP-CLI core downloads.

## Testing Decisions

1. **Testing Philosophy**:
   - Tests must assert observable external behavior: verifying files created on disk, directory structure, manifest contents, and error handling, rather than internal function calls or mocks where real filesystem operations are practical.

2. **Core Cache Unit Tests (`internal/core`)**:
   - Use `t.TempDir()` as custom cache and destination roots.
   - Use `httptest.Server` to mock the WordPress version check API and download endpoints.
   - Test cases:
     - Successful download, verification, manifest generation, and old version cleanup.
     - Offline fallback using valid existing cache when API server is unreachable.
     - Error reporting when cache is empty and API server is unreachable.
     - Safe zip extraction: verify root `wordpress/` prefix stripping, proper permissions, creation of `wp-content/themes/` and `wp-content/plugins/`, and rejection of Zip Slip entry names (`../`).

3. **Provisioning Orchestrator Tests (`internal/create`)**:
   - Verify `Creator.Create()` end-to-end using a mock core resolver or pre-seeded test core archive.
   - Confirm complete creation flow succeeds without executing `wp core download`.
   - Confirm rollback removes the website directory if core extraction fails.

4. **Prior Art**:
   - `internal/packages/cache_test.go` (manifest management, file locking, atomic `.part` rename, corrupted manifest recovery).
   - `internal/packages/download_test.go` (HTTP streaming, size checking, ZIP validation).
   - `internal/create/create_test.go` (orchestrator execution, mock runners, rollback tracking).

## Out of Scope

- Retaining multiple concurrent versions of WordPress core in Core Cache (only the latest verified version is kept).
- User-configurable custom core download mirrors (official `api.wordpress.org` endpoint is used).
- Standalone CLI subcommands for manual cache pruning or prefetching outside of the Website creation flow.

## Further Notes

- The official WordPress API endpoint `https://api.wordpress.org/core/version-check/1.7/` is guaranteed to return `packages.no_content` for all standard releases.
- The `no_content` archive contains identical core files to the full zip release but omits `wp-content/plugins` and `wp-content/themes`, matching the exact semantics of `--skip-content`.
