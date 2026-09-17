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
		{Name: "WP Mail SMTP Pro", Slug: "wp-mail-smtp-pro", Type: "plugin"},
		{Name: "Xstore", Slug: "xstore", Type: "theme"},
	}

	// 1. Direct substring match
	res := packages.FilterCatalog(items, packages.PackageTypePlugin, "admin")
	if len(res) != 1 || res[0].Slug != "admin-site-enhancements-pro" {
		t.Errorf("expected admin-site-enhancements-pro, got %v", res)
	}

	// 2. Acronym / initialism match ("acf" -> "Advanced Custom Fields PRO")
	res = packages.FilterCatalog(items, packages.PackageTypePlugin, "acf")
	if len(res) != 1 || res[0].Slug != "advanced-custom-fields-pro" {
		t.Errorf("expected acf acronym match to return advanced-custom-fields-pro, got %v", res)
	}

	// 3. Hyphen initialism match ("smtp")
	res = packages.FilterCatalog(items, packages.PackageTypePlugin, "smtp")
	if len(res) != 1 || res[0].Slug != "wp-mail-smtp-pro" {
		t.Errorf("expected smtp match to return wp-mail-smtp-pro, got %v", res)
	}

	// 4. Empty query returns all of matching type
	res = packages.FilterCatalog(items, packages.PackageTypeTheme, "")
	if len(res) != 1 || res[0].Slug != "xstore" {
		t.Errorf("expected 1 theme, got %v", res)
	}
}

func TestDeduplicatePackages(t *testing.T) {
	refs := []packages.PackageRef{
		{Type: packages.PackageTypePlugin, Slug: "acf"},
		{Type: packages.PackageTypePlugin, Slug: "rank-math"},
		{Type: packages.PackageTypePlugin, Slug: "acf"},
		{Type: packages.PackageTypeTheme, Slug: "acf"},
	}

	deduped := packages.DeduplicatePackages(refs)
	if len(deduped) != 3 {
		t.Fatalf("expected 3 unique packages, got %d", len(deduped))
	}
}
