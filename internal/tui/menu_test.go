package tui_test

import (
	"testing"

	"wptui/internal/tui"
)

func TestMenuItems(t *testing.T) {
	items := tui.GetMenuItems()
	if len(items) != 7 {
		t.Fatalf("expected 7 menu items (create, config, delete, backup, restore, settings, exit), got %d", len(items))
	}

	createFound := false
	for _, item := range items {
		if item.Key == "create" {
			createFound = true
			if item.Disabled {
				t.Errorf("create item must be enabled")
			}
		} else if item.Key != "exit" {
			if !item.Disabled {
				t.Errorf("expected %s to be disabled in v1", item.Key)
			}
		}
	}

	if !createFound {
		t.Errorf("create item was not found in menu items")
	}
}
