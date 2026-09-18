package backup_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wptui/internal/backup"
	"wptui/internal/packages"
)

type mockAI1WMResolver struct {
	resolveFn func(ctx context.Context, ref packages.PackageRef, stageDir string) (*packages.Artifact, error)
}

func (m *mockAI1WMResolver) ResolvePackage(ctx context.Context, ref packages.PackageRef, stageDir string) (*packages.Artifact, error) {
	if m.resolveFn != nil {
		return m.resolveFn(ctx, ref, stageDir)
	}
	return &packages.Artifact{Ref: ref, Path: filepath.Join(stageDir, ref.Slug+".zip")}, nil
}

func (m *mockAI1WMResolver) ResolveAll(ctx context.Context, refs []packages.PackageRef, stageDir string) ([]packages.Artifact, error) {
	arts := make([]packages.Artifact, 0, len(refs))
	for _, r := range refs {
		art, err := m.ResolvePackage(ctx, r, stageDir)
		if err != nil {
			return nil, err
		}
		arts = append(arts, *art)
	}
	return arts, nil
}

type mockAI1WMWPClient struct {
	runs        []string
	pluginVerFn func(slug string) (string, bool)
	runAI1WMFn  func() (string, error)
}

func (m *mockAI1WMWPClient) ConfigSet(ctx context.Context, dir, key, value string, raw bool) error {
	return nil
}
func (m *mockAI1WMWPClient) RewriteStructure(ctx context.Context, dir, value string) error {
	return nil
}
func (m *mockAI1WMWPClient) OptionUpdate(ctx context.Context, dir, key, value string) error {
	return nil
}
func (m *mockAI1WMWPClient) LanguageCore(ctx context.Context, dir, action, value string) error {
	return nil
}
func (m *mockAI1WMWPClient) PluginInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error {
	m.runs = append(m.runs, fmt.Sprintf("plugin install %s activate:%v", pathOrSlug, activate))
	return nil
}
func (m *mockAI1WMWPClient) ThemeInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error {
	return nil
}

func (m *mockAI1WMWPClient) Run(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
	call := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
	m.runs = append(m.runs, call)

	if len(args) >= 3 && args[0] == "plugin" && args[1] == "get" {
		if m.pluginVerFn != nil {
			ver, ok := m.pluginVerFn(args[2])
			if ok {
				return ver, "", nil
			}
			return "", "Error: not found", fmt.Errorf("not found")
		}
		return "1.0.0", "", nil
	}

	if len(args) >= 2 && args[0] == "ai1wm" && args[1] == "backup" {
		if m.runAI1WMFn != nil {
			stdout, err := m.runAI1WMFn()
			return stdout, "", err
		}
		return "Backup in progress...\nSuccess: Backup complete.\nBackup location: dummy.wpress", "", nil
	}

	return "", "", nil
}

func TestFormatAI1WMExcludes(t *testing.T) {
	excludes := []string{".idea", ".vscode", "node_modules", "wp-content/cache/"}
	got := backup.FormatAI1WMExcludes(excludes)
	expected := ".idea,.vscode,node_modules,wp-content/cache"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestParseBackupLocation(t *testing.T) {
	sampleOutput := `wp ai1wm backup --exclude-cache --exclude-files=node_modules
Backup in progress...
Success: Backup complete.
Backup file: flatsome-test-20260918-091210-d94gl7ofzrzz.wpress
Backup location: F:/laravel-herd/wordpress/flatsome/wp-content/ai1wm-backups/flatsome-test-20260918-091210-d94gl7ofzrzz.wpress`

	loc, err := backup.ParseBackupLocation(sampleOutput)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if !strings.Contains(filepath.ToSlash(loc), "flatsome-test-20260918-091210-d94gl7ofzrzz.wpress") {
		t.Errorf("unexpected parsed location: %q", loc)
	}
}

func TestRunAI1WMBackup_Success(t *testing.T) {
	siteDir := t.TempDir()
	slug := "ai1wm-site"
	backupDir := filepath.Join(t.TempDir(), "backups")

	// Create a dummy wpress file in siteDir
	wpressDir := filepath.Join(siteDir, "wp-content", "ai1wm-backups")
	_ = os.MkdirAll(wpressDir, 0755)
	sourceWpress := filepath.Join(wpressDir, "generated.wpress")
	_ = os.WriteFile(sourceWpress, []byte("wpress archive content"), 0644)

	mockCli := &mockAI1WMWPClient{
		runAI1WMFn: func() (string, error) {
			return fmt.Sprintf("Backup in progress...\nSuccess: Backup complete.\nBackup location: %s", sourceWpress), nil
		},
	}
	mockRes := &mockAI1WMResolver{}

	excludes := []string{"node_modules", ".idea"}
	res, err := backup.RunAI1WMBackup(context.Background(), siteDir, slug, backupDir, excludes, mockRes, mockCli, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res == nil || res.Strategy != backup.StrategyAI1WM {
		t.Fatalf("expected StrategyAI1WM, got: %+v", res)
	}

	// 1. Verify destination artifact exists in backupDir
	if _, err := os.Stat(res.FilePath); err != nil {
		t.Errorf("expected backup file at %s: %v", res.FilePath, err)
	}

	// 2. Verify destination filename starts with ai1wm_<slug>_ and ends with .wpress
	filename := filepath.Base(res.FilePath)
	if !strings.HasPrefix(filename, "ai1wm_"+slug+"_") || !strings.HasSuffix(filename, ".wpress") {
		t.Errorf("expected filename pattern 'ai1wm_%s_*.wpress', got: %s", slug, filename)
	}

	// 3. Verify command was called with --exclude-cache and --exclude-files
	foundAI1WMCall := false
	for _, run := range mockCli.runs {
		if strings.Contains(run, "wp ai1wm backup") {
			foundAI1WMCall = true
			if !strings.Contains(run, "--exclude-cache") {
				t.Errorf("expected --exclude-cache flag in %s", run)
			}
			if !strings.Contains(run, "--exclude-files=node_modules,.idea") {
				t.Errorf("expected --exclude-files flag in %s", run)
			}
		}
	}
	if !foundAI1WMCall {
		t.Errorf("expected wp ai1wm backup command to be executed, calls: %v", mockCli.runs)
	}
}
