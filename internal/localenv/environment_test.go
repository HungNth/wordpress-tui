package localenv_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"wptui/internal/config"
	"wptui/internal/localenv"
)

type mockRunner struct {
	calls        []string
	runErr       error
	lookPathErrs map[string]error
	securedOut   string
	customOutput map[string]string
	customErr    map[string]error
}

func (m *mockRunner) Run(ctx context.Context, dir string, name string, args []string, stdin string) (string, string, error) {
	cmdStr := name + " " + strings.Join(args, " ")
	if dir != "" {
		cmdStr = "(" + dir + ") " + cmdStr
	}
	m.calls = append(m.calls, cmdStr)
	if m.runErr != nil {
		return "", "mock error", m.runErr
	}
	joinedArgs := strings.Join(args, " ")
	for k, err := range m.customErr {
		if strings.Contains(cmdStr, k) || strings.Contains(joinedArgs, k) {
			return "", "mock error", err
		}
	}
	for k, out := range m.customOutput {
		if strings.Contains(cmdStr, k) || strings.Contains(joinedArgs, k) {
			return out, "", nil
		}
	}
	if strings.Contains(cmdStr, "secured") {
		return m.securedOut, "", nil
	}
	return "ok", "", nil
}

func (m *mockRunner) LookPath(file string) (string, error) {
	if err, ok := m.lookPathErrs[file]; ok {
		return "", err
	}
	return "/bin/" + file, nil
}

func TestLocalenv_New_StrictValidation(t *testing.T) {
	// Rejects empty command
	for _, app := range []string{config.AppNameHerd, config.AppNameFlock, config.AppNameLaragon} {
		if _, err := localenv.New(app, ""); err == nil {
			t.Errorf("New(%q, \"\") expected error for empty command, got nil", app)
		}
		if _, err := localenv.New(app, "   "); err == nil {
			t.Errorf("New(%q, \"   \") expected error for whitespace command, got nil", app)
		}
	}

	// Rejects nonexact enums (case folding or leading/trailing whitespace)
	invalidEnums := []string{
		"HERD", "Herd", " herd ",
		"FLOCK", "Flock", " flock ",
		"LARAGON", "Laragon", " laragon ",
		"valet", "docker", "",
	}
	for _, app := range invalidEnums {
		if _, err := localenv.New(app, "some-cmd"); err == nil {
			t.Errorf("New(%q, \"some-cmd\") expected error for nonexact enum, got nil", app)
		}
	}

	// Accepts exact enums with valid command
	exactCases := []struct {
		appName  string
		cmd      string
		wantName string
	}{
		{config.AppNameHerd, "herd", "herd"},
		{config.AppNameFlock, "/usr/local/bin/flock", "flock"},
		{config.AppNameLaragon, "C:\\laragon\\laragon.exe", "laragon"},
	}
	for _, tc := range exactCases {
		env, err := localenv.New(tc.appName, tc.cmd)
		if err != nil {
			t.Fatalf("New(%q, %q) unexpected error: %v", tc.appName, tc.cmd, err)
		}
		if env.Name() != tc.wantName {
			t.Errorf("expected Name %q, got %q", tc.wantName, env.Name())
		}
		if env.Command() != tc.cmd {
			t.Errorf("expected Command %q, got %q", tc.cmd, env.Command())
		}
	}

	// NewFromConfig nil check
	if _, err := localenv.NewFromConfig(nil); err == nil {
		t.Errorf("NewFromConfig(nil) expected error, got nil")
	}
}

