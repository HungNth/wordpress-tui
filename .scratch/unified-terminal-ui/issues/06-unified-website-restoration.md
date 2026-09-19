# 06: Unified Website Restoration with Interactive Progress Handoff

**Parent specification:** [Unified Terminal UI](../spec.md)

**Design contract:** [WPTUI Terminal Design](../../../DESIGN.md)

**What to build:** A complete Website Restoration experience for Full ZIP and AI1WM Backup Archives, with consistent archive/input navigation, preserved pending values, coordinated progress, safe cancellation, and truthful results. In the Full ZIP path, an execution-stage SQL dump selection prompt must temporarily own the terminal without spinner interference, then return control to progress.

**Blocked by:** 02: Website Backup with Coordinated Progress — supplies shared contextual selection, operation-display ownership, and cancellation lifecycle. Restore does not depend on the Create or Package selection UI migrations; its shared Website input form already exists.

**Status:** resolved

## Acceptance criteria

- [x] Strategy selection, Backup Archive selection, custom-path input, Website and administrator inputs, and dump selection follow the shared contextual layout, plain English labels, dark palette, and current keyboard help. Restore does not display Create-only tweak or Package-selection prompts.
- [x] Enter Custom Path is first in the archive list; Back is last in navigation menus. Archive validation retains its format, regular-file, readability, and existing path-safety behavior. Empty archive lists remain usable through custom input and Back.
- [x] Pre-execution Back returns through the relevant screens while retaining entered values and compatible choices within the workflow. Restoring a retained choice never bypasses validation against the currently selected strategy or input. Back from the initial Restore screen returns to the main menu.
- [x] Ctrl+C before execution cancels neutrally without creating a Website. Website Slug validation, collision checks, administrator defaults, and strategy preflight behavior remain intact.
- [x] Both restoration engines use the shared running-step spinner and persistent completed-step results. Replace the independent spinner-plus-print arrangement so there is one active display/input owner and no premature step-success reporting.
- [x] When Full ZIP restoration requires choosing among SQL dumps, stop progress rendering and its keyboard handling before presenting the actual interactive choice. After a confirmed choice, resume progress and continue with that dump; no input is consumed by a hidden competing renderer.
- [x] Aborting the execution-stage dump prompt cancels the workflow with applicable cleanup rather than navigating backwards over completed execution work. A cancelled prompt is not reported as an unexplained restore failure.
- [x] Ctrl+C during either engine requests cancellation, shows stopping feedback, and waits for active work and existing restoration cleanup obligations before returning to the main menu. Another operation can then run in the same session.
- [x] Critical failures and cancellations retain current resource ownership and rollback protections. Reports state actual directory/database/TLS/staging cleanup outcomes where applicable and preserve the source Backup Archive; the UI introduces no new destructive cleanup or rollback guarantee.
- [x] Successful reports distinguish completed from completed with warnings and retain meaningful Website details, complete path and URL, Website Database, and administrator username as metadata without decorative success markers. Passwords and API keys remain absent.
- [x] Existing nonfatal TLS behavior and actual final URL are reported accurately. A TLS warning does not become an unconditional success message or a new reason to roll back a restored Website.
- [x] After success, keep completed progress and the report in scrollback and return directly to the main menu without a continuation gate. Genuine errors retain failed-step, redacted cause, and cleanup information, without automatic retries.
- [x] Forms, prompts, progress, and reports remain usable at ordinary and narrow widths and without color. Final resource locations remain complete and copyable.
- [x] Shared Website input form changes migrate all affected callers together while preserving Create's distinct choices. Actual terminal evidence covers both engines and, critically, the real Full ZIP progress-to-dump-prompt-to-progress sequence, its cancellation path, and a controlled cleanup failure.

## Testing seam

Use the application Restore workflow as the primary integration boundary with real UI interaction, fixture archives, and isolated external operation effects. If Restore lacks the injection needed to control work or cleanup deterministically, extend the existing application-level pattern minimally instead of adding lower-level test-only interfaces.

Reuse existing restoration fixtures and cleanup tests for domain safety. The key regression drives the composed execution path with multiple SQL dumps: progress starts, the real selection prompt becomes usable, the selected answer resumes restoration, and outcomes remain truthful. A second path cancels at that prompt and verifies cleanup before menu return and source archive preservation. Renderer-only or single-callback tests do not cover this boundary.

## Demo path

1. Enter Restore with isolated archives, choose Full ZIP, visit custom input, and navigate backwards through pre-execution screens. Verify retained values and active-strategy validation.
2. Restore a fixture containing multiple candidate SQL dumps. Select a dump while execution is paused and observe resumed progress, the final report, and return to the main menu.
3. Cancel at the execution-stage prompt, then cancel a running fixture restoration. Inspect cleanup and source-archive preservation, start another workflow, and repeat the core progress/result checks with the AI1WM fixture.

## Scope boundary

This ticket changes Website Restoration's UI composition and cancellation handling while preserving its engines and safety contracts. It does not implement a new archive parser, migrate backup formats, or reuse Create's business workflow as a shortcut.
