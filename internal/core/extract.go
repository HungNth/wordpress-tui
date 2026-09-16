package core

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractCoreArchive extracts a content-stripped WordPress core ZIP into destDir,
// stripping the leading "wordpress/" directory prefix from entry paths and
// guaranteeing safe bounds against Zip Slip attacks. It automatically creates
// standard WordPress directories (wp-content, wp-content/themes, wp-content/plugins)
// with 0755 permissions.
func ExtractCoreArchive(archivePath, destDir string) error {
	cleanDest := filepath.Clean(destDir)
	if err := os.MkdirAll(cleanDest, 0755); err != nil {
		return fmt.Errorf("failed to prepare destination directory: %w", err)
	}

	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open core archive: %w", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		name := f.Name
		if name == "wordpress" || name == "wordpress/" {
			continue
		}

		relPath := strings.TrimPrefix(name, "wordpress/")
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath == "" {
			continue
		}

		// Security: Prevent Zip Slip
		targetPath := filepath.Join(cleanDest, filepath.FromSlash(relPath))
		cleanTarget := filepath.Clean(targetPath)
		if cleanTarget == cleanDest {
			continue
		}
		if !strings.HasPrefix(cleanTarget, cleanDest+string(filepath.Separator)) {
			return fmt.Errorf("illegal file path escaping destination: %s", name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTarget, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", cleanTarget, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", cleanTarget, err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open archive entry %s: %w", name, err)
		}

		mode := f.Mode() & 0777
		if mode < 0644 {
			mode = 0644
		}

		outFile, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			_ = rc.Close()
			return fmt.Errorf("failed to create target file %s: %w", cleanTarget, err)
		}

		if _, err := io.Copy(outFile, rc); err != nil {
			_ = outFile.Close()
			_ = rc.Close()
			return fmt.Errorf("failed to extract file %s: %w", cleanTarget, err)
		}

		_ = outFile.Close()
		_ = rc.Close()
	}

	// Scaffold standard WordPress directories
	standardDirs := []string{
		filepath.Join(cleanDest, "wp-content"),
		filepath.Join(cleanDest, "wp-content", "themes"),
		filepath.Join(cleanDest, "wp-content", "plugins"),
	}
	for _, d := range standardDirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed to scaffold standard directory %s: %w", d, err)
		}
	}

	return nil
}
