# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.3] - 2026-11-13

### Added

- `Toggle Debug Mode` action in **Websites Hub** to inspect `WP_DEBUG` on a selected **Website** and switch **Website Debug Mode** between enabled (`WP_DEBUG=true`, `WP_DEBUG_LOG=true`, `WP_DEBUG_DISPLAY=false`) and disabled (`false`, `false`, `false`) with **In-Pane Progress Monitor** progress, cancellation (`q`/`Ctrl+C`), and per-constant outcome reporting.

### Changed

- Updated newly generated default `wp_tweaks` in `config.json` to set `WP_DEBUG_DISPLAY` to `false` while retaining `WP_DEBUG=true`, `WP_DEBUG_LOG=true`, `WP_MEMORY_LIMIT='256M'`, `AUTOSAVE_INTERVAL=600`, `WP_POST_REVISIONS=5`, and `EMPTY_TRASH_DAYS=21` (existing `config.json` files must be updated manually).

### Removed

- Legacy Huh-based **Website Configuration** flow and retired `config` branch from the legacy menu, completing the cutover to **Websites Hub** and the **Master-Detail Layout**.

## [0.2.2] - 2026-11-12

### Added

- Pre-TUI command-line flags: `-v`/`--version`, `-u`/`--update` (read-only **Application Update Check** against the `main` branch manifest with SemVer comparison and release link), `-c`/`--config` (prints expected `config.json` path without creating directories or files), and `-h`/`--help`.
- Compile-time embedded `manifest.json` release identity.

## [0.2.1] - 2026-11-11

### Added

- Explicit **Local Development Environment** selection (`herd` for Laravel Herd, `flock` for Flock, and `laragon` on Windows) with ownership-aware **Website Registration**, TLS configuration, and environment-specific URLs.
- **Batch Website Backup** workflow in **Websites Hub** with sequential execution, cancellation support, **Backup Archive** collision protection, secret redaction, and retained-artifact error reporting.
- Custom **Restore Archive** absolute path entry in **Website Restoration**, including quoted clipboard path normalization and in-place source preservation.

### Changed

- Replaced static package lists and the built-in default theme with API-only **Package Catalog** loading from the configured **Package Service**, conditional package step/action visibility, and optional **Active Theme** selection.
- Replaced legacy `used_herd`, `themes`, `plugins`, and `default_theme_slug` settings with `app_name` and `app_command` under strict configuration validation (no automatic legacy schema migration).
- Centered the **Master-Detail Layout** horizontally and improved **In-Pane Progress Monitor** log viewport padding and hanging-indent wrapping.

### Fixed

- Cleared **Websites Hub** multi-selection and returned to the Websites list after deletion.

## [0.2.0] - 2026-09-24

### Added

- Fullscreen two-column **Master-Detail Layout** with **Sidebar Navigation**, **Content Pane**, **Websites Hub**, and **In-Pane Progress Monitor** supporting real-time logs and operation cancellation.
- **Website Configuration** actions for applying `wp_tweaks`, updating administrator credentials via MySQL prepared statements and WP-CLI, and version-aware plugin/theme installation.
- **Website De-provisioning** and **Batch De-provisioning** with **Website Database** extraction/re-verification and `delete_excludes` protection.
- **Website Backup** (consolidated ZIP and All-in-One WP Migration `.wpress` **Backup Archives**) and **Website Restoration** (**Full ZIP Restore Engine** and **AI1WM Restore Engine**).
- **Application Settings** menu with **External Launcher** support (edit configuration in VS Code, open cache directory, open Websites in browser/editor) and config hot-reload.
- Go-managed **Core Cache** (`Core Archive` verification and native extraction) and live debounced **Package Catalog** search with **Package Selection Accumulator**.
- Tag-triggered cross-platform release workflow, `Makefile`, and user-local installers for macOS and Windows.

### Fixed

- UTF-8 rune and Vietnamese IME backspace handling across wizard and package search inputs.
- Sliding-window scrolling and compact single-line rendering in **Websites Hub**, delete, and restore lists.
- Linux directory opening via `xdg-open` and Herd TLS URL reconciliation during AI1WM restoration.

## [0.1.0] - 2026-09-21

### Added

- Initial WPTUI release featuring first-run setup, atomic **Provisioning** (database creation, `wp-config.php` generation, `wp core install`, best-effort `wp_tweaks`, and Herd TLS), and a verified **Package Cache** with stale fallback.

[unreleased]: https://github.com/HungNth/wordpress-tui/compare/v0.2.3...HEAD
[0.2.3]: https://github.com/HungNth/wordpress-tui/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/HungNth/wordpress-tui/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/HungNth/wordpress-tui/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/HungNth/wordpress-tui/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/HungNth/wordpress-tui/releases/tag/v0.1.0
