package localenv

import (
	"context"
	"fmt"
	"strings"
	"wptui/internal/config"
)

// Runner represents the command execution dependency.
type Runner interface {
	Run(ctx context.Context, dir string, name string, args []string, stdin string) (stdout string, stderr string, err error)
	LookPath(file string) (string, error)
}

// Environment defines the contract for local development environments.
type Environment interface {
	Name() string
	Command() string
	CheckDependencies(ctx context.Context, runner Runner) error
	SiteURL(slug string) string
	Register(ctx context.Context, runner Runner, siteDir, slug string) (bool, error)
	Secure(ctx context.Context, runner Runner, siteDir, slug string) error
	Unsecure(ctx context.Context, runner Runner, siteDir, slug string) error
	Unregister(ctx context.Context, runner Runner, siteDir, slug string) error
	IsSiteSecured(ctx context.Context, runner Runner, slug string) (bool, error)
}

// New creates a new Environment instance for the given app name and command.
// Rejects empty command and nonexact enums without defaulting.
func New(appName, appCommand string) (Environment, error) {
	if strings.TrimSpace(appCommand) == "" {
		return nil, fmt.Errorf("app command cannot be empty")
	}
	switch appName {
	case config.AppNameHerd:
		return &herdEnv{cmd: appCommand}, nil
	case config.AppNameFlock:
		return &flockEnv{cmd: appCommand}, nil
	case config.AppNameLaragon:
		return &laragonEnv{cmd: appCommand}, nil
	default:
		return nil, fmt.Errorf("unsupported local development environment %q", appName)
	}
}

// NewFromConfig returns the configured Environment.
func NewFromConfig(cfg *config.Config) (Environment, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	return New(cfg.AppName, cfg.AppCommand)
}

// SiteURL returns the canonical site URL for a given slug and config.
func SiteURL(cfg *config.Config, slug string) string {
	if cfg == nil {
		return ""
	}
	switch cfg.AppName {
	case config.AppNameHerd, config.AppNameFlock, config.AppNameLaragon:
		return fmt.Sprintf("https://%s.test", slug)
	default:
		return ""
	}
}

func checkCommonDependencies(runner Runner, appCmd string) error {
	required := []string{"php", "wp", "mysql"}
	if appCmd != "" {
		required = append(required, appCmd)
	}

	var missing []string
	for _, bin := range required {
		if _, err := runner.LookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required executable(s) in PATH: %s", strings.Join(missing, ", "))
	}
	return nil
}
