# Batch Website Actions and Backup Specification

Status: ready-for-agent

## Problem Statement

A local WordPress developer can select multiple Websites in the Websites Hub, but pressing Enter currently assumes the selection is for deletion and opens the batch Delete confirmation immediately. The developer cannot choose another action for the selected Websites and must run Website Backup separately for each Website.

The existing single-Website Backup workflow already supports Full ZIP and All-in-One WP Migration exports, progress reporting, cancellation, and Backup Archive storage. Batch support must expose those same strategies for a multi-selection without weakening Delete safety, duplicating backup engines, overwriting an existing Backup Archive, or hiding partial failures and retained recovery artifacts.

## Solution

When more than one Website is selected in the Websites Hub, Enter opens a batch action chooser instead of opening Delete confirmation. The chooser offers `Backup` first and `Delete` second, with `Backup` focused by default. The existing `d` shortcut continues to open batch Delete confirmation directly, and Enter with zero or one selected Website retains its current behavior.

Choosing `Delete` enters the existing batch Delete flow unchanged. Choosing `Backup` presents the existing two strategy choices:

1. `Full source code & database (.zip)`
2. `All-in-One WP Migration (.wpress)`

One selected strategy applies to every selected Website. Selecting a strategy starts Batch Website Backup without another confirmation screen. Websites are processed sequentially. A failure is recorded for the affected Website and processing continues with the remaining Websites. Cancelling stops the active backup, waits for its existing cleanup behavior, and skips every queued Website.

One In-Pane Progress Monitor remains active for the whole batch. Its title identifies the active Website and its `i/N` position, its task stepper reuses the selected strategy's existing four steps, and its log retains output from all attempted Websites. The final report states the aggregate outcome and a concise outcome for every selected Website. After the user dismisses the report, the Websites Hub refreshes and clears the selection.

Backup Archives remain directly under `backup_path` and retain the current naming conventions. Relocation must never overwrite an existing destination. A collision fails only the affected Website, reports the conflicting path, and does not stop the batch. Recovery artifacts retained after failure or cancellation remain available and are reported with their path and cleanup outcome whenever known.

## User Stories

