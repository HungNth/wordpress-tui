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
		} else if item.Key == "config" {
			if item.Disabled {
				t.Errorf("config item must be enabled")
			}
		} else if item.Key == "delete" {
			if item.Disabled {
				t.Errorf("delete item must be enabled")
			}
		} else if item.Key == "settings" {
			if item.Disabled {
				t.Errorf("settings item must be enabled")
			}
		} else if item.Key != "exit" {
			if !item.Disabled {
				t.Errorf("expected %s to be disabled in v1", item.Key)
			}
		}
	}

	var choice string
	form := tui.BuildMainMenuForm(&choice)
	if form == nil {
		t.Fatal("expected non-nil form")
	}

	if !createFound {
		t.Errorf("create item was not found in menu items")
	}
}
