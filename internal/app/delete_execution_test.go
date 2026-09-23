package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"wptui/internal/config"
	"wptui/internal/deprovision"
	"wptui/internal/tui"
)

func TestApp_DeleteExecution_Wiring(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}

	application := New(Options{HomeDir: tempHome})
	_, err := application.InitConfig(false)
	if err != nil {
		t.Fatal(err)
	}

	appModel := application.BuildAppModel(context.Background())
	if appModel.DeleteModel() == nil {
		t.Fatalf("expected DeleteModel to be initialized on AppModel")
	}
	if appModel.DeleteModel().OnDelete == nil {
		t.Fatalf("expected OnDelete callback to be wired on DeleteModel")
	}
	if appModel.WebsitesHub().OnBatchDelete == nil {
		t.Fatalf("expected OnBatchDelete callback to be wired on WebsitesHub")
	}

	cands := []deprovision.Candidate{
		{Slug: "site-one", Path: filepath.Join(tempHome, "sites", "site-one")},
		{Slug: "site-two", Path: filepath.Join(tempHome, "sites", "site-two")},
	}

	cmd := appModel.DeleteModel().OnDelete(cands)
	if cmd == nil {
		t.Fatalf("expected non-nil cmd when OnDelete is called")
	}

	if !appModel.IsExecuting() {
		t.Errorf("expected AppModel to enter executing state")
	}

	view := appModel.ViewString()
	if !strings.Contains(view, "Deleting 2 website(s)") {
		t.Errorf("expected progress monitor title to mention 2 websites, got:\n%s", view)
	}
}

func TestApp_DeleteExecution_ExecutionRemovesDirectories(t *testing.T) {
	tempHome := t.TempDir()
	cfgPath := filepath.Join(tempHome, ".config", "wptui", "config.json")
	cfg := config.DefaultConfig(tempHome)
	sitesDir := filepath.Join(tempHome, "sites")
	cfg.WebsitesPath = sitesDir
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}

	// Create 2 test sites on disk
	siteA := filepath.Join(sitesDir, "del-a")
	siteB := filepath.Join(sitesDir, "del-b")
	_ = os.MkdirAll(siteA, 0755)
	_ = os.MkdirAll(siteB, 0755)

	application := New(Options{HomeDir: tempHome})
	_, err := application.InitConfig(false)
	if err != nil {
		t.Fatal(err)
	}

	cands := []deprovision.Candidate{
		{Slug: "del-a", Path: siteA},
		{Slug: "del-b", Path: siteB},
	}

	ch := make(chan tea.Msg, 64)
	go application.runDeleteExecution(context.Background(), cands, ch)

	var lastComplete *tui.OperationCompleteMsg
	for msg := range ch {
		if complete, ok := msg.(tui.OperationCompleteMsg); ok {
			lastComplete = &complete
		}
	}

	if lastComplete == nil || !lastComplete.Success {
		t.Fatalf("expected deprovisioning to complete successfully, got %+v", lastComplete)
	}

	// Directories must be removed from disk
	if _, err := os.Stat(siteA); !os.IsNotExist(err) {
		t.Errorf("expected %s to be deleted from disk", siteA)
	}
	if _, err := os.Stat(siteB); !os.IsNotExist(err) {
		t.Errorf("expected %s to be deleted from disk", siteB)
	}
}
