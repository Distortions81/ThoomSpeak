//go:build plugin

package main

import (
	"strings"

	"gt"
)

var PluginName = "Chain Swap"

var savedID uint16
var lastFrame int

func Init() {
	gt.RegisterCommand("swapchain", func(string) { swapChain() })
	gt.AddHotkey("WheelUp", "/swapchain")
	gt.AddHotkey("WheelDown", "/swapchain")
}

func swapChain() {
	frame := gt.FrameNumber()
	if frame == lastFrame {
		return
	}
	lastFrame = frame

	var chainID uint16
	var equipped *gt.InventoryItem
	for _, it := range gt.Inventory() {
		if strings.EqualFold(it.Name, "chain") {
			chainID = it.ID
		}
		if it.Equipped && !strings.EqualFold(it.Name, "chain") {
			item := it
			equipped = &item
		}
	}
	if chainID == 0 {
		return
	}
	if equipped != nil {
		savedID = equipped.ID
		gt.Equip(chainID)
	} else if savedID != 0 {
		gt.Equip(savedID)
	}
}
