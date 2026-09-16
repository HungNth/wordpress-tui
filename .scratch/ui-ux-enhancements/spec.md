# UI/UX Enhancements: Unified Slug Input, High-Visibility Focus, and Inline Search

Status: ready-for-agent

## Problem Statement

Users of WPTUI face three friction points in the interactive experience:
1. Entering the Website Name and Website Slug is split across two separate interactive steps, creating unnecessary screen transitions.
2. The active field indicator (the vertical line on the left of the focused field) uses a muted gray tone that blends into the background, making it hard to see which input currently has focus.
3. After selecting default packages, a separate confirmation prompt (`Search catalog? [Yes/No]`) interrupts the flow before allowing users to search the package catalog.

## Solution

1. **Unified Website Name & Slug Form View**:
   - Place `Website Name` and `Website Slug` into a single form view.
   - Allow leaving `Website Slug` empty to automatically derive it from `Website Name`.
   - If a slug is entered or derived, validate its syntax and availability (directory & database collision checks) immediately.
2. **Prominent Cyan Focus Indicator**:
   - Apply a custom WPTUI theme where `Focused.Base` border foreground is styled with high-visibility bright cyan (`#00FFFF`), making the active field immediately recognizable in any terminal theme.
3. **Inline Catalog Search Option**:
   - Append a terminal option `[🔍 Type to search catalog...]` directly into the package MultiSelect choices.
   - If selected, the search prompt opens immediately. If unselected, package selection completes without any extra confirmation dialogs.

## User Stories

1. As a developer, I want to see Website Name and Website Slug in a single form view, so that I don't navigate through multiple screens for basic site identity.
2. As a developer, I want Website Slug to automatically derive from Website Name if left blank, so that I can create standard sites with minimal keystrokes.
3. As a developer, I want to explicitly edit the Website Slug in the same view when I need a custom slug, so that identity configuration is seamless.
4. As a user, I want the active input field to have a bright cyan vertical border indicator, so that I instantly know which field has focus.
5. As a developer selecting packages, I want a `[🔍 Type to search catalog...]` option at the bottom of the default list, so that I can search the catalog without an extra confirmation screen.
6. As a developer selecting packages, I want package selection to conclude immediately when I don't select the search option, so that default-only installations are fast.

## Implementation Decisions

1. **Unified Identity Inputs**:
   - In `internal/tui/create_wizard.go`, combine Website Name and Website Slug into the initial group.
   - The slug validator accepts an empty value as valid during form interaction; when submitted, if the slug is empty, it derives `Slugify(WebsiteName)` and validates availability.
2. **WPTUI Theme Styling**:
   - In `internal/tui/theme.go`, define `AppTheme() *huh.Styles` configuring:
     `theme := huh.ThemeBase(true)`
     `theme.Focused.Base = theme.Focused.Base.BorderForeground(lipgloss.Color("#00FFFF"))`
   - Apply `form.WithTheme(AppTheme())` across all WPTUI interactive forms.
3. **Inline Search in Package Picker**:
   - In `internal/tui/packages_picker.go`, if `catalog` is available, append `huh.NewOption("🔍 Type to search catalog...", "__search__")` to options.
   - When submitted, filter out `__search__` from the selected package list; if `__search__` was selected, immediately prompt for the search query and present matching items.
   - Remove the redundant `Search %s catalog?` confirmation form.

## Testing Decisions

1. **`internal/tui` Seam**:
   - Test `BuildCreateForm` with unified inputs, empty slug auto-derivation, and custom slug overrides.
   - Test `AppTheme` verifies that the focused border foreground is configured to cyan.
   - Test `SelectPackagesFlow` with and without `__search__` selection.

## Out of Scope

- Changing backend provisioning or package installation contracts.
- Altering core preflights or rollback mechanisms.