func TestLocalenv_Flock_Register_Safety(t *testing.T) {
	ctx := t.Context()
	siteDir := "/sites/mysite"
	slug := "mysite"

	flockSitesHeader := "SITE      PHP                     URL                     ROOT  PATH\n"

	t.Run("fresh site links and returns owned true", func(t *testing.T) {
		env, err := localenv.New(config.AppNameFlock, "flock")
		if err != nil {
			t.Fatal(err)
		}
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader + "othersite 8.3                     http://othersite.test   -     /sites/othersite\n",
			},
		}

		owned, err := env.Register(ctx, runner, siteDir, slug)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !owned {
			t.Errorf("expected owned true for fresh registration, got false")
		}
		var linkCallFound bool
		for _, call := range runner.calls {
			if strings.Contains(call, "flock link mysite") {
				linkCallFound = true
				break
			}
		}
		if !linkCallFound {
			t.Errorf("expected flock link mysite to be called, got calls: %v", runner.calls)
		}
	})

	t.Run("preexisting site same path preserves registration and returns owned false", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameFlock, "flock")
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader + "mysite    8.3                     https://mysite.test     -     " + siteDir + "\n",
			},
		}

		owned, err := env.Register(ctx, runner, siteDir, slug)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if owned {
			t.Errorf("expected owned false for preexisting registration, got true")
		}
		for _, call := range runner.calls {
			if strings.Contains(call, "link") {
				t.Fatalf("flock link should not be called for preexisting registration, got call: %s", call)
			}
		}
	})

	t.Run("preexisting parked site same path returns owned false without link", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameFlock, "flock")
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader + "mysite    8.3                     http://mysite.test      -     " + siteDir + "\n",
			},
		}

		owned, err := env.Register(ctx, runner, siteDir, slug)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if owned {
			t.Errorf("expected owned false for parked same-path site, got true")
		}
		for _, call := range runner.calls {
			if strings.Contains(call, "link") {
				t.Fatalf("flock link should not be called for parked site, got call: %s", call)
			}
		}
	})

	t.Run("path with spaces preserved and correctly matched", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameFlock, "flock")
		spacedDir := "C:\\Users\\John Doe\\Sites\\mysite"
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader + "mysite    8.3                     https://mysite.test     -     " + spacedDir + "\n",
			},
		}

		owned, err := env.Register(ctx, runner, spacedDir, slug)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if owned {
			t.Errorf("expected owned false for matching path with spaces, got true")
		}
		for _, call := range runner.calls {
			if strings.Contains(call, "link") {
				t.Fatalf("unexpected link call: %s", call)
			}
		}
	})

	t.Run("collision rejects before link and returns owned false", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameFlock, "flock")
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader + "mysite    8.3                     http://mysite.test      -     /other/path/mysite\n",
			},
		}

		owned, err := env.Register(ctx, runner, siteDir, slug)
		if err == nil {
			t.Fatalf("expected collision error, got nil")
		}
		if owned {
			t.Errorf("expected owned false on preflight collision failure, got true")
		}
		for _, call := range runner.calls {
			if strings.Contains(call, "link") {
				t.Fatalf("flock link must not be called when name collides, got call: %s", call)
			}
		}
	})

	t.Run("malformed target row rejects before link and returns owned false", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameFlock, "flock")
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader + "mysite    8.3                     ⚠ port 80 in use        -     /sites/mysite\n",
			},
		}

		owned, err := env.Register(ctx, runner, siteDir, slug)
		if err == nil {
			t.Fatalf("expected error for unparseable target row, got nil")
		}
		if owned {
			t.Errorf("expected owned false on malformed target preflight error, got true")
		}
		for _, call := range runner.calls {
			if strings.Contains(call, "link") {
				t.Fatalf("link must not be called on malformed target row, got call: %s", call)
			}
		}
	})

	t.Run("unrelated malformed row does not block target site registration", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameFlock, "flock")
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader +
					"broken    8.3                     ⚠ daemon error          -     /sites/broken\n" +
					"other     8.2 FrankenPHP (Octane)  http://other.test       -     /sites/other\n",
			},
		}

		owned, err := env.Register(ctx, runner, siteDir, slug)
		if err != nil {
			t.Fatalf("unrelated malformed row should not fail target registration: %v", err)
		}
		if !owned {
			t.Errorf("expected owned true, got false")
		}
	})

	t.Run("link failure returns owned true and error for rollback", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameFlock, "flock")
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader,
			},
			customErr: map[string]error{
				"link": errors.New("daemon commit failed"),
			},
		}

		owned, err := env.Register(ctx, runner, siteDir, slug)
		if err == nil {
			t.Fatalf("expected link error, got nil")
		}
		if !owned {
			t.Errorf("expected owned true on link failure so caller records ownership before error, got false")
		}
	})

	t.Run("missing table header rejects with preflight error", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameFlock, "flock")
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": "",
			},
		}

		owned, err := env.Register(ctx, runner, siteDir, slug)
		if err == nil {
			t.Fatalf("expected error on empty sites output, got nil")
		}
		if owned {
			t.Errorf("expected owned false on preflight header error, got true")
		}
	})
}

