package main

import (
	"sync"
	"testing"
)

// Test that the macros window lists registered macros.
func TestMacrosWindowListsMacros(t *testing.T) {
	// Reset state.
	macroMu = sync.RWMutex{}
	macroMaps = map[string]map[string]string{}
	macrosWin = nil
	macrosList = nil

	makeMacrosWindow()
	if macrosList == nil {
		t.Fatalf("macros window not initialized")
	}
	if len(macrosList.Contents) != 0 {
		t.Fatalf("expected empty macros list")
	}

	pluginAddMacro("tester", "yy", "/yell ")
	if len(macrosList.Contents) != 1 {
		t.Fatalf("macro not added to list: %d", len(macrosList.Contents))
	}
	if got := macrosList.Contents[0].Text; got != "yy = /yell" {
		t.Fatalf("unexpected macro text: %q", got)
	}
}
