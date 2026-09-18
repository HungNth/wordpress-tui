# Title: Configuration field backup_path and wizard prompt
Status: resolved
Labels: ready-for-agent

## Parent spec
.scratch/website-backup/spec.md

## Required behavior
1. Add `BackupPath string `json:"backup_path"` to `config.Config` in `internal/config/config.go`.
2. In `internal/tui/wizard.go`:
   - Add `BackupPath string` to `WizardInputs`.
   - In `BuildMainWizardForm`, add an input field for Backup Storage Path with description "Path to store website backup archives".
   - Default value: if empty, defaults to `filepath.Join(inputs.WebsitesPath, "backups")`.
3. In `internal/config/config_test.go` and `internal/tui/wizard_test.go`:
   - Test JSON serialization and wizard input defaults for `backup_path`.

## Acceptance criteria
- [x] `backup_path` is serialized/deserialized cleanly in `config.json` (`TestConfig_BackupPathSerialization`).
- [x] Wizard defaults to `<websites_path>/backups` when user leaves the field blank (`TestConvertInputsToConfig_BackupPath`).
- [x] A legacy `config.json` with no `backup_path` key still loads, with no injected fallback (`TestConfig_LegacyLoadWithoutBackupPath`).
- [x] Unit tests pass in `internal/config` and `internal/tui`.