func TestLocalenv_Flock_IsSiteSecured_ProbeTLS(t *testing.T) {
	ctx := t.Context()
	flockSitesHeader := "SITE      PHP                     URL                     ROOT  PATH\n"

	t.Run("https site returns true", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameFlock, "flock")
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader + "mysite    8.3                     https://mysite.test     -     /sites/mysite\n",
			},
		}

		secured, err := env.IsSiteSecured(ctx, runner, "mysite")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !secured {
			t.Errorf("expected secured true for HTTPS URL, got false")
		}
	})

	t.Run("http site returns false", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameFlock, "flock")
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader + "mysite    8.3                     http://mysite.test      -     /sites/mysite\n",
			},
		}

		secured, err := env.IsSiteSecured(ctx, runner, "mysite")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if secured {
			t.Errorf("expected secured false for HTTP URL, got true")
		}
	})

	t.Run("spaces in path and FrankenPHP in PHP column parsed correctly", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameFlock, "flock")
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader + "mysite    8.2 FrankenPHP (Octane)  https://mysite.test     -     C:\\Users\\John Doe\\Sites\\mysite\n",
			},
		}

		secured, err := env.IsSiteSecured(ctx, runner, "mysite")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !secured {
			t.Errorf("expected secured true, got false")
		}
	})

	t.Run("malformed target row error propagates", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameFlock, "flock")
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader + "mysite    8.3                     ⚠ certificate expired   -     /sites/mysite\n",
			},
		}

		_, err := env.IsSiteSecured(ctx, runner, "mysite")
		if err == nil {
			t.Fatalf("expected error propagating for unparseable target URL, got nil")
		}
	})

	t.Run("unrelated row error does not fail target probe", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameFlock, "flock")
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader +
					"broken    8.3                     ⚠ port error            -     /sites/broken\n" +
					"mysite    8.3                     https://mysite.test     -     /sites/mysite\n",
			},
		}

		secured, err := env.IsSiteSecured(ctx, runner, "mysite")
		if err != nil {
			t.Fatalf("unrelated error should not fail target probe: %v", err)
		}
		if !secured {
			t.Errorf("expected secured true, got false")
		}
	})

	t.Run("unknown site returns false without error", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameFlock, "flock")
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader + "othersite 8.3                     https://othersite.test  -     /sites/othersite\n",
			},
		}

		secured, err := env.IsSiteSecured(ctx, runner, "mysite")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if secured {
			t.Errorf("expected secured false for nonexistent site, got true")
		}
	})
}

