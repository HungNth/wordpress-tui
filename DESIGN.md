# WPTUI Terminal Design

This document defines WPTUI's approved terminal UI contract. It describes the target design, not the implementation status.

## Scope

Apply the same design conventions to Create, Config, Delete, Backup, Restore, Settings, first-run setup, and Package Catalog search. Keep each workflow's business behavior and safety checks intact while unifying presentation and interaction.

## Terminal presentation

Use inline interfaces in the normal terminal buffer rather than a full-screen dashboard. Preserve completed progress lines and reports in scrollback for review and copying.

Optimize for dark terminal backgrounds only. Inherit the terminal background; a light palette and a theme-selection menu are outside scope. Output remains understandable without ANSI color.

Use a left-aligned, single-column layout that fits the terminal width. Wrap long descriptions and report values, and scroll long option lists within the available height. Preserve complete paths and URLs in final reports rather than replacing them with ellipses.

## Language

Use English consistently for UI titles, descriptions, labels, validation messages, keyboard help, progress, and results. Retain user-provided names and content as entered.

## Screen layout and wording

Use the same hierarchy on each screen:

1. Contextual title, such as `WPTUI / Backup`.
2. A short description naming the current task or Website where relevant.
3. The form fields or choice list.
4. Keyboard help for actions available on that screen.

Use consistent spacing, with two-space indentation for report details. Prefer concise headings over decorative banners, and plain labels over decorative emoji or numbered menu options that have no numeric shortcut. Numbered progress counts still communicate real progress.

Place Search Catalog and Enter Custom Path actions first in their respective lists, and an explicit Back action last in navigation menus. Keep Back out of selectable data in multi-select lists; expose it through keyboard help instead. Shared controls must use task-specific wording: a Backup picker describes backup, not configuration.

For example:

```text
WPTUI / Backup
Choose a backup format for narrow01.

> Full ZIP
  All-in-One WP Migration
  Back

Up/Down Navigate · Enter Select · Esc Back · Ctrl+C Cancel
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
| Up / Down | Navigate list items |
| Left / Right | Change a confirmation choice |
| Tab / Shift+Tab | Move between form fields |
| Space | Toggle a multi-select item |
| Enter | Select, continue, or submit as stated in the current help |
| Esc | Close an active filter/search first; otherwise return to the previous screen |
| Ctrl+C | Cancel the current workflow; exit at the main menu or during first-run setup |

Back preserves entered values and Package selections within the current workflow. Query changes retain accumulated Package selections. Cancelling the workflow abandons pending input. At the first screen of an operation, Back returns to the main menu; at the main menu, Esc stays on the menu and Exit or Ctrl+C exits.

User cancellation is a neutral outcome, not an operation failure. During execution, request cancellation and wait for the operation and its applicable cleanup to finish before returning to the main menu. Keep the display active while stopping. Report cleanup failures separately; never claim cancellation or rollback completed while work is still running.

Back navigation does not undo mutations. A prompt needed during execution temporarily replaces progress output; aborting that execution-stage prompt cancels the workflow rather than navigating across already-applied mutations. Delete retains its explicit confirmation, defaults to No, and identifies the affected directories and databases. Already deleted resources cannot be restored by cancellation; report which resources were removed and which were retained or failed.

After successful execution, preserve the report and return directly to the following destination, without an extra `Press Enter to continue` prompt:

| Workflow | Destination |
| --- | --- |
| Create / Restore / Delete | Main menu |
| Config | Action menu for the current Website |
| Backup | Website selection |
| Settings | Settings menu |
| First-run setup | Main menu |

## Progress

Combine a single spinner for the running step with persistent result lines for completed steps. Mark a step successful only after its work succeeds. Show `[n/total]` only when the total is known; omit fabricated percentages or estimates.

Give one renderer ownership of terminal output at a time. Coordinate progress and external-command output through that owner rather than mixing animated rendering with independent writes. Pause progress rendering and keyboard handling while an interactive prompt is active, including SQL dump selection during Restore, and resume after the answer.

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
