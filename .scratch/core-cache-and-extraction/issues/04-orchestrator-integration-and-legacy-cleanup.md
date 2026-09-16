# 04: Provisioning Orchestrator Integration and Legacy WP-CLI Removal

**What to build:** Complete end-to-end integration of Core Cache resolution and native extraction into the `Creator.Create()` website provisioning orchestrator. Replaces the legacy `wp core download` command with Core Archive resolution and extraction, reports distinct progress steps (`core_resolve`, `core_extract`), preserves directory rollback on failure, and updates existing unit and e2e smoke tests.

**Blocked by:** 02: Network-First Core Downloader and Offline Fallback, 03: Safe Native Core Zip Extraction and Standard Directory Layout

**Status:** resolved

- [x] Integrates Core Cache resolver and extractor into `Creator.Create()` Step 2
- [x] Emits progress events for resolving and extracting core files
- [x] Removes runtime dependency on `wp core download`
- [x] Ensures atomic rollback cleans up destination directory if core extraction or subsequent steps fail
- [x] All unit, integration, and e2e smoke tests pass without requiring WP-CLI core downloads
