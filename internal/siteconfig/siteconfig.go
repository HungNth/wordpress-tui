package siteconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"wptui/internal/config"
	"wptui/internal/packages"
)

var validTablePrefixRe = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type DBConfig struct {
	Name        string
	User        string
	Password    string
	Host        string
	Port        int
	TablePrefix string
}

type AdminUser struct {
	ID        int    `json:"ID"`
	UserLogin string `json:"user_login"`
	UserEmail string `json:"user_email"`
}

type AdminInput struct {
	UserID      int
	NewUsername string
	NewPassword string
	NewEmail    string
}

type WPClient interface {
	ConfigSet(ctx context.Context, dir, key, value string, raw bool) error
	RewriteStructure(ctx context.Context, dir, value string) error
	OptionUpdate(ctx context.Context, dir, key, value string) error
	LanguageCore(ctx context.Context, dir, action, value string) error
	PluginInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error
	ThemeInstall(ctx context.Context, dir, pathOrSlug string, activate bool) error
	Run(ctx context.Context, dir, name string, args []string, stdin string) (stdout string, stderr string, err error)
}

type ProgressFunc func(step, total int, message string)

type DBConnector func(driverName, dataSourceName string) (*sql.DB, error)

var DefaultDBConnector DBConnector = sql.Open

func ExtractDBConfig(ctx context.Context, siteDir string, client WPClient) (*DBConfig, error) {
	keys := []string{"DB_NAME", "DB_USER", "DB_PASSWORD", "DB_HOST"}
	vals := make(map[string]string)
	for _, k := range keys {
		out, stderr, err := client.Run(ctx, siteDir, "wp", []string{"config", "get", k}, "")
		if err != nil {
			return nil, fmt.Errorf("failed to get %s: %w (%s)", k, err, strings.TrimSpace(stderr))
		}
		vals[k] = strings.TrimSpace(out)
	}

	prefixOut, stderr, err := client.Run(ctx, siteDir, "wp", []string{"config", "get", "table_prefix", "--type=variable"}, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get table_prefix: %w (%s)", err, strings.TrimSpace(stderr))
	}
	prefix := strings.TrimSpace(prefixOut)
	if !validTablePrefixRe.MatchString(prefix) {
		return nil, fmt.Errorf("invalid table_prefix %q", prefix)
	}

	rawHost := vals["DB_HOST"]
	host := rawHost
	port := 3306
	if strings.Contains(rawHost, ":") {
		h, p, err := net.SplitHostPort(rawHost)
		if err == nil {
			host = h
			if parsedPort, err := strconv.Atoi(p); err == nil {
				port = parsedPort
			}
		}
	}

	return &DBConfig{
		Name:        vals["DB_NAME"],
		User:        vals["DB_USER"],
		Password:    vals["DB_PASSWORD"],
		Host:        host,
		Port:        port,
		TablePrefix: prefix,
	}, nil
}

func DiscoverAdministrators(ctx context.Context, siteDir string, client WPClient) ([]AdminUser, error) {
	stdout, stderr, err := client.Run(ctx, siteDir, "wp", []string{"user", "list", "--role=administrator", "--fields=ID,user_login,user_email", "--format=json"}, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list administrators: %w (%s)", err, strings.TrimSpace(stderr))
	}

	var users []AdminUser
	if err := json.Unmarshal([]byte(stdout), &users); err != nil {
		return nil, fmt.Errorf("failed to parse administrators output: %w", err)
	}

	if len(users) == 0 {
		return nil, errors.New("no users with administrator role found on this website")
	}

	return users, nil
}

