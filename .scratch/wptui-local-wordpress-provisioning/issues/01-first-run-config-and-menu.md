# 01: First-run config and menu WPTUI

Status: resolved
Blocked by: None (can start immediately)
Parent spec: ../spec.md

## What to build

Let a user launch `wptui`, complete first-run configuration or load an existing valid config, and reach an accessible terminal menu without exposing secrets. This ticket delivers the complete onboarding/configuration behavior; create becomes actionable in Ticket 02, while unfinished product options remain visibly unavailable.

## Acceptance criteria

- [x] A first launch with no config opens a Huh wizard and persists every approved user-entered field plus the approved non-interactive theme, plugin, tweak, and exclude defaults.
- [x] Herd defaults to enabled; the Website path defaults to `~/Herd` with Herd and `~/Sites` without Herd; Package API URL and key default to empty; administrator and database defaults match the specification.
- [x] Password fields are masked, secret values are never rendered in command output or errors, and the config receives user-only permissions where supported.
- [x] A valid existing config loads without being rewritten.
- [x] Invalid JSON, missing/invalid values, unsupported tweak shapes, or inconsistent config stop startup with a precise error and leave the existing file unchanged.
- [x] The menu shows create plus config, delete, backup, restore, and settings; unfinished options are disabled rather than routed to placeholders.
- [x] The terminal interaction supports Huh accessible mode and clean cancellation.
- [x] A runnable demo using an isolated home directory proves first-run creation, second-run loading, and invalid-config refusal.

## Testing seam

Config load/create public interface and menu presentation model: test with isolated filesystem paths without running external binaries.

## Demo path

Launch with an isolated `HOME`/`USERPROFILE` environment, step through the wizard inputs, inspect the generated configuration, and observe the main menu with `create` enabled and remaining options disabled.

## Answer

Implemented `internal/config`, `internal/tui`, `internal/app`, and `cmd/wptui/main.go`:
- Config storage at `~/.config/wptui/config.json` with 0600 file permissions and 0700 dir permissions.
- Full validation for required fields, port ranges, email validity, and supported `wp_tweaks` types.
- Interactive first-run Huh wizard with password masking (`EchoModePassword`) and accessible mode support.
- Main menu presenting `create`, `config`, `delete`, `backup`, `restore`, `settings`, and `exit`, with non-v1 options cleanly disabled.
- Unit and demo tests (`internal/app/demo_test.go`) demonstrating first-run wizard creation, existing config load without rewrites, and strict halt on invalid/corrupt configs.
