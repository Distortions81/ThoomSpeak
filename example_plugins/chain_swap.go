//go:build plugin

package main

import "gt"

// PluginName identifies this plugin in the client.
var PluginName = "Chain Swap"

var savedID uint16
var lastFrame int

// Init wires up our command and mouse-wheel hotkeys.
func Init() {
	gt.RegisterCommand("swapchain", func(string) { swapChain() })
	gt.AddHotkey("WheelUp", "/swapchain")
	gt.AddHotkey("WheelDown", "/swapchain")
}

// swapChain toggles between a chain weapon and whatever was equipped before.
func swapChain() {
	frame := gt.FrameNumber()
	if frame == lastFrame {
		// Ignore repeated triggers on the same frame.
		return
	}
	lastFrame = frame

	var chainID uint16
	var equipped *gt.InventoryItem
	for _, it := range gt.Inventory() {
		if gt.IgnoreCase(it.Name, "chain") {
			chainID = it.ID
		}
		if it.Equipped && !gt.IgnoreCase(it.Name, "chain") {
			item := it // capture for pointer
			equipped = &item
		}
	}
	if chainID == 0 {
		// No chain? Nothing to do.
		return
	}
	if equipped != nil {
		// Remember what we unequipped so we can switch back later.
		savedID = equipped.ID
		gt.Equip(chainID)
	} else if savedID != 0 {
		// Chain already equipped, so swap back.
		gt.Equip(savedID)
	}
}
