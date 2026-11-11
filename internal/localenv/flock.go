package localenv

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

type flockEnv struct {
	cmd string
}

func (f *flockEnv) Name() string {
	return "flock"
}

func (f *flockEnv) Command() string {
	return f.cmd
}

func (f *flockEnv) CheckDependencies(ctx context.Context, runner Runner) error {
	return checkCommonDependencies(runner, f.cmd)
}

func (f *flockEnv) SiteURL(slug string) string {
	return fmt.Sprintf("https://%s.test", slug)
}

func (f *flockEnv) Register(ctx context.Context, runner Runner, siteDir, slug string) (bool, error) {
	stdout, stderr, err := runner.Run(ctx, "", f.cmd, []string{"sites"}, "")
	if err != nil {
		return false, fmt.Errorf("flock sites failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}

	sites, err := parseFlockSites(stdout)
	if err != nil {
		return false, fmt.Errorf("failed to parse flock sites table: %w", err)
	}

	if s, found := findTargetSite(sites, slug); found {
		if err := validateTargetRow(*s); err != nil {
			return false, fmt.Errorf("unparseable target row for %q: %w", slug, err)
		}
		if pathsEqual(s.Path, siteDir) {
			// Site already mapped to the same path (parked or linked); preserve existing registration.
			return false, nil
		}
		// Name collision: slug is mapped to a different path elsewhere.
		return false, fmt.Errorf("site %q is already registered to %q", slug, s.Path)
	}

	_, stderr, err = runner.Run(ctx, siteDir, f.cmd, []string{"link", slug}, "")
	if err != nil {
		// Upstream flock commits config before daemon commit can error; registration may exist.
		return true, fmt.Errorf("flock link failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return true, nil
}

func (f *flockEnv) Secure(ctx context.Context, runner Runner, siteDir, slug string) error {
	_, stderr, err := runner.Run(ctx, siteDir, f.cmd, []string{"secure", slug}, "")
	if err != nil {
		return fmt.Errorf("flock secure failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return nil
}

func (f *flockEnv) Unsecure(ctx context.Context, runner Runner, siteDir, slug string) error {
	stdout, stderr, err := runner.Run(ctx, "", f.cmd, []string{"sites"}, "")
	if err != nil {
		return fmt.Errorf("flock sites failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}

	sites, err := parseFlockSites(stdout)
	if err != nil {
		return fmt.Errorf("failed to parse flock sites table: %w", err)
	}

	site, found := findTargetSite(sites, slug)
	if !found {
		// Missing site is a harmless no-op during cleanup.
		return nil
	}

	if err := validateTargetRow(*site); err != nil {
		return fmt.Errorf("cannot unsecure site %q: %w", slug, err)
	}

	if !pathsEqual(site.Path, siteDir) {
		return fmt.Errorf("cannot unsecure site %q: registered path %q does not match %q", slug, site.Path, siteDir)
	}

	_, stderr, err = runner.Run(ctx, siteDir, f.cmd, []string{"unsecure", slug}, "")
	if err != nil {
		return fmt.Errorf("flock unsecure failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return nil
}

func (f *flockEnv) Unregister(ctx context.Context, runner Runner, siteDir, slug string) error {
	stdout, stderr, err := runner.Run(ctx, "", f.cmd, []string{"sites"}, "")
	if err != nil {
		return fmt.Errorf("flock sites failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}

	sites, err := parseFlockSites(stdout)
	if err != nil {
		return fmt.Errorf("failed to parse flock sites table: %w", err)
	}

	site, found := findTargetSite(sites, slug)
	if !found {
		// Missing site is a harmless no-op during cleanup.
		return nil
	}

	if err := validateTargetRow(*site); err != nil {
		return fmt.Errorf("cannot unregister site %q: %w", slug, err)
	}

	if !pathsEqual(site.Path, siteDir) {
		return fmt.Errorf("cannot unregister site %q: registered path %q does not match %q", slug, site.Path, siteDir)
	}

	_, stderr, err = runner.Run(ctx, siteDir, f.cmd, []string{"unlink", slug}, "")
	if err != nil {
		return fmt.Errorf("flock unlink failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}
	return nil
}

func (f *flockEnv) IsSiteSecured(ctx context.Context, runner Runner, slug string) (bool, error) {
	stdout, stderr, err := runner.Run(ctx, "", f.cmd, []string{"sites"}, "")
	if err != nil {
		return false, fmt.Errorf("flock sites failed: %w (output: %s)", err, strings.TrimSpace(stderr))
	}

	sites, err := parseFlockSites(stdout)
	if err != nil {
		return false, fmt.Errorf("failed to parse flock sites table: %w", err)
	}

	if s, found := findTargetSite(sites, slug); found {
		if err := validateTargetRow(*s); err != nil {
			return false, fmt.Errorf("cannot probe TLS for site %q: %w", slug, err)
		}
		u, err := url.Parse(s.URL)
		if err != nil {
			return false, fmt.Errorf("invalid URL %q for site %q: %w", s.URL, slug, err)
		}
		return u.Scheme == "https", nil
	}

	return false, nil
}

type flockSiteRow struct {
	Site string
	PHP  string
	URL  string
	Root string
	Path string
	Err  error
}

var reColSep = regexp.MustCompile(`(?: {2,}|\t+)`)

func findTargetSite(sites []flockSiteRow, slug string) (*flockSiteRow, bool) {
	for i := range sites {
		if strings.EqualFold(sites[i].Site, slug) {
			return &sites[i], true
		}
	}
	return nil, false
}

func parseFlockSites(stdout string) ([]flockSiteRow, error) {
	lines := strings.Split(stdout, "\n")
	headerIdx := -1

	for i, l := range lines {
		trimmed := strings.TrimRight(l, "\r\n")
		parts := reColSep.Split(strings.TrimSpace(trimmed), 5)
		if len(parts) == 5 && parts[0] == "SITE" && parts[1] == "PHP" && parts[2] == "URL" && parts[3] == "ROOT" && parts[4] == "PATH" {
			headerIdx = i
			break
		}
	}

	if headerIdx == -1 {
		return nil, errors.New("flock sites output missing valid table header")
	}

	var rows []flockSiteRow
	for _, rawLine := range lines[headerIdx+1:] {
		trimmed := strings.TrimRight(rawLine, "\r\n")
		if strings.TrimSpace(trimmed) == "" {
			continue
		}

		parts := reColSep.Split(strings.TrimSpace(trimmed), 5)
		var row flockSiteRow
		if len(parts) > 0 {
			row.Site = strings.TrimSpace(parts[0])
		}

		if len(parts) < 5 || row.Site == "" || strings.TrimSpace(parts[4]) == "" {
			row.Err = fmt.Errorf("malformed site row: %q", trimmed)
		} else {
			row.PHP = strings.TrimSpace(parts[1])
			row.URL = strings.TrimSpace(parts[2])
			row.Root = strings.TrimSpace(parts[3])
			row.Path = strings.TrimSpace(parts[4])
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func validateTargetRow(r flockSiteRow) error {
	if r.Err != nil {
		return r.Err
	}
	if r.Site == "" {
		return errors.New("empty site name")
	}
	if r.Path == "" {
		return errors.New("empty site path")
	}
	if strings.HasPrefix(r.URL, "⚠") {
		return fmt.Errorf("site reported error: %s", r.URL)
	}
	u, err := url.Parse(r.URL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid site URL: %q", r.URL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported site URL scheme: %q", u.Scheme)
	}
	return nil
}

func pathsEqual(p1, p2 string) bool {
	c1 := filepath.Clean(filepath.FromSlash(p1))
	c2 := filepath.Clean(filepath.FromSlash(p2))
	if runtime.GOOS == "windows" {
		return strings.EqualFold(c1, c2)
	}
	return c1 == c2
}
