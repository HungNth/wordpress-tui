package tui_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/tui"
)

func TestProgressMonitor_StepProgressionAndStatusMarkers(t *testing.T) {
	steps := []tui.ProgressStep{
		{ID: "step1", Title: "Download WordPress Core"},
		{ID: "step2", Title: "Create Database"},
		{ID: "step3", Title: "Configure Herd TLS"},
	}

	monitor := tui.NewProgressMonitorModel("Site Provisioning", steps)

	// Initially all pending
	view := monitor.Render(70, 24)
	if !strings.Contains(view, "Download WordPress Core") {
		t.Errorf("expected view to contain initial steps, got:\n%s", view)
	}

	// 1. Start Step 1
	monitor.Update(tui.StepStartMsg{ID: "step1"})
	// Complete Step 1 with Success
	monitor.Update(tui.StepCompleteMsg{ID: "step1", Status: tui.StepStatusSuccess})
	view = monitor.Render(70, 24)
	if !strings.Contains(view, "[✓]") {
		t.Errorf("expected view to contain [✓] for completed step 1, got:\n%s", view)
	}

	// 2. Start Step 2 and Complete with Warning
	monitor.Update(tui.StepStartMsg{ID: "step2"})
	monitor.Update(tui.StepCompleteMsg{ID: "step2", Status: tui.StepStatusWarning, Detail: "fallback used"})
	view = monitor.Render(70, 24)
	if !strings.Contains(view, "[!]") || !strings.Contains(view, "fallback used") {
		t.Errorf("expected view to contain [!] and detail for warning step 2, got:\n%s", view)
	}

	// 3. Start Step 3 and Complete with Failed
	monitor.Update(tui.StepStartMsg{ID: "step3"})
	monitor.Update(tui.StepCompleteMsg{ID: "step3", Status: tui.StepStatusFailed, Detail: "TLS failed"})
	view = monitor.Render(70, 24)
	if !strings.Contains(view, "[✗]") || !strings.Contains(view, "TLS failed") {
		t.Errorf("expected view to contain [✗] and detail for failed step 3, got:\n%s", view)
	}
}

func TestProgressMonitor_LogStreamingAndViewport(t *testing.T) {
	steps := []tui.ProgressStep{
		{ID: "step1", Title: "Run WP-CLI Commands"},
	}

	monitor := tui.NewProgressMonitorModel("Website Configuration", steps)
	monitor.SetSize(70, 24)

	// Stream log lines
	monitor.Update(tui.LogLineMsg("Executing: wp core is-installed"))
	monitor.Update(tui.LogLineMsg("Success: WordPress is installed."))
	monitor.Update(tui.LogLineMsg("Updating admin password via mysql prepared statement..."))

	view := monitor.Render(70, 24)
	if !strings.Contains(view, "Executing: wp core is-installed") || !strings.Contains(view, "WordPress is installed.") {
		t.Errorf("expected view to contain streamed log lines, got:\n%s", view)
	}
}

func TestProgressMonitor_CompletionAndExit(t *testing.T) {
	steps := []tui.ProgressStep{
		{ID: "step1", Title: "Export Database"},
	}

	monitor := tui.NewProgressMonitorModel("Website Backup", steps)
	monitor.SetSize(70, 24)

	if monitor.IsCompleted() {
		t.Errorf("expected monitor not completed initially")
	}

	// Dispatch OperationCompleteMsg
	monitor.Update(tui.OperationCompleteMsg{
		Title:   "Backup Completed Successfully",
		Success: true,
		Summary: "Archive saved to F:/backups/site.zip",
	})

	if !monitor.IsCompleted() {
		t.Fatalf("expected monitor to be completed")
	}

	view := monitor.Render(70, 24)
	if !strings.Contains(view, "Backup Completed Successfully") || !strings.Contains(view, "Archive saved to") {
		t.Errorf("expected view to contain completion summary, got:\n%s", view)
	}

	// Press Enter to finish
	if monitor.IsFinished() {
		t.Errorf("expected not finished before Enter")
	}
	monitor.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !monitor.IsFinished() {
		t.Errorf("expected IsFinished to be true after Enter")
	}
}
