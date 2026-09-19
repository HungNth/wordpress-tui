# 04: Unified Website Provisioning

**Parent specification:** [Unified Terminal UI](../spec.md)

**Design contract:** [WPTUI Terminal Design](../../../DESIGN.md)

**What to build:** A complete Create experience from Website Name and Website Slug through administrator inputs, Package choices, Provisioning, cleanup, and the final report. Users can revise earlier inputs without losing their work, observe truthful progress, cancel safely, and return to the main menu with usable results preserved in scrollback.

**Blocked by:** 02: Website Backup with Coordinated Progress — supplies the shared operation-display and cancellation lifecycle; 03: Unified Package Selection and Catalog Search — supplies the reusable selection and Back/Cancel contract.

**Status:** resolved

## Acceptance criteria

- [x] All Create screens, opt-in prompts, validation, and notifications use the shared layout, task-specific English wording, dark palette, and correct keyboard help.
- [x] Website Name and Website Slug behavior, ASCII normalization, explicit slug overrides, early collision checks, administrator defaults, optional tweaks, and existing Package API behavior remain intact.
- [x] Back through pre-execution screens preserves entered values and selected Packages within the current workflow. A collision or validation error retains useful input and explains the correction rather than erasing unrelated choices.
- [x] Back from the first Create screen returns to the main menu. Ctrl+C before execution cancels neutrally and abandons pending input without creating a Website. The integrated Package search follows Ticket 03's contract.
- [x] Package resolution and Provisioning use the same single-owner progress lifecycle as Backup: a running-step spinner, persistent completed-step outcomes, truthful known-total counts, and no independent raw progress printing or premature success.
- [x] Ctrl+C during execution propagates cancellation to active work, displays stopping feedback, and waits for existing Provisioning cleanup obligations to finish before returning to the main menu. Another operation remains usable afterwards.
- [x] Critical-failure and cancellation reporting describes the failed or interrupted step and actual cleanup outcome. Existing ownership safeguards, unrelated Websites, source material, and reusable caches are preserved according to the current Provisioning contract.
- [x] Best-effort tweak failures, default-theme skips, stale Package fallbacks, and other nonfatal outcomes remain visible. A completed Website with warnings is not reported as unconditional success or treated as a reason to remove that Website.
- [x] The completion report uses an accurate overall outcome and neutral Label: Value metadata, including the complete Website path and URL and applicable active-theme information. It retains meaningful per-Package and per-tweak outcomes.
- [x] Passwords remain masked in forms; credentials and API keys do not appear in progress, errors, reports, or scrollback. Useful non-secret error details remain available, with no automatic retries.
- [x] Successful Create preserves its report and returns directly to the main menu without a continuation prompt. Presentation remains usable at ordinary and narrow widths and without color.
- [x] Shared Website input forms continue to work for Restore without showing Create-only choices there. Any shared interface changes migrate affected callers together; remaining operation business behavior stays unchanged.
- [x] Actual terminal evidence covers pre-execution Back/retention, a normal creation, completed-with-warnings output, a controlled critical failure, and cancellation with cleanup followed by a new usable workflow.

## Testing seam

Use the existing composed Create/application boundary with temporary configuration, fixture core/Package inputs, and isolated external operation effects. Prior art is the composed Create smoke scenario that verifies created resources, failure cleanup, unaffected prior Websites, cache preservation, and secret handling.

Keep regression checks focused on navigation state, truthful outcome classification, cancellation/cleanup ordering, and resource preservation. Exercise real Create forms and progress in the terminal smoke; scripted prompt callbacks alone do not prove keyboard or visual correctness. Reuse existing business tests rather than duplicate them to satisfy a UI checklist.

## Demo path

1. Start Create with an isolated fixture configuration, fill Website and administrator inputs, select Packages, and navigate backwards. Confirm retained values, then complete a successful creation and observe return to the main menu.
2. Run a fixture scenario with a nonfatal tweak or Package warning and inspect the complete report and copyable resource locations.
3. Trigger an isolated critical failure and a separate cancellation during execution. Verify the cleanup report, preservation of unrelated resources, and ability to start another operation.

## Scope boundary

This ticket changes Create's presentation and interaction end to end while preserving Provisioning business rules. It does not add provisioning features, new rollback guarantees, persistent UI state, or configuration schema changes.
