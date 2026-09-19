# 07: Unified De-provisioning and Partial Results

**Parent specification:** [Unified Terminal UI](../spec.md)

**Design contract:** [WPTUI Terminal Design](../../../DESIGN.md)

**What to build:** A complete Delete experience for one or more Websites with clear selection, explicit destructive confirmation, coordinated execution feedback, and accurate per-resource results. Cancellation must wait for in-flight deletion work to stop and must distinguish resources already removed from resources retained or failed, without implying that deletion was undone.

**Blocked by:** 02: Website Backup with Coordinated Progress — supplies shared operation-display ownership and cancellation lifecycle. This ticket does not depend on the Create, Config, or Restore UI migrations.

**Status:** resolved

## Acceptance criteria

- [x] Website discovery/selection, empty-list feedback, destructive confirmation, progress, and results follow the shared contextual English layout, dark palette, markers, and relevant keyboard help.
- [x] The multi-select list contains Website data rather than a selectable Back entry. Focus remains distinct from selection; Back from confirmation restores the selected set and Back from initial selection returns to the main menu.
- [x] The destructive confirmation remains explicit, defaults to No, and identifies the Website directories and configured Website Databases affected. Unknown database identity remains clearly identified; the UI does not assume that a Website Slug is the database name.
- [x] Declining confirmation or cancelling before execution performs no deletion and returns neutrally rather than reporting an operation error. Existing exclusion, identity, and resource-safety rules remain intact.
- [x] Single-Website and batch execution provide coordinated progress through one terminal display owner rather than concurrent worker writes or silent execution. A running indication remains visible, and completed Website/resource outcomes become persistent results without claiming success before completion.
- [x] Known Website or resource totals may produce truthful progress counts; unknown work does not produce invented percentages or estimates. Existing concurrent operation behavior is not rewritten solely to simplify rendering.
- [x] Ctrl+C during execution requests cancellation, retains stopping feedback, and waits for active workers and applicable cleanup to settle before returning to the main menu. The application remains usable for another workflow afterwards.
- [x] Reports distinguish completed, completed with warnings, failed, and cancelled batch outcomes. They retain applicable per-Website and per-resource details for Herd TLS, Website Database, and directory operations, plus meaningful existing aggregate information.
- [x] Removed resources are reported as removed even when another resource fails or cancellation interrupts the batch. Retained, unknown, skipped, and failed resources remain distinguishable; reports never imply that completed deletion was rolled back.
- [x] Resource metadata uses Label: Value fields without decorative success markers. Status markers describe actual outcomes. The final report uses concise headings and consistent indentation rather than a separate wide banner style.
- [x] Errors identify the failed resource or step, useful redacted cause, and applicable cleanup status. Passwords and API keys remain absent from output. No automatic retry, restoration of removed resources, or new destructive action is introduced.
- [x] After successful deletion, preserve the report and return directly to the main menu without an extra continuation prompt. Paths remain complete and copyable; selection and reporting remain usable at ordinary and narrow widths and without color.
- [x] Shared-interface changes migrate affected callers cleanly. Existing domain safety tests remain effective, and actual terminal evidence covers selection, default-No confirmation, a successful isolated batch, mixed outcomes, cancellation, and a usable subsequent workflow.

## Testing seam

Use the existing composed application Delete flow and its multi-Website fixture patterns, with temporary directories and controlled database/TLS effects. Observe actual retained or removed fixture resources and reported outcomes, not simply worker calls or command arguments.

For cancellation, hold in-flight fixture work at a deterministic boundary and verify that the main menu cannot resume early. Include a batch in which one resource has already been removed before a later operation fails or cancellation occurs; the report must not lose or misclassify that partial result. Keep concurrency tests deterministic and isolate all destructive effects from user Websites. Use real terminal interaction to verify selection, confirmation, progress, and report presentation.

## Demo path

1. Select several isolated fixture Websites, enter confirmation, verify default No and the exact resource identities, then go back and confirm that the selection remains intact.
2. Run a successful fixture deletion and a separate mixed-outcome batch. Inspect per-resource results, full paths, aggregate information, and return to the main menu.
3. Cancel a running fixture batch after one resource completes. Observe stopping, the truthful partial report, preserved unrelated resources, and the ability to start another workflow.

## Scope boundary

This ticket unifies De-provisioning's presentation and interaction; it does not make irreversible deletion transactional. It preserves existing resource identity, deletion policy, concurrency, and safety behavior. Completion is independently demoable after its stated blockers, not gated by unrelated workflow migrations.
