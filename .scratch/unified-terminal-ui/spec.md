# Unified Terminal UI Specification

Status: ready-for-agent

## Problem Statement

WPTUI's options feel like separate interfaces rather than one application. Although forms already share a Visual Theme Palette, individual workflows use different menu labels, navigation conventions, progress indicators, status colors, report layouts, and cancellation handling. Website Backup even reuses a Website Configuration picker whose title and description describe the wrong task.

A local WordPress developer must therefore relearn how to go back, cancel, interpret progress, and find results when moving between Create, Config, Delete, Backup, Restore, and Settings. Some summaries attach success markers to neutral metadata, while other outcomes use unrelated headings or unstyled messages. Restore starts an animated spinner while its progress callback independently prints lines, creating competing owners of the same terminal display.

The developer uses a dark terminal and wants a consistent, readable, keyboard-driven interface without a full-screen dashboard or changes to the underlying WordPress operations.

## Solution

Unify every terminal surface under the approved design contract: the main menu, all six options, first-run setup, Package selection and Package Catalog search, validation, notifications, progress, and completion reports.

Use an inline, left-aligned, single-column interface in English, optimized for dark backgrounds. Every screen follows a contextual title, short description, form or choice list, and relevant keyboard help. Keep completed progress and results in terminal scrollback. Use one semantic palette and consistent symbols, with readable text even when color is unavailable.

Make Back preserve pending input, distinguish cancellation from failure, and return users to predictable menus. Combine a single running-step spinner with persistent completed-step results, suspending progress whenever a prompt needs the terminal. Report actual success, warnings, failure, or cancellation with complete resource information and redacted technical details.

Preserve each workflow's existing business behavior, validation, destructive confirmations, and applicable cleanup obligations. UI consistency must not erase differences such as irreversible De-provisioning versus rollback-capable Provisioning and Website Restoration.

## User Stories

