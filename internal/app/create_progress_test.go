package app_test

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/app"
	"wptui/internal/config"
	"wptui/internal/create"
	"wptui/internal/tui"
)

func TestCreateProgress_StepProgressionAndSummary(t *testing.T) {
	cfg := &config.Config{
		DefaultAdminUsername: "admin",
		WPTweaks: []config.WPTweak{
			{Type: config.TweakTypeOptionUpdate, Key: "blogdescription", Value: "Just another WP site"},
		},
		AppName:        config.AppNameHerd,
		AppCommand:     "herd",
		PackagesAPIURL: "https://api.example.com",
	}

	inputs := tui.CreateInputs{
		WebsiteName:   "test 02",
		WebsiteSlug:   "test02",
		AdminUsername: "admin",
		ApplyTweaks:   true,
	}
	pkgs := []string{"admin-site-enhancements-pro", "advanced-custom-fields-pro"}

	steps := app.BuildCreateSteps(inputs, pkgs, cfg)

	// Verify steps include tweaks, packages, and herd
	stepIDs := make([]string, len(steps))
	for i, s := range steps {
		stepIDs[i] = s.ID
	}
	expectedIDs := []string{"resolve", "register", "download", "database", "install", "tweaks", "packages", "environment"}
	if strings.Join(stepIDs, ",") != strings.Join(expectedIDs, ",") {
		t.Fatalf("expected step IDs %v, got %v", expectedIDs, stepIDs)
	}

	monitor := tui.NewProgressMonitorModel("Provisioning test 02", steps)
	monitor.SetSize(80, 24)

	ch := make(chan tea.Msg, 100)
	tracker := app.NewCreateProgressTracker(ch, steps)

	// Simulate sequence from create.go as seen in the user's screenshot
	ch <- tui.StepStartMsg{ID: "resolve", Title: "Resolving packages..."}
	ch <- tui.StepCompleteMsg{ID: "resolve", Status: tui.StepStatusSuccess, Detail: "Resolved"}

	progress := tracker.StepFunc()
	progress("create_directory", "Creating website directory...")
	progress("environment_register", "Registering environment...")
	progress("core_extract", "Extracting WordPress core...")
	progress("config_create", "Generating wp-config.php...")
	progress("db_create", "Creating database...")
	progress("core_install", "Running WordPress installation...")
	progress("tweaks", "Applying WordPress tweaks...")
	progress("theme_install", "Installing default theme flatsome...")
	progress("plugin_install", "Installing plugin admin-site-enhancements-pro...")
	progress("plugin_install", "Installing plugin advanced-custom-fields-pro...")
	progress("environment_secure", "Securing site with environment TLS...")

	// Successful finish
	tracker.CompleteAllSuccess()

	result := &create.Result{
		WebsitePath:      "F:/laravel-herd/wordpress/test02",
		WebsiteURL:       "https://test02.test",
		ActiveTheme:      "flatsome",
		InstalledPlugins: []string{"admin-site-enhancements-pro", "advanced-custom-fields-pro"},
	}

	ch <- tui.OperationCompleteMsg{
		Title:   "Website Provisioned Successfully",
		Success: true,
		Summary: app.FormatCreateSummary(inputs, result, cfg.DefaultAdminUsername),
	}

	// Drain messages into monitor
	close(ch)
	for msg := range ch {
		monitor.Update(msg)
	}

	view := monitor.Render(80, 24)

	// 1. Database step MUST NOT be stuck in running state
	if strings.Contains(view, "Creating database...") && !strings.Contains(view, "[✓]") {
		t.Errorf("database step is still stuck in running state:\n%s", view)
	}

	// 2. All 7 steps must be ticked [✓] and none pending [ ]
	if strings.Contains(view, "[ ]") {
		t.Errorf("found unticked [ ] steps in view:\n%s", view)
	}
	checkCount := strings.Count(view, "[✓]")
	if checkCount != 8 {
		t.Errorf("expected 8 completed [✓] steps, got %d:\n%s", checkCount, view)
	}

	// 3. Summary details must be present and formatted
	if !strings.Contains(view, "https://test02.test") {
		t.Errorf("Summary URL not found in view:\n%s", view)
	}
	if !strings.Contains(view, "flatsome") {
		t.Errorf("Summary Active Theme not found in view:\n%s", view)
	}
	if !strings.Contains(view, "admin-site-enhancements-pro, advanced-custom-fields-pro") {
		t.Errorf("Summary Installed Plugins not found in view:\n%s", view)
	}
	if !strings.Contains(view, "Press [ Enter / Esc ] to return") {
		t.Errorf("Return prompt not found in view:\n%s", view)
	}

	// 4. View must fit within 24 rows without overflow
	renderedLines := len(strings.Split(strings.TrimRight(view, "\n"), "\n"))
	if renderedLines > 24 {
		t.Errorf("expected rendered view to fit in 24 lines, got %d lines:\n%s", renderedLines, view)
	}
}

