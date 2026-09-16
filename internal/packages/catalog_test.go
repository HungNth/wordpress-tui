package packages_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wptui/internal/packages"
)

func TestFetchCatalog_PreservesPathAndFiltersGeneric(t *testing.T) {
	fixtureData := []packages.CatalogItem{
		{Name: "66Uptime", Slug: "66uptime", Type: "generic"},
		{Name: "Advanced Custom Fields PRO", Slug: "advanced-custom-fields-pro", Type: "plugin"},
		{Name: "Xstore", Slug: "xstore", Type: "theme"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/packages" {
			t.Errorf("expected path /api/v1/packages, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("license_key") != "my_key" {
			t.Errorf("expected license_key my_key, got %s", r.URL.Query().Get("license_key"))
		}

		resp := packages.CatalogResponse{
			Total: len(fixtureData),
			Data:  fixtureData,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	baseURL := server.URL + "/api/v1"
	items, err := packages.FetchCatalog(context.Background(), server.Client(), baseURL, "my_key")
	if err != nil {
		t.Fatalf("FetchCatalog failed: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items after filtering generic, got %d", len(items))
	}

	// Verify generic 66Uptime is excluded
	for _, it := range items {
		if it.Type == "generic" {
			t.Errorf("generic item was not filtered out: %+v", it)
		}
	}
}

func TestFilterCatalog(t *testing.T) {
	items := []packages.CatalogItem{
		{Name: "Advanced Custom Fields PRO", Slug: "advanced-custom-fields-pro", Type: "plugin"},
		{Name: "Admin and Site Enhancements (ASE) Pro", Slug: "admin-site-enhancements-pro", Type: "plugin"},
		{Name: "Xstore", Slug: "xstore", Type: "theme"},
	}

	plugins := packages.FilterCatalog(items, "plugin", "admin")
	if len(plugins) != 1 || plugins[0].Slug != "admin-site-enhancements-pro" {
		t.Errorf("expected 1 admin plugin, got %v", plugins)
	}

	themes := packages.FilterCatalog(items, "theme", "XSTORE")
	if len(themes) != 1 || themes[0].Slug != "xstore" {
		t.Errorf("expected 1 theme case-insensitive match, got %v", themes)
	}
}

func TestDeduplicatePackages(t *testing.T) {
	refs := []packages.PackageRef{
		{Type: "plugin", Slug: "acf"},
		{Type: "plugin", Slug: "rank-math"},
		{Type: "plugin", Slug: "acf"}, // duplicate
		{Type: "theme", Slug: "acf"},  // different type
	}

	deduped := packages.DeduplicatePackages(refs)
	if len(deduped) != 3 {
		t.Fatalf("expected 3 unique packages, got %d", len(deduped))
	}
}
