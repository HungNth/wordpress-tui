# 02: Website Backup with Coordinated Progress

**Parent specification:** [Unified Terminal UI](../spec.md)

**Design contract:** [WPTUI Terminal Design](../../../DESIGN.md)

**What to build:** A complete Website Backup workflow with task-correct Website selection, consistent Back/Cancel, one running-step spinner, persistent completed-step results, and a truthful Backup Archive report. Prove the common operation-display and cancellation lifecycle through both existing backup strategies so other workflows can reuse it.

**Blocked by:** 01: Unified Entry, Setup, and Application Settings — supplies the shared presentation and navigation conventions.

**Status:** resolved

## Acceptance criteria

- [x] Entering Backup presents a Website picker whose title and description explicitly describe backup rather than Website Configuration. Empty lists, configuration problems, and discovery failures receive the appropriate contextual message.
- [x] Website and strategy selection follow the shared layout, palette, keyboard help, and plain-label conventions. Back is last in navigation menus. Strategy Back returns to Website selection; Back from the initial picker returns to the main menu.
- [x] Pending selections survive Back within this workflow. Both Full ZIP and AI1WM backup retain their existing artifact, exclusion, cleanup, and error-propagation behavior.
- [x] During either strategy, one active display owner renders a spinner describing the current step and leaves persistent outcome lines as steps finish. A running step is not marked successful before its work succeeds.
- [x] Relevant operation and external-command output is coordinated with the active renderer rather than independently printed over animation. A known total may produce a truthful step count; unknown totals do not produce invented percentages or estimates.
- [x] The operation-display lifecycle has a clear handoff for prompts: progress rendering and its input handling can stop while a prompt owns the terminal, then resume. Reuse this same lifecycle for later workflows; do not add a general-purpose event framework or test-only architecture.
- [x] Ctrl+C before execution cancels neutrally. During execution it requests cancellation, shows stopping feedback, and waits for the backup and its applicable cleanup to finish before returning to the main menu. A later operation can run in the same application session.
- [x] Cancellation and genuine failure are distinguishable. Reports preserve the underlying cause and any partial-artifact or cleanup outcome required by the existing backup contract; the UI neither fabricates rollback nor deletes additional resources to simplify the report.
- [x] A successful report identifies the actual strategy and final Backup Archive location, size, and elapsed time as Label: Value metadata without decorative success markers. Paths remain complete; status symbols describe action outcomes only.
- [x] Results and completed progress remain in scrollback. After success, Backup returns directly to Website selection with no extra Press Enter or continuation question. Failures do not trigger automatic retries.
- [x] Credentials, API keys, and sensitive external-command details are redacted from progress, reports, and scrollback. Errors retain useful non-secret technical context.
- [x] The reusable Website picker remains correctly worded in its other callers. Shared progress/cancellation interface changes migrate affected callers cleanly and keep the application working; this ticket does not leave obsolete APIs or compatibility aliases.
- [x] Actual terminal evidence covers both strategies, ordinary and narrow widths, colorless readability, success, a controlled error, and cancellation followed by another usable operation.

## Testing seam

Use the existing composed Backup flow boundary with temporary Website and backup locations and deterministic operation fixtures. Reuse the Full ZIP and AI1WM Backup test patterns. Verify produced artifact metadata, preserved resources on failure, navigation destinations, and cancellation timing rather than command forwarding or style internals.

A cancellation regression should hold fixture work or cleanup at a deterministic boundary and establish that the menu cannot resume before it finishes. The visual smoke must run real form/progress rendering; a captured string or mocked prompt is not evidence of single-owner rendering. The real execution-stage prompt scenario is owned by Ticket 06.

## Demo path

1. Choose Backup, select a fixture Website, move between Website and strategy selection with Back, then perform Full ZIP backup.
2. Observe the running step, completed results, full artifact details, and automatic return to Website selection. Repeat with the AI1WM fixture.
3. Inject a safe failure and then a cancellable long-running fixture step. Cancel, observe stopping and cleanup results, and start another workflow without restarting WPTUI.

## Scope boundary

This is the vertical slice that establishes the reusable operation-display lifecycle, not a standalone renderer project. Backup business behavior remains unchanged. Other workflows adopt the lifecycle in their own tickets, and destructive verification uses only isolated fixture resources.
