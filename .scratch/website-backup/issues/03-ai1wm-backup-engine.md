# Title: Strategy 2 All-in-One WP Migration backup engine
Status: resolved
Labels: ready-for-agent

## Parent spec
.scratch/website-backup/spec.md

## Required behavior
Implement Strategy 2 in package `internal/backup`:
1. `EnsureAI1WMExtension(ctx context.Context, siteDir string, resolver PackageResolver, client WPClient, onProgress ProgressFunc) error`:
   - Checks and installs `all-in-one-wp-migration-unlimited-extension` using version-aware logic (`siteconfig.InstallPackages`).
2. `RunAI1WMBackup(ctx context.Context, siteDir, slug, backupPath string, excludes []string, client WPClient, onProgress ProgressFunc) (*BackupResult, error)`:
   - Formats excludes: comma-separated string from `excludes`.
   - Runs `wp ai1wm backup --exclude-cache --exclude-files=<excludes>`.
   - Parses output to find `Backup location: <path>`.
   - Moves `.wpress` file to `backupPath` as `ai1wm_<slug>_YYYY-MM-DD_HH-mm-ss.wpress`.
   - Leaves plugin installed on site.

## Acceptance criteria
- [x] Command arguments correctly include `--exclude-cache` and `--exclude-files=...`.
- [x] Parses `Backup location: <path>` accurately across Windows and macOS path separators.
- [x] Moves and renames file into `backupPath` according to `ai1wm_<slug>_YYYY-MM-DD_HH-mm-ss.wpress`.
- [x] Unit tests with mock runner verifying command invocation and output parsing in `internal/backup/ai1wm_backup_test.go`.
