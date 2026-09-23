# Master-Detail Bubble Tea TUI Specification

Status: ready-for-agent

## Problem Statement

WPTUI's interface currently relies on sequential Huh forms and select menus rendered inline in the standard terminal scrollback buffer. This presentation feels flat, static, and disjointed:
1. Moving between operations requires repeatedly navigating through the root menu and disparate pickers.
2. Managing existing websites (configuring, backing up, or deleting) requires selecting the target website multiple times across separate workflows rather than managing it from a centralized dashboard.
3. Synchronous database checks during website discovery can introduce perceptible delays and stutter.
4. Terminal resizing causes text wrapping and display glitches in the scrollback buffer.
5. Long-running task logs mix with prompt rendering, lacking a dedicated, scrollable execution viewport.

A local WordPress developer using a dark terminal wants a modern, responsive, two-column Master-Detail TUI powered by Bubble Tea, featuring a persistent navigation sidebar, instantaneous website management, smooth keyboard transitions, and an in-pane progress monitor.

## Solution

WPTUI adopts a full-screen alternate-screen terminal architecture (`tea.WithAltScreen()`) implementing a Master-Detail Layout:
1. **Header & Footer**: An application title and system context header at the top, and a dynamic contextual keyboard shortcut footer at the bottom.
2. **Sidebar Navigation (Left Column)**: A persistent, fixed-width (~28 columns) navigation menu managing primary destinations: `Websites`, `Create`, `Restore`, `Settings`, and `Exit`.
3. **Content Pane (Right Column)**: A dynamic operational surface rendering context-specific views:
   - **Websites Hub**: Instantaneous inventory of local WordPress websites discovered by filesystem scan (without synchronous database calls), supporting single/batch selection and a contextual action menu (`Config`, `Backup`, `Open in Browser`, `Open in Editor`, `Delete`).
   - **Create Wizard**: A multi-step Bubble Tea form collecting website parameters (Name, Slug, Admin credentials, Tweaks) followed by package selection via the existing live search model.
   - **Restore Wizard**: A multi-step reconstitution wizard selecting backup format, archive source, and target credentials.
   - **Settings View**: Interactive tools to launch the external editor for configuration and open the system cache folder.
4. **Visual Focus Mode**: High-contrast border and title highlighting. The active pane is styled with a bright Cyan (`#00FFFF`) border and bold title; the inactive pane is styled with a muted gray (`#888888`) border.
5. **Keyboard Navigation & Safety**:
   - `Up` / `Down` or `k` / `j` navigate items in the active pane.
   - `Enter` activates the Content Pane from the Sidebar, or advances/submits within forms.
   - `Esc` returns focus to the Sidebar Navigation. If form fields were modified (dirty), a confirmation prompt (`Discard changes? (y/n)`) prevents accidental data loss.
   - `Tab` / `Shift+Tab` or `Left` / `Right` toggle focus between Sidebar and Content Pane.
   - `Space` toggles multi-select items (e.g. batch selecting websites for deletion).
   - `q` or `Ctrl+C` exits the application when on the Sidebar, or halts running tasks.
6. **In-Pane Progress Monitor**: During long-running operations, the Content Pane displays a task stepper with an active spinner and status markers (`[✓]`, `[!]`, `[✗]`), paired with a scrollable log viewport (`bubbles/viewport`) capturing real-time command output.
7. **Terminal Geometry**: Enforces an 80x24 minimum terminal threshold, rendering a clean notice when resized below limits and auto-restoring the layout upon enlargement.

## User Stories