1. As a local WordPress developer, I want Enter on a multi-selection to open a batch action chooser, so that selecting Websites does not imply deletion.
2. As a keyboard user, I want `Backup` focused by default in the batch action chooser, so that repeated Enter presses do not prioritize a destructive operation.
3. As a developer who frequently removes disposable Websites, I want the `d` shortcut to keep opening batch Delete confirmation directly, so that the established fast path remains available.
4. As a developer working with zero or one selected Website, I want Enter to retain its current behavior, so that this feature does not change the single-Website workflow.
5. As a developer choosing `Delete`, I want the existing destructive confirmation to remain unchanged and default to cancellation, so that batch action selection does not weaken deletion safety.
6. As a developer choosing `Backup`, I want to see `Full source code & database (.zip)`, so that I can create the same consolidated source-and-database Backup Archive available for one Website.
7. As a developer choosing `Backup`, I want to see `All-in-One WP Migration (.wpress)`, so that I can create the same portable AI1WM Backup Archive available for one Website.
8. As a developer backing up several Websites, I want to select the backup strategy once for the whole batch, so that I do not repeat the same choice for every Website.
9. As a developer who has explicitly selected Websites, an action, and a backup strategy, I want execution to start without another confirmation, so that a non-destructive workflow is not padded with a redundant prompt.
10. As a developer running a disk-intensive backup, I want Websites processed sequentially, so that archive creation, database export, logs, and cleanup remain predictable.
11. As a developer watching Batch Website Backup, I want the progress title to show the active Website and `i/N`, so that I know both the current target and overall position.
12. As a developer familiar with single-Website Backup, I want the active Website to use the same four strategy-specific progress steps, so that batch progress has familiar meaning.
13. As a developer diagnosing a long-running batch, I want the log to retain output from earlier Websites, so that moving to the next Website does not erase evidence.
14. As a developer backing up independent Websites, I want a failure for one Website not to block later Websites, so that one fault does not discard the rest of the batch's value.
15. As a developer who cancels a running batch, I want the active backup cancelled and its cleanup allowed to finish, so that cancellation does not abandon an operation mid-cleanup.
16. As a developer who cancels a running batch, I want queued Websites skipped rather than started, so that cancellation takes effect for all remaining work.
17. As a developer reviewing a successful Website outcome, I want to see its Backup Archive path and size, so that I can locate and assess the artifact.
18. As a developer reviewing a failed Website outcome, I want to see the failure cause, so that I know what prevented its backup.
19. As a developer reviewing a cancelled batch, I want unstarted Websites marked as skipped or cancelled, so that they are not mistaken for successful or failed attempts.
20. As a developer reviewing the final report, I want the common backup strategy displayed once, so that per-Website rows stay concise.
21. As a developer reviewing the final report, I want an outcome heading that distinguishes complete success, completion with warnings, total failure, and cancellation, so that the overall result is immediately accurate.
22. As a developer who uses `backup_path`, I want every generated Backup Archive stored directly there under the current naming convention, so that batch and single-Website archives remain discoverable in the same place.
23. As a developer with an existing Backup Archive, I want a same-name collision to fail without overwriting that archive, so that an immutable recovery artifact is never destroyed.
24. As a developer whose one Website hits an archive collision, I want the conflicting path reported and the remaining Websites attempted, so that the collision is actionable without aborting unrelated work.
25. As a developer whose failed Full ZIP backup retains a database dump, I want its path and cleanup outcome reported, so that sensitive recovery data is not left unnoticed.
26. As a developer whose failed AI1WM backup retains a source `.wpress`, I want its path and cleanup outcome reported whenever known, so that I can recover or remove it deliberately.
27. As a developer using AI1WM, I want the required extension to remain installed as it does today, so that batch support does not silently alter the existing strategy semantics.
28. As a developer reading a batch result, I want the report to remain visible until I dismiss it, so that navigation does not erase outcomes before I can inspect them.
29. As a developer returning to the Websites Hub after a batch report, I want the previous selection cleared, so that I do not accidentally create duplicate backups.
30. As a keyboard user who changes direction before execution, I want Esc or the established Back action to return through strategy selection, action selection, and the Websites list without losing the Website selection, so that navigation is reversible.
31. As a terminal user, I want contextual keyboard help and non-color status markers throughout the batch workflow, so that interaction and outcomes remain understandable without ANSI color.
32. As a terminal user at the minimum supported geometry, I want the action chooser, progress, logs, and report to remain usable at 80x24, so that batch support respects the terminal design contract.
33. As a developer facing a failed backup, I want WPTUI not to retry automatically, so that expensive or unsafe commands are not repeated without my decision.
34. As a maintainer, I want Batch Website Backup to orchestrate the existing backup engines rather than duplicate them, so that single and batch semantics cannot drift.
35. As a maintainer, I want archive collision protection enforced by the shared relocation boundary, so that every caller receives the same no-overwrite guarantee.

## Implementation Decisions

1. **Authoritative entry behavior**
   - The feature extends the Websites Hub multi-selection flow; it does not create another top-level Backup destination or a separate batch picker.
   - Enter opens the batch action chooser only when more than one Website is selected.
   - Enter with zero or one selected Website remains unchanged.
   - The `d` shortcut remains a direct route to the existing batch Delete confirmation whenever the current Delete shortcut is valid.

2. **Batch action chooser**
   - The actionable choices are `Backup` and `Delete`.
   - `Backup` appears first and receives initial focus; `Delete` appears second.
   - Choosing `Delete` delegates to the existing batch Delete state and handler without changing its confirmation, execution, failure, or reporting behavior.
   - Esc and the established Back interaction return to the Websites list while preserving the current selection because execution has not started.
   - Footer help reflects the chooser's actual navigation and activation keys.

3. **Backup strategy selection**
   - Reuse the existing labels and semantics for Full ZIP and AI1WM.
   - One strategy is selected once and applied to every Website in selection order.
   - Back returns to the batch action chooser and preserves the selected Websites.
   - Enter on a strategy starts execution immediately; Batch Website Backup adds no confirmation screen.

4. **Batch execution contract**
   - Introduce one batch orchestration path above the existing single-Website backup engines.
   - The batch handler accepts the selected Websites and one strategy as one operation rather than invoking the UI callback once per Website.
   - Process Websites sequentially in stable selection-list order. Do not add a worker pool or strategy-specific concurrency.
   - Reuse the existing preflight, Full ZIP, AI1WM, package-resolution, cancellation, and archive-result behavior for each Website.
   - Record a structured outcome for every selected Website: success, failure, cancelled active work, or skipped queued work.
   - A Website failure does not terminate the loop. No automatic retry occurs.

5. **Cancellation contract**
   - `q` and Ctrl+C request cancellation through the existing operation context while Batch Website Backup is running.
   - The active backup receives cancellation and completes its existing cleanup path before navigation is re-enabled.
   - No queued Website starts after cancellation is requested.
   - The active Website is reported as cancelled unless its operation had already reached a terminal success or failure before cancellation took effect.
   - Every queued Website is reported as skipped due to cancellation.
   - Cancellation is the aggregate outcome even if earlier Websites succeeded or failed.

