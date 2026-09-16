# 02: Early interactive Website collision check in TUI wizard

**What to build:**
In the interactive `create` wizard, validate Website Slug availability immediately upon entry rather than waiting until all administrator fields and package selections are completed. If the destination directory or database already exists, or if permission/database query errors occur, display an inline error on the Slug field and keep the user in the field to choose an available slug before asking for administrator credentials.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

## Acceptance criteria

- [ ] `tui.BuildCreateForm` accepts an optional `SlugAvailabilityChecker` callback (`func(slug string) error`).
- [ ] Syntactic slug validation runs first: empty, malformed, or Windows-reserved slugs fail immediately without invoking the availability checker.
- [ ] For syntactically valid slugs, the availability checker is called:
  - Checks if the destination directory exists: only `os.IsNotExist(err)` is considered available; any permission/I/O error returns an error.
  - Checks if the MySQL database exists using `client.CheckDatabaseExists`: any query/connection failure returns an error.
  - If the directory exists, returns `"directory <path> already exists"`.
  - If the database exists, returns `"database <name> already exists"`.
- [ ] In `internal/app/app.go`, the availability checker is injected into the TUI prompt before collecting administrator credentials and package selections.
- [ ] Programmatic preflight checks inside `create.Creator.Create()` remain in place as defense-in-depth against race conditions.

## Testing seam

- `internal/tui`: unit tests for `BuildCreateForm` / `PromptCreateInputs`:
  - Syntactically invalid slug rejects without invoking the availability checker.
  - Directory collision returns an inline error on the slug field.
  - Database collision returns an inline error on the slug field.
  - Available slug succeeds and proceeds to administrator inputs.
- `internal/app`: unit tests for `RunCreateFlowWithDeps` asserting availability checker wiring and early rejection before package prompts.

## Demo path

Launch `wptui`, enter an existing website name whose directory or database already exists, observe an immediate inline error on the slug field, change the slug to an available name, and observe the form proceed to administrator credentials.
