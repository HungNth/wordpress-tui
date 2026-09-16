# 02: Provision a basic WordPress Website

Status: resolved
Blocked by: 01
Parent spec: ../spec.md

## What to build

Let a user create a complete local WordPress Website without optional tweaks or Packages. The flow collects Website identity and administrator values, validates the environment, creates the database before WordPress installation, configures the local URL, and removes only current-run resources after critical failure or cancellation.

## Acceptance criteria

- [x] Create is enabled and collects Website Name, a separately editable Website Slug prefilled from the Name, and optional administrator overrides.
- [x] Website Slug validation enforces the approved ASCII param-case, length, edge-hyphen, and Windows-reserved-name rules.
- [x] Directory and database collisions are detected before mutation and return the user to Website identity input without reusing or deleting existing resources.
- [x] Required executables are checked before mutation; Herd is required only in Herd mode.
- [x] WordPress core is downloaded, `wp-config.php` is created with database checking skipped, the database is created, and `wp core install` runs only after database creation succeeds.
- [x] Database and administrator passwords use secret-safe stdin prompting where WP-CLI supports it and never appear in rendered command lines.
- [x] Herd mode uses the configured Website path, secures the Website without linking it as the final step, and returns an HTTPS `.test` URL; non-Herd mode returns an HTTP `.test` URL and relies on the documented wildcard-stack prerequisite.
- [x] Database creation failure never invokes core install and leaves no Website directory or run-owned database behind.
- [x] Critical failures and user cancellation produce named, sanitized progress and rollback only the directory, database, and Herd TLS state created by the run.
- [x] A behavioral test at the create-use-case seam proves a successful basic Website and representative rollback paths.

## Testing seam

Create orchestrator use-case seam: invoke the orchestrator with controlled external command execution and verify resulting filesystem, database call sequence, and rollback cleanup.

## Demo path

Select `create`, input a valid Website Name, confirm default slug and admin values, observe sequential progress through core download, DB creation, and installation, and view the completion summary with the `.test` URL.

## Answer

Implemented `internal/create` and `internal/wpcli`:
- Validated Website Slug generation without truncation, enforcing 1-63 ASCII lowercase, digits, and hyphens without edge hyphens or reserved Windows names.
- Preflight collision detection for both directory and database before any mutation begins:
  - Database collision check uses `mysql --defaults-extra-file` without exposing passwords in argv, and exact-matches the schema name without raw string concatenation.
  - Preflight DB check failure immediately halts execution before any mutation occurs.
- Atomic leaf directory creation via `os.Mkdir` ensures no race conditions overwrite or delete pre-existing directories.
- Sequential WP-CLI database/install ordering: directory creation -> `wp core download --skip-content` -> `wp config create --skip-check` -> `wp db create` -> `wp core install` -> tweaks/packages -> `herd secure` (terminal step).
- Passwords for DB and Admin supplied through stdin via `--prompt` to prevent process/log exposure.
- Strict ownership tracking where `createdDB` is set only upon actual creation by `DBCreate`, preventing destruction of external databases during rollback.
- Verified with unit and demo tests in `internal/create/create_test.go`, `internal/create/demo_test.go`, and `internal/wpcli/wpcli_test.go`.
