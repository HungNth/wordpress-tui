# 01: Early interactive Website collision check in TUI wizard

Status: ready-for-agent
Blocked by: None (can start immediately)
Parent spec: ../spec.md

## What to build

In the interactive `create` wizard, validate Website Slug availability immediately upon entry rather than waiting until administrator fields and package selections are completed. If the destination directory or database already exists, or if permission/database query errors occur, display an inline error on the Slug field and keep the user in the field to choose an available slug before asking for administrator credentials or prompting for packages.

## Acceptance criteria

- [ ] `tui.SlugAvailabilityChecker` is defined as `func(slug string) error`.
- [ ] `tui.BuildCreateForm` accepts an optional `SlugAvailabilityChecker` callback.
- [ ] Syntactic slug validation runs first: empty, malformed, or Windows-reserved slugs fail immediately without invoking the availability checker.
- [ ] For syntactically valid slugs, the availability checker is called:
  - Checks if the destination directory exists: only `os.IsNotExist(err)` is considered available; any permission/I/O error returns an error.
  - Checks if the MySQL database exists using `client.CheckDatabaseExists`: any query/connection failure returns an error.
  - If the directory exists, returns `"directory <path> already exists"`.
  - If the database exists, returns `"database <name> already exists"`.
- [ ] `tui.PromptCreateInputs` accepts `checker tui.SlugAvailabilityChecker` (or a dedicated signature) to pass into `BuildCreateForm`.
- [ ] In `internal/app/app.go`, `RunCreateFlowWithDeps` wires a production `SlugAvailabilityChecker` combining directory `os.Stat` and `client.CheckDatabaseExists` and passes it to the prompt before package selection.
- [ ] `CreateFlowDependencies.PromptCreate` is updated or wrapped so the injected checker is preserved, ensuring no collision bypasses the new UX.
- [ ] Programmatic preflight checks inside `create.Creator.Create()` remain in place as defense-in-depth against race conditions.

## Testing seam

- `internal/tui`: Unit tests for `BuildCreateForm` / `PromptCreateInputs`:
  - Syntactically invalid slug rejects without calling the checker.
  - Directory collision produces an inline error on the slug field.
  - Database collision produces an inline error on the slug field.
  - Available slug succeeds and allows the form to advance.
- `internal/app`: Unit test for `RunCreateFlowWithDeps` verifying that a colliding slug rejects at the prompt phase before package prompts are called.

## Demo path

Launch `wptui`, enter an existing website name whose directory or database already exists, observe an immediate inline error on the slug field, change the slug to an available name, and observe the form proceed to administrator credentials.
