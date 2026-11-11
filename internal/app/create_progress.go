package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/config"
	"wptui/internal/create"
	"wptui/internal/tui"
)

// BuildCreateSteps returns the dynamic list of progress steps matching the actual provisioning plan.
func BuildCreateSteps(inputs tui.CreateInputs, pkgs []string, cfg *config.Config) []tui.ProgressStep {
	steps := []tui.ProgressStep{
		{ID: "resolve", Title: "Resolve Packages", Status: tui.StepStatusPending},
		{ID: "register", Title: "Prepare Website & Environment", Status: tui.StepStatusPending},
		{ID: "download", Title: "Download & Extract WordPress Core", Status: tui.StepStatusPending},
		{ID: "database", Title: "Create Database & Config", Status: tui.StepStatusPending},
		{ID: "install", Title: "Install WordPress", Status: tui.StepStatusPending},
	}
	if inputs.ApplyTweaks && len(cfg.WPTweaks) > 0 {
		steps = append(steps, tui.ProgressStep{ID: "tweaks", Title: "Apply Tweaks", Status: tui.StepStatusPending})
	}
	if len(pkgs) > 0 {
		steps = append(steps, tui.ProgressStep{ID: "packages", Title: "Install Selected Packages", Status: tui.StepStatusPending})
	}
	if cfg != nil && (cfg.AppName == config.AppNameHerd || cfg.AppName == config.AppNameFlock) {
		steps = append(steps, tui.ProgressStep{ID: "environment", Title: "Configure Environment TLS", Status: tui.StepStatusPending})
	}
	return steps
}

// CreateProgressTracker translates create.ProgressFunc callbacks into stepper updates.
type CreateProgressTracker struct {
	ch          chan<- tea.Msg
	currentStep string
	steps       []tui.ProgressStep
}

// NewCreateProgressTracker creates a new tracker instance.
func NewCreateProgressTracker(ch chan<- tea.Msg, steps []tui.ProgressStep) *CreateProgressTracker {
	return &CreateProgressTracker{
		ch:    ch,
		steps: steps,
	}
}

func (t *CreateProgressTracker) transition(targetStep, detail string) {
	if t.currentStep == targetStep {
		return
	}
	if t.currentStep != "" {
		t.ch <- tui.StepCompleteMsg{ID: t.currentStep, Status: tui.StepStatusSuccess, Detail: "Complete"}
	}
	t.currentStep = targetStep
	t.ch <- tui.StepStartMsg{ID: targetStep, Title: detail}
}

// StepFunc returns the progress callback passed to creator.Create.
func (t *CreateProgressTracker) StepFunc() create.ProgressFunc {
	return func(step, detail string) {
		t.ch <- tui.LogLineMsg(detail)
		switch step {
		case "create_directory", "environment_register":
			t.transition("register", detail)

		case "core_resolve", "core_extract", "core_download":
			t.transition("download", detail)

		case "config_create", "db_create":
			t.transition("database", detail)

		case "core_install":
			t.transition("install", detail)

		case "tweaks", "tweak":
			t.transition("tweaks", detail)

		case "theme_install", "plugin_install":
			t.transition("packages", detail)

		case "environment_secure":
			t.transition("environment", detail)
		}
	}
}

// CompleteAllSuccess marks any remaining active or pending steps as succeeded.
func (t *CreateProgressTracker) CompleteAllSuccess() {
	if t.currentStep != "" {
		t.ch <- tui.StepCompleteMsg{ID: t.currentStep, Status: tui.StepStatusSuccess, Detail: "Complete"}
	}
	for _, s := range t.steps {
		t.ch <- tui.StepCompleteMsg{ID: s.ID, Status: tui.StepStatusSuccess, Detail: "Complete"}
	}
}

// MarkFailed marks the current active step as failed.
func (t *CreateProgressTracker) MarkFailed(err error) {
	if t.currentStep != "" {
		t.ch <- tui.StepCompleteMsg{ID: t.currentStep, Status: tui.StepStatusFailed, Detail: "Failed"}
	}
}

// FormatCreateSummary creates a clean, structured summary card of the provisioned site.
func FormatCreateSummary(inputs tui.CreateInputs, result *create.Result, defaultAdminUser string) string {
	var lines []string
	if result.WebsiteURL != "" {
		lines = append(lines, fmt.Sprintf("URL:          %s", result.WebsiteURL))
	}
	if result.WebsitePath != "" {
		lines = append(lines, fmt.Sprintf("Directory:    %s", result.WebsitePath))
	}
	lines = append(lines, fmt.Sprintf("Database:     %s", inputs.WebsiteSlug))

	adminUser := inputs.AdminUsername
	if adminUser == "" {
		adminUser = defaultAdminUser
	}
	if adminUser != "" {
		lines = append(lines, fmt.Sprintf("Admin User:   %s", adminUser))
	}

	if result.ActiveTheme != "" {
		lines = append(lines, fmt.Sprintf("Active Theme: %s", result.ActiveTheme))
	} else {
		lines = append(lines, "Active Theme: none (administration interface available; frontend has no active renderer)")
	}
	if len(result.InstalledPlugins) > 0 {
		lines = append(lines, fmt.Sprintf("Plugins:      %s", strings.Join(result.InstalledPlugins, ", ")))
	}
	if len(result.FailedTweaks) > 0 {
		lines = append(lines, fmt.Sprintf("[!] Tweaks:   %d failed (see log)", len(result.FailedTweaks)))
	}
	return strings.Join(lines, "\n")
}
