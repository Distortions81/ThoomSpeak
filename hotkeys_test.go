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
	addHotkeyCommand("", "ponder", "")
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

// Test that selecting a function without clicking add saves the hotkey.
func TestHotkeyFunctionSelectionSaves(t *testing.T) {
	hotkeys = nil
	openHotkeyEditor(-1)
	hotkeyComboText.Text = "Ctrl-H"
	selectHotkeyFunction("ponder", "")
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

// Test that a hotkey with an empty command saves correctly.
func TestHotkeyEmptyCommandSaved(t *testing.T) {
	hotkeys = nil
	openHotkeyEditor(-1)
	hotkeyComboText.Text = "Ctrl-E"
	finishHotkeyEdit(true)

	if len(hotkeys) != 1 {
		t.Fatalf("hotkey not saved")
	}
	if len(hotkeys[0].Commands) != 0 {
		t.Fatalf("expected no commands, got: %+v", hotkeys[0].Commands)
	}
	if hotkeyEditWin != nil {
		hotkeyEditWin.Close()
	}
}

// Test that a function-only hotkey persists through save/load cycles.
func TestHotkeyFunctionPersisted(t *testing.T) {
	hotkeys = []Hotkey{{Combo: "Ctrl-P", Commands: []HotkeyCommand{{Function: "ponder"}}}}
	dir := t.TempDir()
	origDir := dataDirPath
	dataDirPath = dir
	defer func() { dataDirPath = origDir }()

	saveHotkeys()
	hotkeys = nil
	loadHotkeys()

	if len(hotkeys) != 1 {
		t.Fatalf("hotkey not loaded")
	}
	cmd := hotkeys[0].Commands[0]
	if cmd.Command != "" || cmd.Function != "ponder" {
		t.Fatalf("unexpected hotkey after load: %+v", cmd)
	}
}

// Test that editing a hotkey preselects its function in the dropdown.
func TestHotkeyEditPreselectsFunction(t *testing.T) {
	hotkeys = []Hotkey{{Combo: "Ctrl-P", Commands: []HotkeyCommand{{Function: "ponder"}}}}
	pluginMu.Lock()
	origFuncs := pluginFuncs
	pluginFuncs = map[string]map[string]PluginFunc{"": {"ponder": nil}}
	pluginMu.Unlock()
	defer func() {
		pluginMu.Lock()
		pluginFuncs = origFuncs
		pluginMu.Unlock()
	}()

	openHotkeyEditor(0)
	flow := hotkeyEditWin.Contents[0]
	fnRow := flow.Contents[4]
	fnDD := fnRow.Contents[1]
	if fnDD.Selected != 1 {
		t.Fatalf("function dropdown not preselected: %d", fnDD.Selected)
	}
	hotkeyEditWin.Close()
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

// Test that @clicked in commands expands to the last clicked mobile name.
func TestApplyHotkeyVars(t *testing.T) {
	lastClickMu.Lock()
	lastClick = ClickInfo{OnMobile: true, Mobile: Mobile{Name: "Target"}}
	lastClickMu.Unlock()
	got, ok := applyHotkeyVars("/use @clicked")
	if !ok || got != "/use Target" {
		t.Fatalf("got %q, ok %v", got, ok)
	}
}

// Test that @hovered in commands expands to the currently hovered mobile name.
func TestApplyHotkeyVarsHovered(t *testing.T) {
	lastHoverMu.Lock()
	lastHover = ClickInfo{OnMobile: true, Mobile: Mobile{Name: "Hover"}}
	lastHoverMu.Unlock()
	got, ok := applyHotkeyVars("/inspect @hovered")
	if !ok || got != "/inspect Hover" {
		t.Fatalf("got %q, ok %v", got, ok)
	}
}

// Test that commands referencing @clicked don't fire without a target.
func TestApplyHotkeyVarsNoClicked(t *testing.T) {
	lastClickMu.Lock()
	lastClick = ClickInfo{}
	lastClickMu.Unlock()
	if got, ok := applyHotkeyVars("/use @clicked"); ok || got != "" {
		t.Fatalf("got %q, ok %v", got, ok)
	}
}

// Test that commands referencing @hovered don't fire without a target.
func TestApplyHotkeyVarsNoHovered(t *testing.T) {
	lastHoverMu.Lock()
	lastHover = ClickInfo{}
	lastHoverMu.Unlock()
	if got, ok := applyHotkeyVars("/inspect @hovered"); ok || got != "" {
		t.Fatalf("got %q, ok %v", got, ok)
	}
}

// Test that hotkey equip commands skip already equipped items.
func TestHotkeyEquipAlreadyEquipped(t *testing.T) {
	resetInventory()
	addInventoryItem(100, -1, "Sword", true)
	consoleLog = messageLog{max: maxMessages}
	if !hotkeyEquipAlreadyEquipped("/equip 100") {
		t.Fatalf("expected command to be skipped")
	}
	msgs := getConsoleMessages()
	if len(msgs) == 0 || msgs[len(msgs)-1] != "Sword already equipped, skipping" {
		t.Fatalf("unexpected console messages %v", msgs)
	}
}