6. **Progress contract**
   - Use one In-Pane Progress Monitor and one persistent log viewport for the whole batch.
   - The progress title identifies the active Website and its one-based `i/N` position.
   - For each active Website, the task stepper uses the four existing labels and transition semantics for the chosen strategy.
   - Moving to the next Website resets the active task stepper for that Website but does not clear prior log lines.
   - Navigation remains locked and the Sidebar remains visible with inactive styling during execution.

7. **Aggregate report contract**
   - Use `Batch Backup Complete` when every selected Website succeeds.
   - Use `Batch Backup Completed with Warnings` when execution finishes with a mixture of successful and failed Websites.
   - Use `Batch Backup Failed` when execution finishes and no Website succeeds.
   - Use `Batch Backup Cancelled` whenever the user cancels, regardless of earlier outcomes.
   - Show the selected format once at report level.
   - Show one status-marked row or block per Website in original batch order.
   - A successful outcome includes Website identity, final Backup Archive path, and formatted and raw size.
   - A failed outcome includes Website identity and the underlying cause.
   - A cancelled or skipped outcome identifies why no Backup Archive was produced.
   - Include retained artifact paths and cleanup outcomes when known. Do not expose credentials, API keys, or unrelated command output.
   - Keep full technical logs in the viewport; do not replace the report with counts only.

8. **Archive storage and naming**
   - Store every final Backup Archive directly in the configured `backup_path`.
   - Keep the established Full ZIP and AI1WM filename patterns. Do not add a batch directory, manifest, batch identifier, or unique suffix.
   - A destination collision is a normal per-Website failure and names the conflicting destination.

9. **No-overwrite invariant**
   - Backup Archive relocation must provide a cross-platform no-replace guarantee. Platform-dependent overwrite behavior is not acceptable.
   - Enforce the invariant at the shared relocation boundary used by both strategies rather than adding a batch-only precheck. Single-Website Backup therefore receives the same data-safety guarantee.
   - A check-then-rename sequence alone is insufficient because another process can create the destination between those operations.
   - If relocation cannot finalize without replacing an existing destination, leave that destination untouched, preserve the recoverable source artifact, and return an explicit error.
   - Clean any newly created partial destination owned by the failed relocation without deleting a pre-existing Backup Archive.

10. **Retained recovery artifacts**
    - Preserve the existing Full ZIP invariant: remove the exported database dump only after archive relocation succeeds; otherwise retain it.
    - Preserve a generated AI1WM source archive when relocation fails or cancellation prevents successful relocation.
    - Return retained artifact metadata from the backup execution boundary when it is known, rather than relying on parsing human-readable errors in the batch layer.
    - Report the retained path and cleanup outcome. The feature does not add automatic artifact purging.
    - AI1WM extension installation remains persistent under the current strategy semantics.

11. **Completion lifecycle**
    - Keep the aggregate report visible until the user dismisses it.
    - On dismissal, refresh the Websites inventory, clear all multi-selection state, and return to the Websites list.
    - Clear the selection after success, warnings, total failure, or cancellation because execution may already have produced one or more Backup Archives.
    - Do not clear the selection when the user backs out before execution starts.

12. **Reuse and dependency boundaries**
    - Reuse the existing Website candidate representation, backup strategy definitions, backup engines, progress-step vocabulary, package resolver, and In-Pane Progress Monitor.
    - Add only the batch result and orchestration state needed to represent per-Website outcomes and the aggregate lifecycle.
    - Do not create a second Full ZIP or AI1WM implementation.
    - No new configurable concurrency, confirmation option, archive layout option, or dependency is justified by this feature.

13. **Domain language**
    - Use **Website**, **Website Backup**, **Batch Website Backup**, and **Backup Archive** consistently in code-facing concepts, reports, specifications, and tests.
    - Keep the established user-facing `Delete` wording even though the domain operation is De-provisioning.
    - Avoid `site`, `bulk backup loop`, and `multi-site backup` in new user-visible copy.

## Testing Decisions

1. **Test externally observable behavior**
   - Tests assert state transitions, callbacks at workflow boundaries, progress/report messages, filesystem outcomes, and user-visible text that forms part of the approved contract.
   - Tests do not pin private field layout, helper call counts, goroutine structure, copied values, or incidental formatting.
   - Existing Full ZIP and AI1WM engine tests remain authoritative for strategy internals; batch tests do not repeat them.

