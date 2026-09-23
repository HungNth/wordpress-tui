# 02: Websites Hub and Contextual Action Menu

**Parent specification:** [Master-Detail Bubble Tea TUI](../spec.md)

**Design contract:** [WPTUI Terminal Design](../../../DESIGN.md)

**What to build:** The Websites Hub view rendered inside the Content Pane when `Websites` is selected. Discovers local websites instantaneously via filesystem scan without synchronous database queries. Displays a searchable and selectable list showing Website Slug, `.test` URL, and directory path. Supports multi-selection with `Space` for batch deletion (`d` key) with confirmation. Pressing `Enter` on a site opens a contextual action menu: `Config`, `Backup`, `Open in Browser`, `Open in Editor`, `Delete`, and `Back`.

**Blocked by:** 01 (needs master-detail shell and focus engine).

**Status:** resolved

## Acceptance criteria

- [x] Websites Hub discovers local WordPress websites via filesystem directory scan (`WebsitesPath`, respecting `DeleteExcludes`) without synchronous database connections or WP-CLI queries.
- [x] Content Pane renders the website list with cursor highlighting (`>`), displaying Website Slug, URL (`http://<slug>.test`), and directory path.
- [x] An empty state (`No websites found in <WebsitesPath>`) is rendered when no local sites exist.
- [x] Users can navigate the list with `Up` / `Down` or `k` / `j`.
- [x] Pressing `Space` toggles multi-select checkboxes (`[x]` / `[ ]`) on website items.
- [x] Pressing `d` when one or more websites are selected opens a batch deletion confirmation modal listing all target directories and databases, defaulting to `No`.
- [x] Pressing `Enter` on a single website opens its Action Menu with 6 actions:
  - `Config`: Triggers configuration flow for the site.
  - `Backup`: Triggers backup flow for the site.
  - `Open in Browser`: Launches default browser with `http://<slug>.test` via `launcher.OpenURL`.
  - `Open in Editor`: Launches configured code editor for the website directory via `launcher.OpenEditor`.
  - `Delete`: Displays single-site de-provisioning confirmation dialog.
  - `Back`: Returns to the website list.
- [x] Pressing `Esc` in the Action Menu returns to the website list; pressing `Esc` in the website list returns focus to Sidebar Navigation.

## Testing seam

The `tea.Model` for Websites Hub and the root model. Test by providing mock directory fixtures, dispatching keys (`j`/`k`, `Space`, `d`, `Enter`, `Esc`), and asserting on rendered list items, checkbox markers, action menu entries, and launcher calls.

## Demo path

1. Select `Websites` in Sidebar and press `Enter` to focus into Content Pane.
2. Observe instantaneous display of local websites with `.test` URLs and paths.
3. Move cursor over a site and press `Enter`. Verify 6-item Action Menu appears.
4. Select `Open in Browser` and observe browser launch.
5. Press `Esc` to return to list. Press `Space` on 2 websites, then press `d`. Verify batch deletion confirmation dialog appears with affected paths. Press `Esc` to cancel.

## Scope boundary

Delivers the Websites Hub list, multi-selection, and action menu. The actual execution stepper/logs for long-running workflows (Config, Backup, Delete) wire into the progress monitor delivered in Ticket 04.
