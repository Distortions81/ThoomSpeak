package main

import "testing"

// Test that plugin equip command skips already equipped items.
func TestPluginEquipAlreadyEquipped(t *testing.T) {
	resetInventory()
	addInventoryItem(200, -1, "Shield", true)
	consoleLog = messageLog{max: maxMessages}
	pendingCommand = ""
	pluginEquip(200)
	msgs := getConsoleMessages()
	if len(msgs) == 0 || msgs[len(msgs)-1] != "Shield already equipped, skipping" {
		t.Fatalf("unexpected console messages %v", msgs)
	}
	if pendingCommand != "" {
		t.Fatalf("pending command queued: %q", pendingCommand)
	}
}