func TestLocalenv_Flock_CleanupSafety(t *testing.T) {
	env, err := localenv.New(config.AppNameFlock, "flock")
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	flockSitesHeader := "SITE      PHP                     URL                     ROOT  PATH\n"
	siteDir := "/sites/mysite"
	slug := "mysite"

	t.Run("matching path invokes unsecure and unregister", func(t *testing.T) {
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader + "mysite    8.3                     https://mysite.test     -     " + siteDir + "\n",
			},
		}

		if err := env.Unsecure(ctx, runner, siteDir, slug); err != nil {
			t.Fatalf("unexpected unsecure error: %v", err)
		}
		var unsecureCalled bool
		for _, call := range runner.calls {
			if strings.Contains(call, "flock unsecure mysite") {
				unsecureCalled = true
				break
			}
		}
		if !unsecureCalled {
			t.Errorf("expected flock unsecure call, got: %v", runner.calls)
		}

		if err := env.Unregister(ctx, runner, siteDir, slug); err != nil {
			t.Fatalf("unexpected unregister error: %v", err)
		}
		var unlinkCalled bool
		for _, call := range runner.calls {
			if strings.Contains(call, "flock unlink mysite") {
				unlinkCalled = true
				break
			}
		}
		if !unlinkCalled {
			t.Errorf("expected flock unlink call, got: %v", runner.calls)
		}
	})

	t.Run("missing site is harmless no-op without mutation", func(t *testing.T) {
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader + "othersite 8.3                     https://othersite.test  -     /sites/othersite\n",
			},
		}

		if err := env.Unsecure(ctx, runner, siteDir, slug); err != nil {
			t.Fatalf("expected nil for missing site unsecure, got: %v", err)
		}
		if err := env.Unregister(ctx, runner, siteDir, slug); err != nil {
			t.Fatalf("expected nil for missing site unregister, got: %v", err)
		}
		for _, call := range runner.calls {
			if strings.Contains(call, "unsecure") || strings.Contains(call, "unlink") {
				t.Fatalf("expected no mutation calls for missing site, got: %s", call)
			}
		}
	})

	t.Run("mismatched registration returns clear error without mutation", func(t *testing.T) {
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader + "mysite    8.3                     https://mysite.test     -     /different/path\n",
			},
		}

		if err := env.Unsecure(ctx, runner, siteDir, slug); err == nil {
			t.Fatalf("expected error when unsecuring mismatched path, got nil")
		}
		if err := env.Unregister(ctx, runner, siteDir, slug); err == nil {
			t.Fatalf("expected error when unregistering mismatched path, got nil")
		}
		for _, call := range runner.calls {
			if strings.Contains(call, "unsecure") || strings.Contains(call, "unlink") {
				t.Fatalf("expected no mutation calls when path is mismatched, got: %s", call)
			}
		}
	})

	t.Run("malformed target row rejects unsecure and unregister without mutation", func(t *testing.T) {
		runner := &mockRunner{
			customOutput: map[string]string{
				"sites": flockSitesHeader + "mysite    8.3                     ⚠ daemon error          -     " + siteDir + "\n",
			},
		}

		if err := env.Unsecure(ctx, runner, siteDir, slug); err == nil {
			t.Fatalf("expected error when unsecuring malformed target row, got nil")
		}
		if err := env.Unregister(ctx, runner, siteDir, slug); err == nil {
			t.Fatalf("expected error when unregistering malformed target row, got nil")
		}
		for _, call := range runner.calls {
			if strings.Contains(call, "unsecure") || strings.Contains(call, "unlink") {
				t.Fatalf("expected no mutation calls on malformed target row, got: %s", call)
			}
		}
	})

	t.Run("secure calls flock secure", func(t *testing.T) {
		runner := &mockRunner{}
		if err := env.Secure(ctx, runner, siteDir, slug); err != nil {
			t.Fatal(err)
		}
		if len(runner.calls) != 1 || !strings.Contains(runner.calls[0], "flock secure mysite") {
			t.Errorf("unexpected Secure call: %v", runner.calls)
		}
	})
}

