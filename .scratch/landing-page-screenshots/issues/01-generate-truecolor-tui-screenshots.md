# 01: Generate TrueColor TUI screenshots for the four core workflows

**Parent specification:** [Landing Page TUI Screenshots](../spec.md)

**What to build:** Extend the automated media-generation pipeline so a single execution captures and exports crisp 1080x600 TrueColor terminal screenshots for the four canonical WPTUI workflows—Provisioning, Websites Hub, In-Pane Progress Monitor, and Website Backup and Restoration—alongside the existing walkthrough video, using safe isolated fixtures and no manual screen capture.

**Blocked by:** None (can start immediately).

**Status:** resolved

- [x] A single invocation builds the current WPTUI executable, prepares an isolated temporary user home and Website directory with non-sensitive sample Websites, and cleans up all temporary staging after completion or failure.
- [x] TrueColor rendering is preserved throughout recording and capture so active cyan focus borders and green status markers match WPTUI's terminal palette rather than falling back to monochrome output.
- [x] An isolated WP-CLI executable stub inside the temporary execution environment handles `WP_DEBUG` read and write calls so the real WPTUI binary drives the genuine In-Pane Progress Monitor for Website Debug Mode without touching real databases or local services.
- [x] Four canonical screenshots are captured at 1080x600: Provisioning (Create Wizard form with Website inputs), Websites Hub (local Websites with `.test` URLs and multi-select checkboxes), In-Pane Progress Monitor (Task Stepper `[✓]` items, live log lines, and summary), and Website Backup and Restoration (format selection showing full ZIP and All-in-One WP Migration `.wpress`).
- [x] Each screenshot is exported in both optimized WebP and PNG fallback formats alongside the existing walkthrough video assets inside the website source directory.
- [x] Generated screenshots contain no personal filesystem paths, credentials, or production Website names.
- [x] Demo: run the media-generation script in a clean environment, confirm temporary staging is removed, inspect all four generated screenshot pairs at 1080x600, and verify TrueColor cyan/green rendering in each capture.
