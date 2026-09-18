package restore

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ErrNoWordPressRoot         = errors.New("no WordPress root directory found in archive")
	ErrAmbiguousWordPressRoot  = errors.New("ambiguous WordPress root: multiple candidates found in archive")
	ErrMissingSQLDump          = errors.New("no SQL database dump found in archive")
	ErrUndeterminedTablePrefix = errors.New("unable to determine table prefix from wp-config.php or SQL dump")
)

type DumpPromptFunc func(candidates []string) (string, error)

// FindWordPressRoot scans extractedDir for the unique directory containing
// wp-content, wp-includes, and wp-settings.php.
func FindWordPressRoot(extractedDir string) (string, error) {
	var candidates []string

	err := filepath.Walk(extractedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}

		// Check for core markers in this directory
		wpContent := filepath.Join(path, "wp-content")
		wpIncludes := filepath.Join(path, "wp-includes")
		wpSettings := filepath.Join(path, "wp-settings.php")

		contentInfo, errContent := os.Stat(wpContent)
		includesInfo, errIncludes := os.Stat(wpIncludes)
		settingsInfo, errSettings := os.Stat(wpSettings)

		if errContent == nil && contentInfo.IsDir() &&
			errIncludes == nil && includesInfo.IsDir() &&
			errSettings == nil && !settingsInfo.IsDir() {
			candidates = append(candidates, path)
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error scanning extracted archive: %w", err)
	}

	if len(candidates) == 0 {
		return "", ErrNoWordPressRoot
	}
	if len(candidates) > 1 {
		return "", ErrAmbiguousWordPressRoot
	}

	return candidates[0], nil
}

var fullBackupRegex = regexp.MustCompile(`^full_([a-zA-Z0-9_\-]+)_\d{4}-\d{2}-\d{2}_\d{2}-\d{2}-\d{2}\.zip$`)

// ResolveSQLDump identifies the SQL database dump to import.
func ResolveSQLDump(archivePath, wpRoot string, prompt DumpPromptFunc) (string, error) {
	base := filepath.Base(archivePath)
	matches := fullBackupRegex.FindStringSubmatch(base)
	if len(matches) == 2 {
		expected := filepath.Join(wpRoot, matches[1]+".sql")
		if info, err := os.Stat(expected); err == nil && !info.IsDir() {
			return expected, nil
		}
		return "", fmt.Errorf("standard backup archive is missing expected database dump: %s", matches[1]+".sql")
	}

	entries, err := os.ReadDir(wpRoot)
	if err != nil {
		return "", fmt.Errorf("failed to read WordPress root directory: %w", err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".sql") {
			sqlFiles = append(sqlFiles, filepath.Join(wpRoot, entry.Name()))
		}
	}

	if len(sqlFiles) == 0 {
		return "", ErrMissingSQLDump
	}
	if len(sqlFiles) == 1 {
		return sqlFiles[0], nil
	}

	if prompt != nil {
		chosen, err := prompt(sqlFiles)
		if err != nil {
			return "", err
		}
		if chosen != "" {
			return chosen, nil
		}
	}

	return "", fmt.Errorf("multiple SQL dumps found and none selected")
}

var tablePrefixRegex = regexp.MustCompile(`\$table_prefix\s*=\s*['"]([^'"]+)['"]\s*;`)
var createTableRegex = regexp.MustCompile(`(?i)CREATE\s+TABLE(?:\s+IF\s+NOT\s+EXISTS)?\s+[` + "`" + `]?([a-zA-Z0-9_]+?)(?:users|posts|options|comments|terms|postmeta|usermeta)[` + "`" + `]?\s*[\(]`)

// ExtractTablePrefix determines the WordPress table prefix.
func ExtractTablePrefix(wpRoot, sqlDumpPath string) (string, error) {
	wpConfigPath := filepath.Join(wpRoot, "wp-config.php")
	if data, err := os.ReadFile(wpConfigPath); err == nil {
		matches := tablePrefixRegex.FindSubmatch(data)
		if len(matches) == 2 && len(matches[1]) > 0 {
			return string(matches[1]), nil
		}
	}

	if sqlDumpPath != "" {
		data, err := os.ReadFile(sqlDumpPath)
		if err == nil {
			allMatches := createTableRegex.FindAllSubmatch(data, -1)
			if len(allMatches) > 0 {
				counts := make(map[string]int)
				for _, m := range allMatches {
					if len(m) == 2 && len(m[1]) > 0 {
						counts[string(m[1])]++
					}
				}
				var bestPrefix string
				var maxCount int
				var tie bool
				for p, c := range counts {
					if c > maxCount {
						maxCount = c
						bestPrefix = p
						tie = false
					} else if c == maxCount {
						tie = true
					}
				}
				if !tie && bestPrefix != "" {
					return bestPrefix, nil
				}
			}
		}
	}

	return "", ErrUndeterminedTablePrefix
}

// ExtractZipArchive extracts a zip archive safely, preventing Zip Slip attacks.
func ExtractZipArchive(archivePath, destDir string) error {
	cleanDest := filepath.Clean(destDir)
	if err := os.MkdirAll(cleanDest, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip archive: %w", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		targetPath := filepath.Join(cleanDest, filepath.FromSlash(f.Name))
		cleanTarget := filepath.Clean(targetPath)
		if cleanTarget == cleanDest {
			continue
		}
		if !strings.HasPrefix(cleanTarget, cleanDest+string(filepath.Separator)) {
			return fmt.Errorf("illegal file path escaping destination: %s", f.Name)
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
			return fmt.Errorf("failed to open file inside zip %s: %w", f.Name, err)
		}

		outFile, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			_ = rc.Close()
			return fmt.Errorf("failed to create target file %s: %w", cleanTarget, err)
		}

		_, copyErr := io.Copy(outFile, rc)
		closeOut := outFile.Close()
		closeRc := rc.Close()

		if copyErr != nil {
			return fmt.Errorf("failed to write file %s: %w", cleanTarget, copyErr)
		}
		if closeOut != nil {
			return fmt.Errorf("failed to close file %s: %w", cleanTarget, closeOut)
		}
		if closeRc != nil {
			return fmt.Errorf("failed to close zip entry %s: %w", f.Name, closeRc)
		}
	}

	return nil
}
