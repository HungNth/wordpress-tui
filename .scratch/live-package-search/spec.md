# Top-Positioned Search and Live Interactive Package Catalog Selection Specification

Status: ready-for-agent

## Problem Statement

When local WordPress developers provision a new website, they frequently want to install specific plugins or themes that are not part of the default configuration list. Currently, the search option is placed at the very bottom of the defaults list, forcing users to scroll through all default options before discovering the search capability. Furthermore, the existing search mechanism requires a multi-step form cycle: enter search keyword -> submit form -> render static results -> pick results -> confirm -> trigger search again. This repetitive workflow breaks the developer's flow, does not retain previously selected packages across successive queries, and creates high interaction friction.

## Solution

WPTUI introduces an optimized, top-positioned live package search and selection experience:
1. Position the search option (`🔍 Type to search catalog...`) as the very first item (index 0) in the package multi-select options list when catalog items are available.
2. Provide an interactive live search picker operating directly against the already-loaded in-memory Package Catalog, ensuring sub-millisecond filtering without network latency or rate-limiting.
3. Maintain a Package Selection Accumulator that persistently preserves all user-selected packages across multiple search queries: a developer can search for "acf", select Advanced Custom Fields PRO, clear or modify the search term to "smtp", select WP Mail SMTP Pro, and both selections remain safely accumulated.
4. Display an active status indicator showing currently accumulated packages during the interactive search session.
5. Confirm and return the deduplicated aggregated list of packages when the user confirms with `Enter`.

## User Stories

1. As a local WordPress developer, I want the search option to be the first item in the package list, so that I can immediately search without scrolling through defaults.
2. As a local WordPress developer, I want to select default packages and also invoke search in the same initial form, so that I can combine default and custom packages effortlessly.
3. As a local WordPress developer, I want search filtering to be instant as I type, so that I get immediate visual feedback on available catalog items.
4. As a local WordPress developer, I want to select a package using the Space key and immediately keep typing another search term, so that I can find multiple packages in one seamless session.
5. As a local WordPress developer, I want previously selected packages to stay selected when I change my search query, so that I do not lose my choices when searching for a different tool.
6. As a local WordPress developer, I want to see a clear indicator of how many and which packages are currently selected, so that I have continuous visibility of my choices.
7. As a local WordPress developer, I want to deselect an accumulated package if I change my mind, so that I can correct accidental selections before provisioning.
8. As a local WordPress developer, I want search to filter across both package human-readable names and machine slugs, so that I can find packages regardless of which identifier I recall.
9. As a local WordPress developer, I want pressing Enter to finalize my selection and continue website provisioning, so that the interaction feels natural and efficient.
10. As a local WordPress developer, I want pressing Escape to cancel the search session cleanly while retaining previously confirmed defaults, so that I can back out without errors.
11. As a local WordPress developer, I want the returned package list to be automatically deduplicated, so that accidentally selecting the same package from both defaults and search does not produce duplicate installation attempts.

## Implementation Decisions

1. **Top-Level Search Positioning**:
   - In `BuildPackageOptions`, when catalog items exist (`hasCatalog = true`), prepend `huh.NewOption("🔍 Type to search catalog...", SearchOptionKey)` at index 0 before appending default package options.
   - When `hasCatalog = false`, no search option is added and default options remain in their configured order.

2. **Package Selection Accumulator and Interactive Flow**:
   - In `SelectPackagesFlow`, when the user selects `SearchOptionKey` (either standalone or along with default packages), transition immediately into the interactive search mode.
   - Initialize the accumulator with any default packages selected in the initial form.
   - The interactive search interface maintains:
     - The current search input query string.
     - The filtered catalog results based on `packages.FilterCatalog(catalog, itemType, query)`.
     - The persistent set of selected package slugs.
   - When the user presses `Space` on an item in the search results, toggle its membership in the selection set.
   - Changing the query string re-evaluates `packages.FilterCatalog` without clearing the persistent selection set.
   - When the user presses `Enter` on the search view, the accumulator returns all accumulated selections.

3. **In-Memory Catalog Performance**:
   - The interactive search operates strictly on the in-memory catalog slice passed to `SelectPackagesFlow`. No HTTP calls or network I/O occur during interactive typing.

4. **Visual Feedback**:
   - Display an active selection summary line showing the count and names/slugs of all accumulated packages.

## Testing Decisions

Testing is focused at the **`internal/tui` module seam**:
- **Option Ordering Test**:
  - Test `BuildPackageOptions` with `hasCatalog = true`: verify that option 0 has value `SearchOptionKey` and that all default options follow in their original order.
  - Test `BuildPackageOptions` with `hasCatalog = false`: verify that option count equals defaults count and search option is absent.
- **Selection Extraction Test**:
  - Test `ExtractSelectedPackages`: verify clean separation of `SearchOptionKey` from user-selected default package slugs.
- **Selection Accumulation Test**:
  - Test interactive search flow with scripted queries and selections: verify that selecting package A under query 1, changing query to query 2, and selecting package B yields both package A and package B.
  - Verify deduplication logic ensures that picking a slug present in both defaults and search appears exactly once.

## Out of Scope

- Remote search via HTTP API during live typing (in-memory catalog filtering is preferred for sub-millisecond response).
- Changing the package installation pipeline or backend provisioning steps.
- Modifying theme or plugin disk caching mechanisms.

## Further Notes

- ADR `0009-head-search-option-and-live-package-selection.md` records the architectural decision for this specification.
- Domain term `Package Selection Accumulator` is defined in `CONTEXT.md`.
