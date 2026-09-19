# 03: Unified Package Selection and Catalog Search

**Parent specification:** [Unified Terminal UI](../spec.md)

**Design contract:** [WPTUI Terminal Design](../../../DESIGN.md)

**What to build:** A consistent Package selection journey from Create or Config into live Package Catalog search and back to the calling workflow. Users can search repeatedly, distinguish focus from selection, go back without losing choices, confirm exactly the selected Packages for the existing installation path, or cancel the workflow without installing pending choices.

**Blocked by:** 01: Unified Entry, Setup, and Application Settings — supplies shared form presentation and navigation semantics. This selection-only slice does not require the new running-operation renderer.

**Status:** resolved

## Acceptance criteria

- [x] Plugin and theme selection, their opt-in prompts, and live Package Catalog search use the shared contextual layout, English wording, dark palette, and relevant keyboard help in both Create and Config.
- [x] Search Catalog is the first action in the appropriate Package list, with a plain-text label instead of a decorative emoji. It is an action, never an installable Package. Back is not inserted as selectable Package data.
- [x] Focus uses the cyan cursor and checked items remain visibly green. A checked item under the cursor retains both selection and focus cues. Unchecked items remain distinguishable through their marker even without color.
- [x] Arrow keys navigate results, Space toggles selection, and Enter confirms as stated in the active help. Empty and no-match result states remain usable; confirmation cannot invent a Package from an empty result list.
- [x] Changing or clearing the query preserves the Package Selection Accumulator. Repeated selection, deselection, and searches produce the intended final choices without duplicates or lost prior selections.
- [x] Esc exits an active search/filter interaction before leaving its parent screen. Back preserves pending choices and other entered values within the current workflow; closing search does not silently discard checked Packages.
- [x] Ctrl+C is distinguishable from Back: it cancels the calling workflow, abandons pending choices, and returns neutrally to the main menu. Neither pending Packages nor a cancelled selection trigger installation.
- [x] Confirmed choices reach the existing Create and Config installation paths correctly. The caller can continue and act on the selected plugin/theme set, including selections accumulated across multiple queries; this ticket is not satisfied by a standalone picker demo.
- [x] Existing optional Package API behavior, default Package lists, validation, theme activation choices, and Package business rules remain intact. Selection-state retention introduces no persistent UI storage or new Package service API.
- [x] Long Package names and result lists fit or scroll within ordinary and narrow terminal dimensions; current input, selection state, and keyboard help remain usable. Empty results do not create invalid cursor behavior.
- [x] Shared selection interfaces and both calling flows are migrated together for any changed Back/Cancel contract, with no legacy behavior hidden behind compatibility aliases. Existing flows remain runnable before their remaining UI migration tickets.
- [x] Actual terminal evidence covers plugin and theme selection from both callers, multiple queries, no results, deselection, Esc/Back, cancellation, and confirmation into an isolated installation scenario.

## Testing seam

Exercise selection through the application workflow boundary using real UI interaction with fixture Packages and isolated operation effects. Assert the resulting selection behavior and observable installation or non-installation outcome, rather than merely echoing forwarded arguments.

Reuse the existing interactive multi-query Live Search test pattern for the genuinely uncertain state transitions: selections across query changes, focused checked items, empty results, Back retention, and workflow cancellation. Update the old Escape-aborts expectation to the new contextual Back contract without preserving obsolete incidental wording or style snapshots. Do not add tests for each style property.

## Demo path

1. Enter Create's plugin selection with fixture Packages. Open Search Catalog, select results from two queries, clear the query, and confirm that all intended selections remain checked.
2. Go back to the parent picker, reopen search, deselect one item, and confirm into the existing isolated installation flow. Verify the final Package set rather than a picker-only result.
3. Repeat from Config for themes, including a no-match query. Cancel instead of confirming and verify return to the main menu with no installation of pending choices.

## Scope boundary

This ticket owns Package selection/navigation and the necessary caller integration. It preserves current Package service and installation behavior. General Create/Config field navigation, running progress, reports, and final menu destinations are delivered by Tickets 04 and 05.
