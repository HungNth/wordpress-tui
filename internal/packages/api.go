package packages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// buildBaseRelativeURL normalizes the base URL with a trailing slash and resolves the relative path.
func buildBaseRelativeURL(baseURL, relPath, licenseKey string) (*url.URL, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("packages_api_url is empty")
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base url %q: %w", baseURL, err)
	}

	rel, err := url.Parse(relPath)
	if err != nil {
		return nil, err
	}

	full := base.ResolveReference(rel)
	if licenseKey != "" {
		q := full.Query()
		q.Set("license_key", licenseKey)
		full.RawQuery = q.Encode()
	}
	return full, nil
}

// BuildMetadataURL resolves the relative metadata path against the normalized trailing-slash base URL.
func BuildMetadataURL(baseURL, slug, licenseKey string) (*url.URL, error) {
	relPath := fmt.Sprintf("package/%s/metadata", url.PathEscape(slug))
	return buildBaseRelativeURL(baseURL, relPath, licenseKey)
}

// SanitizeURL replaces any query string with [REDACTED] to prevent logging credentials.
func SanitizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "[MALFORMED_URL]"
	}
	if u.RawQuery != "" {
		u.RawQuery = "[REDACTED]"
	}
	return u.String()
}

// SanitizedURLError wraps a url.Error with a redacted URL while preserving the error cause chain.
type SanitizedURLError struct {
	Op  string
	URL string
	Err error
}

func (e *SanitizedURLError) Error() string {
	return fmt.Sprintf("%s %s: %v", e.Op, e.URL, e.Err)
}

func (e *SanitizedURLError) Unwrap() error {
	return e.Err
}

func (e *SanitizedURLError) Timeout() bool {
	var netErr net.Error
	if errors.As(e.Err, &netErr) {
		return netErr.Timeout()
	}
	return errors.Is(e.Err, context.DeadlineExceeded)
}

// SanitizeError strips sensitive queries from url.Error values while preserving unwrap and timeout capabilities.
func SanitizeError(err error) error {
	if err == nil {
		return nil
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return &SanitizedURLError{
			Op:  urlErr.Op,
			URL: SanitizeURL(urlErr.URL),
			Err: urlErr.Err,
		}
	}
	return err
}

// FetchMetadata retrieves and validates package metadata without leaking secrets in error bodies.
func FetchMetadata(ctx context.Context, client *http.Client, baseURL, expectedType, slug, licenseKey string) (*Metadata, error) {
	u, err := BuildMetadataURL(baseURL, slug, licenseKey)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata request: %w", err)
	}

	if client == nil {
		client = &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("stopped after 5 redirects")
				}
				if req.URL.Scheme != "https" {
					return fmt.Errorf("redirect to non-HTTPS URL %s rejected", SanitizeURL(req.URL.String()))
				}
				req.Header.Del("Referer")
				if len(via) > 0 && req.URL.Host != via[0].URL.Host {
					q := req.URL.Query()
					q.Del("license_key")
					req.URL.RawQuery = q.Encode()
				}
				return nil
			},
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("metadata request failed: %w", SanitizeError(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("metadata request for package %q failed", slug)}
	}

	var meta Metadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, &MetadataContractError{Err: fmt.Errorf("invalid metadata JSON: %w", err)}
	}

	if meta.Slug != slug {
		return nil, &MetadataContractError{Err: fmt.Errorf("metadata slug %q does not match requested slug %q", meta.Slug, slug)}
	}
	if meta.Type != expectedType {
		return nil, &MetadataContractError{Err: fmt.Errorf("metadata type %q does not match requested type %q", meta.Type, expectedType)}
	}
	if strings.TrimSpace(meta.Version) == "" {
		return nil, &MetadataContractError{Err: errors.New("metadata version is empty")}
	}

	sizeInt, err := strconv.ParseInt(strings.TrimSpace(meta.Size), 10, 64)
	if err != nil || sizeInt <= 0 {
		return nil, &MetadataContractError{Err: fmt.Errorf("invalid metadata size: %q", meta.Size)}
	}
	if sizeInt > MaxArtifactCap {
		return nil, &MetadataContractError{Err: fmt.Errorf("metadata size %d exceeds 1 GiB limit", sizeInt)}
	}

	downURL, err := url.Parse(meta.DownloadURL)
	if err != nil || downURL.Scheme != "https" || downURL.Hostname() == "" {
		return nil, &MetadataContractError{Err: fmt.Errorf("invalid download URL %s (must be HTTPS with valid host)", SanitizeURL(meta.DownloadURL))}
	}

	return &meta, nil
}
