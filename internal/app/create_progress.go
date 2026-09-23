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
		{ID: "download", Title: "Download & Extract WordPress Core", Status: tui.StepStatusPending},
		{ID: "database", Title: "Create Database & Config", Status: tui.StepStatusPending},
		{ID: "install", Title: "Install WordPress", Status: tui.StepStatusPending},
	}
	if inputs.ApplyTweaks && len(cfg.WPTweaks) > 0 {
		steps = append(steps, tui.ProgressStep{ID: "tweaks", Title: "Apply Tweaks", Status: tui.StepStatusPending})
	}
	hasPackages := len(pkgs) > 0 || (cfg.DefaultThemeSlug != "" && strings.TrimSpace(cfg.PackagesAPIURL) != "")
	if hasPackages {
		steps = append(steps, tui.ProgressStep{ID: "packages", Title: "Install Selected Packages", Status: tui.StepStatusPending})
	}
	if cfg.UsedHerd {
		steps = append(steps, tui.ProgressStep{ID: "herd", Title: "Configure Herd TLS", Status: tui.StepStatusPending})
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

// StepFunc returns the progress callback passed to creator.Create.
func (t *CreateProgressTracker) StepFunc() create.ProgressFunc {
	return func(step, detail string) {
		t.ch <- tui.LogLineMsg(detail)
		switch step {
		case "create_directory", "core_resolve", "core_extract", "core_download":
			t.currentStep = "download"
			t.ch <- tui.StepStartMsg{ID: "download", Title: detail}

		case "config_create", "db_create":
			if t.currentStep == "download" {
				t.ch <- tui.StepCompleteMsg{ID: "download", Status: tui.StepStatusSuccess, Detail: "Complete"}
			}
			t.currentStep = "database"
			t.ch <- tui.StepStartMsg{ID: "database", Title: detail}

		case "core_install":
			if t.currentStep == "database" {
				t.ch <- tui.StepCompleteMsg{ID: "database", Status: tui.StepStatusSuccess, Detail: "Complete"}
			}
			t.currentStep = "install"
			t.ch <- tui.StepStartMsg{ID: "install", Title: detail}

		case "tweaks", "tweak":
			if t.currentStep == "install" {
				t.ch <- tui.StepCompleteMsg{ID: "install", Status: tui.StepStatusSuccess, Detail: "Complete"}
			}
			t.currentStep = "tweaks"
			t.ch <- tui.StepStartMsg{ID: "tweaks", Title: detail}

		case "theme_install", "plugin_install":
			if t.currentStep == "tweaks" {
				t.ch <- tui.StepCompleteMsg{ID: "tweaks", Status: tui.StepStatusSuccess, Detail: "Complete"}
			} else if t.currentStep == "install" {
				t.ch <- tui.StepCompleteMsg{ID: "install", Status: tui.StepStatusSuccess, Detail: "Complete"}
			}
			t.currentStep = "packages"
			t.ch <- tui.StepStartMsg{ID: "packages", Title: detail}

		case "herd_secure":
			if t.currentStep == "packages" {
				t.ch <- tui.StepCompleteMsg{ID: "packages", Status: tui.StepStatusSuccess, Detail: "Complete"}
			} else if t.currentStep == "tweaks" {
				t.ch <- tui.StepCompleteMsg{ID: "tweaks", Status: tui.StepStatusSuccess, Detail: "Complete"}
			} else if t.currentStep == "install" {
				t.ch <- tui.StepCompleteMsg{ID: "install", Status: tui.StepStatusSuccess, Detail: "Complete"}
			}
			t.currentStep = "herd"
			t.ch <- tui.StepStartMsg{ID: "herd", Title: detail}
		}
	}
}

// CompleteAllSuccess marks any remaining active or pending steps as succeeded.
func (t *CreateProgressTracker) CompleteAllSuccess() {
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
	}
	if len(result.InstalledPlugins) > 0 {
		lines = append(lines, fmt.Sprintf("Plugins:      %s", strings.Join(result.InstalledPlugins, ", ")))
	}
	if len(result.FailedTweaks) > 0 {
		lines = append(lines, fmt.Sprintf("[!] Tweaks:   %d failed (see log)", len(result.FailedTweaks)))
	}
	return strings.Join(lines, "\n")
}
