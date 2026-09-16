package packages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type CatalogItem struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Version     string `json:"version"`
	Homepage    string `json:"homepage"`
	Author      string `json:"author"`
	Type        string `json:"type"`
	LastUpdated string `json:"last_updated"`
}

type CatalogResponse struct {
	Total int           `json:"total"`
	Data  []CatalogItem `json:"data"`
}

// BuildCatalogURL resolves the relative `packages` reference against the normalized base URL.
func BuildCatalogURL(baseURL, licenseKey string) (*url.URL, error) {
	return buildBaseRelativeURL(baseURL, "packages", licenseKey)
}

// FetchCatalog retrieves the full package catalog from the API and removes generic items.
func FetchCatalog(ctx context.Context, client *http.Client, baseURL, licenseKey string) ([]CatalogItem, error) {
	u, err := BuildCatalogURL(baseURL, licenseKey)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create catalog request: %w", err)
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
		return nil, fmt.Errorf("catalog request failed: %w", SanitizeError(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog request returned HTTP status %d", resp.StatusCode)
	}

	var catResp CatalogResponse
	if err := json.NewDecoder(resp.Body).Decode(&catResp); err != nil {
		return nil, fmt.Errorf("invalid catalog response JSON: %w", err)
	}

	var filtered []CatalogItem
	for _, item := range catResp.Data {
		if item.Type == "generic" {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

// FilterCatalog filters catalog items in memory by type and search query (case-insensitive).
func FilterCatalog(items []CatalogItem, expectedType PackageType, query string) []CatalogItem {
	q := strings.ToLower(strings.TrimSpace(query))
	var out []CatalogItem
	for _, it := range items {
		if it.Type != string(expectedType) {
			continue
		}
		if q == "" || strings.Contains(strings.ToLower(it.Name), q) || strings.Contains(strings.ToLower(it.Slug), q) {
			out = append(out, it)
		}
	}
	return out
}

// DeduplicatePackages removes duplicate references preserving order.
func DeduplicatePackages(refs []PackageRef) []PackageRef {
	seen := make(map[string]bool)
	var out []PackageRef
	for _, r := range refs {
		key := string(r.Type) + ":" + r.Slug
		if !seen[key] {
			seen[key] = true
			out = append(out, r)
		}
	}
	return out
}
