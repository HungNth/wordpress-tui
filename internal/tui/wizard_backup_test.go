package tui_test

import (
	"path/filepath"
	"testing"

	"wptui/internal/tui"
)

func TestConvertInputsToConfig_BackupPath(t *testing.T) {
	tempDir := t.TempDir()

	// 1. When user leaves BackupPath empty, it defaults to <websites_path>/backups
	inputsDefault := tui.WizardInputs{
		UsedHerd:             true,
		WebsitesPath:         filepath.Join(tempDir, "sites"),
		BackupPath:           "",
		DefaultAdminUsername: "admin",
		DefaultAdminPassword: "password123",
		DefaultAdminEmail:    "admin@example.com",
		DatabaseHost:         "localhost",
		DatabasePortStr:      "3306",
		DBUsername:           "root",
	}

	cfgDefault, err := tui.ConvertInputsToConfig(inputsDefault, tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDefaultBackupPath := filepath.Join(tempDir, "sites", "backups")
	if cfgDefault.BackupPath != expectedDefaultBackupPath {
		t.Errorf("expected default backup path %q, got %q", expectedDefaultBackupPath, cfgDefault.BackupPath)
	}

	// 2. When user specifies a custom BackupPath
	customBackupPath := filepath.Join(tempDir, "custom-backups")
	inputsCustom := tui.WizardInputs{
		UsedHerd:             true,
		WebsitesPath:         filepath.Join(tempDir, "sites"),
		BackupPath:           customBackupPath,
		DefaultAdminUsername: "admin",
		DefaultAdminPassword: "password123",
		DefaultAdminEmail:    "admin@example.com",
		DatabaseHost:         "localhost",
		DatabasePortStr:      "3306",
		DBUsername:           "root",
	}

	cfgCustom, err := tui.ConvertInputsToConfig(inputsCustom, tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfgCustom.BackupPath != customBackupPath {
		t.Errorf("expected custom backup path %q, got %q", customBackupPath, cfgCustom.BackupPath)
	}
}
