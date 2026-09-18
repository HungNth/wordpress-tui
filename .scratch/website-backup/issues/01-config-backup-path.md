# Title: Configuration field backup_path and wizard prompt
Status: ready-for-agent
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
- [ ] `backup_path` is serialized/deserialized cleanly in `config.json`.
- [ ] Wizard defaults to `<websites_path>/backups` when user leaves the field blank.
- [ ] Unit tests pass in `internal/config` and `internal/tui`.