func UpdateAdminCredentials(ctx context.Context, siteDir string, dbCfg *DBConfig, input AdminInput, client WPClient, connector DBConnector, onProgress ProgressFunc) error {
	if input.UserID <= 0 {
		return fmt.Errorf("invalid user ID %d", input.UserID)
	}

	if connector == nil {
		connector = DefaultDBConnector
	}

	totalSteps := 0
	if input.NewUsername != "" {
		totalSteps++
	}
	if input.NewPassword != "" {
		totalSteps++
	}
	if input.NewEmail != "" {
		totalSteps++
	}

	currentStep := 0

	// 1. Update username directly in MySQL if provided
	if input.NewUsername != "" {
		currentStep++
		if onProgress != nil {
			onProgress(currentStep, totalSteps, fmt.Sprintf("Updating username to %q in database...", input.NewUsername))
		}

		if !validTablePrefixRe.MatchString(dbCfg.TablePrefix) {
			return fmt.Errorf("invalid table_prefix %q", dbCfg.TablePrefix)
		}

		addr := fmt.Sprintf("%s:%d", dbCfg.Host, dbCfg.Port)
		dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?timeout=5s", dbCfg.User, dbCfg.Password, addr, dbCfg.Name)

		db, err := connector("mysql", dsn)
		if err != nil {
			return fmt.Errorf("failed to open database connection: %w", err)
		}
		defer db.Close()

		query := fmt.Sprintf("UPDATE `%susers` SET user_login = ?, user_nicename = ? WHERE ID = ?", dbCfg.TablePrefix)
		res, err := db.ExecContext(ctx, query, input.NewUsername, input.NewUsername, input.UserID)
		if err != nil {
			return fmt.Errorf("failed to update user_login in database: %w", err)
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to check rows affected: %w", err)
		}
		if rows == 0 {
			var count int
			checkQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%susers` WHERE ID = ?", dbCfg.TablePrefix)
			if err := db.QueryRowContext(ctx, checkQuery, input.UserID).Scan(&count); err != nil || count == 0 {
				return fmt.Errorf("user ID %d not found in database", input.UserID)
			}
		}
	}

	// 2. Update password if provided
	if input.NewPassword != "" {
		currentStep++
		if onProgress != nil {
			onProgress(currentStep, totalSteps, "Updating administrator password via WP-CLI...")
		}
		_, stderr, err := client.Run(ctx, siteDir, "wp", []string{"user", "update", strconv.Itoa(input.UserID), "--prompt=user_pass"}, input.NewPassword)
		if err != nil {
			return fmt.Errorf("failed to update password: %w (%s)", err, strings.TrimSpace(stderr))
		}
	}

	// 3. Update email if provided
	if input.NewEmail != "" {
		currentStep++
		if onProgress != nil {
			onProgress(currentStep, totalSteps, fmt.Sprintf("Updating email and site admin_email to %q...", input.NewEmail))
		}
		_, stderr, err := client.Run(ctx, siteDir, "wp", []string{"user", "update", strconv.Itoa(input.UserID), "--user_email=" + input.NewEmail}, "")
		if err != nil {
			return fmt.Errorf("failed to update user_email: %w (%s)", err, strings.TrimSpace(stderr))
		}

		if err := client.OptionUpdate(ctx, siteDir, "admin_email", input.NewEmail); err != nil {
			return fmt.Errorf("failed to synchronize site admin_email: %w", err)
		}
	}

	return nil
}

type TweakStatus struct {
	Tweak   config.WPTweak
	Success bool
	Err     error
}

func ApplyTweaks(ctx context.Context, siteDir string, tweaks []config.WPTweak, client WPClient, onProgress ProgressFunc) []TweakStatus {
	results := make([]TweakStatus, 0, len(tweaks))
	total := len(tweaks)
	for i, tw := range tweaks {
		desc := fmt.Sprintf("[%s] %s=%s", tw.Type, tw.Key, tw.Value)
		if onProgress != nil {
			onProgress(i+1, total, fmt.Sprintf("Applying tweak: %s", desc))
		}

		status := TweakStatus{Tweak: tw, Success: true}
		var err error
		switch tw.Type {
		case "config_set":
			err = client.ConfigSet(ctx, siteDir, tw.Key, tw.Value, tw.Raw)
		case "rewrite_structure":
			err = client.RewriteStructure(ctx, siteDir, tw.Value)
		case "option_update":
			err = client.OptionUpdate(ctx, siteDir, tw.Key, tw.Value)
		case "language_core":
			err = client.LanguageCore(ctx, siteDir, tw.Key, tw.Value)
		default:
			err = fmt.Errorf("unsupported tweak type %q", tw.Type)
		}
		if err != nil {
			status.Success = false
			status.Err = err
		}
		results = append(results, status)
	}
	return results
}

type PackageStatus struct {
	Slug       string
	Type       packages.PackageType
	Success    bool
	Skipped    bool
	SkipReason string
	Activated  bool
	Err        error
}

// CompareVersions compares two version strings (e.g. "6.8.10" vs "6.8.9").
// Returns 1 if v1 > v2, -1 if v1 < v2, and 0 if v1 == v2.
func CompareVersions(v1, v2 string) int {
	v1 = strings.TrimSpace(strings.TrimPrefix(v1, "v"))
	v2 = strings.TrimSpace(strings.TrimPrefix(v2, "v"))
	if v1 == v2 {
		return 0
	}

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			n1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			n2, _ = strconv.Atoi(parts2[i])
		}
		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
			return -1
		}
	}
	return 0
}

// GetInstalledVersion queries currently installed version of a plugin or theme.
func GetInstalledVersion(ctx context.Context, siteDir string, pkgType packages.PackageType, slug string, client WPClient) (string, bool) {
	args := []string{string(pkgType), "get", slug, "--field=version"}
	stdout, _, err := client.Run(ctx, siteDir, "wp", args, "")
	if err != nil {
		return "", false
	}
	ver := strings.TrimSpace(stdout)
	return ver, true
}

func InstallPackages(ctx context.Context, siteDir string, pkgType packages.PackageType, artifacts []packages.Artifact, activate bool, client WPClient, onProgress ProgressFunc) []PackageStatus {
	results := make([]PackageStatus, 0, len(artifacts))
	total := len(artifacts)
	for idx, art := range artifacts {
		slug := art.Ref.Slug
		targetVer := art.Version
		if onProgress != nil {
			onProgress(idx+1, total, fmt.Sprintf("Checking %s %q...", pkgType, slug))
		}

		status := PackageStatus{
			Slug:      slug,
			Type:      pkgType,
			Success:   true,
			Activated: activate,
		}

		pathOrSlug := art.Path
		if pathOrSlug == "" {
			pathOrSlug = slug
		}

		installedVer, installed := GetInstalledVersion(ctx, siteDir, pkgType, slug, client)
		force := false

		if installed {
			if targetVer != "" && installedVer != "" {
				cmp := CompareVersions(targetVer, installedVer)
				if cmp == 0 {
					status.Skipped = true
					status.SkipReason = fmt.Sprintf("Already up to date (v%s)", installedVer)
					results = append(results, status)
					continue
				} else if cmp < 0 {
					status.Skipped = true
					status.SkipReason = fmt.Sprintf("Current installed version is newer (v%s > target v%s)", installedVer, targetVer)
					results = append(results, status)
					continue
				} else {
					// targetVer > installedVer -> force install to upgrade
					force = true
				}
			} else {
				// Version unknown -> force install
				force = true
			}
		}

		if onProgress != nil {
			actionText := "Installing"
			if force {
				actionText = "Upgrading (force)"
			}
			onProgress(idx+1, total, fmt.Sprintf("%s %s %q...", actionText, pkgType, slug))
		}

		var err error
		args := []string{string(pkgType), "install", pathOrSlug}
		if force {
			args = append(args, "--force")
		}
		if activate && pkgType == packages.PackageTypePlugin {
			args = append(args, "--activate")
		} else if activate && pkgType == packages.PackageTypeTheme {
			args = append(args, "--activate")
		}

		_, stderr, runErr := client.Run(ctx, siteDir, "wp", args, "")
		if runErr != nil {
			err = fmt.Errorf("wp %s install failed: %w (%s)", pkgType, runErr, strings.TrimSpace(stderr))
		}

		if err != nil {
			status.Success = false
			status.Err = err
		}
		results = append(results, status)
	}
	return results
}
