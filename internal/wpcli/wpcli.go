package wpcli

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type DBConnection struct {
	Host   string
	Port   int
	User   string
	Pass   string
	Socket string
}

type Runner interface {
	Run(ctx context.Context, dir string, name string, args []string, stdin string) (stdout string, stderr string, err error)
	LookPath(file string) (string, error)
}

type DefaultRunner struct{}

func (r *DefaultRunner) Run(ctx context.Context, dir string, name string, args []string, stdin string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func (r *DefaultRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

type Client struct {
	runner Runner
}

func NewClient() *Client {
	return &Client{
		runner: &DefaultRunner{},
	}
}

func NewClientWithRunner(r Runner) *Client {
	return &Client{
		runner: r,
	}
}

func (c *Client) CheckDependencies(usedHerd bool) error {
	required := []string{"php", "wp", "mysql"}
	if usedHerd {
		required = append(required, "herd")
	}

	var missing []string
	for _, bin := range required {
		if _, err := c.runner.LookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required executable(s) in PATH: %s", strings.Join(missing, ", "))
	}
	return nil
}

func (c *Client) CoreDownload(ctx context.Context, dir, locale string) error {
	args := []string{"core", "download"}
	if locale != "" {
		args = append(args, "--locale="+locale)
	}
	_, stderr, err := c.runner.Run(ctx, dir, "wp", args, "")
	if err != nil {
		return fmt.Errorf("wp core download failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return nil
}

func (c *Client) ConfigCreate(ctx context.Context, dir, dbName string, conn DBConnection, skipCheck bool) error {
	hostVal := conn.Host
	if conn.Socket != "" {
		hostVal = "localhost:" + conn.Socket
	} else if conn.Port > 0 {
		hostVal = net.JoinHostPort(conn.Host, strconv.Itoa(conn.Port))
	}

	args := []string{
		"config", "create",
		"--dbname=" + dbName,
		"--dbuser=" + conn.User,
		"--dbhost=" + hostVal,
	}

	var stdin string
	if conn.Pass != "" {
		args = append(args, "--prompt=dbpass")
		stdin = conn.Pass + "\n"
	}

	if skipCheck {
		args = append(args, "--skip-check")
	}

	_, stderr, err := c.runner.Run(ctx, dir, "wp", args, stdin)
	if err != nil {
		return fmt.Errorf("wp config create failed: %w (output: %s)", err, sanitizeStderr(stderr, conn.Pass))
	}
	return nil
}

func (c *Client) DBCreate(ctx context.Context, dir string) (bool, error) {
	stdout, stderr, err := c.runner.Run(ctx, dir, "wp", []string{"db", "create"}, "")
	if err != nil {
		return false, fmt.Errorf("wp db create failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return strings.Contains(stdout, "Database created") || strings.Contains(stdout, "Success"), nil
}

func (c *Client) DBDrop(ctx context.Context, dir string) error {
	_, stderr, err := c.runner.Run(ctx, dir, "wp", []string{"db", "drop", "--yes"}, "")
	if err != nil {
		return fmt.Errorf("wp db drop failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return nil
}

func (c *Client) CoreInstall(ctx context.Context, dir, url, title, adminUser, adminPass, adminEmail string) error {
	args := []string{
		"core", "install",
		"--url=" + url,
		"--title=" + title,
		"--admin_user=" + adminUser,
		"--admin_email=" + adminEmail,
		"--skip-email",
	}

	var stdin string
	if adminPass != "" {
		args = append(args, "--prompt=admin_password")
		stdin = adminPass + "\n"
	}
	_, stderr, err := c.runner.Run(ctx, dir, "wp", args, stdin)
	if err != nil {
		return fmt.Errorf("wp core install failed: %w (output: %s)", err, sanitizeStderr(stderr, adminPass))
	}
	return nil
}

func (c *Client) HerdSecure(ctx context.Context, dir, slug string) error {
	_, stderr, err := c.runner.Run(ctx, dir, "herd", []string{"secure", slug}, "")
	if err != nil {
		return fmt.Errorf("herd secure failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return nil
}

func (c *Client) HerdUnsecure(ctx context.Context, dir, slug string) error {
	_, stderr, err := c.runner.Run(ctx, dir, "herd", []string{"unsecure", slug}, "")
	if err != nil {
		return fmt.Errorf("herd unsecure failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return nil
}
func (c *Client) IsPathParked(ctx context.Context, targetPath string) (bool, error) {
	stdout, stderr, err := c.runner.Run(ctx, "", "herd", []string{"paths"}, "")
	if err != nil {
		var errParked error
		stdout, stderr, errParked = c.runner.Run(ctx, "", "herd", []string{"parked"}, "")
		if errParked != nil {
			return false, fmt.Errorf("failed to check herd parked paths: %w (output: %s)", err, strings.TrimSpace(stderr))
		}
	}
	cleanTarget := filepath.Clean(targetPath)
	lines := strings.Split(stdout, "\n")
	for _, l := range lines {
		trimmed := strings.Trim(l, "|\r\n\t ")
		if strings.HasPrefix(trimmed, "+-") || strings.EqualFold(trimmed, "path") {
			continue
		}
		cleanL := filepath.Clean(trimmed)
		if runtime.GOOS == "windows" {
			if strings.EqualFold(cleanL, cleanTarget) {
				return true, nil
			}
		} else {
			if cleanL == cleanTarget {
				return true, nil
			}
		}
	}
	return false, nil
}

func (c *Client) IsSiteSecured(ctx context.Context, slug string) (bool, error) {
	stdout, stderr, err := c.runner.Run(ctx, "", "herd", []string{"secured"}, "")
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

func sanitizeStderr(stderr string, secrets ...string) string {
	clean := strings.TrimSpace(stderr)
	for _, s := range secrets {
		if s != "" {
			clean = strings.ReplaceAll(clean, s, "[REDACTED]")
		}
	}
	return clean
}

func (c *Client) PluginInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error {
	args := []string{"plugin", "install", pathOrSlug}
	if activate {
		args = append(args, "--activate")
	}
	_, stderr, err := c.runner.Run(ctx, dir, "wp", args, "")
	if err != nil {
		return fmt.Errorf("wp plugin install failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return nil
}

func (c *Client) ThemeInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error {
	args := []string{"theme", "install", pathOrSlug}
	if activate {
		args = append(args, "--activate")
	}
	_, stderr, err := c.runner.Run(ctx, dir, "wp", args, "")
	if err != nil {
		return fmt.Errorf("wp theme install failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return nil
}

func (c *Client) CheckDatabaseExists(ctx context.Context, conn DBConnection, dbName string) (bool, error) {
	tmpFile, err := os.CreateTemp("", "wptui-mysql-*.cnf")
	if err != nil {
		return false, fmt.Errorf("failed to create temporary mysql config: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	var cnfContent strings.Builder
	cnfContent.WriteString("[client]\n")
	if conn.User != "" {
		cnfContent.WriteString("user=" + conn.User + "\n")
	}
	if conn.Pass != "" {
		cnfContent.WriteString("password=" + conn.Pass + "\n")
	}
	if conn.Socket != "" {
		cnfContent.WriteString("socket=" + conn.Socket + "\n")
	} else if conn.Host != "" {
		cnfContent.WriteString("host=" + conn.Host + "\n")
		if conn.Port > 0 {
			cnfContent.WriteString("port=" + strconv.Itoa(conn.Port) + "\n")
		}
	}

	if _, err := tmpFile.WriteString(cnfContent.String()); err != nil {
		_ = tmpFile.Close()
		return false, fmt.Errorf("failed to write mysql credentials: %w", err)
	}
	_ = tmpFile.Close()
	_ = os.Chmod(tmpPath, 0600)

	args := []string{
		"--defaults-extra-file=" + tmpPath,
		"--batch",
		"--skip-column-names",
		"-e", "SHOW DATABASES;",
	}

	stdout, stderr, err := c.runner.Run(ctx, "", "mysql", args, "")
	if err != nil {
		return false, fmt.Errorf("failed to list database schemas: %w (output: %s)", err, strings.TrimSpace(stderr))
	}

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == dbName {
			return true, nil
		}
	}

	return false, nil
}

func (c *Client) ConfigSet(ctx context.Context, dir, key, value string, raw bool) error {
	args := []string{"config", "set", key, value}
	if raw {
		args = append(args, "--raw")
	}
	_, stderr, err := c.runner.Run(ctx, dir, "wp", args, "")
	if err != nil {
		return fmt.Errorf("wp config set %s failed: %w (output: %s)", key, err, strings.TrimSpace(stderr))
	}
	return nil
}

func (c *Client) RewriteStructure(ctx context.Context, dir, value string) error {
	args := []string{"rewrite", "structure", value}
	_, stderr, err := c.runner.Run(ctx, dir, "wp", args, "")
	if err != nil {
		return fmt.Errorf("wp rewrite structure failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return nil
}

func (c *Client) OptionUpdate(ctx context.Context, dir, key, value string) error {
	args := []string{"option", "update", key, value}
	_, stderr, err := c.runner.Run(ctx, dir, "wp", args, "")
	if err != nil {
		return fmt.Errorf("wp option update %s failed: %w (output: %s)", key, err, strings.TrimSpace(stderr))
	}
	return nil
}

func (c *Client) LanguageCore(ctx context.Context, dir, action, value string) error {
	args := []string{"language", "core", action, value}
	_, stderr, err := c.runner.Run(ctx, dir, "wp", args, "")
	if err != nil {
		return fmt.Errorf("wp language core %s %s failed: %w (output: %s)", action, value, err, strings.TrimSpace(stderr))
	}
	return nil
}