1. As a local WordPress developer, I want WPTUI to launch in full-screen alternate-screen mode, so that my terminal history is uncluttered and the interface utilizes the entire window cleanly.
2. As a local WordPress developer, I want a two-column Master-Detail layout, so that navigation options remain visible while I interact with specific workflows.
3. As a local WordPress developer, I want a persistent Sidebar Navigation on the left, so that I can switch between major application sections without returning to a separate main menu screen.
4. As a local WordPress developer, I want a dedicated Content Pane on the right, so that active forms, lists, and progress indicators have ample space to render details.
5. As a keyboard user, I want the active pane highlighted with a bright Cyan border and bold title, so that I always know which pane currently receives my keystrokes.
6. As a keyboard user, I want the inactive pane displayed with a muted gray border, so that visual noise is minimized.
7. As a keyboard user, I want the footer to display contextual keyboard shortcuts matching the active pane, so that I never have to guess valid keybindings.
8. As a local WordPress developer, I want `Up` / `Down` and `k` / `j` to navigate menu items and lists, so that standard navigation conventions work naturally.
9. As a keyboard user, I want `Enter` on a Sidebar option to focus into the Content Pane, so that I can begin interacting with that section's actions.
10. As a keyboard user, I want `Esc` in the Content Pane to return focus to the Sidebar, so that I can easily navigate to another section.
11. As a local WordPress developer, I want `Esc` on a modified form to prompt `Discard changes? (y/n)`, so that I do not accidentally lose unsaved inputs.
12. As a keyboard user, I want `Esc` on an unmodified form to return directly to the Sidebar without prompting, so that clean navigation remains fast.
13. As a keyboard user, I want `Tab` and `Shift+Tab` to toggle focus between the Sidebar and Content Pane, so that I have standard TUI pane switching.
14. As a keyboard user, I want `q` or `Ctrl+C` on the Sidebar to cleanly exit WPTUI and restore my shell terminal, so that leaving the app is immediate.
15. As a local WordPress developer, I want the `Websites` section to serve as a central Websites Hub, so that I can inspect and manage all my local sites in one place.
16. As a local WordPress developer, I want the Websites Hub to discover local sites via filesystem scan without synchronous database queries, so that the list loads instantly without lag.
17. As a local WordPress developer, I want each item in the Websites list to display its Name/Slug, `.test` URL, and directory path, so that I can identify my sites at a glance.
18. As a local WordPress developer, I want pressing `Enter` on a Website in the list to open its Action Menu, so that I can choose what action to perform on that specific site.
19. As a local WordPress developer, I want the Website Action Menu to include `Config`, `Backup`, `Open in Browser`, `Open in Editor`, `Delete`, and `Back`, so that all site operations are accessible from one menu.
20. As a local WordPress developer, I want selecting `Open in Browser` to launch the site's `.test` URL in my default web browser, so that I can preview the site immediately.
21. As a local WordPress developer, I want selecting `Open in Editor` to open the website directory in my configured code editor, so that I can start coding without finding the folder manually.
22. As a local WordPress developer, I want selecting `Config` from the Website Action Menu to open configuration for that site, so that I do not have to select the site a second time.
23. As a local WordPress developer, I want selecting `Backup` from the Website Action Menu to start backup for that site, so that backup is directly linked to the target site.
24. As a local WordPress developer, I want selecting `Delete` from the Website Action Menu to initiate de-provisioning for that site with an explicit confirmation step, so that accidental deletion is prevented.
25. As a local WordPress developer, I want pressing `Space` in the Websites list to toggle selection checkboxes on multiple sites, so that I can target multiple sites at once.
26. As a local WordPress developer, I want pressing `d` with multiple websites selected to prompt for batch deletion, so that I can cleanly delete obsolete testing sites in bulk.
27. As a local WordPress developer, I want the batch deletion prompt to list every affected directory and database before proceeding, so that destructive actions remain safe.
28. As a local WordPress developer, I want selecting `Create` in the Sidebar to open the Create Wizard in the Content Pane, so that I can provision a new WordPress site.
29. As a local WordPress developer, I want the Create Wizard to use clean Bubble Tea inputs with `Tab` / `Shift+Tab` / `Down` / `Up` field navigation, so that filling in details is smooth.
30. As a local WordPress developer, I want entering a Website Name to auto-derive a valid Website Slug if the slug is left blank, so that I do not have to retype names.
31. As a local WordPress developer, I want the Create Wizard to validate slug syntax and directory availability, so that collision errors are caught before provisioning begins.
32. As a local WordPress developer, I want the Create Wizard to transition to the Live Package Search picker in the Content Pane, so that I can search and select plugins and themes.
33. As a local WordPress developer, I want package selections preserved while filtering queries, so that searching multiple times accumulates my chosen packages.
34. As a local WordPress developer, I want submitting the Create Wizard to transition the Content Pane to the In-Pane Progress Monitor, so that I can watch the provisioning process.
35. As a local WordPress developer, I want selecting `Restore` in the Sidebar to open the Restore Wizard in the Content Pane, so that I can reconstitute a site from an archive.
36. As a local WordPress developer, I want the Restore Wizard to support selecting Full ZIP or AI1WM archive formats and picking archive files, so that both restore engines are available.
37. As a local WordPress developer, I want selecting `Settings` in the Sidebar to display application settings in the Content Pane, so that I can edit config or inspect caches.
38. As a local WordPress developer, I want the In-Pane Progress Monitor to display a Task Stepper at the top with a spinner and persistent checkmarks (`[✓]`, `[!]`, `[✗]`), so that I see step-by-step progress.
39. As a local WordPress developer, I want the In-Pane Progress Monitor to provide a scrollable Log Viewport at the bottom, so that I can read live WP-CLI, database, and extraction logs.
40. As a local WordPress developer, I want to use `Up` / `Down` or `PageUp` / `PageDown` in the Log Viewport to scroll log history, so that I can inspect previous error messages.
41. As a local WordPress developer, I want Sidebar navigation disabled and dimmed during task execution, so that I cannot trigger conflicting operations while background work runs.
42. As a local WordPress developer, I want completed operations to display a color-coded outcome summary in the Content Pane, so that results and warnings are clearly reported.
43. As a local WordPress developer, I want terminal windows narrower than 80 columns or shorter than 24 rows to show a polite resize notice, so that layout distortion is prevented.
44. As a local WordPress developer, I want resizing the terminal back to at least 80x24 to immediately re-render the active screen, so that resizing is seamless.

