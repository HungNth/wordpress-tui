# Title: Strategy 1 Full source code and database backup engine
Status: ready-for-agent
Labels: ready-for-agent

## Parent spec
.scratch/website-backup/spec.md

## Required behavior
Implement Strategy 1 in package `internal/backup`:
1. `ExportDatabase(ctx context.Context, siteDir, slug string, client WPClient) (string, error)`:
   - Runs `wp db export <slug>.sql` in `siteDir`.
   - Returns path to generated SQL dump: `filepath.Join(siteDir, slug+".sql")`.
2. `CreateFullZipArchive(ctx context.Context, siteDir, tempZipPath string, excludes []string, onProgress ProgressFunc) error`:
   - Packages `siteDir` into `tempZipPath` located in an isolated temporary directory outside `siteDir` (`os.MkdirTemp("", "wptui-backup-*")`).
   - Walks `siteDir`, skipping entries matching `excludes`, existing `*.zip` or `*.wpress` files.
   - Normalizes all zip entry headers to clean relative paths using forward slashes (`/`), stripping Windows backslashes and drive letters for cross-platform portability across Windows and macOS.
   - Packages files with `zip.Deflate`.
3. `RunFullBackup(ctx context.Context, siteDir, slug, backupPath string, excludes []string, client WPClient, onProgress ProgressFunc) (*BackupResult, error)`:
   - Coordinates:
     1. Export DB to `<slug>.sql`.
     2. Create zip archive in an isolated temporary directory outside `siteDir`.
     3. Ensure `backupPath` exists and safely relocate the temporary zip archive into `backupPath` as `full_<slug>_YYYY-MM-DD_HH-mm-ss.zip` (handling cross-volume moves cleanly via copy-then-delete if rename fails across partitions).
     4. **Strict Cleanup Invariant**: Delete the temporary `<slug>.sql` file **strictly after both archive creation AND relocation to `backup_path` succeed**.
   - If zip creation, closing, or relocation fails for any reason, retains `<slug>.sql` on disk and returns the underlying error.
   - Calculates artifact size and elapsed execution duration.

## Acceptance criteria
- [ ] Archive is constructed in a temporary folder outside `siteDir` to avoid recursive self-archiving.
- [ ] Temporary SQL dump is deleted strictly after both successful zip creation and move into `backup_path`.
- [ ] Temporary SQL dump is explicitly retained on disk if zip creation OR file move fails.
- [ ] Safe relocation supports cross-volume moves between temp dir and `backup_path`.
- [ ] Zip entries use relative paths with forward slashes (`/`) for Windows/macOS extraction portability.
- [ ] Excludes properly skip `backup_excludes` patterns and previous backup archives.
- [ ] Comprehensive unit tests in `internal/backup/full_backup_test.go` verify:
  - Cross-platform forward-slash path separators in zip headers.
  - Relocation failure preserves `<slug>.sql` on disk.
  - Successful backup cleans up `<slug>.sql`.
