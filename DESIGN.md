# WPTUI Terminal Design

This document defines WPTUI's approved terminal UI contract. It describes the target design, not the implementation status.

## Scope

Apply the same design conventions to Create, Config, Delete, Backup, Restore, Settings, first-run setup, and Package Catalog search. Keep each workflow's business behavior and safety checks intact while unifying presentation and interaction.

## Terminal presentation

Use a full-screen alternate-screen buffer (`tea.WithAltScreen()`) with a two-column Master-Detail layout. When exiting, restore the terminal screen cleanly.

Optimize for dark terminal backgrounds only. Inherit the terminal background; a light palette and a theme-selection menu are outside scope. Output remains understandable without ANSI color.

Require a minimum terminal geometry of 80 columns by 24 rows. When terminal dimensions fall below this threshold, display a polite resize notice (`Terminal window too small (minimum 80x24 required)`) and restore the layout when resized.

## Language

Use English consistently for UI titles, descriptions, labels, validation messages, keyboard help, progress, and results. Retain user-provided names and content as entered.

## Screen layout and wording

Partition the display into three primary vertical zones:

1. **Header**: Application title `WPTUI — WordPress Local Manager` and active system context.
2. **Body (Master-Detail columns)**:
   - **Sidebar Navigation** (left column, fixed width ~28 chars): Top-level navigation items (`Websites`, `Create`, `Delete`, `Restore`, `Settings`, `Exit`).
   - **Content Pane** (right column, remaining width): Active operational view, multi-step wizard, website action panel, or in-pane progress monitor.
3. **Footer**: Contextual keyboard help reflecting the currently active Focus Mode (Sidebar vs Content).

Visual focus is indicated by high-contrast border and title highlighting:
- **Active pane**: Bright Cyan (`#00FFFF`) border and bold cyan title.
- **Inactive pane**: Muted Gray (`#888888`) border and muted title.

For example:

```text
┌─ WPTUI ──────────────────┬─ Websites Hub ─────────────────────────────────────┐
│                          │ narrow01                                           │
│ > Websites               │   URL:       http://narrow01.test                  │
│   Create                 │   Directory: F:/laravel-herd/wordpress/narrow01    │
│   Restore                │                                                    │
│   Settings               │ demo-site                                          │
│   Exit                   │   URL:       http://demo-site.test                 │
│                          │   Directory: F:/laravel-herd/wordpress/demo-site   │
│                          │                                                    │
└──────────────────────────┴────────────────────────────────────────────────────┘
  Up/Down Navigate · Enter Open Details · Space Multi-Select · q Exit
```

## Visual Theme Palette

Use one shared palette for forms, search, progress, notifications, and reports. These dark-background values consolidate the existing colors from ADR-0010:

| Role | Color | Use |
| --- | --- | --- |
| Focus / highlight | `#00FFFF` | Active cursor, focused field, paths and URLs |
| Success / selected | `#04B575` | Successful outcomes and checked items |
| Warning | `#FFA500` | Warnings, stale fallbacks and skipped actions |
| Error | `#FF4444` | Failed actions and validation errors |
| Muted | `#888888` | Secondary descriptions and keyboard help |

Use bold styling for focus, checked items, and status emphasis. Distinguish states with text or symbols as well as color:

- `>` identifies the active option; `[x]` / `[ ]` identify selected / unselected items.
- A focused checked item retains its green checked label and cyan cursor.
- `[✓]`, `[!]`, `[✗]`, and `[-]` identify successful, warning, failed, and skipped action results respectively.
- Neutral metadata uses `Label: Value` without a success marker.

## Navigation and cancellation

Use conventional field and list controls with context-specific help:

| Key | Behavior |
| --- | --- |
| Up / Down or k / j | Navigate menu choices or list items in the active pane |
| Left / Right | Toggle focus between panes, or change confirmation choices |
| Tab / Shift+Tab | Move focus between Sidebar and Content Pane, or between form fields |
| Space | Toggle multi-select items (e.g., batch selecting Websites or Packages) |
| Enter | Activate/drill into Content Pane from Sidebar; advance or submit within Content Pane |
| Esc | Return focus to Sidebar Navigation from Content Pane (prompts discard confirmation if form is dirty); dismiss active filter |
| q / Ctrl+C | Exit application when on Sidebar Navigation; cancel running background workflow |

Navigating back to the Sidebar preserves entered form values until explicitly submitted or discarded. Dirty forms prompt `Discard changes? (y/n)` upon Esc to prevent accidental data loss. User cancellation during execution stops background tasks gracefully before re-enabling navigation.

## Progress

Display ongoing operations inside the Content Pane via the In-Pane Progress Monitor, leaving the Sidebar visible with muted styling and locked navigation.

Partition the progress view into two distinct vertical sections:
1. **Task Stepper**: Shows a single spinner for the currently executing step alongside persistent checkmarks (`[✓]`, `[!]`, `[✗]`) for completed steps.
2. **Log Viewport**: A scrollable viewport (`bubbles/viewport`) streaming real-time WP-CLI, database, and archive logs.

## Results and errors

Start each report with the actual overall outcome: completed, completed with warnings, failed, or cancelled. Use concise `Label: Value` fields for Website details, archive paths, sizes, durations, and other metadata. Reserve status markers for action outcomes.

Retain applicable per-Package, per-tweak, and per-resource outcomes, including partial success in batch operations. Keep warnings visible rather than presenting an unconditional success headline. Short actions such as opening an editor can use a concise status message instead of an empty report template.

On failure, identify the failed step, the underlying cause, and the cleanup or rollback outcome where applicable. Preserve useful technical detail while redacting credentials and API keys. Keep passwords masked in forms and out of progress, reports, and scrollback. Let the user choose the next action rather than retrying automatically.

```text
Restore completed with warnings

  Website:   narrow01
  URL:       http://narrow01.test
  Directory: F:/laravel-herd/wordpress/narrow01

  [!] HTTPS setup failed; the website remains available over HTTP.
```

## Verification

Verify UI changes on the actual terminal surface: selection, focus, Back/Cancel, running progress, and the resulting report. Include a narrow-window check and colorless output readability. Exercise destructive or failure scenarios with isolated fixtures rather than user Websites. Record any surface that could not be exercised; unit-test success alone is not visual verification.

These rules apply across the full scope above. Feature-specific content and safety requirements remain intact; UI unification does not make different operations share a business workflow.
