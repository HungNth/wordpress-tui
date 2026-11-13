# 01: Hide WordPress debug output in default Apply Tweaks

**What to build:** Make newly generated WPTUI configuration keep WordPress debugging and debug logging enabled while disabling debug display. The existing Apply Tweaks workflows for Provisioning and Website Configuration must apply that safer default alongside the established local-development memory, autosave, revision, and trash-retention values. Prove the complete default mutation contract through the confirmed application-level Website Configuration seam and keep the provisioning documentation aligned.

**Blocked by:** None (can start immediately).

**Status:** resolved

**Parent specification:** [WordPress Debug and Tweak Defaults Specification](../spec.md)

- [x] Newly generated configuration represents `WP_DEBUG=true`, `WP_DEBUG_LOG=true`, and `WP_DEBUG_DISPLAY=false` as raw PHP values.
- [x] Newly generated configuration retains `WP_MEMORY_LIMIT=256M` as a string value and retains `AUTOSAVE_INTERVAL=600`, `WP_POST_REVISIONS=5`, and `EMPTY_TRASH_DAYS=21` as raw PHP values.
- [x] Running Website Configuration → Apply Tweaks with the generated default configuration sends all seven expected constant names, values, and raw/string modes to the WP client boundary.
- [x] The regression proof uses the existing application-level Website Configuration flow seam with scripted Website/action selection; it does not add a duplicate lower-level default-configuration test.
- [x] Provisioning still has Apply Tweaks selected by default and uses the same configured tweak list; no standalone Debug Mode action or Create wizard control is added.
- [x] Existing `config.json` files remain untouched and authoritative; no migration, backfill, or forced value outside `wp_tweaks` is introduced.
- [x] Apply Tweaks retains its current configured ordering, per-tweak progress, best-effort execution, and partial-failure reporting.
- [x] Provisioning documentation states that the default debug-display value is `false` and continues to describe the accepted raw/string modes accurately.
