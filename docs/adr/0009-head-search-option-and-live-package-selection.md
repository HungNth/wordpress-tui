# 0009: Head Search Option and Interactive Live Package Catalog Selection

Current presentation follows [DESIGN.md](../../DESIGN.md), including plain-text action labels in place of the decorative search emoji recorded below. Under [ADR-0017](0017-selected-local-development-environment-integration.md), static default package lists in `config.json` are removed; selectable packages originate exclusively from the configured Package Service API, and package UI is hidden entirely when no catalog is available. When a catalog is available, the search-first placement and selection-accumulation behavior remain unchanged across separate plugin and theme selections.

## Context

When selecting Packages (plugins or themes) during Website Provisioning, the search option was previously placed at the bottom of the default choices list. In addition, catalog search required a multi-step form workflow (input search term -> submit -> render static result multiselect -> confirm -> ask to search again). This created significant interaction friction for developers who want to quickly find and install multiple non-default packages without losing previously chosen selections or repeatedly restarting the search flow.

## Decision

1. **Top-Level Search Trigger**:
   - In package selection lists, the search trigger is positioned at the very top of the live catalog picker.
   - Developers immediately see the search capability without needing to scroll past default items.

2. **Interactive Live In-Memory Search with Selection Accumulation**:
   - The interactive search interface leverages the already-loaded in-memory Package Catalog, performing instant local string filtering over package names and slugs.
   - Developers can type queries in real-time; the matching results update live below the input.
   - Selecting a package with `Space` marks it as selected.
   - Modifying or clearing the query to search for another package **preserves all previously selected packages across queries** (accumulated selections).
   - An active status counter/list displays all currently accumulated packages.
   - Pressing `Enter` confirms and returns the aggregated list of selected packages.