func TestBuildCreateSteps_OmitsInactiveSteps(t *testing.T) {
	cfg := &config.Config{
		AppName: "",
	}
	inputs := tui.CreateInputs{
		ApplyTweaks: false,
	}
	steps := app.BuildCreateSteps(inputs, nil, cfg)

	for _, s := range steps {
		if s.ID == "tweaks" {
			t.Errorf("tweaks step should be omitted when ApplyTweaks is false")
		}
		if s.ID == "packages" {
			t.Errorf("packages step should be omitted when no packages chosen")
		}
		if s.ID == "environment" {
			t.Errorf("environment step should be omitted when AppName is empty")
		}
	}
}
func TestCreateProgress_EarlyRegistrationFailure_LaragonNoFinalTLS(t *testing.T) {
	laragonCfg := &config.Config{
		AppName:    config.AppNameLaragon,
		AppCommand: "laragon.exe",
	}
	inputs := tui.CreateInputs{
		WebsiteName: "Laragon Site",
		WebsiteSlug: "laragon-site",
	}
	steps := app.BuildCreateSteps(inputs, nil, laragonCfg)

	// 1. Verify Laragon has NO final "environment" TLS step
	for _, s := range steps {
		if s.ID == "environment" {
			t.Errorf("Laragon should omit final environment TLS step, found in steps: %v", steps)
		}
	}

	ch := make(chan tea.Msg, 50)
	tracker := app.NewCreateProgressTracker(ch, steps)
	prog := tracker.StepFunc()

	// Simulate early registration failure
	prog("create_directory", "Creating website directory...")
	prog("environment_register", "Registering environment...")
	tracker.MarkFailed(errors.New("registration failed"))

	close(ch)
	var msgs []tea.Msg
	for m := range ch {
		msgs = append(msgs, m)
	}

	monitor := tui.NewProgressMonitorModel("Provisioning Laragon Site", steps)
	monitor.SetSize(80, 24)
	for _, m := range msgs {
		monitor.Update(m)
	}
	var registerFailed bool
	var downloadStartedOrCompleted bool
	for _, m := range msgs {
		switch msg := m.(type) {
		case tui.StepCompleteMsg:
			if msg.ID == "register" && msg.Status == tui.StepStatusFailed {
				registerFailed = true
			}
			if msg.ID == "download" {
				downloadStartedOrCompleted = true
			}
		case tui.StepStartMsg:
			if msg.ID == "download" {
				downloadStartedOrCompleted = true
			}
		}
	}

	// 2. Register step must have emitted StepCompleteMsg with StepStatusFailed
	if !registerFailed {
		t.Errorf("expected emitted StepCompleteMsg with ID 'register' and Status Failed, got msgs: %+v", msgs)
	}

	// 3. Download step must NOT have been started or completed
	if downloadStartedOrCompleted {
		t.Errorf("expected download step not started or completed, got msgs: %+v", msgs)
	}

	// 4. In rendered view, failure marker [✗] must be present and no [✓] completed steps
	view := monitor.Render(80, 24)
	if !strings.Contains(view, "[✗]") {
		t.Errorf("expected failed marker [✗] in view, got:\n%s", view)
	}
	if strings.Contains(view, "[✓]") {
		t.Errorf("expected no completed [✓] steps in view upon early registration failure, got:\n%s", view)
	}
}
