//go:build plugin

package main

import (
	"strings"

	"gt"
)

// PluginName identifies this plugin in the UI.
var PluginName = "Weapon Cycle"

var cycleItems = []string{"Axe", "Short Sword", "Dagger", "Chocolate"}

// Init binds F3 to cycle through weapons.
func Init() {
	gt.RegisterCommand("cycleweapon", func(string) { cycleWeapon() })
	gt.AddHotkey("F3", "/cycleweapon")
}

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
		if strings.EqualFold(current, name) {
			next = cycleItems[(i+1)%len(cycleItems)]
			break
		}
	}
	for _, it := range inv {
		if strings.EqualFold(it.Name, next) {
			gt.Equip(it.ID)
			return
		}
	}
}
