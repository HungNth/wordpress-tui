# 01: Unified Entry, Setup, and Application Settings

**Parent specification:** [Unified Terminal UI](../spec.md)

**Design contract:** [WPTUI Terminal Design](../../../DESIGN.md)

**What to build:** A consistent, dark-terminal entry experience from first-run setup through the main menu and Application Settings. Users can recognize focus, read contextual help, navigate back without losing pending input, open the existing settings tools, and distinguish cancellation from failure. Establish shared presentation conventions through these working workflows rather than ship an unused UI foundation.

**Blocked by:** None (can start immediately).

**Status:** resolved

## Acceptance criteria

- [x] First-run setup, the main menu, and Application Settings use the shared inline, left-aligned, single-column hierarchy: contextual title, short description, content, and relevant keyboard help. All application-owned wording is English; user-entered content is preserved.
- [x] Their screens and notifications use the single dark-background Visual Theme Palette and markers defined by the design contract. Focus, checked items, status, and secondary text remain distinguishable without color. No light theme, background override, theme picker, or new configuration fields are introduced.
- [x] Menus use plain action labels instead of decorative emoji, banners, and numeric prefixes without numeric shortcuts. Explicit Back is last where a navigation menu has a parent; the main menu retains its Exit action.
- [x] Field and list controls match the design contract. Help advertises only actions available on the current screen. Back through setup retains pending input; input validation retains values and explains the invalid field.
- [x] Esc at the main menu keeps the menu open. Exit or Ctrl+C exits. Ctrl+C during incomplete first-run setup exits without saving incomplete configuration or entering the configured application menu.
- [x] Completed first-run setup preserves its result and enters the main menu directly. Existing configuration loading and validation retain their business behavior.
- [x] Back from Application Settings returns to the main menu. A cancelled Settings workflow returns neutrally to the main menu rather than printing an operation error; a subsequent workflow remains usable.
- [x] Existing Settings actions still invoke the intended External Launcher behavior. Success and genuine failure use concise semantic status messages, useful redacted detail, and no empty report template. Successful actions return directly to Settings without an extra continuation prompt.
- [x] Password inputs remain masked; configuration credentials and API keys do not appear in setup notifications, Settings output, or scrollback.
- [x] At ordinary and narrow terminal sizes, content fits or wraps and long lists remain navigable. Completed messages stay in scrollback; any reported path remains complete and copyable.
- [x] The shared conventions are reusable by the remaining workflows. Perform required prefactoring before the behavior changes within this slice, reuse the current terminal dependencies, and migrate every affected caller if a shared interface changes. Existing workflows remain runnable; no compatibility shim or duplicate design framework is introduced.
- [x] Actual terminal evidence covers fresh setup, configured startup, Settings success/failure, Back, cancellation, and exit. Record the operating system and sizes exercised, and identify any unverified surface instead of treating noninteractive tests as visual proof.

## Testing seam

Use the existing application startup/options boundary and Application Settings flow dependency injection. Temporary configuration and fixture launchers isolate external effects. Verify resulting navigation and persisted configuration, not helper calls or style-object fields. Include cancellation followed by another workflow and preservation of existing configuration. Feed real keys through the terminal UI for the visual smoke scenarios; prompt stubs alone are insufficient.

Update tests affected by the deliberate menu or cancellation contract changes. Keep behavioral regressions for meaningful boundaries; replace no obsolete wording snapshot with another wording snapshot.

## Demo path

1. Launch WPTUI with an isolated fresh configuration location. Enter setup values, go back, verify retained values, then complete setup and reach the main menu.
2. Open Settings and exercise its existing actions with safe fixture launch effects. Observe the result, remain in Settings, and return to the main menu with Back.
3. Cancel a Settings workflow, open it again, then exit from the main menu. Repeat the presentation checks at a narrow width and without color.

## Scope boundary

This slice delivers startup and Settings end to end, including the common presentation they use. Coordinated long-running progress and migration of the other operation workflows belong to the dependent tickets. It does not redesign their business behavior or authorize commits.