2. **Primary TUI model seam**
   - Drive the Websites Hub with key events and injected operation callbacks.
   - Cover more-than-one selection opening the batch action chooser, initial `Backup` focus, navigation to `Delete`, direct `d` behavior, and unchanged zero/one-selection Enter behavior.
   - Cover strategy labels, one strategy submitted with the complete candidate set, immediate execution without confirmation, Back/Esc navigation, selection preservation before execution, and selection clearing after report dismissal.
   - Cover contextual footer help and status markers without depending on ANSI color.

3. **Application workflow seam**
   - Exercise Batch Website Backup through an injected deterministic backup runner rather than WP-CLI or MySQL.
   - Prove stable sequential order and that no second Website starts before the first reaches a terminal outcome.
   - Cover all success, one failure followed by later success, all failure, cancellation during active work, and queued Websites skipped after cancellation.
   - Assert `i/N` progress, strategy-specific step reuse, persistent cross-Website logs, aggregate outcome precedence, and per-Website report data.
   - Confirm that no automatic retry occurs.

4. **Filesystem safety seam**
   - Use isolated temporary directories and real filesystem operations to verify that a pre-existing destination is byte-for-byte unchanged after a collision.
   - Verify the collision identifies the destination, preserves the source artifact, removes only partial files owned by the attempted relocation, and permits later Websites in the batch to run.
   - Exercise the cross-volume fallback where the test environment can provide distinct volumes; otherwise record that environmental limit and keep the shared relocation contract covered at its deterministic boundary.

5. **Retained artifact outcomes**
   - Extend strategy tests only where needed to prove structured retained-artifact metadata on archive creation, relocation, cleanup failure, and cancellation.
   - Assert that reports include known retained paths and never include credentials.
   - Do not add tests that merely duplicate error strings already covered by behavior assertions.

6. **Actual terminal verification**
   - Use isolated Website fixtures, never user Websites.
   - Exercise selection, action focus, strategy Back/Esc, running progress, one partial failure, cancellation, final report dismissal, and cleared selection on the actual TUI surface.
   - Verify the 80x24 layout and readability without ANSI color.
   - Record any real WP-CLI, database, cross-volume, or AI1WM surface that cannot be exercised in the verification environment.

7. **Prior art**
   - Follow the existing Websites Hub navigation and backup-strategy model tests for key-driven interaction.
   - Follow the existing application backup execution tests for injected orchestration and progress completion.
   - Follow the existing Full ZIP relocation and cancellation tests for temporary-directory filesystem assertions.
   - Follow the existing batch De-provisioning tests for aggregate continuation and partial-outcome behavior, while retaining sequential execution for Batch Website Backup.

## Out of Scope

- Choosing a different backup strategy for each Website in one batch.
- Concurrent backup execution, worker pools, or configurable concurrency.
- Automatic retries, retry prompts, resume support, or rerunning only failed Websites from the report.
- Additional backup strategies or changes to Full ZIP and AI1WM archive contents.
- A new top-level Backup destination, a second Website picker, or changes to the legacy standalone Backup flow beyond shared no-overwrite safety.
- Changes to batch Delete confirmation, execution, concurrency, or reporting.
- An extra Backup confirmation screen.
- Batch directories, manifests, combined archives, batch identifiers, or unique collision suffixes.
- Changing `backup_path`, backup exclusion configuration, or archive filename timestamp precision.
- Automatically deleting retained SQL or `.wpress` recovery artifacts after failure or cancellation.
- Automatically uninstalling the AI1WM extension.
- Repeating URL, directory, and format fields for every successful Website in the concise aggregate report.
- Persisting a batch queue across application restarts.
- Light-terminal theming or support below the established 80x24 minimum geometry.

## Further Notes

- `Batch Website Backup` is the canonical glossary term for this operation: multiple selected Websites backed up with one shared backup approach.
- No ADR is required. The workflow composes established Website Backup and Websites Hub concepts, and its sequential orchestration and UI choices are straightforward to reverse.
- The existing relocation behavior is not a reliable immutability boundary because destination replacement differs by host filesystem. The no-overwrite invariant is part of this feature's data-safety contract, not an optional hardening task.
- A cancelled batch may contain successful, failed, cancelled, and skipped Website outcomes. The aggregate heading remains `Batch Backup Cancelled`, while every completed result remains visible.
- The specification deliberately reuses current strategy semantics. In particular, Full ZIP may retain an exported SQL dump after failure, and AI1WM may retain a generated source archive and its installed extension; the new requirement is to surface known retained artifacts clearly.