## Implementation Decisions

1. **Master-Detail Two-Column Layout**:
   - The root application model manages the overall full-screen alternate buffer (`tea.WithAltScreen()`).
   - The display layout partitions terminal width into:
     - Left column: Sidebar Navigation (fixed width of 28 columns).
     - Right column: Content Pane (remaining available width).
     - Top row: Application title bar with active context.
     - Bottom row: Contextual keyboard shortcut status bar.
   - Enforce minimum dimensions of 80x24: if terminal width < 80 or height < 24, render a centered resize notice: `Terminal window too small (minimum 80x24 required)`.

2. **Focus Mode State Machine**:
   - Focus is represented as a state enum: `FocusSidebar` or `FocusContent`.
   - In `FocusSidebar`:
     - Arrow keys `Up` / `Down` or `k` / `j` change the active sidebar section.
     - `Enter` or `Right` or `Tab` shifts focus to `FocusContent`.
     - `q` or `Ctrl+C` initiates clean application shutdown.
   - In `FocusContent`:
     - Keys are handled by the currently active content view model.
     - `Esc` returns focus to `FocusSidebar`. If the active content model reports dirty/uncommitted form changes, a confirmation dialog intercept is shown.
     - `Shift+Tab` or `Left` (when not within an editable text input) returns focus to `FocusSidebar`.
   - Borders and titles dynamically reflect focus state: Cyan `#00FFFF` with bold styling for active pane; muted gray `#888888` for inactive pane.

3. **Websites Hub Architecture**:
   - Replaces disconnected website discovery pickers across Config, Backup, and Delete.
   - Discovers websites asynchronously via directory listing of `WebsitesPath` (respecting `DeleteExcludes`), without synchronous database queries or WP-CLI checks.
   - Content Pane renders the website list with search/filter, cursor selection, and multi-select checkboxes (`Space`).
   - Selecting a website opens the Website Action Panel with six actions:
     - `Config`: Loads Site Configuration view for the target site.
     - `Backup`: Loads Website Backup view for the target site.
     - `Open in Browser`: Invokes `launcher.OpenURL("http://" + slug + ".test")`.
     - `Open in Editor`: Invokes `launcher.OpenEditor(websitePath)`.
     - `Delete`: Displays single-site de-provisioning confirmation.
     - `Back`: Returns to the website list.
   - Pressing `d` when one or more websites are checked triggers batch de-provisioning confirmation.

4. **Pure Bubble Tea Form Engine**:
   - Replaces standalone Huh form modals with native Bubble Tea form models embedded inside the Content Pane.
   - Form models encapsulate text input controls (`bubbles/textinput`), toggles, and buttons.
   - Field navigation uses `Tab` / `Shift+Tab` and `Down` / `Up`.
   - Integrates the existing `LiveSearchModel` as a secondary step in Create and Config wizards for package selection.
   - Dirty-tracking state prevents accidental abandonment on `Esc`.

5. **In-Pane Progress Monitor**:
   - Execution views implement a split view in the Content Pane:
     - Upper pane: Task Stepper with animated spinner and status icons (`[✓]`, `[!]`, `[✗]`).
     - Lower pane: Log Viewport (`bubbles/viewport`) capturing piped stdout/stderr streams.
   - Background tasks execute via `tea.Cmd` messaging without blocking the UI event loop.
   - Sidebar navigation is locked and dimmed during active execution.

## Testing Decisions

- **Testing Principles**:
  - Test external behavior, keyboard interactions, state transitions, and view rendering—not internal private helper functions.
  - Assert that specific inputs produce expected model state transitions, visual focus changes, and rendered ANSI/text substrings.
- **Highest Testing Seam**:
  - The root `tea.Model` interface at `internal/tui`.
  - Tests feed sequences of `tea.Msg` (e.g. `tea.WindowSizeMsg`, `tea.KeyPressMsg`, and action completions) to `model.Update(msg)` and assert on `model.View()` and exported inspector properties.
- **Modules Tested**:
  - `internal/tui`: Master-Detail layout, Sidebar navigation, Focus Mode switching, Websites Hub list and actions, Create/Restore wizards, In-Pane Progress monitor, resize handling.
  - `internal/app`: Integration bridging between the Bubble Tea root model, configuration, and execution flows.
- **Prior Art**:
  - `internal/tui/live_search_test.go`: Existing testing seam feeding `tea.KeyPressMsg` into `Update` and asserting on rendered Lipgloss styles in `ViewString()`.

## Out of Scope

1. Light terminal theme and theme configuration selector.
2. Synchronous database health/version checks during website list discovery.
3. Fallback inline scrollback mode (superseded by full-screen alt-screen).
4. Remote server provisioning or cloud deployment workflows.

## Further Notes

- Supersedes ADR-0001 (Use Huh for terminal interface).
- Implements architectural decisions recorded in ADR-0016 (`0016-fullscreen-master-detail-bubbletea-tui.md`) and the updated `DESIGN.md`.
