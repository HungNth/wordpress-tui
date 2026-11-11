package localenv

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type laragonEnv struct {
	cmd string
}

func (l *laragonEnv) Name() string {
	return "laragon"
}

func (l *laragonEnv) Command() string {
	return l.cmd
}

func (l *laragonEnv) CheckDependencies(ctx context.Context, runner Runner) error {
	return checkCommonDependencies(runner, l.cmd)
}

func (l *laragonEnv) SiteURL(slug string) string {
	return fmt.Sprintf("https://%s.test", slug)
}

func (l *laragonEnv) Register(ctx context.Context, runner Runner, siteDir, slug string) (bool, error) {
	// Laragon automatically generates virtual hosts for directories in Document Root when reloaded.
	_, stderr, err := runner.Run(ctx, siteDir, l.cmd, []string{"reload"}, "")
	if err != nil {
		return true, fmt.Errorf("laragon reload failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return true, nil
}

func (l *laragonEnv) Secure(ctx context.Context, runner Runner, siteDir, slug string) error {
	// Auto SSL is managed by Laragon externally.
	return nil
}

func (l *laragonEnv) Unsecure(ctx context.Context, runner Runner, siteDir, slug string) error {
	// Auto SSL is managed by Laragon externally.
	return nil
}

func (l *laragonEnv) Unregister(ctx context.Context, runner Runner, siteDir, slug string) error {
	dir := filepath.Dir(siteDir)
	_, stderr, err := runner.Run(ctx, dir, l.cmd, []string{"reload"}, "")
	if err != nil {
		return fmt.Errorf("laragon reload failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return nil
}

func (l *laragonEnv) IsSiteSecured(ctx context.Context, runner Runner, slug string) (bool, error) {
	// Laragon sites with Auto SSL enabled are accessible via HTTPS.
	return true, nil
}
