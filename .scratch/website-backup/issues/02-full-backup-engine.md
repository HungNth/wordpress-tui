# Title: Strategy 1 Full source code and database backup engine
Status: resolved
Labels: ready-for-agent

## Parent spec
.scratch/website-backup/spec.md

## Required behavior
Implement Strategy 1 in package `internal/backup`:
1. `ExportDatabase(ctx context.Context, siteDir, slug string, client WPClient) (string, error)`:
   - Runs `wp db export <slug>.sql` in `siteDir`.
   - Returns path to generated SQL dump: `filepath.Join(siteDir, slug+".sql")`.
2. `CreateFullZipArchive(ctx context.Context, siteDir, tempZipPath, backupPath string, excludes []string, onProgress ProgressFunc) error`:
   - Packages `siteDir` into `tempZipPath` located in an isolated temporary directory outside `siteDir` (`os.MkdirTemp("", "wptui-backup-*")`).
   - Walks `siteDir`, skipping entries matching `excludes`, existing `*.zip` or `*.wpress` files, and the `backupPath` subtree when it is nested inside the website.
   - Normalizes all zip entry headers to clean relative paths using forward slashes (`/`), stripping Windows backslashes and drive letters for cross-platform portability across Windows and macOS.
   - Packages files with `zip.Deflate`, and returns any `archive.Close`/file-close finalization error.
3. `RunFullBackup(ctx context.Context, siteDir, slug, backupPath string, excludes []string, client WPClient, onProgress ProgressFunc) (*BackupResult, error)`:
   - Coordinates:
     1. Export DB to `<slug>.sql`.
     2. Create zip archive in an isolated temporary directory outside `siteDir`.
     3. Ensure `backupPath` exists and relocate the temporary zip archive into `backupPath` as `full_<slug>_YYYY-MM-DD_HH-mm-ss.zip`; same-volume moves use `os.Rename`, cross-volume moves copy into a temp file in `backupPath`, flush, close, and rename into place.
     4. **Strict Cleanup Invariant**: Delete the temporary `<slug>.sql` file **strictly after both archive creation AND relocation to `backup_path` succeed**; a deletion failure returns an explicit error instead of a false success.
   - If zip creation, archive finalization, relocation, or dump deletion fails for any reason, retains `<slug>.sql` on disk and returns the underlying error.
   - Calculates artifact size and elapsed execution duration.

## Acceptance criteria
- [x] Archive is constructed in a temporary folder outside `siteDir` to avoid recursive self-archiving.
- [x] A nested `backup_path` inside the website tree is excluded from the archive (`TestRunFullBackup_NestedBackupDirExcluded`).
- [x] Temporary SQL dump is deleted strictly after both successful zip creation and move into `backup_path`.
- [x] Temporary SQL dump is explicitly retained on disk when archive relocation fails after successful zip creation (`TestRunFullBackup_RelocationFailureRetainsSQLDump`).
- [x] Relocation uses `os.Rename` on the same volume and a flush-then-rename copy fallback across volumes; failures propagate instead of reporting success.
- [x] Relocation tests verify success removes the source, failure retains the source, and destination temp files are cleaned (`TestRelocateFile_SuccessMovesContentAndRemovesSource`, `TestRelocateFile_FailureRetainsSourceAndCleansTempDestination`).
- [x] Archive finalization errors (`archive.Close`, `zipFile.Close`) and dump-deletion failure are returned rather than swallowed (`TestRunFullBackup_CleanupFailureReturnsError`).
- [x] Zip entries use relative paths with forward slashes (`/`) for Windows/macOS extraction portability.
- [x] Excludes properly skip `backup_excludes` patterns and previous backup archives.
- [x] Comprehensive unit tests in `internal/backup/full_backup_test.go` verify:
  - Cross-platform forward-slash path separators in zip headers.
  - Relocation failure preserves `<slug>.sql` on disk.
  - Cleanup failure is reported after the archive is produced.
  - Successful backup cleans up `<slug>.sql`.