1. As a local WordPress developer, I want Create, Config, Delete, Backup, Restore, and Settings to share the same visual conventions, so that changing options does not require learning a different interface.
2. As a first-time WPTUI user, I want setup to follow the same conventions as the main application, so that the interface is predictable from the first launch.
3. As a local WordPress developer, I want Package selection and Package Catalog search to match the surrounding forms, so that searching does not feel like entering another application.
4. As a local WordPress developer, I want every screen to identify its current action and relevant Website, so that I can tell what I am about to modify.
5. As a local WordPress developer, I want the Website Backup picker to describe backup rather than configuration, so that shared controls do not give misleading instructions.
6. As a local WordPress developer, I want titles, descriptions, labels, validation, help, progress, and results to use English consistently, so that terminology does not change between screens.
7. As a local WordPress developer, I want my entered Website Names and other content preserved, so that UI language standardization does not alter my data.
8. As a terminal user, I want WPTUI to render inline rather than replace the terminal with a full-screen dashboard, so that I retain access to terminal history.
9. As a terminal user, I want completed progress and reports to remain in scrollback, so that I can review and copy them after returning to a menu.
10. As a terminal user, I want a left-aligned single-column layout with consistent spacing, so that forms and results are easy to scan.
11. As a terminal user, I want concise headings and plain action labels rather than decorative banners, emoji, or nonfunctional numeric shortcuts, so that decoration does not compete with useful information.
12. As a terminal user, I want long descriptions to wrap and long lists to scroll within the window, so that a narrow terminal remains usable.
13. As a terminal user, I want final paths and URLs preserved in full, so that I can copy accurate resource locations even when they wrap.
14. As a dark-terminal user, I want one high-contrast palette optimized for my environment, so that important states are readable without a theme configuration step.
15. As a terminal user, I want cyan focus and a visible cursor marker, so that I can identify the active option immediately.
16. As a terminal user, I want checked items to remain green when focus moves elsewhere, so that selected items remain distinct from the current cursor position.
17. As a terminal user, I want a focused checked item to retain its green selection and cyan cursor, so that focus does not obscure selection state.
18. As a terminal user, I want success, warning, failure, and skipped outcomes to use consistent text, symbols, and colors, so that I can interpret them across all workflows.
19. As a terminal user, I want statuses to remain understandable without ANSI color, so that color is not the only source of information.
20. As a keyboard user, I want help to describe only the actions currently available, so that displayed shortcuts reliably match their behavior.
21. As a keyboard user, I want consistent arrow, Tab, Shift+Tab, Space, and Enter behavior, so that familiar controls work across forms and lists.
22. As a keyboard user, I want Search Catalog and Enter Custom Path actions first in their respective lists, so that alternate input is immediately discoverable.
23. As a keyboard user, I want Back last in navigation menus and available through keyboard help in multi-select forms, so that navigation cannot be mistaken for a selected Package or Website.
24. As a keyboard user, I want Esc to close active search or filtering before navigating back, so that I can leave a local interaction without accidentally leaving its parent workflow.
25. As a local WordPress developer, I want Back to retain entered values and Package selections within a workflow, so that correcting an earlier choice does not require re-entering unrelated information.
26. As a local WordPress developer, I want Package selections retained across multiple search queries, so that finding another Package does not discard earlier choices.
27. As a local WordPress developer, I want Back from the first operation screen to return to the main menu, so that I always have a predictable way out before execution.
28. As a keyboard user, I want Esc at the main menu to leave the menu open, so that an extra Back press does not unexpectedly exit WPTUI.
29. As a local WordPress developer, I want Ctrl+C to cancel the current workflow and discard pending input, so that I can abandon a task without an erroneous failure report.
30. As a first-time WPTUI user, I want Ctrl+C during setup to exit, so that I can defer setup rather than enter a menu that requires completed configuration.
31. As a keyboard user, I want Exit or Ctrl+C at the main menu to exit WPTUI, so that application exit is distinct from returning from a workflow.
32. As a local WordPress developer, I want cancellation during execution to wait for stopping and applicable cleanup before returning to a menu, so that I do not start another action while the previous one still runs.
33. As a local WordPress developer, I want a stopping indication and a truthful cleanup outcome, so that cancellation never implies successful rollback before it has completed.
34. As a local WordPress developer, I want cancellation of one workflow to leave the main application usable, so that I can start another operation without restarting WPTUI.
35. As a local WordPress developer, I want De-provisioning to retain explicit confirmation defaulting to No and identifying directories and Website Databases, so that UI simplification does not weaken destructive-action safety.
36. As a local WordPress developer, I want cancelled or partially failed De-provisioning to identify removed, retained, and failed resources, so that I know what remains without assuming deletion can be undone.
37. As a local WordPress developer, I want Create, Restore, and Delete to return to the main menu after completion, so that one-off operations have a consistent endpoint.
38. As a local WordPress developer, I want Config to return to the action menu for the current Website, so that I can perform another configuration action without a redundant continuation question.
39. As a local WordPress developer, I want Backup to return to Website selection, so that I can choose the next Website to back up.
40. As a local WordPress developer, I want Settings to remain in its menu after an action and completed setup to enter the main menu, so that each workflow returns to a useful destination.
41. As a keyboard user, I want results preserved without an extra Press Enter to continue prompt, so that a completed operation does not introduce an unnecessary keystroke.
42. As a local WordPress developer, I want one spinner describing the running step and persistent results for completed steps, so that I can distinguish current activity from finished work.
43. As a local WordPress developer, I want progress counts only when the total is known, so that WPTUI does not show invented percentages or misleading estimates.
44. As a local WordPress developer, I want a step marked successful only after it succeeds, so that progress does not announce an outcome prematurely.
45. As a local WordPress developer, I want progress and command output coordinated by one display owner, so that animation does not interfere with messages or keyboard input.
46. As a developer restoring a Website, I want progress suspended while selecting a database dump and resumed after the answer, so that an execution-stage prompt remains readable and interactive.
47. As a local WordPress developer, I want aborting an execution-stage prompt to cancel safely rather than navigate backwards over mutations, so that Back cannot imply undoing completed work.
48. As a local WordPress developer, I want a report headed by the actual overall outcome, so that completed-with-warnings, failure, and cancellation are not presented as unconditional success.
49. As a local WordPress developer, I want Website and Backup Archive metadata displayed as Label: Value fields without decorative success markers, so that resource attributes are distinct from action outcomes.
50. As a local WordPress developer, I want applicable per-Package, per-tweak, and per-resource results retained, so that a compact report does not hide meaningful details or partial failures.
51. As a local WordPress developer, I want short Settings actions to produce concise status messages, so that a simple action does not produce an empty report template.
52. As a local WordPress developer, I want errors to identify the failed step, underlying cause, and cleanup or rollback result where applicable, so that I can decide how to recover.
53. As a local WordPress developer, I want passwords masked and credentials and API keys excluded from terminal output, so that copied reports and scrollback do not reveal secrets.
54. As a local WordPress developer, I want to choose the next action after failure rather than have WPTUI retry automatically, so that recovery stays under my control.
55. As a WPTUI maintainer, I want one authoritative design document referenced by agent guidance and historical UI decisions, so that future options follow the same conventions instead of introducing another style.
56. As a local WordPress developer, I want terminal interaction verified across every affected workflow, so that passing noninteractive tests is not mistaken for a working UI.

