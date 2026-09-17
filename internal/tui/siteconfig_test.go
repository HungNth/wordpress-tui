package tui_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"wptui/internal/config"
	"wptui/internal/siteconfig"
	"wptui/internal/tui"
)

func TestSelectWebsiteForConfig_Empty(t *testing.T) {
	_, err := tui.SelectWebsiteForConfig(nil)
	if err == nil {
		t.Fatal("expected error on empty candidates list, got nil")
	}
}

func TestSelectAdminUser_Single(t *testing.T) {
	admins := []siteconfig.AdminUser{
		{ID: 1, UserLogin: "admin1", UserEmail: "admin1@test.com"},
	}
	selected, err := tui.SelectAdminUser(admins)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected == nil || selected.ID != 1 {
		t.Errorf("expected admin 1, got %+v", selected)
	}
}

func TestSelectAdminUser_Empty(t *testing.T) {
	_, err := tui.SelectAdminUser(nil)
	if err == nil {
		t.Fatal("expected error on empty administrators list, got nil")
	}
}

func captureStdout(fn func()) string {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestPrintTweakSummary(t *testing.T) {
	results := []siteconfig.TweakStatus{
		{
			Tweak:   config.WPTweak{Type: "config_set", Key: "WP_DEBUG", Value: "true"},
			Success: true,
		},
		{
			Tweak:   config.WPTweak{Type: "rewrite_structure", Value: "/%postname%/"},
			Success: false,
			Err:     errors.New("timeout"),
		},
	}

	out := captureStdout(func() {
		tui.PrintTweakSummary(results)
	})

	if !strings.Contains(out, "WP Tweaks Summary") {
		t.Errorf("expected summary title, got %q", out)
	}
	if !strings.Contains(out, "WP_DEBUG=true") {
		t.Errorf("expected tweak 1 in output, got %q", out)
	}
	if !strings.Contains(out, "timeout") {
		t.Errorf("expected error in output, got %q", out)
	}
}

func TestPrintPackageInstallSummary(t *testing.T) {
	results := []siteconfig.PackageStatus{
		{
			Slug:      "test-plugin",
			Type:      "plugin",
			Success:   true,
			Activated: true,
		},
		{
			Slug:      "test-theme",
			Type:      "theme",
			Success:   false,
			Activated: false,
			Err:       errors.New("failed download"),
		},
	}

	out := captureStdout(func() {
		tui.PrintPackageInstallSummary(results)
	})

	if !strings.Contains(out, "Package Installation Summary") {
		t.Errorf("expected summary title, got %q", out)
	}
	if !strings.Contains(out, "test-plugin (Activated)") {
		t.Errorf("expected activated plugin in output, got %q", out)
	}
	if !strings.Contains(out, "failed download") {
		t.Errorf("expected failure error in output, got %q", out)
	}
}
