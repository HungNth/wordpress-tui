# 03: Native Create Wizard and Package Selection

**Parent specification:** [Master-Detail Bubble Tea TUI](../spec.md)

**Design contract:** [WPTUI Terminal Design](../../../DESIGN.md)

**What to build:** The Create Wizard rendered inside the Content Pane when `Create` is selected in the Sidebar. Built with pure Bubble Tea form components (`bubbles/textinput`, toggles, buttons) across two steps: Step 1 collects Website Name (auto-deriving Slug), Admin Username, Password, Email, and Apply Tweaks checkbox; Step 2 integrates the existing `LiveSearchModel` for plugin and theme package selection. Include dirty form tracking so pressing `Esc` on a modified form prompts `Discard changes? (y/n)`.

**Blocked by:** 01 (needs master-detail shell and focus engine).

**Status:** resolved

## Acceptance criteria

- [x] Selecting `Create` in the Sidebar renders the native Bubble Tea Create Wizard in the Content Pane.
- [x] Step 1 form fields:
  - Website Name (text input)
  - Website Slug (text input, auto-derived from Website Name if left empty)
  - Admin Username (defaults to `cfg.DefaultAdminUsername`)
  - Admin Password (masked text input, defaults to `cfg.DefaultAdminPassword`)
  - Admin Email (text input, defaults to `cfg.DefaultAdminEmail`)
  - Apply WordPress Tweaks (checkbox, toggleable with `Space`)
  - `[ Next: Select Packages ]` button
- [x] Field navigation supports `Tab` / `Shift+Tab` and `Down` / `Up`.
- [x] Slug availability checker verifies non-collision with existing directory and database prior to advancing.
- [x] Advancing past Step 1 transitions the Content Pane to Step 2: Package Selection, reusing `LiveSearchModel` for plugins and themes with query filtering and preserved selections.
- [x] Dirty form protection: Pressing `Esc` when any input has been edited displays a confirmation dialog: `Discard changes? (y/n)`. Selecting `y` discards input and returns focus to Sidebar; `n` resumes editing. If no inputs were modified, `Esc` returns directly to Sidebar.
- [x] Pressing `Esc` in Step 2 returns to Step 1 preserving entered form values.
- [x] Submitting Step 2 accumulates validated inputs and selected packages, preparing execution payload.

## Testing seam

The `tea.Model` for Create Wizard. Test by sending key presses to fill inputs, verify auto-slug derivation, validate collision rejection, check package search filtering, and assert that dirty form triggers the discard confirmation on `Esc`.

## Demo path

1. Select `Create` in Sidebar and press `Enter`.
2. Type website name `Demo Site`. Observe auto-derived slug `demo-site`.
3. Press `Tab` to navigate through admin fields. Toggle tweaks with `Space`.
4. Press `Enter` on Next. Observe transition to Package Selection.
5. Type `acf`, press `Space` to select. Press `Esc` to return to Step 1, verify entered values remain intact.
6. Press `Esc` on Step 1. Verify `Discard changes? (y/n)` appears. Press `y` and verify return to Sidebar.

## Scope boundary

Delivers the native Create form wizard and package selection integration. Progress rendering during the actual provisioning execution connects to Ticket 04.