## Implementation Decisions

1. **Scope and preserved behavior.** Apply the design to the main menu, Create, Config, Delete, Backup, Restore, Settings, first-run setup, Package selection and search, validation, notifications, progress, and results. Preserve existing business requirements, input validation, resource identity, data handling, and cleanup policies. This is a presentation and interaction cutover, not a redesign of WordPress operations.

2. **Existing terminal stack.** Reuse the installed Huh forms, Bubble Tea interaction model, and Lip Gloss styling. Extend their existing integration where necessary rather than introduce another terminal framework or a general-purpose UI framework. Keep the application inline in the normal terminal buffer and inherit the terminal background.

3. **Presentation ownership.** The terminal UI module owns layout, wording, keyboard help, semantic styles, progress rendering, and reports. Application orchestration owns workflow transitions and supplies task-specific context and operation outcomes. Operation modules retain domain behavior and communicate progress or results without independently taking ownership of the display. Move scattered presentation into the shared UI conventions rather than retain parallel legacy renderers.

4. **Dark-only Visual Theme Palette.** Consolidate focus and highlighted resource locations to cyan `#00FFFF`, successful outcomes and selected items to green `#04B575`, warnings/stale fallbacks/skipped actions to amber `#FFA500`, failures to red `#FF4444`, and secondary text to muted `#888888`. Use bold emphasis for focus, checked items, and status emphasis. Preserve semantic text and symbols when ANSI color is unavailable. No light palette, adaptive theme selection, or theme configuration is required.

5. **State markers.** Use `>` for the active option, `[x]` and `[ ]` for checked and unchecked data, and `[✓]`, `[!]`, `[✗]`, and `[-]` for successful, warning, failed, and skipped action results. A checked option under the cursor keeps its green label and cyan cursor. Metadata is not a successful action and receives no success marker.

6. **Screen composition.** Use a contextual WPTUI/action heading, a short task description, the form or options, and relevant help. Use a single left-aligned column and consistent whitespace; report details use two-space indentation. Avoid decorative banners, emoji, and numbered menu labels without numeric shortcuts. Real step counts remain useful information. Shared selectors receive context-appropriate titles and descriptions.

7. **Option ordering and sizing.** Search Catalog and Enter Custom Path appear first in their respective lists. Explicit Back appears last in navigation menus; multi-select data lists do not contain a selectable Back item. Fit the current terminal width, wrap descriptions and report values, and scroll long lists within the available height. Final resource paths and URLs remain complete rather than ellipsized.

8. **Keyboard contract.** Up/Down navigates lists; Left/Right changes confirmation choices; Tab/Shift+Tab moves between form fields; Space toggles a multi-select item; Enter selects, continues, or submits as advertised by the current help. Esc first closes active search/filtering, otherwise returns to the previous screen. Esc on the main menu stays there. Exit or Ctrl+C on the main menu exits. Help changes with the screen and active interaction.

