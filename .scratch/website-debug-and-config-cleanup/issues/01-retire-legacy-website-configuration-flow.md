# 01: Retire the legacy Website Configuration flow

**What to build:** Complete the Website Configuration cutover to the fullscreen Website Hub and Bubble Tea execution model. Remove the obsolete Huh-era Website Configuration orchestration, injection surface, prompts, summaries, and tests while preserving the current Website Hub actions and shared Website Configuration domain operations. Move still-valid Apply Tweaks coverage to current seams and align the active Website Configuration documentation with the product users actually run.

**Blocked by:** None (can start immediately).

**Status:** resolved

- [x] Normal WPTUI startup continues to expose Apply Tweaks, administrator credential changes, and Package installation through Website Hub/Bubble Tea with unchanged current behavior.
- [x] The obsolete Huh-era Website Configuration orchestration and its dedicated dependency-injection surface are removed completely rather than retained as aliases or compatibility shims.
- [x] The legacy menu no longer exposes or dispatches its obsolete Website Configuration branch; legacy seams for unrelated workflows remain outside this ticket.
- [x] Huh-era Website Configuration selectors, prompts, progress printers, summaries, and tests are deleted when they have no remaining caller.
- [x] Shared Website Configuration domain operations for tweaks, administrator credentials, and Package installation remain available to the Bubble Tea executors.
- [x] Existing behavior that was unique to the obsolete flow, including multi-administrator selection, is not ported; the current Bubble Tea behavior is authoritative.
- [x] The regression contract for applying the generated default WordPress tweaks is preserved at a current Website Configuration domain or Bubble Tea seam before its legacy test is removed.
- [x] Active Website Configuration documentation describes the Website Hub actions, In-Pane Progress Monitor, cancellation, and result reporting; historical ADRs remain unchanged.
- [x] The deleted infinite-loop tail no longer causes an unreachable-code diagnostic, and repository-wide vet, build, and test checks pass without adding suppression.
