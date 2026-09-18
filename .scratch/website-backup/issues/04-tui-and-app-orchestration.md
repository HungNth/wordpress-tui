# Title: Backup TUI views, menu enablement, and app flow orchestration
Status: resolved
Labels: ready-for-agent

## Parent spec
.scratch/website-backup/spec.md

## Required behavior
1. In `internal/tui/menu.go`:
   - Enable `backup` menu item (`Disabled: false`, description: "Backup an existing WordPress website").
   - Update footer description ("Coming soon: Restore").
2. In `internal/tui/backup_wizard.go`:
   - `SelectBackupStrategy(siteSlug string) (BackupStrategy, error)`: displays options:
     1. Full source code & database (.zip)
     2. All-in-One WP Migration (.wpress)
     `← Back`
   - `PrintBackupSummary(res *BackupResult)`: prints color-coded summary (archive path, size, duration).
3. In `internal/app/app.go` and `internal/app/backup_flow.go`:
   - Wire `case "backup":` to `a.backupFn(ctx, a.config)`.
## Acceptance criteria
- [x] Backup option enabled and selectable from main menu.
- [x] End-to-end flow test verifying full and AI1WM backups in `internal/app/backup_flow_test.go`.
- [x] Full test suite passes under `-race` with 0 warnings from `go vet`.

## Limitations
- Direct keystroke-level coverage for Back and Esc is not present; flow-level back selection is verified while keystroke handling relies on Huh's tested Select component.
