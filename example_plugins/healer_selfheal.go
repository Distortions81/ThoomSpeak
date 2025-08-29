//go:build plugin

package main

import (
	"time"

	"gt"
)

// PluginName is how the client lists this plugin.
var PluginName = "Healer Helper"

// PluginCategory groups this plugin.
var PluginCategory = "Profession"

// PluginSubCategory refines the category.
var PluginSubCategory = "Healer"

// Init launches a tiny loop that watches for right clicks on ourselves.
func Init() {
	go func() {
		for {
			if gt.MouseJustPressed("right") {
				c := gt.LastClick()
				if c.OnMobile && gt.IgnoreCase(c.Mobile.Name, gt.PlayerName()) {
					equipMoonstone()
					gt.RunCommand("/use 10")
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()
}

// equipMoonstone equips the moonstone if it isn't already in hand.
func equipMoonstone() {
	for _, it := range gt.Inventory() {
		if gt.IgnoreCase(it.Name, "moonstone") {
			if !it.Equipped {
				gt.Equip(it.ID)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)

	}
}
