package tui_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
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

func TestPrintDeleteSummary_ColorOutput(t *testing.T) {
	results := []deprovision.Result{
		{
			Candidate: deprovision.Candidate{Slug: "test-site", Path: "/sites/test-site", DetectedDB: "test_db"},
			HerdDone:  true,
			DBDone:    true,
			DirDone:   true,
		},
		{
			Candidate: deprovision.Candidate{Slug: "fail-site", Path: "/sites/fail-site"},
			HerdErr:   errors.New("stale cert"),
			DirErr:    errors.New("permission denied"),
		},
	}
	// Capture stdout
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	tui.PrintDeleteSummary(results)

	_ = w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Assert rendered ANSI color tokens for Success (Green), Warning (Yellow), and Error (Red)
	expectedSuccess := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true).Render("unsecured")
	expectedWarning := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Bold(true).Render("warning: stale cert")
	expectedError := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444")).Bold(true).Render("error: permission denied")
	expectedPath := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Bold(true).Render("test-site")

	if !strings.Contains(output, expectedSuccess) {
		t.Errorf("expected summary to contain green styled %q, got:\n%s", expectedSuccess, output)
	}
	if !strings.Contains(output, expectedWarning) {
		t.Errorf("expected summary to contain yellow styled %q, got:\n%s", expectedWarning, output)
	}
	if !strings.Contains(output, expectedError) {
		t.Errorf("expected summary to contain red styled %q, got:\n%s", expectedError, output)
	}
	if !strings.Contains(output, expectedPath) {
		t.Errorf("expected summary to contain cyan styled slug %q, got:\n%s", expectedPath, output)
	}
}
