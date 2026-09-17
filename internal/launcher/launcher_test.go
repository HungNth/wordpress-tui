package launcher_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wptui/internal/launcher"
)

type mockRunner struct {
	lookPathFn func(file string) (string, error)
	runFn      func(ctx context.Context, name string, args ...string) error
	startFn    func(ctx context.Context, name string, args ...string) error
	calls      []string
}

func (m *mockRunner) LookPath(file string) (string, error) {
	if m.lookPathFn != nil {
		return m.lookPathFn(file)
	}
	return "/usr/bin/" + file, nil
}

func (m *mockRunner) Run(ctx context.Context, name string, args ...string) error {
	m.calls = append(m.calls, name+" "+strings.Join(args, " "))
	if m.runFn != nil {
		return m.runFn(ctx, name, args...)
	}
	return nil
}

func (m *mockRunner) Start(ctx context.Context, name string, args ...string) error {
	m.calls = append(m.calls, "start:"+name+" "+strings.Join(args, " "))
	if m.startFn != nil {
		return m.startFn(ctx, name, args...)
	}
	return nil
}

func TestOpenInVSCode_Success(t *testing.T) {
	mock := &mockRunner{}
	ctx := context.Background()

	err := launcher.OpenInVSCode(ctx, "/path/to/config.json", mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.calls) != 1 || mock.calls[0] != "code --wait /path/to/config.json" {
		t.Errorf("expected 'code --wait /path/to/config.json', got: %v", mock.calls)
	}
}

func TestOpenInVSCode_NotFound(t *testing.T) {
	mock := &mockRunner{
		lookPathFn: func(file string) (string, error) {
			return "", errors.New("binary not found")
		},
	}
	ctx := context.Background()

	err := launcher.OpenInVSCode(ctx, "/path/to/config.json", mock)
	if err == nil {
		t.Fatal("expected error when code CLI is missing, got nil")
	}

	if !strings.Contains(err.Error(), "VS Code ('code' CLI) not found in PATH") {
		t.Errorf("expected 'VS Code ('code' CLI) not found in PATH', got: %v", err)
	}
	if len(mock.calls) != 0 {
		t.Errorf("expected no commands to be executed on missing binary, got: %v", mock.calls)
	}
}

func TestOpenInVSCode_RunFails(t *testing.T) {
	mock := &mockRunner{
		runFn: func(ctx context.Context, name string, args ...string) error {
			return errors.New("exit status 1")
		},
	}
	ctx := context.Background()

	err := launcher.OpenInVSCode(ctx, "/path/to/config.json", mock)
	if err == nil {
		t.Fatal("expected error when editor process exits non-zero, got nil")
	}
	if !strings.Contains(err.Error(), "failed to run VS Code") {
		t.Errorf("expected 'failed to run VS Code' error, got: %v", err)
	}
	if len(mock.calls) != 1 || mock.calls[0] != "code --wait /path/to/config.json" {
		t.Errorf("expected exactly one 'code --wait' invocation, got: %v", mock.calls)
	}
}

func TestOpenDirectory_NotADirectory(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	err := launcher.OpenDirectory(context.Background(), filePath, &mockRunner{}, "windows")
	if err == nil {
		t.Fatal("expected error when path is not a directory, got nil")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("expected 'not a directory' error, got: %v", err)
	}
}

func TestOpenDirectory_Windows(t *testing.T) {
	tempDir := t.TempDir()
	mock := &mockRunner{}
	ctx := context.Background()

	err := launcher.OpenDirectory(ctx, tempDir, mock, "windows")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.calls) != 1 || mock.calls[0] != "start:explorer.exe "+tempDir {
		t.Errorf("expected 'start:explorer.exe %s', got: %v", tempDir, mock.calls)
	}
}

func TestOpenDirectory_MacOS(t *testing.T) {
	tempDir := t.TempDir()
	mock := &mockRunner{}
	ctx := context.Background()

	err := launcher.OpenDirectory(ctx, tempDir, mock, "darwin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.calls) != 1 || mock.calls[0] != "start:open "+tempDir {
		t.Errorf("expected 'start:open %s', got: %v", tempDir, mock.calls)
	}
}

func TestOpenDirectory_UnsupportedOS(t *testing.T) {
	tempDir := t.TempDir()
	mock := &mockRunner{}
	ctx := context.Background()

	err := launcher.OpenDirectory(ctx, tempDir, mock, "linux")
	if err == nil {
		t.Fatal("expected error on unsupported OS, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported operating system") {
		t.Errorf("expected unsupported OS error, got: %v", err)
	}
}

func TestOpenDirectory_NotExist(t *testing.T) {
	mock := &mockRunner{}
	ctx := context.Background()
	nonExistentPath := filepath.Join(t.TempDir(), "does-not-exist")

	err := launcher.OpenDirectory(ctx, nonExistentPath, mock, "windows")
	if err == nil {
		t.Fatal("expected error on non-existent directory, got nil")
	}

	if !strings.Contains(err.Error(), "cache directory does not exist yet") {
		t.Errorf("expected 'cache directory does not exist yet' error, got: %v", err)
	}
}
