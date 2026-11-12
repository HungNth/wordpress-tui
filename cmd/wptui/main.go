package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"wptui"
	"wptui/internal/app"
	"wptui/internal/config"
	"wptui/internal/release"
)

type cliAction uint8

const (
	actionLaunchTUI cliAction = iota
	actionVersion
	actionUpdate
	actionConfig
	actionHelp
)

const usageText = `Usage:
  wptui [flag]

Action flags (choose at most one):
  -v, --version   Print the installed WPTUI release version
  -u, --update    Perform a read-only Application Update Check
  -c, --config    Print the expected config.json path
  -h, --help      Print command-line usage
`

type runDeps struct {
	manifestJSON      []byte
	remoteManifestURL string
	httpClient        *http.Client
	updateTimeout     time.Duration
	userHomeDir       func() (string, error)
	launchApp         func() error
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, runDeps{}))
}

func run(args []string, stdout, stderr io.Writer, deps runDeps) int {
	action, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n\n%s", err, usageText)
		return 2
	}

	manifestBytes := deps.manifestJSON
	if manifestBytes == nil {
		manifestBytes = wptui.ManifestJSON
	}

	userHomeDir := deps.userHomeDir
	if userHomeDir == nil {
		userHomeDir = os.UserHomeDir
	}

	launchApp := deps.launchApp
	if launchApp == nil {
		launchApp = defaultLaunchApp
	}

	switch action {
	case actionLaunchTUI:
		if err := launchApp(); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case actionHelp:
		_, _ = io.WriteString(stdout, usageText)
		return 0
	case actionVersion:
		manifest, err := release.ParseManifest(manifestBytes)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s v%s\n", manifest.Name, manifest.Version)
		return 0
	case actionConfig:
		home, err := userHomeDir()
		if err != nil {
			fmt.Fprintf(stderr, "Error: could not determine user home directory: %v\n", err)
			return 1
		}
		if home == "" {
			fmt.Fprintln(stderr, "Error: user home directory is empty")
			return 1
		}
		cfgPath, err := config.ConfigPath(home)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, cfgPath)
		return 0
	case actionUpdate:
		res, err := release.CheckUpdate(context.Background(), manifestBytes, release.CheckOptions{
			ManifestURL: deps.remoteManifestURL,
			HTTPClient:  deps.httpClient,
			Timeout:     deps.updateTimeout,
		})
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		if res.UpdateAvailable {
			fmt.Fprintf(stdout, "Update available: v%s -> v%s\nRelease: %s\n", res.CurrentVersion, res.RemoteVersion, res.ReleaseURL)
			return 0
		}
		fmt.Fprintf(stdout, "%s v%s is already up to date.\n", release.ExpectedProduct, res.CurrentVersion)
		return 0
	default:
		fmt.Fprintln(stderr, "Error: unsupported command action")
		return 1
	}
}

func parseArgs(args []string) (cliAction, error) {
	if len(args) == 0 {
		return actionLaunchTUI, nil
	}

	var selected cliAction
	for _, arg := range args {
		action, ok := matchActionFlag(arg)
		if !ok {
			if strings.HasPrefix(arg, "-") {
				return 0, fmt.Errorf("unknown flag %q", arg)
			}
			return 0, fmt.Errorf("unexpected argument %q", arg)
		}
		if selected != actionLaunchTUI {
			if selected == action {
				return 0, fmt.Errorf("duplicate action flag %q", arg)
			}
			return 0, fmt.Errorf("cannot combine multiple action flags")
		}
		selected = action
	}
	return selected, nil
}

func matchActionFlag(arg string) (cliAction, bool) {
	switch arg {
	case "-v", "--version":
		return actionVersion, true
	case "-u", "--update":
		return actionUpdate, true
	case "-c", "--config":
		return actionConfig, true
	case "-h", "--help":
		return actionHelp, true
	default:
		return 0, false
	}
}

func defaultLaunchApp() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	application := app.New(app.Options{})
	return application.RunWithContext(ctx)
}
