package backup

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type WPClient interface {
	Run(ctx context.Context, dir, name string, args []string, stdin string) (stdout string, stderr string, err error)
}

type ProgressFunc func(step, total int, message string)

type BackupResult struct {
	Strategy BackupStrategy
	FilePath string
	FileSize int64
	Duration time.Duration
}

type BackupStrategy string

const (
	StrategyFull  BackupStrategy = "full"
	StrategyAI1WM BackupStrategy = "ai1wm"
)

// ExportDatabase runs "wp db export <slug>.sql" in siteDir.
func ExportDatabase(ctx context.Context, siteDir, slug string, client WPClient) (string, error) {
	sqlFile := slug + ".sql"
	sqlPath := filepath.Join(siteDir, sqlFile)

	args := []string{"db", "export", sqlFile}
	_, stderr, err := client.Run(ctx, siteDir, "wp", args, "")
	if err != nil {
		return "", fmt.Errorf("wp db export failed: %w (%s)", err, strings.TrimSpace(stderr))
	}

	if _, err := os.Stat(sqlPath); err != nil {
		return "", fmt.Errorf("exported database file not found at %s: %w", sqlPath, err)
	}

	return sqlPath, nil
}

// shouldExclude checks whether a relative or base path matches the exclusion rules.
func shouldExclude(relPath, baseName string, excludes []string) bool {
	// Exclude existing backup archives
	lower := strings.ToLower(baseName)
	if strings.HasSuffix(lower, ".zip") || strings.HasSuffix(lower, ".wpress") {
		return true
	}

	normalizedRel := filepath.ToSlash(relPath)
	parts := strings.Split(normalizedRel, "/")

	for _, ex := range excludes {
		cleanEx := strings.Trim(filepath.ToSlash(ex), "/")
		if cleanEx == "" {
			continue
		}
		if baseName == cleanEx || normalizedRel == cleanEx || strings.HasPrefix(normalizedRel, cleanEx+"/") {
			return true
		}
		for _, part := range parts {
			if part == cleanEx {
				return true
			}
		}
	}
	return false
}

// CreateFullZipArchive creates a .zip archive of siteDir at destZipPath.
// Entry headers inside the zip are normalized with clean forward slashes (/) for cross-platform portability.
func CreateFullZipArchive(ctx context.Context, siteDir, destZipPath, backupPath string, excludes []string, onProgress ProgressFunc) error {
	zipFile, err := os.Create(destZipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file at %s: %w", destZipPath, err)
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	destAbs, _ := filepath.Abs(destZipPath)
	backupAbs, _ := filepath.Abs(backupPath)

	err = filepath.Walk(siteDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		pathAbs, _ := filepath.Abs(path)
		// Skip the temporary archive itself and the target backup directory if nested inside siteDir
		if pathAbs == destAbs || (backupPath != "" && (pathAbs == backupAbs || strings.HasPrefix(pathAbs, backupAbs+string(filepath.Separator)))) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(siteDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		baseName := info.Name()
		if shouldExclude(relPath, baseName, excludes) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Forward-slash path normalization for zip header
		zipEntryName := filepath.ToSlash(relPath)
		if info.IsDir() {
			zipEntryName += "/"
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("failed to create zip header for %s: %w", relPath, err)
		}
		header.Name = zipEntryName
		header.Method = zip.Deflate

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("failed to create archive entry for %s: %w", relPath, err)
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open source file %s: %w", path, err)
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		return err
	}

	return archive.Close()
}

// safeRelocate moves a file from src to dest. If os.Rename fails across different drives/volumes,
// it falls back to copy-then-remove.
type RelocatorFunc func(src, dest string) error

var defaultRelocator RelocatorFunc = safeRelocate

// SetRelocatorForTesting allows unit tests to simulate relocation failures.
func SetRelocatorForTesting(fn RelocatorFunc) func() {
	orig := defaultRelocator
	defaultRelocator = fn
	return func() {
		defaultRelocator = orig
	}
}

func safeRelocate(src, dest string) error {
	if err := os.Rename(src, dest); err == nil {
		return nil
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		_ = os.Remove(dest)
		return err
	}
	_ = destFile.Close()
	_ = srcFile.Close()
	_ = os.Remove(src)
	return nil
}

// RunFullBackup executes Strategy 1:
// 1. Export DB to <slug>.sql
// 2. Create zip archive in temporary directory outside siteDir
// 3. Move zip to backupPath under full_<slug>_YYYY-MM-DD_HH-mm-ss.zip
// 4. Delete <slug>.sql only if steps 2 and 3 succeed.
func RunFullBackup(ctx context.Context, siteDir, slug, backupPath string, excludes []string, client WPClient, onProgress ProgressFunc) (*BackupResult, error) {
	start := time.Now()

	if onProgress != nil {
		onProgress(1, 4, fmt.Sprintf("Exporting database to %s.sql...", slug))
	}

	sqlPath, err := ExportDatabase(ctx, siteDir, slug, client)
	if err != nil {
		return nil, err
	}

	// Create temp dir outside siteDir
	tempDir, err := os.MkdirTemp("", "wptui-full-backup-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary workspace: %w", err)
	}
	defer os.RemoveAll(tempDir)

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	archiveName := fmt.Sprintf("full_%s_%s.zip", slug, timestamp)
	tempZipPath := filepath.Join(tempDir, archiveName)

	if onProgress != nil {
		onProgress(2, 4, "Packaging source code and database into zip archive...")
	}

	if err := CreateFullZipArchive(ctx, siteDir, tempZipPath, backupPath, excludes, onProgress); err != nil {
		// Strict invariant: keep <slug>.sql on disk when zip creation fails
		return nil, fmt.Errorf("failed to create zip archive: %w (database dump retained at %s)", err, sqlPath)
	}

	if onProgress != nil {
		onProgress(3, 4, "Relocating archive to backup storage...")
	}

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w (database dump retained at %s)", err, sqlPath)
	}

	finalDestPath := filepath.Join(backupPath, archiveName)
	if err := defaultRelocator(tempZipPath, finalDestPath); err != nil {
		return nil, fmt.Errorf("failed to move archive to %s: %w (database dump retained at %s)", finalDestPath, err, sqlPath)
	}

	// Step 4: Clean up <slug>.sql strictly after successful archive and move
	if onProgress != nil {
		onProgress(4, 4, "Cleaning up temporary database dump...")
	}
	_ = os.Remove(sqlPath)

	info, err := os.Stat(finalDestPath)
	var fileSize int64
	if err == nil {
		fileSize = info.Size()
	}

	return &BackupResult{
		Strategy: StrategyFull,
		FilePath: finalDestPath,
		FileSize: fileSize,
		Duration: time.Since(start),
	}, nil
}