9. **Navigation state and outcomes.** Distinguish confirmation, Back, workflow cancellation, and actual failure at the form/workflow boundary. Back preserves entered values and Package selections within the current workflow; query changes preserve the Package Selection Accumulator. At the first screen of an operation, Back returns to the main menu. Cancelling a workflow abandons pending input rather than reporting it as a failure. Ordinary validation stays attached to the relevant input and retains the user's entries.

10. **Cancellation lifetime and safety.** Ctrl+C during a workflow requests cancellation, propagates it to active work, waits for stopping and applicable cleanup, and then returns to the main menu. A workflow cancellation must not permanently cancel the application lifetime and prevent subsequent operations. During first-run setup, Ctrl+C exits. Keep a stopping indication while shutdown is in progress and report cleanup failures separately. Preserve existing rollback obligations; do not promise rollback for irreversible deletion or claim completion while work remains active.

11. **Destructive actions and execution-stage prompts.** De-provisioning retains its explicit resource confirmation and default No choice. Back navigation is a pre-execution interaction, not an undo operation. If execution needs input, suspend progress and give the prompt sole control; aborting that prompt cancels the workflow with applicable cleanup instead of navigating backwards over mutations. A cancelled or partially failed deletion reports the resource outcomes already produced.

12. **Return destinations.** After successful execution, Create/Restore/Delete return to the main menu, Config returns to the current Website's action menu, Backup returns to Website selection, Settings remains in its menu, and first-run setup enters the main menu. Preserve the report and show the destination directly, removing the redundant Config continuation question and introducing no generic Press Enter to continue gate. Normal Back remains available at each destination.

13. **Progress lifecycle.** Display a single spinner for the running step and persistent result lines after steps finish. Mark success only when the step succeeds. Use `[n/total]` only when the total is known, with no fabricated percentages or estimates. Coordinate all display writes, including relevant external-command output, through the active owner. Pause both progress rendering and its input handling for required prompts, including Restore's SQL dump selection, and resume after an answer. Preserve completed lines in scrollback.

14. **Results and errors.** Reports state the actual overall outcome: completed, completed with warnings, failed, or cancelled. Use Label: Value metadata and retain all applicable per-Package, per-tweak, and per-resource outcomes, including partial success. Short actions can use a concise status message. Errors retain the failed step, underlying cause, and applicable cleanup or rollback outcome, with useful technical detail. Mask password input and redact credentials and API keys from progress, reports, and scrollback. Do not introduce automatic retries.

15. **Scope of interfaces and persistence.** Make only the UI-facing interface changes needed to carry navigation outcomes, retained input state, progress, and cancellation through the existing application flows. Migrate all affected callers as a clean cutover. No configuration schema, Package service API, backup format, database schema, or command-line entrypoint change is required. Reuse existing dependencies and avoid new persistence for UI state.

16. **Design governance.** Maintain a single authoritative design contract. Agent guidance points to it for terminal changes; historical UI decisions explicitly defer to it for current presentation while retaining their behavioral decisions. Preserve search-first placement and multi-query selection accumulation even though decorative search labels change.

## Testing Decisions

1. **Primary integration seam.** Prefer the existing application entry/flow boundary, using the application options and composed workflow dependency injection already exercised by the test suite. Test user-observable transitions and outcomes through the real orchestration while isolating filesystem locations and external WordPress, database, Package, and launcher effects. Reuse this boundary across workflows instead of adding separate seams for each style or widget. If a workflow lacks the injection needed for deterministic cancellation or progress, extend this same high-level boundary minimally rather than create another test-only architecture.

2. **What constitutes a useful test.** Assert behavior that can fail under a plausible defect: the next visible menu, retained user input and Package selections, cancellation versus failure, cleanup finishing before return, truthful partial outcomes, secret redaction, and preservation of unrelated resources. Avoid assertions about helper invocation order, copied fields, style-object internals, arbitrary wording, or exact escape-sequence snapshots. Update existing tests broken by the intentional contract change; remove obsolete tests that merely pin incidental wording or implementation rather than re-pin them.

