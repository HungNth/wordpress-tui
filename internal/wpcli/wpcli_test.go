package wpcli_test

import (
	"context"
	"strings"
	"testing"

	"wptui/internal/wpcli"
)

type mockRunner struct {
	calls []string
	stdin []string
}

func (m *mockRunner) Run(ctx context.Context, dir string, name string, args []string, stdin string) (string, string, error) {
	call := name + " " + strings.Join(args, " ")
	m.calls = append(m.calls, call)
	m.stdin = append(m.stdin, stdin)

	if strings.Contains(call, "SHOW DATABASES;") {
		return "information_schema\nmysql\nexisting-db\nperformance_schema\n", "", nil
	}
	if strings.Contains(call, "herd paths") || strings.Contains(call, "herd parked") {
		return "+------------------------------+\n| Path                         |\n+------------------------------+\n| /var/www/sites               |\n+------------------------------+\n", "", nil
	}
	if strings.Contains(call, "herd secured") {
		return "+----------------------------+\n| Site                       |\n+----------------------------+\n| existing-secure.test       |\n+----------------------------+\n", "", nil
	}
	return "Success: Database created.", "", nil
}

func (m *mockRunner) LookPath(file string) (string, error) {
	return "/usr/bin/" + file, nil
}

func TestWPCLIClient_Flow(t *testing.T) {
	runner := &mockRunner{}
	client := wpcli.NewClientWithRunner(runner)

	ctx := context.Background()
	dir := "/var/www/test-site"

	if err := client.CoreDownload(ctx, dir, "en_US"); err != nil {
		t.Fatal(err)
	}
	conn := wpcli.DBConnection{
		Host: "localhost",
		Port: 3306,
		User: "root",
		Pass: "secret",
	}
	if err := client.ConfigCreate(ctx, dir, "test-db", conn, true); err != nil {
		t.Fatal(err)
	}
	created, err := client.DBCreate(ctx, dir)
	if err != nil || !created {
		t.Fatalf("expected db created, got %v, err %v", created, err)
	}
	if err := client.CoreInstall(ctx, dir, "http://test-site.test", "Test Site", "admin", "adminpass", "admin@admin.com"); err != nil {
		t.Fatal(err)
	}

	expectedCalls := []string{
		"wp core download https://wordpress.org/latest.zip",
		"wp config create --dbname=test-db --dbuser=root --dbhost=localhost:3306 --prompt=dbpass --skip-check",
		"wp db create",
		"wp core install --url=http://test-site.test --title=Test Site --admin_user=admin --admin_email=admin@admin.com --skip-email --prompt=admin_password",
	}

	if len(runner.calls) != len(expectedCalls) {
		t.Fatalf("expected %d calls, got %d", len(expectedCalls), len(runner.calls))
	}

	for i, expected := range expectedCalls {
		if runner.calls[i] != expected {
			t.Errorf("call[%d] = %q, want %q", i, runner.calls[i], expected)
		}
	}

	// Verify stdin contained passwords
	if runner.stdin[1] != "secret\n" {
		t.Errorf("expected stdin 'secret\\n', got %q", runner.stdin[1])
	}
	if runner.stdin[3] != "adminpass\n" {
		t.Errorf("expected stdin 'adminpass\\n', got %q", runner.stdin[3])
	}
}

func TestWPCLIClient_CheckDatabaseExists(t *testing.T) {
	runner := &mockRunner{}
	client := wpcli.NewClientWithRunner(runner)
	ctx := context.Background()

	conn := wpcli.DBConnection{
		Host: "localhost",
		Port: 3306,
		User: "root",
		Pass: "super_secret_password",
	}

	exists, err := client.CheckDatabaseExists(ctx, conn, "existing-db")
	if err != nil {
		t.Fatalf("CheckDatabaseExists failed: %v", err)
	}
	if !exists {
		t.Errorf("expected existing-db to exist")
	}

	// Verify that password was NOT passed in argv
	lastCall := runner.calls[len(runner.calls)-1]
	if strings.Contains(lastCall, "super_secret_password") {
		t.Errorf("password must not appear in command line arguments: %s", lastCall)
	}
	if !strings.Contains(lastCall, "--defaults-extra-file=") {
		t.Errorf("expected command to use --defaults-extra-file: %s", lastCall)
	}

	exists, err = client.CheckDatabaseExists(ctx, conn, "non-existing-db")
	if err != nil {
		t.Fatalf("CheckDatabaseExists failed: %v", err)
	}
	if exists {
		t.Errorf("expected non-existing-db to not exist")
	}
}

func TestWPCLIClient_IsSiteSecured(t *testing.T) {
	runner := &mockRunner{}
	client := wpcli.NewClientWithRunner(runner)
	ctx := context.Background()

	secured, err := client.IsSiteSecured(ctx, "existing-secure")
	if err != nil || !secured {
		t.Errorf("expected existing-secure to be secured, got %v, err: %v", secured, err)
	}

	secured, err = client.IsSiteSecured(ctx, "unsecured-site")
	if err != nil || secured {
		t.Errorf("expected unsecured-site to NOT be secured, got %v", secured)
	}
}
