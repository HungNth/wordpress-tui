# 02: Selection-Accumulating Multi-Query Search Flow

Status: ready-for-agent
Blocked by: 01-top-positioned-search-option.md
Parent spec: ../spec.md

## What to build

Implement the interactive multi-query catalog search experience that persistently preserves selected packages across successive search queries (Package Selection Accumulator). When a user searches for a term, selects a package, and then modifies or clears the search query to find another package, all previously selected packages remain preserved. The view displays a live counter of currently accumulated packages and confirms the aggregated, deduplicated selection upon pressing Enter.

## Acceptance criteria

- [ ] Selecting a package under one search query and subsequently searching for another term keeps the previously selected package preserved in the accumulated set.
- [ ] Changing the search query filters catalog items instantly in memory without resetting accumulated selections.
- [ ] An active indicator displays the count and names/slugs of all currently accumulated packages.
- [ ] Pressing Enter confirms the accumulated package selection and proceeds with the provisioning flow.
- [ ] Returned package list deduplicates any package selected both in defaults and via search.
- [ ] Unit and flow tests verify selection accumulation across multiple search queries.

## Testing seam

- `internal/tui/packages_picker_test.go`: test interactive search accumulation across multiple simulated queries and selections.

## Demo path

Run `wptui`, choose `Create`, select search, enter "acf", pick Advanced Custom Fields PRO, change search query to "rank", pick Rank Math SEO PRO, press Enter, and observe that both packages are installed.
