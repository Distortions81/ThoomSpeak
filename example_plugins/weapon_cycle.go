//go:build plugin

package main

import "gt"

// PluginName identifies this plugin in the UI.
var PluginName = "Weapon Cycle"

var cycleItems = []string{"Axe", "Short Sword", "Dagger", "Chocolate"}

// Init binds F3 to cycle through weapons.
func Init() {
	gt.RegisterCommand("cycleweapon", func(string) { cycleWeapon() })
	gt.AddHotkey("F3", "/cycleweapon")
}

// cycleWeapon equips the next item in cycleItems.
func cycleWeapon() {
	inv := gt.Inventory()
	current := ""
	for _, it := range inv {
		if it.Equipped {
			current = it.Name
			break
		}
	}
	next := cycleItems[0]
	for i, name := range cycleItems {
		if gt.IgnoreCase(current, name) {
			next = cycleItems[(i+1)%len(cycleItems)]
			break
		}
	}
	for _, it := range inv {
		if gt.IgnoreCase(it.Name, next) {
			gt.Equip(it.ID)
			return
		}
	}
}
