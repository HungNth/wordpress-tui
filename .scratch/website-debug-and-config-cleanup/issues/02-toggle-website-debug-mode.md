# 02: Toggle Website Debug Mode from Websites Hub

**What to build:** Let a user select one Website and immediately toggle Website Debug Mode from its Website Hub action menu. The operation must determine the current state safely, apply the canonical enabled or disabled values only to that Website, show preflight and mutation progress in the existing In-Pane Progress Monitor, and report complete, partial, failed, or cancelled outcomes without running unrelated WordPress tweaks.

**Blocked by:** 01: Retire the legacy Website Configuration flow.

**Status:** resolved

- [x] Every single-Website action menu contains `Toggle Debug Mode`; no keyboard shortcut or batch action is added.
- [x] The action remains visible for discovered Websites and reports a preflight failure when the Website Directory or `wp-config.php` is unavailable or invalid.
- [x] `WP_DEBUG=true` or `1` is normalized as enabled; `false`, `0`, or a missing constant is normalized as disabled.
- [x] A non-boolean `WP_DEBUG` value or a read failure other than a missing constant stops the operation before any configuration write and reports the cause.
- [x] Enabling writes raw values `WP_DEBUG=true`, `WP_DEBUG_LOG=true`, and `WP_DEBUG_DISPLAY=false`.
- [x] Disabling writes raw values `WP_DEBUG=false`, `WP_DEBUG_LOG=false`, and `WP_DEBUG_DISPLAY=false`.
- [x] The operation changes only the selected Website's `wp-config.php`; it does not mutate `config.json`, remove debug entries from Apply Tweaks, or prevent a later Apply Tweaks run from changing the Website again.
- [x] Once a target state is known, all three writes are attempted in deterministic order despite an individual write failure; completed writes remain in place, no rollback is attempted, and each failed constant is retained in the warning report.
- [x] q/Ctrl+C cancellation stops before the next WP-CLI command, retains completed writes, performs no rollback, and reports a cancelled/partial outcome accurately.
- [x] The operation uses the existing navigation lock, task stepper, log viewport, status markers, and overall-outcome-first report conventions, including the Website, Website Directory, target state, and per-constant failures.
- [x] Website Configuration domain tests cover state normalization, missing and invalid values, canonical enable/disable writes, deterministic best-effort behavior, read failure, and cancellation through a fake narrow config client.
- [x] Website Hub/AppModel tests drive the action through keyboard interaction and verify discoverability, handler routing, progress locking, and success/warning/cancelled reports without repeating WP-CLI details.
- [x] No direct `wp-config.php` parser, duplicate configuration source, retry loop, rollback mechanism, or new dependency is introduced.
