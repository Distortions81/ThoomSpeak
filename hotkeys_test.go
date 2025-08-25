package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	if len(hotkeyCmdInputs) == 0 {
		t.Fatalf("command input not initialized")
	}

	hotkeyComboText.Text = "Ctrl-A"
	hotkeyNameInput.Text = "Test"
	hotkeyCmdInputs[0].Text = "say hi"
	finishHotkeyEdit(true)

	if len(hotkeys) != 1 {
		t.Fatalf("hotkey not saved")
	}
	if hotkeys[0].Combo != "Ctrl-A" || hotkeys[0].Name != "Test" || len(hotkeys[0].Commands) != 1 || hotkeys[0].Commands[0].Command != "say hi" {
		t.Fatalf("unexpected hotkey data: %+v", hotkeys[0])
	}
}

// Test that a hotkey invoking a function can omit the command text.
func TestHotkeyFunctionWithoutCommand(t *testing.T) {
	hotkeys = nil
	openHotkeyEditor(-1)
	hotkeyComboText.Text = "Ctrl-F"
	addHotkeyCommand("", "ponder")
	finishHotkeyEdit(true)

	if len(hotkeys) != 1 {
		t.Fatalf("hotkey not saved")
	}
	cmd := hotkeys[0].Commands[0]
	if cmd.Command != "" || cmd.Function != "ponder" {
		t.Fatalf("unexpected hotkey command: %+v", cmd)
	}
	if hotkeyEditWin != nil {
		hotkeyEditWin.Close()
	}
}

// Test that editing a hotkey with no name still saves changes.
func TestHotkeyEditWithoutName(t *testing.T) {
	hotkeys = []Hotkey{{Combo: "Ctrl-A", Commands: []HotkeyCommand{{Command: "say hi"}}}}
	dir := t.TempDir()
	origDir := dataDirPath
	dataDirPath = dir
	defer func() { dataDirPath = origDir }()

	openHotkeyEditor(0)
	hotkeyCmdInputs[0].Text = "say bye"
	finishHotkeyEdit(true)

	if len(hotkeys) != 1 || hotkeys[0].Commands[0].Command != "say bye" {
		t.Fatalf("hotkey not updated without name: %+v", hotkeys)
	}
}

// Test that a hotkey without a name still saves and refreshes.
func TestHotkeySavedWithoutName(t *testing.T) {
	hotkeys = nil
	openHotkeyEditor(-1)
	hotkeyComboText.Text = "Ctrl-C"
	hotkeyCmdInputs[0].Text = "say hi"
	finishHotkeyEdit(true)
	if len(hotkeys) != 1 || hotkeys[0].Name != "" {
		t.Fatalf("hotkey not saved or name unexpectedly set: %+v", hotkeys)
	}
	if hotkeyEditWin != nil {
		hotkeyEditWin.Close()
	}
}

// Test that adding a hotkey without a name updates the window list.
func TestHotkeyListUpdatesForNamelessHotkey(t *testing.T) {
	hotkeys = nil
	hotkeysWin = nil
	hotkeysList = nil

	makeHotkeysWindow()
	if hotkeysList == nil {
		t.Fatalf("hotkeys window not initialized")
	}
	if len(hotkeysList.Contents) != 0 {
		t.Fatalf("expected empty list")
	}

	openHotkeyEditor(-1)
	hotkeyComboText.Text = "Ctrl-X"
	hotkeyCmdInputs[0].Text = "say hi"
	finishHotkeyEdit(true)

	if len(hotkeysList.Contents) != 1 {
		t.Fatalf("hotkeys list not refreshed: %d", len(hotkeysList.Contents))
	}
	row := hotkeysList.Contents[0]
	if row == nil || len(row.Contents) == 0 {
		t.Fatalf("hotkey row malformed")
	}
	if got := row.Contents[0].Text; got != "Ctrl-X -> say hi" {
		t.Fatalf("unexpected hotkey text: %q", got)
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
	hk := []Hotkey{{Combo: "Ctrl-B", Name: "Bye", Commands: []HotkeyCommand{{Command: "say bye"}}}}
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
	row := hotkeysList.Contents[0]
	if row == nil || len(row.Contents) == 0 {
		t.Fatalf("hotkey row malformed")
	}
	if got := row.Contents[0].Text; got != "Bye : Ctrl-B -> say bye" {
		t.Fatalf("unexpected hotkey text: %q", got)
	}
}

// Test that long input lines wrap and cause the window to grow.
func TestHotkeyEditorWrapsAndResizes(t *testing.T) {
	hotkeyEditWin = nil
	openHotkeyEditor(-1)
	if hotkeyEditWin == nil {
		t.Fatalf("editor not opened")
	}
	base := hotkeyEditWin.Size.Y
	long := "this is a very long command line that should wrap across multiple lines for testing"
	hotkeyCmdInputs[0].Text = long
	wrapHotkeyInputs()
	if !strings.Contains(hotkeyCmdInputs[0].Text, "\n") || hotkeyCmdInputs[0].Size.Y <= 20 {
		t.Fatalf("command input did not wrap or grow: %q size %v", hotkeyCmdInputs[0].Text, hotkeyCmdInputs[0].Size.Y)
	}
	if hotkeyEditWin.Size.Y <= base {
		t.Fatalf("window did not resize: %v <= %v", hotkeyEditWin.Size.Y, base)
	}
	hotkeyEditWin.Close()
}
