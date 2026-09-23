package backup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wptui/internal/packages"
	"wptui/internal/siteconfig"
)

const AI1WMExtensionSlug = "all-in-one-wp-migration-unlimited-extension"

type PackageResolver interface {
	ResolvePackage(ctx context.Context, ref packages.PackageRef, stageDir string) (*packages.Artifact, error)
	ResolveAll(ctx context.Context, refs []packages.PackageRef, stageDir string) ([]packages.Artifact, error)
}

// EnsureAI1WMExtension checks whether all-in-one-wp-migration-unlimited-extension is installed and up-to-date.
// It installs or updates it via the version-aware package installer if needed.
func EnsureAI1WMExtension(ctx context.Context, siteDir string, resolver PackageResolver, client siteconfig.WPClient, onProgress ProgressFunc) error {
	if resolver == nil {
		if _, installed := siteconfig.GetInstalledVersion(ctx, siteDir, packages.PackageTypePlugin, AI1WMExtensionSlug, client); installed {
			return nil
		}
		return errors.New("package resolver is not configured (Packages API URL missing) and all-in-one-wp-migration-unlimited-extension is not installed")
	}

	stageDir, err := os.MkdirTemp("", "wptui-ai1wm-stage-*")
	if err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	defer os.RemoveAll(stageDir)

	ref := packages.PackageRef{Slug: AI1WMExtensionSlug, Type: packages.PackageTypePlugin}
	artifact, err := resolver.ResolvePackage(ctx, ref, stageDir)
	if err != nil {
		return fmt.Errorf("failed to resolve %s: %w", AI1WMExtensionSlug, err)
	}

	results := siteconfig.InstallPackages(ctx, siteDir, packages.PackageTypePlugin, []packages.Artifact{*artifact}, true, client, siteconfig.ProgressFunc(onProgress))
	if len(results) == 0 || (!results[0].Success && !results[0].Skipped) {
		if len(results) > 0 && results[0].Err != nil {
			return fmt.Errorf("failed to install %s: %w", AI1WMExtensionSlug, results[0].Err)
		}
		return fmt.Errorf("failed to install %s", AI1WMExtensionSlug)
	}

	return nil
}

// ParseBackupLocation parses WP-CLI ai1wm output to extract the generated .wpress path.
func ParseBackupLocation(output string) (string, error) {
	lines := strings.Split(output, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "Backup location:") {
			pathVal := strings.TrimSpace(strings.TrimPrefix(trimmed, "Backup location:"))
			if pathVal != "" {
				return filepath.Clean(pathVal), nil
			}
		}
	}
	return "", fmt.Errorf("failed to locate 'Backup location:' in WP-CLI output: %s", output)
}

// FormatAI1WMExcludes formats backup_excludes into a comma-separated string for --exclude-files.
func FormatAI1WMExcludes(excludes []string) string {
	clean := make([]string, 0, len(excludes))
	for _, ex := range excludes {
		t := strings.Trim(filepath.ToSlash(ex), "/")
		if t != "" {
			clean = append(clean, t)
		}
	}
	return strings.Join(clean, ",")
}

// RunAI1WMBackup executes Strategy 2:
// 1. Ensure extension plugin is installed
// 2. Run wp ai1wm backup --exclude-cache --exclude-files=<excludes>
// 3. Move .wpress artifact to backupPath under ai1wm_<slug>_YYYY-MM-DD_HH-mm-ss.wpress
// 4. Retain plugin on website.
func RunAI1WMBackup(ctx context.Context, siteDir, slug, backupPath string, excludes []string, resolver PackageResolver, client siteconfig.WPClient, onProgress ProgressFunc) (*BackupResult, error) {
	start := time.Now()

	if onProgress != nil {
		onProgress(1, 3, fmt.Sprintf("Verifying %s...", AI1WMExtensionSlug))
	}

	if err := EnsureAI1WMExtension(ctx, siteDir, resolver, client, onProgress); err != nil {
		return nil, err
	}

	if onProgress != nil {
		onProgress(2, 3, "Generating All-in-One WP Migration backup archive...")
	}

	args := []string{"ai1wm", "backup", "--exclude-cache"}
	if len(excludes) > 0 {
		excludeStr := FormatAI1WMExcludes(excludes)
		if excludeStr != "" {
			args = append(args, "--exclude-files="+excludeStr)
		}
	}

	stdout, stderr, err := client.Run(ctx, siteDir, "wp", args, "")
	if err != nil {
		return nil, fmt.Errorf("wp ai1wm backup failed: %w (%s)", err, strings.TrimSpace(stderr))
	}

	sourceWpressPath, err := ParseBackupLocation(stdout)
	if err != nil {
		return nil, err
	}

	if onProgress != nil {
		onProgress(3, 3, "Relocating .wpress archive to backup storage...")
	}

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup storage directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	archiveName := fmt.Sprintf("ai1wm_%s_%s.wpress", slug, timestamp)
	destPath := filepath.Join(backupPath, archiveName)

	if err := RelocateFile(sourceWpressPath, destPath); err != nil {
		return nil, fmt.Errorf("failed to relocate .wpress backup from %s to %s: %w", sourceWpressPath, destPath, err)
	}

	info, err := os.Stat(destPath)
	var fileSize int64
	if err == nil {
		fileSize = info.Size()
	}

	return &BackupResult{
		Strategy: StrategyAI1WM,
		FilePath: destPath,
		FileSize: fileSize,
		Duration: time.Since(start),
	}, nil
}
