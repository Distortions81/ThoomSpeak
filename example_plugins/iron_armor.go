//go:build plugin

package main

import (
	"time"

	"gt"
)

// PluginName identifies the plugin.
var PluginName = "Iron Armor Manager"

var armorCondition string

// Init wires up commands, hotkeys, and a chat watcher for examine results.
func Init() {
	gt.RegisterCommand("ironarmortoggle", func(args string) { ironArmorToggler() })
	gt.RegisterCommand("examinearmor", func(args string) { examineArmor() })
	gt.AddHotkey("Ctrl-F10", "/ironarmortoggle")
	gt.AddHotkey("Ctrl-F11", "/examinearmor")
	gt.RegisterTriggers("", []string{"perfect", "good", "look"}, func(msg string) {
		armorCondition = msg
	})
}

func hasEquipped(name string) bool {
	for _, it := range gt.EquippedItems() {
		if gt.IgnoreCase(it.Name, name) {
			return true
		}
	}
	return false
}

func ironArmorToggler() {
	if hasEquipped("iron breastplate") && hasEquipped("iron helmet") && hasEquipped("iron shield") {
		gt.RunCommand("/unequip ironbreastplate")
		gt.RunCommand("/unequip ironhelmet")
		gt.RunCommand("/unequip ironshield")
		return
	}
	equipIronArmor()
}

func equipIronArmor() {
	equipItem("iron breastplate", "ironbreastplate", "Iron Breastplate")
	equipItem("iron helmet", "ironhelmet", "Iron Helmet")
	equipItem("iron shield", "ironshield", "Iron Shield")
}

func equipItem(name, cmd, display string) {
	if hasEquipped(name) {
		return
	}
	gt.RunCommand("/equip " + cmd)
	time.Sleep(100 * time.Millisecond)
	if !hasEquipped(name) {
		gt.RunCommand("/unequip " + cmd)
		gt.Console("* " + display + " unequipped due to durability.")
	}
}

func examineArmor() {
	gt.Console("* Armor Examiner:")
	if gt.HasItem("iron breastplate") {
		gt.RunCommand("/examine ironbreastplate")
		time.Sleep(100 * time.Millisecond)
		armorLabeler("5")
	}
	if gt.HasItem("iron helmet") {
		gt.RunCommand("/examine ironhelmet")
		time.Sleep(100 * time.Millisecond)
		armorLabeler("4")
	}
	if gt.HasItem("iron shield") {
		gt.RunCommand("/examine ironshield")
		time.Sleep(100 * time.Millisecond)
		armorLabeler("3")
	}
}

func armorLabeler(slot string) {
	lower := gt.Lower(armorCondition)
	switch {
	case gt.Includes(lower, "perfect"):
		gt.RunCommand("/name " + slot + " (perfect)")
	case gt.Includes(lower, "good"):
		gt.RunCommand("/name " + slot + " (good)")
	case gt.Includes(lower, "look"):
		gt.RunCommand("/name " + slot + " (worn)")
	}
}
