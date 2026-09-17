package tui_test

import (
	"testing"

	"wptui/internal/deprovision"
	"wptui/internal/tui"
)

func TestBuildDeleteSelectionForm(t *testing.T) {
	candidates := []deprovision.Candidate{
		{Slug: "site-one", Path: "/sites/site-one", DetectedDB: "site_one_db"},
		{Slug: "empty-site", Path: "/sites/empty-site", DetectedDB: ""},
	}

	var selectedSlugs []string
	form := tui.BuildDeleteSelectionForm(candidates, &selectedSlugs)
	if form == nil {
		t.Fatal("expected non-nil form from BuildDeleteSelectionForm")
	}
}

func TestBuildDeleteConfirmMultiForm(t *testing.T) {
	selected := []deprovision.Candidate{
		{Slug: "site-one", Path: "/sites/site-one", DetectedDB: "site_one_db"},
	}

	var confirmed bool
	form := tui.BuildDeleteConfirmMultiForm(selected, &confirmed)
	if form == nil {
		t.Fatal("expected non-nil form from BuildDeleteConfirmMultiForm")
	}

	if confirmed != false {
		t.Errorf("expected confirmed to default to false, got true")
	}
}
