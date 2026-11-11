package app_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"wptui/internal/app"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/packages"
	"wptui/internal/tui"
	"wptui/internal/wpcli"
)

type environmentCommand struct {
	dir, name string
	args      []string
}
type environmentRunner struct {
	commands        []environmentCommand
	env, fail       string
	path            string
	linked, secured bool
	database        bool
}

func (r *environmentRunner) LookPath(name string) (string, error) { return name, nil }
func (r *environmentRunner) Run(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
	if dir != "" {
		if _, err := os.Stat(dir); err != nil {
			return "", "working directory does not exist", err
		}
	}
	r.commands = append(r.commands, environmentCommand{dir, name, slices.Clone(args)})
	op := strings.Join(args, " ")
	if name == r.env {
		if op == "sites" {
			output := "SITE  PHP  URL  ROOT  PATH\n"
			if r.linked {
				scheme := "http"
				if r.secured {
					scheme = "https"
				}
				output += fmt.Sprintf("demo  8.4  %s://demo.test  .  %s\n", scheme, r.path)
			}
			return output, "", nil
		}
		if op == "secured" {
			if r.secured {
				return "demo.test\n", "", nil
			}
			return "", "", nil
		}
		if strings.HasPrefix(op, "link ") {
			r.path = dir
			r.linked = true
		}
		if op == r.fail {
			return "", "injected environment failure", errors.New("injected environment failure")
		}
		if strings.HasPrefix(op, "secure ") {
			r.secured = true
		}
		if strings.HasPrefix(op, "unsecure ") {
			r.secured = false
		}
		if strings.HasPrefix(op, "unlink ") {
			r.linked = false
		}
	}
	if name == "wp" {
		switch op {
		case "db create":
			r.database = true
			return "Database created", "", nil
		case "db drop --yes":
			r.database = false
		case "config get DB_NAME":
			return "demo_database", "", nil
		case "theme list --status=active --field=name":
			return "", "", errors.New("no active theme")
		}
	}
	return "", "", nil
}

type environmentCore struct{}

func (environmentCore) Resolve(context.Context) (string, string, error) {
	return "fixture-core.zip", "7.1", nil
}

func provisionEnvironment(t *testing.T, cfg *config.Config, runner *environmentRunner) error {
	t.Helper()
	return app.RunCreateFlowWithDeps(t.Context(), cfg, app.CreateFlowDependencies{
		Runner: runner, CoreResolver: environmentCore{},
		CoreExtractor: func(_ string, dir string) error {
			return os.WriteFile(filepath.Join(dir, "wp-config.php"), []byte("<?php"), 0600)
		},
		PromptCreate: func(*config.Config, ...tui.SlugAvailabilityChecker) (*tui.CreateInputs, error) {
			return &tui.CreateInputs{WebsiteName: "Demo", WebsiteSlug: "demo"}, nil
		},
		PromptPackages: func(context.Context, *config.Config, []packages.CatalogItem) ([]string, []string, error) {
			t.Error("Package picker shown with disabled API")
			return nil, nil, nil
		},
	})
}

func TestEnvironmentCreateDeleteComposed(t *testing.T) {
	for _, name := range []string{"herd", "flock", "laragon"} {
		t.Run(name, func(t *testing.T) {
			cfg := config.DefaultConfig(t.TempDir())
			cfg.AppName = name
			cfg.AppCommand = name
			runner := &environmentRunner{env: name}
			if err := provisionEnvironment(t, cfg, runner); err != nil {
				t.Fatal(err)
			}
			siteDir := filepath.Join(cfg.WebsitesPath, "demo")
			if _, err := os.Stat(siteDir); err != nil {
				t.Fatalf("Website missing: %v", err)
			}
			if !runner.database {
				t.Fatal("Website database missing")
			}
			if name == "flock" && !runner.linked {
				t.Fatal("Flock Website not linked")
			}
			if name != "laragon" && !runner.secured {
				t.Fatal("Website not secured")
			}
			client := wpcli.NewClientWithRunnerForConfig(runner, cfg)
			err := app.RunDeleteFlowWithDeps(t.Context(), cfg, app.DeleteFlowDependencies{
				WPClient: client,
				Select:   func(c []deprovision.Candidate) ([]deprovision.Candidate, error) { return c, nil },
				Confirm: func(c []deprovision.Candidate) (bool, error) {
					if len(c) != 1 || c[0].DetectedDB != "demo_database" {
						t.Fatalf("unsafe deletion preview: %+v", c)
					}
					return true, nil
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(siteDir); !os.IsNotExist(err) {
				t.Fatalf("Website directory retained: %v", err)
			}
			if runner.database {
				t.Fatal("database retained")
			}
			if name == "flock" && (runner.linked || runner.secured) {
				t.Fatal("Flock resources retained")
			}
			if name == "laragon" {
				reloads := 0
				for _, cmd := range runner.commands {
					if cmd.name == name && slices.Equal(cmd.args, []string{"reload"}) {
						reloads++
					}
				}
				if reloads < 2 {
					t.Fatal("Laragon not reloaded after deletion")
				}
				last := runner.commands[len(runner.commands)-1]
				if last.name != name || last.dir != cfg.WebsitesPath {
					t.Fatalf("post-delete reload must use surviving Websites directory: %+v", last)
				}
			}
		})
	}
}

func TestEnvironmentFailureRollsBackComposed(t *testing.T) {
	for _, tc := range []struct{ name, fail string }{{"flock", "secure demo"}, {"laragon", "reload"}} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.DefaultConfig(t.TempDir())
			cfg.AppName = tc.name
			cfg.AppCommand = tc.name
			runner := &environmentRunner{env: tc.name, fail: tc.fail}
			if err := provisionEnvironment(t, cfg, runner); err == nil {
				t.Fatal("critical environment failure reported as success")
			}
			siteDir := filepath.Join(cfg.WebsitesPath, "demo")
			if _, err := os.Stat(siteDir); !os.IsNotExist(err) {
				t.Fatalf("rollback retained directory: %v", err)
			}
			if runner.database {
				t.Fatal("rollback retained database")
			}
			if tc.name == "flock" && runner.linked {
				t.Fatal("rollback retained newly created Flock registration")
			}
		})
	}
}
