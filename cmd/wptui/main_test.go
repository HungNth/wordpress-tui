package main

// Command-level CLI tests at the run() seam.

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRun_VersionPrintsEmbeddedRelease(t *testing.T) {
	for _, flag := range []string{"--version", "-v"} {
		t.Run(flag, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			appLaunched := false

			code := run([]string{flag}, &stdout, &stderr, runDeps{
				manifestJSON: []byte(`{"name":"wptui","version":"1.4.2","author":{"name":"HungNth","url":"https://github.com/HungNth"}}`),
				launchApp: func() error {
					appLaunched = true
					return nil
				},
			})

			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr=%q)", code, stderr.String())
			}
			if got, want := stdout.String(), "wptui v1.4.2\n"; got != want {
				t.Fatalf("stdout = %q, want %q", got, want)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			if appLaunched {
				t.Fatal("interactive application launched during --version")
			}
		})
	}
}

func TestRun_VersionRejectsInvalidEmbeddedManifest(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
	}{
		{name: "malformed json", manifest: `{`},
		{name: "wrong product", manifest: `{"name":"other","version":"1.0.0"}`},
		{name: "missing version", manifest: `{"name":"wptui"}`},
		{name: "leading v prefix", manifest: `{"name":"wptui","version":"v1.0.0"}`},
		{name: "invalid semver", manifest: `{"name":"wptui","version":"01.0.0"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := run([]string{"--version"}, &stdout, &stderr, runDeps{
				manifestJSON: []byte(tc.manifest),
				launchApp:    func() error { return nil },
			})

			if code != 1 {
				t.Fatalf("exit code = %d, want 1", code)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if stderr.Len() == 0 {
				t.Fatal("expected error diagnostic on stderr")
			}
		})
	}
}

func TestRun_ConfigPrintsExpectedPathWithoutCreatingFiles(t *testing.T) {
	for _, flag := range []string{"--config", "-c"} {
		t.Run(flag, func(t *testing.T) {
			home := t.TempDir()
			var stdout, stderr bytes.Buffer
			appLaunched := false

			code := run([]string{flag}, &stdout, &stderr, runDeps{
				userHomeDir: func() (string, error) { return home, nil },
				launchApp: func() error {
					appLaunched = true
					return nil
				},
			})

			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr=%q)", code, stderr.String())
			}
			wantPath := filepath.Join(home, ".config", "wptui", "config.json")
			if got, want := stdout.String(), wantPath+"\n"; got != want {
				t.Fatalf("stdout = %q, want %q", got, want)
			}
			if !filepath.IsAbs(strings.TrimSpace(stdout.String())) {
				t.Fatalf("expected absolute path, got %q", stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			if _, err := os.Stat(filepath.Join(home, ".config")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected .config directory not to be created, stat err = %v", err)
			}
			if appLaunched {
				t.Fatal("interactive application launched during --config")
			}
		})
	}
}

func TestRun_ConfigHomeResolutionFailureExitsOne(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"--config"}, &stdout, &stderr, runDeps{
		userHomeDir: func() (string, error) { return "", errors.New("home unavailable") },
		launchApp:   func() error { return nil },
	})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Error:") {
		t.Fatalf("stderr = %q, want Error: diagnostic", stderr.String())
	}
}

func TestRun_HelpListsAllActionsOnStdout(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			appLaunched := false

			code := run([]string{flag}, &stdout, &stderr, runDeps{
				launchApp: func() error {
					appLaunched = true
					return nil
				},
			})

			if code != 0 {
				t.Fatalf("exit code = %d, want 0", code)
			}
			out := stdout.String()
			for _, expected := range []string{"Usage:", "-v, --version", "-u, --update", "-c, --config", "-h, --help"} {
				if !strings.Contains(out, expected) {
					t.Fatalf("help output missing %q:\n%s", expected, out)
				}
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			if appLaunched {
				t.Fatal("interactive application launched during --help")
			}
		})
	}
}

func TestRun_InvalidSyntaxWritesUsageErrorAndExitsTwo(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{name: "unknown long flag", args: []string{"--unknown"}},
		{name: "unknown short flag", args: []string{"-x"}},
		{name: "positional argument", args: []string{"websites"}},
		{name: "version and config", args: []string{"--version", "--config"}},
		{name: "version and update", args: []string{"-v", "-u"}},
		{name: "help with another flag", args: []string{"--help", "--version"}},
		{name: "duplicate version flags", args: []string{"-v", "--version"}},
		{name: "bundled short flags", args: []string{"-vc"}},
		{name: "flag with value", args: []string{"--version=true"}},
		{name: "action plus positional", args: []string{"--version", "extra"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			appLaunched := false

			code := run(tc.args, &stdout, &stderr, runDeps{
				launchApp: func() error {
					appLaunched = true
					return nil
				},
			})

			if code != 2 {
				t.Fatalf("exit code = %d, want 2", code)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			errOut := stderr.String()
			if !strings.Contains(errOut, "Error:") || !strings.Contains(errOut, "Usage:") {
				t.Fatalf("stderr = %q, want Error: and Usage: guidance", errOut)
			}
			if appLaunched {
				t.Fatal("interactive application launched on invalid syntax")
			}
		})
	}
}

func TestRun_NoArgumentsLaunchesInteractiveApp(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		launched := false

		code := run(nil, &stdout, &stderr, runDeps{
			launchApp: func() error {
				launched = true
				return nil
			},
		})

		if code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !launched {
			t.Fatal("expected interactive application to launch")
		}
		if stdout.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("unexpected output stdout=%q stderr=%q", stdout.String(), stderr.String())
		}
	})

	t.Run("app error exits one", func(t *testing.T) {
		var stdout, stderr bytes.Buffer

		code := run([]string{}, &stdout, &stderr, runDeps{
			launchApp: func() error {
				return errors.New("tui failure")
			},
		})

		if code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(stderr.String(), "Error: tui failure") {
			t.Fatalf("stderr = %q, want tui failure error", stderr.String())
		}
	})
}

func TestRun_UpdateReportsNewerReleaseAndLink(t *testing.T) {
	for _, flag := range []string{"--update", "-u"} {
		t.Run(flag, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.URL.Path != "/manifest.json" {
					http.NotFound(w, r)
					return
				}
				_, _ = w.Write([]byte(`{"name":"wptui","version":"0.3.0","author":{"name":"HungNth","url":"https://github.com/HungNth"}}`))
			}))
			defer server.Close()

			var stdout, stderr bytes.Buffer
			appLaunched := false

			code := run([]string{flag}, &stdout, &stderr, runDeps{
				manifestJSON:      []byte(`{"name":"wptui","version":"0.2.1"}`),
				remoteManifestURL: server.URL + "/manifest.json",
				httpClient:        server.Client(),
				launchApp: func() error {
					appLaunched = true
					return nil
				},
			})

			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr=%q)", code, stderr.String())
			}
			want := "Update available: v0.2.1 -> v0.3.0\nRelease: https://github.com/HungNth/wordpress-tui/releases/tag/v0.3.0\n"
			if got := stdout.String(); got != want {
				t.Fatalf("stdout = %q, want %q", got, want)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			if got := requests.Load(); got != 1 {
				t.Fatalf("HTTP requests = %d, want 1", got)
			}
			if appLaunched {
				t.Fatal("interactive application launched during --update")
			}
		})
	}
}

func TestRun_UpdateSemVerPrecedenceOutcomes(t *testing.T) {
	cases := []struct {
		name       string
		current    string
		remote     string
		wantStdout string
	}{
		{
			name:       "equal version is up to date",
			current:    "0.2.1",
			remote:     "0.2.1",
			wantStdout: "wptui v0.2.1 is already up to date.\n",
		},
		{
			name:       "lower remote version does not recommend downgrade",
			current:    "0.3.0",
			remote:     "0.2.1",
			wantStdout: "wptui v0.3.0 is already up to date.\n",
		},
		{
			name:       "greater prerelease over earlier stable is reported",
			current:    "0.2.1",
			remote:     "0.3.0-beta.1",
			wantStdout: "Update available: v0.2.1 -> v0.3.0-beta.1\nRelease: https://github.com/HungNth/wordpress-tui/releases/tag/v0.3.0-beta.1\n",
		},
		{
			name:       "stable release supersedes same-version prerelease",
			current:    "0.3.0-rc.1",
			remote:     "0.3.0",
			wantStdout: "Update available: v0.3.0-rc.1 -> v0.3.0\nRelease: https://github.com/HungNth/wordpress-tui/releases/tag/v0.3.0\n",
		},
		{
			name:       "same-version prerelease is lower than installed stable",
			current:    "0.3.0",
			remote:     "0.3.0-rc.1",
			wantStdout: "wptui v0.3.0 is already up to date.\n",
		},
		{
			name:       "numeric prerelease identifiers compare numerically",
			current:    "0.3.0-beta.2",
			remote:     "0.3.0-beta.10",
			wantStdout: "Update available: v0.3.0-beta.2 -> v0.3.0-beta.10\nRelease: https://github.com/HungNth/wordpress-tui/releases/tag/v0.3.0-beta.10\n",
		},
		{
			name:       "non-numeric prerelease outranks numeric prerelease",
			current:    "0.3.0-1",
			remote:     "0.3.0-alpha",
			wantStdout: "Update available: v0.3.0-1 -> v0.3.0-alpha\nRelease: https://github.com/HungNth/wordpress-tui/releases/tag/v0.3.0-alpha\n",
		},
		{
			name:       "build metadata alone does not make release newer",
			current:    "0.2.1+build.1",
			remote:     "0.2.1+build.99",
			wantStdout: "wptui v0.2.1+build.1 is already up to date.\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"name":"wptui","version":"` + tc.remote + `"}`))
			}))
			defer server.Close()

			var stdout, stderr bytes.Buffer
			code := run([]string{"--update"}, &stdout, &stderr, runDeps{
				manifestJSON:      []byte(`{"name":"wptui","version":"` + tc.current + `"}`),
				remoteManifestURL: server.URL,
				httpClient:        server.Client(),
				launchApp:         func() error { return nil },
			})

			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr=%q)", code, stderr.String())
			}
			if got := stdout.String(); got != tc.wantStdout {
				t.Fatalf("stdout = %q, want %q", got, tc.wantStdout)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestRun_UpdateOperationalFailuresExitOne(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		body       string
		delay      time.Duration
		closeEarly bool
	}{
		{name: "http 404", statusCode: http.StatusNotFound, body: "not found"},
		{name: "http 500", statusCode: http.StatusInternalServerError, body: "boom"},
		{name: "malformed json", statusCode: http.StatusOK, body: `{"name":"wptui"`},
		{name: "missing version", statusCode: http.StatusOK, body: `{"name":"wptui"}`},
		{name: "wrong product", statusCode: http.StatusOK, body: `{"name":"other","version":"1.0.0"}`},
		{name: "leading v in remote semver", statusCode: http.StatusOK, body: `{"name":"wptui","version":"v1.0.0"}`},
		{name: "invalid remote semver", statusCode: http.StatusOK, body: `{"name":"wptui","version":"1.0"}`},
		{name: "oversized response", statusCode: http.StatusOK, body: strings.Repeat("a", 70*1024)},
		{name: "request timeout", statusCode: http.StatusOK, body: `{"name":"wptui","version":"1.0.0"}`, delay: 150 * time.Millisecond},
		{name: "transport failure", closeEarly: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.delay > 0 {
					select {
					case <-time.After(tc.delay):
					case <-r.Context().Done():
						return
					}
				}
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			if tc.closeEarly {
				server.Close()
			} else {
				defer server.Close()
			}

			var stdout, stderr bytes.Buffer
			appLaunched := false
			code := run([]string{"--update"}, &stdout, &stderr, runDeps{
				manifestJSON:      []byte(`{"name":"wptui","version":"0.2.1"}`),
				remoteManifestURL: server.URL,
				httpClient:        server.Client(),
				updateTimeout:     40 * time.Millisecond,
				launchApp: func() error {
					appLaunched = true
					return nil
				},
			})

			if code != 1 {
				t.Fatalf("exit code = %d, want 1 (stdout=%q stderr=%q)", code, stdout.String(), stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if !strings.HasPrefix(stderr.String(), "Error: ") {
				t.Fatalf("stderr = %q, want Error: prefix", stderr.String())
			}
			if appLaunched {
				t.Fatal("interactive application launched on failed --update")
			}
		})
	}
}
