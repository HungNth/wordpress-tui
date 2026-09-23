# 01: Master-Detail Shell, Sidebar Navigation, and Focus State Machine

**Parent specification:** [Master-Detail Bubble Tea TUI](../spec.md)

**Design contract:** [WPTUI Terminal Design](../../../DESIGN.md)

**What to build:** The root Bubble Tea application shell operating in full-screen alternate-screen mode (`tea.WithAltScreen()`). Partition the screen into a persistent left Sidebar Navigation (~28 columns), a dynamic right Content Pane, a top Header bar, and a contextual Keymap Footer. Implement the Focus Mode state machine (`FocusSidebar` ↔ `FocusContent`) with bright Cyan `#00FFFF` active border highlighting, muted `#888888` inactive border, and terminal resize geometry enforcement (minimum 80x24).

**Blocked by:** None (can start immediately).

**Status:** resolved

## Acceptance criteria

- [x] WPTUI initializes in full-screen alternate-screen buffer (`tea.WithAltScreen()`) and restores the terminal cleanly upon exit.
- [x] Screen partitions into three vertical zones: top Header bar with application title `WPTUI — WordPress Local Manager`, body containing the two-column Master-Detail layout, and bottom Keymap Footer.
- [x] Sidebar Navigation occupies a fixed width of 28 columns on the left and displays 5 top-level items: `Websites`, `Create`, `Restore`, `Settings`, `Exit`.
- [x] Focus Mode state machine toggles between `FocusSidebar` and `FocusContent`:
  - Active pane displays a bright Cyan (`#00FFFF`) border and bold title.
  - Inactive pane displays a muted gray (`#888888`) border and muted title.
- [x] Navigation controls:
  - In `FocusSidebar`: `Up` / `Down` and `k` / `j` navigate menu items; `Enter`, `Right`, or `Tab` shifts focus to `FocusContent`; `q` or `Ctrl+C` cleanly exits WPTUI.
  - In `FocusContent`: `Esc` or `Left` or `Shift+Tab` returns focus to `FocusSidebar`.
- [x] The Keymap Footer dynamically updates keybinding descriptions based on the active Focus Mode.
- [x] Terminal geometry enforcement: If terminal width < 80 or height < 24, a centered notice `Terminal window too small (minimum 80x24 required)` is rendered; layout auto-restores when resized >= 80x24.

## Testing seam

The root `tea.Model` in `internal/tui`. Test by dispatching `tea.WindowSizeMsg` and `tea.KeyPressMsg` to `Update(msg)` and asserting on `model.View()` output, focus state, active sidebar selection, and resize handling.

## Demo path

1. Launch WPTUI in terminal. Verify full-screen alt-screen layout with Header, Sidebar, Content Pane, and Footer.
2. Press `j` / `k` or `Down` / `Up` to move between sidebar options. Observe Cyan cursor.
3. Press `Enter` to focus into Content Pane. Verify border turns Cyan on Content Pane and Muted on Sidebar.
4. Press `Esc` to return focus to Sidebar. Verify border turns Cyan on Sidebar.
5. Resize terminal to < 80x24. Observe polite resize notice. Resize back to >= 80x24 and observe layout restoration.
6. Press `q` on Sidebar and verify clean exit to shell.

## Scope boundary

Delivers the application shell, layout, focus engine, and resize handling. Specific content view implementations (Websites Hub, wizards, progress monitor) belong to subsequent tickets.
