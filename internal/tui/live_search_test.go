package tui_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/packages"
	"wptui/internal/tui"
)

func TestLiveSearchModel_InteractiveMultiQueryFlow(t *testing.T) {
	catalog := []packages.CatalogItem{
		{Name: "Advanced Custom Fields PRO", Slug: "advanced-custom-fields-pro", Type: "plugin"},
		{Name: "Admin and Site Enhancements (ASE) Pro", Slug: "admin-site-enhancements-pro", Type: "plugin"},
		{Name: "WP Mail SMTP Pro", Slug: "wp-mail-smtp-pro", Type: "plugin"},
		{Name: "Rank Math SEO PRO", Slug: "seo-by-rank-math-pro", Type: "plugin"},
	}

	initialSelected := []string{"default-plugin"}
	m := tui.NewLiveSearchModel(packages.PackageTypePlugin, catalog, initialSelected)

	// 1. Initially without typing, default selection is preserved
	if len(m.FinalSelected()) != 1 || m.FinalSelected()[0] != "default-plugin" {
		t.Fatalf("expected initial selection preserved, got %v", m.FinalSelected())
	}

	// 2. Type "acf"
	for _, ch := range "acf" {
		newM, _ := m.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
		m = newM.(*tui.LiveSearchModel)
	}
	rawView := m.ViewString()
	if !strings.Contains(rawView, "advanced-custom-fields-pro") {
		t.Errorf("expected view to contain acf after typing, got:\n%s", rawView)
	}

	// 3. Press Space to select "advanced-custom-fields-pro"
	newM, _ := m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	m = newM.(*tui.LiveSearchModel)
	final := m.FinalSelected()
	if len(final) != 2 || final[0] != "default-plugin" || final[1] != "advanced-custom-fields-pro" {
		t.Errorf("expected [default-plugin, advanced-custom-fields-pro], got %v", final)
	}

	// 4. Backspace query to clear "acf"
	for range 3 {
		newM, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
		m = newM.(*tui.LiveSearchModel)
	}

	// 5. Type "smtp"
	for _, ch := range "smtp" {
		newM, _ = m.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
		m = newM.(*tui.LiveSearchModel)
	}
	view := m.ViewString()
	if !strings.Contains(view, "wp-mail-smtp-pro") {
		t.Errorf("expected view to contain wp-mail-smtp-pro, got:\n%s", view)
	}

	// 6. Press Space to select "wp-mail-smtp-pro"
	newM, _ = m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	m = newM.(*tui.LiveSearchModel)
	final = m.FinalSelected()
	// MUST retain all 3: default-plugin, advanced-custom-fields-pro, wp-mail-smtp-pro
	if len(final) != 3 || final[0] != "default-plugin" || final[1] != "advanced-custom-fields-pro" || final[2] != "wp-mail-smtp-pro" {
		t.Errorf("expected persistent accumulation of all 3 selections, got %v", final)
	}

	// 7. Enter non-matching query e.g. "xyz123"
	for _, ch := range "xyz123" {
		newM, _ = m.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
		m = newM.(*tui.LiveSearchModel)
	}
	view = m.ViewString()
	if !strings.Contains(view, "No matching packages found") {
		t.Errorf("expected no matching packages message, got:\n%s", view)
	}

	// 8. Press Enter on no matches: does NOT select any ghost package, confirms existing accumulated
	newM, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newM.(*tui.LiveSearchModel)
	if m.IsAborted() {
		t.Errorf("expected enter to submit, not abort")
	}
	final = m.FinalSelected()
	if len(final) != 3 {
		t.Errorf("expected exactly 3 selections after enter on empty results, got %v", final)
	}
}

func TestLiveSearchModel_EscapeAborts(t *testing.T) {
	m := tui.NewLiveSearchModel(packages.PackageTypePlugin, nil, []string{"init"})
	newM, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = newM.(*tui.LiveSearchModel)
	if !m.IsAborted() {
		t.Errorf("expected Esc to abort search")
	}
}
