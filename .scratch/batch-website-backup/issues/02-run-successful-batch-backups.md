# 02: Run Successful Batch Website Backups from the Websites Hub

**What to build:** Let a developer select multiple Websites, choose `Backup`, select one existing backup strategy for the whole selection, and complete a sequential Batch Website Backup. Show familiar per-Website progress and a concise all-success report, then return to a refreshed Websites Hub with the selection cleared.

**Blocked by:** 01: Protect Backup Archives and Surface Retained Artifacts.

**Status:** done

**Parent specification:** Batch Website Actions and Backup Specification

- [x] Enter with more than one selected Website opens a batch action chooser instead of opening Delete confirmation.
- [x] The chooser lists `Backup` first with initial focus and `Delete` second; contextual keyboard help matches the available navigation and activation keys.
- [x] Choosing `Delete` enters the existing batch Delete confirmation without changing its default, execution, cancellation, or reporting behavior.
- [x] The `d` shortcut remains a direct path to batch Delete, and Enter with zero or one selected Website retains its current behavior.
- [x] Choosing `Backup` shows exactly `Full source code & database (.zip)` and `All-in-One WP Migration (.wpress)` with their existing semantics.
- [x] One strategy is submitted with the complete selected Website set and applies to every Website; selecting it starts execution without another confirmation.
- [x] Esc and the established Back action return through strategy selection, action selection, and the Websites list while preserving selection before execution starts.
- [x] Websites are backed up sequentially in stable list order, and the next Website does not start until the current Website reaches a terminal outcome.
- [x] Full ZIP and AI1WM both run through their existing engines, package resolution, archive naming, storage, and four-step progress vocabulary rather than duplicated implementations.
- [x] One In-Pane Progress Monitor remains active for the batch; its title shows the active Website and one-based `i/N` position, while its log retains output from earlier Websites.
- [x] An all-success run ends with `Batch Backup Complete`, displays the common format once, and lists each Website with its final Backup Archive path and formatted and raw size.
- [x] The report remains visible until dismissed; dismissal refreshes Website discovery, clears all selection state, and returns to the Websites list.
- [x] Model-level tests cover key-driven action and strategy navigation, and application-level tests use deterministic injected backup runners to prove sequential execution and all-success reporting without WP-CLI or MySQL.
- [x] Actual terminal verification covers selection, focus, Back/Esc, both strategy choices, running progress, report dismissal, cleared selection, the 80x24 layout, and readability without ANSI color using isolated Website fixtures.
