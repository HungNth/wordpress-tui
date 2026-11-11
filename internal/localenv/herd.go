package localenv

import (
	"context"
	"fmt"
	"strings"
)

type herdEnv struct {
	cmd string
}

func (h *herdEnv) Name() string {
	return "herd"
}

func (h *herdEnv) Command() string {
	return h.cmd
}

func (h *herdEnv) CheckDependencies(ctx context.Context, runner Runner) error {
	return checkCommonDependencies(runner, h.cmd)
}

func (h *herdEnv) SiteURL(slug string) string {
	return fmt.Sprintf("https://%s.test", slug)
}

func (h *herdEnv) Register(ctx context.Context, runner Runner, siteDir, slug string) (bool, error) {
	// Herd automatically discovers and serves subdirectories under parked paths.
	return false, nil
}

func (h *herdEnv) Secure(ctx context.Context, runner Runner, siteDir, slug string) error {
	_, stderr, err := runner.Run(ctx, siteDir, h.cmd, []string{"secure", slug}, "")
	if err != nil {
		return fmt.Errorf("herd secure failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return nil
}

func (h *herdEnv) Unsecure(ctx context.Context, runner Runner, siteDir, slug string) error {
	_, stderr, err := runner.Run(ctx, siteDir, h.cmd, []string{"unsecure", slug}, "")
	if err != nil {
		return fmt.Errorf("herd unsecure failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return nil
}

func (h *herdEnv) Unregister(ctx context.Context, runner Runner, siteDir, slug string) error {
	return nil
}

func (h *herdEnv) IsSiteSecured(ctx context.Context, runner Runner, slug string) (bool, error) {
	stdout, stderr, err := runner.Run(ctx, "", h.cmd, []string{"secured"}, "")
	if err != nil {
		return false, fmt.Errorf("failed to check herd secured sites: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	targetDomain := strings.ToLower(slug + ".test")
	lines := strings.Split(stdout, "\n")
	for _, l := range lines {
		trimmed := strings.ToLower(strings.TrimSpace(l))
		trimmed = strings.Trim(trimmed, "| \t\r")
		trimmed = strings.TrimSpace(trimmed)
		if strings.HasPrefix(trimmed, "+-") || strings.HasPrefix(trimmed, "site") {
			continue
		}
		parts := strings.Fields(trimmed)
		for _, part := range parts {
			if part == targetDomain {
				return true, nil
			}
		}
	}
	return false, nil
}
