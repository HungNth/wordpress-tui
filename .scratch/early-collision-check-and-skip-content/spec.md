# Early Interactive Collision Checks and Lean Core Download

Status: ready-for-agent
## Problem Statement

When creating a new WordPress website in WPTUI:
1. Users currently enter the Website Name, Website Slug, Admin Username, Admin Password, and Admin Email across the full form before WPTUI performs collision checks. If the website directory or database already exists, the entire creation fails at the end, forcing the user to re-enter all details.
2. WordPress core download fetches default bundled themes (such as Twenty Twenty-Four) and plugins (Hello Dolly, Akismet). In local development, developers usually install custom themes and plugins, making default bundled themes and plugins unnecessary clutter that slows down the initial installation.

## Solution

1. **Immediate Slug Collision Validation in TUI**:
   - In the interactive creation wizard, after entering the Website Name, the Website Slug is prompted and validated immediately.
   - The Slug field validates not only syntax, but also checks whether the destination directory (`websites_path/<slug>`) or MySQL database (`<slug>`) already exists.
   - If a collision is detected, an inline error is displayed on the Slug field immediately, keeping the user in the field to choose an available slug before proceeding to administrator credential inputs.

2. **Omit Default Themes and Plugins with `--skip-content`**:
   - `wp core download` includes the `--skip-content` flag:
     `wp core download https://wordpress.org/latest.zip --skip-content`
   - Default bundled themes and plugins are skipped, resulting in a cleaner, faster core installation.

## User Stories

1. As a developer creating a Website, I want WPTUI to validate Website Slug availability immediately after input, so that I don't waste time entering administrator credentials for an existing site.
2. As a developer, I want an existing directory collision on the Slug field to display an inline validation error, so that I know the folder is already taken.
3. As a developer, I want an existing database collision on the Slug field to display an inline validation error, so that I know the database name is already taken.
4. As a developer, I want to edit the slug in-place when a collision is reported, so that I can immediately find an available slug without restarting the wizard.
5. As a developer, I want the administrator credential inputs to appear only after a non-colliding slug is confirmed, so that the workflow is smooth and progressive.
6. As a developer, I want core download to pass `--skip-content`, so that default bundled themes (Twenty Twenty-Four, etc.) and plugins (Hello Dolly, Akismet) are not downloaded.
7. As a developer, I want faster WordPress installation and clean `wp-content` directories, so that only explicitly chosen plugins and themes exist on the new Website.
8. As an automated consumer or test harness, I want core `Create()` logic to retain defense-in-depth preflight collision checks, so that programmatic calls remain safe.

## Implementation Decisions

1. **TUI Input Form Progression**:
   - Split `tui.PromptCreateInputs` or update `tui.BuildCreateForm` with a slug preflight checker callback:
     `type SlugAvailabilityChecker func(slug string) error`
   - In `internal/app/app.go`, construct the `SlugAvailabilityChecker` that verifies directory and database availability:
     - Directory check: `os.Stat(targetPath)` — only `os.IsNotExist(err)` means available. Any permission/I/O error returns an error preventing unverified continuation.
     - Database check: `client.CheckDatabaseExists(ctx, dbConn, slug)` — if an error occurs during the check, return the error to fail safely.
     - If either exists, return `"directory %s already exists"` or `"database %s already exists"`.
   - In `tui.BuildCreateForm`, the `WebsiteSlug` input validator first validates syntax (rejecting empty/malformed/Windows-reserved slugs immediately without calling the availability checker); only syntactically valid slugs invoke `SlugAvailabilityChecker`.

2. **Core Download `--skip-content` Flag**:
   - In `internal/wpcli/wpcli.go`, update `CoreDownload`:
     `args := []string{"core", "download", downloadURL, "--skip-content"}`
   - The command executed will be:
     `wp core download https://wordpress.org/latest.zip --skip-content`
   - When Package integration is disabled and a default theme is configured, no bundled default themes are retained: `DefaultThemeSkipped` is reported as true, and active-theme reporting correctly reflects that no default theme was activated. If no default theme is configured, `DefaultThemeSkipped` remains false.
   - All tests in `internal/wpcli` and `internal/create` expecting `wp core download` calls will reflect the `--skip-content` argument.

3. **Defense-in-Depth Preflights**:
   - `create.Creator.Create()` continues to perform its preflight checks (`Preflight 2: Directory collision`, `Preflight 3: Database collision`) to guarantee atomicity and safety for non-interactive or programmatic invocations.

## Testing Decisions

1. **TUI Seam (`internal/tui`)**:
   - Unit tests for `PromptCreateInputs` / `BuildCreateForm` using mock `SlugAvailabilityChecker` confirming:
     - Syntax error prevents collision check.
     - Collision checker error produces form validation error on the slug field.
     - Available slug allows form validation to pass.
2. **WP-CLI Seam (`internal/wpcli`)**:
   - Unit tests for `CoreDownload` verifying that `--skip-content` is passed to the WP-CLI runner.
3. **End-to-End Orchestrator Seam (`internal/create`, `internal/app`)**:
   - Run create flow tests and smoke tests ensuring complete creation succeeds with `--skip-content`.

## Out of Scope

- Removing core preflight checks inside `create.Creator` (they remain as defense-in-depth).
- Selective downloading of individual bundled themes (either skipped with `--skip-content` or downloaded via configured package integration).

## Further Notes

- `--skip-content` is natively supported by `wp core download` across all recent versions of WP-CLI.
