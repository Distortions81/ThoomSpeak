package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Test that closing the hotkey editor clears the reference and allows reopening.
func TestOpenHotkeyEditorReopenAfterClose(t *testing.T) {
	hotkeyEditWin = nil

	// Open and close with OK
	openHotkeyEditor(-1)
	if hotkeyEditWin == nil {
		t.Fatalf("editor not opened")
	}
	finishHotkeyEdit(true)
	if hotkeyEditWin != nil {
		t.Fatalf("editor not cleared after OK")
	}

	// Reopen and close with Cancel
	openHotkeyEditor(-1)
	if hotkeyEditWin == nil {
		t.Fatalf("editor not reopened after OK")
	}
	finishHotkeyEdit(false)
	if hotkeyEditWin != nil {
		t.Fatalf("editor not cleared after Cancel")
	}

	// Reopen and close via 'X'
	openHotkeyEditor(-1)
	if hotkeyEditWin == nil {
		t.Fatalf("editor not reopened after Cancel")
	}
	hotkeyEditWin.Close()
	if hotkeyEditWin != nil {
		t.Fatalf("editor not cleared after Close")
	}

	// Final reopen to ensure no leftovers
	openHotkeyEditor(-1)
	if hotkeyEditWin == nil {
		t.Fatalf("editor not reopened after Close")
	}
	hotkeyEditWin.Close()
}

// Test that entering a command in the hotkey editor saves correctly.
func TestHotkeyCommandInput(t *testing.T) {
	hotkeys = nil
	dir := t.TempDir()
	origDir := dataDirPath
	dataDirPath = dir
	defer func() { dataDirPath = origDir }()

	openHotkeyEditor(-1)
	if hotkeyCmdInput == nil {
		t.Fatalf("command input not initialized")
	}

	hotkeyComboText.Text = "Ctrl-A"
	hotkeyCmdInput.Text = "say"
	hotkeyTextInput.Text = "hi"
	finishHotkeyEdit(true)

	if len(hotkeys) != 1 {
		t.Fatalf("hotkey not saved")
	}
	if hotkeys[0].Combo != "Ctrl-A" || hotkeys[0].Command != "say" || hotkeys[0].Text != "hi" {
		t.Fatalf("unexpected hotkey data: %+v", hotkeys[0])
	}
}

// Test that loading hotkeys from disk refreshes the hotkeys window list.
func TestLoadHotkeysShowsEntriesInWindow(t *testing.T) {
	hotkeys = nil
	hotkeysWin = nil
	hotkeysList = nil

	dir := t.TempDir()
	origDir := dataDirPath
	dataDirPath = dir
	defer func() { dataDirPath = origDir }()

	// Create the hotkeys window initially with no entries.
	makeHotkeysWindow()
	if hotkeysList == nil {
		t.Fatalf("hotkeys window not initialized")
	}
	if len(hotkeysList.Contents) != 0 {
		t.Fatalf("expected empty hotkeys list")
	}

	// Write a hotkey entry to disk and load it.
	hk := []Hotkey{{Combo: "Ctrl-B", Command: "say bye"}}
	data, err := json.Marshal(hk)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	err = os.WriteFile(filepath.Join(dir, hotkeysFile), data, 0o644)
	if err != nil {
		t.Fatalf("write hotkeys: %v", err)
	}

	loadHotkeys()

	if len(hotkeysList.Contents) != 1 {
		t.Fatalf("hotkeys list not refreshed: %d", len(hotkeysList.Contents))
	}
}
