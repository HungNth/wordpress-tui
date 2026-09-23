# 04: In-Pane Progress Monitor and Log Viewport

**Parent specification:** [Master-Detail Bubble Tea TUI](../spec.md)

**Design contract:** [WPTUI Terminal Design](../../../DESIGN.md)

**What to build:** The In-Pane Progress Monitor rendered inside the Content Pane during execution of long-running operations (Provisioning, Configuration, Backup, Restoration, De-provisioning). Partition the Content Pane vertically into: (1) a Task Stepper at the top displaying a single spinner for active task and persistent checkmarks (`[✓]`, `[!]`, `[✗]`) for completed tasks; and (2) a scrollable Log Viewport (`bubbles/viewport`) at the bottom capturing real-time stdout/stderr streams. Lock and dim Sidebar navigation during active execution.

**Blocked by:** 01 (needs master-detail shell and layout).

**Status:** resolved

## Acceptance criteria

- [x] When an operation begins executing, the Content Pane transitions to the In-Pane Progress Monitor.
- [x] During execution, Sidebar navigation is locked and styled with dimmed borders/text; keystrokes are prevented from triggering navigation actions.
- [x] The Task Stepper renders in the upper section of the Content Pane:
  - Displays a single active spinner for the currently running step.
  - Completed steps display persistent status markers: `[✓]` for success (green), `[!]` for warnings (orange), `[✗]` for failures (red).
  - Shows `[n/total]` step counters only when total count is predetermined.
- [x] The Log Viewport renders in the lower section of the Content Pane using `bubbles/viewport`:
  - Streams real-time command output from WP-CLI, database commands, and archive extractions.
  - Automatically scrolls down as new log lines arrive.
  - Supports manual scrolling via `Up` / `Down` or `PageUp` / `PageDown` to inspect history.
- [x] On completion, the progress view renders the final outcome summary (success, completed with warnings, or failed), redacting passwords and sensitive tokens.
- [x] Completed operations enable an `[ Enter / Esc ] Continue` prompt to return to the appropriate destination (Websites Hub or Sidebar) per `DESIGN.md`.

## Testing seam

The `tea.Model` for In-Pane Progress Monitor. Test by dispatching simulated step progress messages and log chunk messages to `Update(msg)`, verifying step checkmark transitions in `View()`, auto-scrolling log lines in the viewport, and asserting that sidebar navigation is rejected during execution.

## Demo path

1. Trigger a test provisioning or backup operation.
2. Observe Content Pane split into Stepper (top) and Log Viewport (bottom).
3. Verify spinner spins on active step and turns to `[✓]` upon completion.
4. Verify live stdout lines append smoothly into the lower viewport.
5. Use `Up` arrow to scroll up in the log viewport and review previous output lines.
6. Upon completion, observe summary report with completed status.

## Scope boundary

Delivers the reusable In-Pane Progress Monitor model and viewport. Integration with individual flows connects to Tickets 02, 03, and 05.
