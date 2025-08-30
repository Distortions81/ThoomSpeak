package main

import (
	"sync"
	"testing"
)

// Test that the macros window lists registered macros.
func TestMacrosWindowListsMacros(t *testing.T) {
	// Reset state and ensure cleanup after the test.
	macroMu = sync.RWMutex{}
	macroMaps = map[string]map[string]string{}
	pluginDisplayNames = map[string]string{}
	pluginCategories = map[string]string{}
	pluginSubCategories = map[string]string{}
	macrosWin = nil
	macrosList = nil
	t.Cleanup(func() {
		macroMu = sync.RWMutex{}
		macroMaps = map[string]map[string]string{}
		pluginDisplayNames = map[string]string{}
		pluginCategories = map[string]string{}
		pluginSubCategories = map[string]string{}
		macrosWin = nil
		macrosList = nil
	})

	makeMacrosWindow()
	if macrosList == nil {
		t.Fatalf("macros window not initialized")
	}
	if len(macrosList.Contents) != 0 {
		t.Fatalf("expected empty macros list")
	}

	pluginAddMacro("tester", "yy", "/yell ")
	if len(macrosList.Contents) != 2 {
		t.Fatalf("items not added to list: %d", len(macrosList.Contents))
	}
	if got := macrosList.Contents[0].Text; got != "tester:" {
		t.Fatalf("unexpected plugin text: %q", got)
	}
	if got := macrosList.Contents[1].Text; got != "  yy = /yell" {
		t.Fatalf("unexpected macro text: %q", got)
	}
}

// Test that removing macros refreshes the window and clears the list.
func TestPluginRemoveMacrosRefresh(t *testing.T) {
	// Reset state and ensure cleanup after the test.
	macroMu = sync.RWMutex{}
	macroMaps = map[string]map[string]string{}
	pluginDisplayNames = map[string]string{}
	pluginCategories = map[string]string{}
	pluginSubCategories = map[string]string{}
	macrosWin = nil
	macrosList = nil
	t.Cleanup(func() {
		macroMu = sync.RWMutex{}
		macroMaps = map[string]map[string]string{}
		pluginDisplayNames = map[string]string{}
		pluginCategories = map[string]string{}
		pluginSubCategories = map[string]string{}
		macrosWin = nil
		macrosList = nil
	})

	makeMacrosWindow()
	if macrosList == nil {
		t.Fatalf("macros window not initialized")
	}

	pluginAddMacro("tester", "yy", "/yell ")
	if len(macrosList.Contents) != 2 {
		t.Fatalf("items not added to list: %d", len(macrosList.Contents))
	}

	// Clear dirty flag so we can detect refresh.
	macrosWin.Dirty = false

	pluginRemoveMacros("tester")
	if len(macrosList.Contents) != 0 {
		t.Fatalf("macros list not cleared: %d", len(macrosList.Contents))
	}
	if !macrosWin.Dirty {
		t.Fatalf("macros window not refreshed")
	}
}

// Test that opening the macros window after macros have been added lists them correctly.
func TestMacrosWindowLoadsExistingMacros(t *testing.T) {
	// Reset state and ensure cleanup after the test.
	macroMu = sync.RWMutex{}
	macroMaps = map[string]map[string]string{}
	pluginDisplayNames = map[string]string{}
	pluginCategories = map[string]string{}
	pluginSubCategories = map[string]string{}
	macrosWin = nil
	macrosList = nil
	t.Cleanup(func() {
		macroMu = sync.RWMutex{}
		macroMaps = map[string]map[string]string{}
		pluginDisplayNames = map[string]string{}
		pluginCategories = map[string]string{}
		pluginSubCategories = map[string]string{}
		macrosWin = nil
		macrosList = nil
	})

	// Add macros before creating the window to mimic plugins registering at startup.
	pluginAddMacro("tester", "yy", "/yell ")

	// Now create the window; it should populate with existing macros.
	makeMacrosWindow()
	if macrosList == nil {
		t.Fatalf("macros window not initialized")
	}
	if len(macrosList.Contents) != 2 {
		t.Fatalf("items not added to list: %d", len(macrosList.Contents))
	}
	if got := macrosList.Contents[0].Text; got != "tester:" {
		t.Fatalf("unexpected plugin text: %q", got)
	}
	if got := macrosList.Contents[1].Text; got != "  yy = /yell" {
		t.Fatalf("unexpected macro text: %q", got)
	}
}