func TestLocalenv_Flock_RealisticLiveOutputFixture(t *testing.T) {
	liveOutput := "SITE                               PHP  URL                                             ROOT                       PATH\n" +
		"db                                 8.4  https://db.test                                 project folder (detected)  F:\\sites\\db\n" +
		"wp-update-server-laravel-react-ts  8.4  https://wp-update-server-laravel-react-ts.test  public (detected)          F:\\Laravel\\laravel13\\wp-update-server-laravel-react-ts\n"

	env, err := localenv.New(config.AppNameFlock, "flock")
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()

	// 1. Probe TLS on live fixture sites
	runner := &mockRunner{
		customOutput: map[string]string{"sites": liveOutput},
	}
	secured, err := env.IsSiteSecured(ctx, runner, "db")
	if err != nil || !secured {
		t.Fatalf("expected db to be secured on live fixture, got secured=%v, err=%v", secured, err)
	}
	secured, err = env.IsSiteSecured(ctx, runner, "wp-update-server-laravel-react-ts")
	if err != nil || !secured {
		t.Fatalf("expected wp-update-server-laravel-react-ts to be secured, got secured=%v, err=%v", secured, err)
	}

	// 2. Register preexisting same path returns owned false without linking
	runner = &mockRunner{
		customOutput: map[string]string{"sites": liveOutput},
	}
	owned, err := env.Register(ctx, runner, "F:\\sites\\db", "db")
	if err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}
	if owned {
		t.Errorf("expected owned false for preexisting db, got true")
	}

	// 3. Register collision returns error
	runner = &mockRunner{
		customOutput: map[string]string{"sites": liveOutput},
	}
	owned, err = env.Register(ctx, runner, "F:\\sites\\other-db", "db")
	if err == nil {
		t.Fatalf("expected collision error for db, got nil")
	}
	if owned {
		t.Errorf("expected owned false on collision, got true")
	}

	// 4. Register fresh site issues link and returns owned true
	runner = &mockRunner{
		customOutput: map[string]string{"sites": liveOutput},
	}
	owned, err = env.Register(ctx, runner, "F:\\sites\\newsite", "newsite")
	if err != nil {
		t.Fatalf("unexpected register error for fresh site: %v", err)
	}
	if !owned {
		t.Errorf("expected owned true for fresh site, got false")
	}

	// 5. Unregister on matching path invokes unlink
	runner = &mockRunner{
		customOutput: map[string]string{"sites": liveOutput},
	}
	if err := env.Unregister(ctx, runner, "F:\\sites\\db", "db"); err != nil {
		t.Fatalf("unexpected unregister error: %v", err)
	}
	var unlinkCalled bool
	for _, call := range runner.calls {
		if strings.Contains(call, "flock unlink db") {
			unlinkCalled = true
			break
		}
	}
	if !unlinkCalled {
		t.Errorf("expected flock unlink db, got calls: %v", runner.calls)
	}

	// 6. Unregister on mismatched path rejects without mutation
	runner = &mockRunner{
		customOutput: map[string]string{"sites": liveOutput},
	}
	if err := env.Unregister(ctx, runner, "F:\\sites\\wrong", "db"); err == nil {
		t.Fatalf("expected error on mismatched path unregister, got nil")
	}
	for _, call := range runner.calls {
		if strings.Contains(call, "unlink") {
			t.Fatalf("unlink should not be called on mismatched path: %s", call)
		}
	}
}

