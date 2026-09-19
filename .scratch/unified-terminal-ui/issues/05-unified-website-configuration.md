# 05: Unified Website Configuration

**Parent specification:** [Unified Terminal UI](../spec.md)

**Design contract:** [WPTUI Terminal Design](../../../DESIGN.md)

**What to build:** A consistent Website Configuration workflow for tweaks, administrative credentials, and plugin/theme installation. Users select a Website, navigate its actions and inputs predictably, receive coherent progress and results, and return directly to the current Website's action menu without the redundant continuation question.

**Blocked by:** 02: Website Backup with Coordinated Progress — supplies shared operation feedback and cancellation; 03: Unified Package Selection and Catalog Search — supplies integrated Package selection and Back/Cancel behavior.

**Status:** resolved

## Acceptance criteria

- [x] Website selection, the Website action menu, administrator selection and inputs, tweak actions, Package choices, theme activation, and all notifications use the shared contextual layout, English wording, dark palette, and accurate keyboard help.
- [x] Plain action labels replace decorative or nonfunctional numeric prefixes. Back is last in navigation menus and never selectable Package data. Action-menu Back returns to Website selection; Back from initial Website selection returns to the main menu.
- [x] Back within pre-execution configuration forms preserves pending inputs and Package choices. Validation and discovery failures retain useful context without presenting a user cancellation as an error.
- [x] Ctrl+C before execution cancels neutrally to the main menu without applying pending changes. Search/filter interactions and confirmed selections follow Ticket 03's contract in both plugin and theme paths.
- [x] Existing tweak behavior, Administrative Credential Reconciler semantics, administrator selection/defaults, Version-Aware Package Upgrader decisions, Package API availability handling, and theme activation choices remain intact.
- [x] Tweaks, credential changes, Package resolution, and installation use coordinated single-owner progress and semantic status output rather than a mixture of raw prints and independent styles. Only completed work receives a success marker.
- [x] Ctrl+C during an action requests cancellation, keeps stopping feedback visible, and waits for active work and applicable cleanup before returning to the main menu. Previously completed configuration changes are reported truthfully; no new transactional rollback guarantee is implied.
- [x] Successful actions preserve their result and return directly to the action menu for the same Website. Remove the redundant Continue configuring question and do not replace it with a generic continuation gate.
- [x] Reports and notifications distinguish clean completion, warnings, failure, and cancellation. Preserve applicable per-tweak and per-Package outcomes, including skipped/current versions, installations, upgrades, and partial failures.
- [x] Credential-update success uses concise semantic feedback without exposing the password. Errors retain the failed step, useful redacted cause, and applicable cleanup outcome; no automatic retry is introduced.
- [x] Neutral Website and Package metadata is distinct from action statuses. Long paths, descriptions, and Package lists remain usable at narrow widths and without color; completed results stay in scrollback.
- [x] After cancellation or a controlled failure, the application remains usable for another action. Existing Website resources are preserved according to the operation's current contract rather than deleted to force an all-or-nothing UI result.
- [x] Actual terminal evidence covers each action category, same-Website menu return, Back/retention, Package search, controlled warning/failure, and cancellation followed by another usable workflow.

## Testing seam

Use the existing composed Website Configuration/application flow boundary with temporary Website data, deterministic Package fixtures, and a fixture Administrative Credential Reconciler/database boundary. Existing composed configuration tests provide prior art for multiple actions against one Website and isolated external effects.

Assert which Website and menu remain active, whether pending changes were applied, preserved selections, meaningful per-item outcomes, and cancellation timing. Exercise real menus/forms in terminal smoke scenarios. Do not reduce the test to command forwarding, copied fields, or new snapshots of old incidental wording.

## Demo path

1. Select a fixture Website in Config and apply tweaks. Observe progress and the result, then confirm immediate return to that Website's action menu.
2. Change administrative credentials using safe fixture effects, then install plugins and themes chosen across multiple search queries. Verify masked input and complete per-action results.
3. Navigate back through inputs, cancel a pending action, and separately cancel running fixture work. Verify retained data for Back, neutral cancellation, accurate partial outcomes, and a usable subsequent workflow.

## Scope boundary

This ticket covers Website Configuration rather than Application Settings. It changes the interaction and output contract without introducing new credential policies, package-upgrade behavior, or rollback semantics. It can proceed independently of the Create migration once its two blockers are complete.
