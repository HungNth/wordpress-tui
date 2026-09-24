package launcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

type ProcessRunner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, name string, args ...string) error
	Start(ctx context.Context, name string, args ...string) error
}

type DefaultRunner struct{}

func (d *DefaultRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (d *DefaultRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (d *DefaultRunner) Start(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Start()
}

// OpenInVSCode launches "code --wait <filePath>", blocking until the user closes the editor.
// If the 'code' CLI is not found in PATH, it returns an explicit error without falling back.
func OpenInVSCode(ctx context.Context, filePath string, runner ProcessRunner) error {
	if runner == nil {
		runner = &DefaultRunner{}
	}

	if _, err := runner.LookPath("code"); err != nil {
		return errors.New("VS Code ('code' CLI) not found in PATH")
	}

	if err := runner.Run(ctx, "code", "--wait", filePath); err != nil {
		return fmt.Errorf("failed to run VS Code: %w", err)
	}

	return nil
}

// OpenDirectory opens dirPath in the native operating system file manager on Windows, macOS, or Linux.
// If dirPath does not exist, it returns an error instead of creating empty directories.
func OpenDirectory(ctx context.Context, dirPath string, runner ProcessRunner, targetOS ...string) error {
	if runner == nil {
		runner = &DefaultRunner{}
	}

	goos := runtime.GOOS
	if len(targetOS) > 0 && targetOS[0] != "" {
		goos = targetOS[0]
	}

	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cache directory does not exist yet: %s", dirPath)
		}
		return fmt.Errorf("failed to inspect directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", dirPath)
	}

	switch goos {
	case "windows":
		return runner.Start(ctx, "explorer.exe", dirPath)
	case "darwin":
		return runner.Start(ctx, "open", dirPath)
	case "linux":
		return runner.Start(ctx, "xdg-open", dirPath)
	default:
		return fmt.Errorf("unsupported operating system %q: only Windows, macOS, and Linux are supported", goos)
	}
}

// OpenURL opens targetURL in the default web browser.
func OpenURL(ctx context.Context, targetURL string, runner ProcessRunner, targetOS ...string) error {
	if runner == nil {
		runner = &DefaultRunner{}
	}

	goos := runtime.GOOS
	if len(targetOS) > 0 && targetOS[0] != "" {
		goos = targetOS[0]
	}

	switch goos {
	case "windows":
		return runner.Start(ctx, "cmd.exe", "/c", "start", targetURL)
	case "darwin":
		return runner.Start(ctx, "open", targetURL)
	default:
		return runner.Start(ctx, "xdg-open", targetURL)
	}
}

// OpenInEditorNonBlocking launches the code editor asynchronously without blocking.
func OpenInEditorNonBlocking(ctx context.Context, targetPath string, runner ProcessRunner) error {
	if runner == nil {
		runner = &DefaultRunner{}
	}

	if _, err := runner.LookPath("code"); err != nil {
		return errors.New("VS Code ('code' CLI) not found in PATH")
	}

	return runner.Start(ctx, "code", targetPath)
}
