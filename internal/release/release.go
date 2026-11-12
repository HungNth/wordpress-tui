// Package release validates WPTUI application manifests and performs read-only Application Update Checks.
package release

import (
	"cmp"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	ExpectedProduct          = "wptui"
	DefaultRemoteManifestURL = "https://raw.githubusercontent.com/HungNth/wordpress-tui/main/manifest.json"
	ReleaseTagBaseURL        = "https://github.com/HungNth/wordpress-tui/releases/tag/v"
	DefaultCheckTimeout      = 5 * time.Second
	maxManifestBytes         = 64 * 1024
)

// Manifest represents the WPTUI application metadata contract.
type Manifest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// CheckOptions configures external boundaries for an Application Update Check.
type CheckOptions struct {
	ManifestURL string
	HTTPClient  *http.Client
	Timeout     time.Duration
}

// CheckResult describes the outcome of comparing the embedded release with the remote manifest.
type CheckResult struct {
	CurrentVersion  string
	RemoteVersion   string
	UpdateAvailable bool
	ReleaseURL      string
}

// ParseManifest decodes and validates a WPTUI application manifest.
func ParseManifest(raw []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return Manifest{}, fmt.Errorf("application manifest is invalid JSON: %w", err)
	}
	if m.Name != ExpectedProduct {
		return Manifest{}, fmt.Errorf("application manifest has unexpected product %q", m.Name)
	}
	if !isStrictSemVer(m.Version) {
		return Manifest{}, fmt.Errorf("application manifest version %q is not valid semantic versioning", m.Version)
	}
	return m, nil
}

// CheckUpdate compares the embedded manifest against the remote repository manifest.
func CheckUpdate(ctx context.Context, localManifestRaw []byte, opts CheckOptions) (CheckResult, error) {
	localManifest, err := ParseManifest(localManifestRaw)
	if err != nil {
		return CheckResult{}, err
	}

	timeout := cmp.Or(opts.Timeout, DefaultCheckTimeout)
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	manifestURL := cmp.Or(opts.ManifestURL, DefaultRemoteManifestURL)
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return CheckResult{}, fmt.Errorf("failed to create update check request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{}, fmt.Errorf("update check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return CheckResult{}, fmt.Errorf("update check returned HTTP %d", resp.StatusCode)
	}

	rawRemote, err := io.ReadAll(io.LimitReader(resp.Body, maxManifestBytes+1))
	if err != nil {
		return CheckResult{}, fmt.Errorf("failed to read remote manifest: %w", err)
	}
	if len(rawRemote) > maxManifestBytes {
		return CheckResult{}, errors.New("remote manifest exceeds size limit")
	}

	remoteManifest, err := ParseManifest(rawRemote)
	if err != nil {
		return CheckResult{}, fmt.Errorf("remote %w", err)
	}

	if semver.Compare("v"+remoteManifest.Version, "v"+localManifest.Version) > 0 {
		return CheckResult{
			CurrentVersion:  localManifest.Version,
			RemoteVersion:   remoteManifest.Version,
			UpdateAvailable: true,
			ReleaseURL:      ReleaseTagBaseURL + remoteManifest.Version,
		}, nil
	}

	return CheckResult{
		CurrentVersion: localManifest.Version,
		RemoteVersion:  remoteManifest.Version,
	}, nil
}

func isStrictSemVer(v string) bool {
	if v == "" || strings.HasPrefix(v, "v") {
		return false
	}
	core, _, _ := strings.Cut(v, "+")
	core, _, _ = strings.Cut(core, "-")
	if strings.Count(core, ".") != 2 {
		return false
	}
	return semver.IsValid("v" + v)
}