3. **Navigation and cancellation coverage.** Exercise Back across screens with data retained; search/filter Esc precedence; first-screen Back; workflow cancellation followed by another usable workflow; main-menu and setup exit; correct post-action destinations; and absence of unnecessary continuation prompts. During running work, hold a fixture operation or cleanup at a deterministic boundary and verify that navigation cannot resume before stopping and cleanup complete. Check partial deletion reporting without implying that removed resources were restored.

4. **Progress/prompt composition coverage.** Drive a multi-step operation with progress, a required interactive prompt, and subsequent progress. Verify one active display/input owner, usable prompt interaction, persistent completed results, and no premature success. Include cancellation from the execution-stage prompt and a warning/failure result. The Restore dump-selection path is the key scenario because it crosses the progress/form boundary.

5. **Modules and prior art.** The application orchestration and terminal UI are the main targets. Existing composed Create smoke coverage, composed Website Configuration coverage, Full ZIP and AI1WM Backup flow tests, multi-Website De-provisioning tests, and Restore cleanup tests provide fixture and failure-injection patterns. The current interactive multi-query Live Search model test provides prior art for feeding key events and asserting accumulated Package selections. Reuse existing domain tests for unchanged safety behavior rather than duplicate every operation's business tests in the UI suite.

6. **Actual terminal verification.** Run the real terminal interface with isolated fixture resources and exercise every in-scope surface: setup, the main menu, all six options, Package selection/search, custom archive input, and execution-stage dump selection. Observe focus, checked items, current help, Back/Cancel, progress, result readability, and return destinations on a dark terminal at ordinary and narrow sizes. Check that colorless output remains understandable and wrapped final paths/URLs remain complete. Scripted prompt callbacks and captured plain output alone do not establish visual or keyboard correctness.

7. **Safety and evidence.** Keep test Websites, configuration, archives, databases, and external launch effects isolated from the user's resources. Use deterministic fixture backends for destructive and failure/cancellation cases. Report exactly which terminal scenarios and supported operating systems were exercised; do not claim Windows and macOS visual verification from a single-platform run. Retain regression tests only for meaningful behavioral boundaries; use throwaway terminal smoke tooling for presentation checks and remove it after recording evidence.

## Out of Scope

- Light-background optimization, automatic dark/light palettes, a theme picker, or configurable color settings.
- A full-screen dashboard, alternate-screen application redesign, mouse-driven navigation, or new GUI/web surfaces.
- Translation to Vietnamese or an internationalization framework; user-provided content remains unchanged.
- New WordPress features or changes to Provisioning, Website Configuration, De-provisioning, Website Backup, or Website Restoration business contracts beyond the agreed interaction and cancellation presentation.
- Changing backup archive formats, Package service endpoints, database identity rules, credential-reconciliation policy, or configuration schema.
- Undoing completed deletion, adding new rollback guarantees, automatic retries, fabricated progress percentages, telemetry, or persistent operation history.
- Introducing a replacement terminal framework, a general-purpose design-system framework, or a broad unit-test suite for style internals.
- Implementation-ticket decomposition, source-code implementation, and Git commits as part of publishing this spec.

## Further Notes

- This spec synthesizes the approved Q1–Q10 design discussion. Q6–Q9 were accepted as recommended; Q10 was explicitly narrowed to dark backgrounds only.
- [DESIGN.md](../../DESIGN.md) is the authoritative UI contract. [AGENTS.md](../../AGENTS.md) requires it for terminal changes.
- [ADR-0009](../../docs/adr/0009-head-search-option-and-live-package-selection.md) retains search-first placement and selection accumulation. [ADR-0010](../../docs/adr/0010-high-contrast-ui-palette-and-summary-system.md) retains the semantic high-contrast roles. Both defer to the design contract for current presentation.
- Existing automated tests often replace prompts or external operations. They are useful behavioral seams, not evidence that the redesigned terminal has already been exercised.
- The specification is ready for agent consumption and subsequent ticket decomposition. This status does not authorize implementation or commits; follow the project's approval workflow.