func TestLocalenv_Laragon_Lifecycle_Safety(t *testing.T) {
	ctx := t.Context()
	siteDir := "C:\\laragon\\www\\mysite"

	t.Run("register calls reload and returns owned true", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameLaragon, "C:\\laragon\\laragon.exe")
		runner := &mockRunner{}

		owned, err := env.Register(ctx, runner, siteDir, "mysite")
		if err != nil {
			t.Fatal(err)
		}
		if !owned {
			t.Errorf("expected owned true for Laragon register, got false")
		}
		if len(runner.calls) != 1 || !strings.Contains(runner.calls[0], "laragon.exe reload") {
			t.Errorf("unexpected register call: %v", runner.calls)
		}
	})

	t.Run("register failure returns owned true for rollback reload", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameLaragon, "C:\\laragon\\laragon.exe")
		runner := &mockRunner{
			customErr: map[string]error{
				"reload": errors.New("reload failed"),
			},
		}

		owned, err := env.Register(ctx, runner, siteDir, "mysite")
		if err == nil {
			t.Fatalf("expected reload error, got nil")
		}
		if !owned {
			t.Errorf("expected owned true on Laragon reload failure for cleanup, got false")
		}
	})

	t.Run("unregister runs reload in filepath.Dir surviving directory deletion", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameLaragon, "C:\\laragon\\laragon.exe")
		runner := &mockRunner{}

		err := env.Unregister(ctx, runner, siteDir, "mysite")
		if err != nil {
			t.Fatal(err)
		}
		if len(runner.calls) != 1 {
			t.Fatalf("expected 1 call, got %v", runner.calls)
		}

		expectedDir := filepath.Dir(siteDir)
		if !strings.Contains(runner.calls[0], "("+expectedDir+")") {
			t.Errorf("expected unregister to run in parent directory %q, got call: %s", expectedDir, runner.calls[0])
		}
	})

	t.Run("secure and unsecure are no-ops and isSiteSecured is true", func(t *testing.T) {
		env, _ := localenv.New(config.AppNameLaragon, "C:\\laragon\\laragon.exe")
		runner := &mockRunner{}

		if err := env.Secure(ctx, runner, siteDir, "mysite"); err != nil {
			t.Fatal(err)
		}
		if err := env.Unsecure(ctx, runner, siteDir, "mysite"); err != nil {
			t.Fatal(err)
		}
		if len(runner.calls) != 0 {
			t.Errorf("expected 0 calls for Secure/Unsecure, got %v", runner.calls)
		}

		secured, err := env.IsSiteSecured(ctx, runner, "mysite")
		if err != nil || !secured {
			t.Errorf("expected Laragon site to be secured by default, got %v, err: %v", secured, err)
		}
	})
}

func TestLocalenv_Herd_Lifecycle(t *testing.T) {
	env, _ := localenv.New(config.AppNameHerd, "herd")
	runner := &mockRunner{
		securedOut: "+-------------------+\n| Site              |\n+-------------------+\n| mysite.test       |\n+-------------------+\n",
	}
	ctx := t.Context()

	// Register is no-op returning owned=false
	owned, err := env.Register(ctx, runner, "/sites/mysite", "mysite")
	if err != nil {
		t.Fatal(err)
	}
	if owned {
		t.Errorf("expected owned false for Herd Register, got true")
	}
	if len(runner.calls) != 0 {
		t.Errorf("expected 0 calls for Herd Register, got %v", runner.calls)
	}

	// Secure
	if err := env.Secure(ctx, runner, "/sites/mysite", "mysite"); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 || !strings.Contains(runner.calls[0], "herd secure mysite") {
		t.Errorf("unexpected Secure call: %v", runner.calls)
	}

	// IsSiteSecured
	secured, err := env.IsSiteSecured(ctx, runner, "mysite")
	if err != nil || !secured {
		t.Errorf("expected mysite to be secured: %v, err: %v", secured, err)
	}

	// Unsecure
	if err := env.Unsecure(ctx, runner, "/sites/mysite", "mysite"); err != nil {
		t.Fatal(err)
	}
	lastCall := runner.calls[len(runner.calls)-1]
	if !strings.Contains(lastCall, "herd unsecure mysite") {
		t.Errorf("unexpected Unsecure call: %s", lastCall)
	}

	// Unregister is no-op
	if err := env.Unregister(ctx, runner, "/sites/mysite", "mysite"); err != nil {
		t.Fatal(err)
	}
}

func TestLocalenv_CheckDependencies(t *testing.T) {
	env, err := localenv.New(config.AppNameFlock, "flock")
	if err != nil {
		t.Fatal(err)
	}

	runner := &mockRunner{
		lookPathErrs: map[string]error{
			"flock": errors.New("not found"),
		},
	}

	err = env.CheckDependencies(t.Context(), runner)
	if err == nil || !strings.Contains(err.Error(), "flock") {
		t.Fatalf("expected missing flock error, got %v", err)
	}
}
