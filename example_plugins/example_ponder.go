//go:build plugin

package main

// This plugin shows many features of the scripting system. It is copied to the
// user's plugin folder the first time the game runs so new users can edit it.

import (
	"fmt"  // used for building strings like "HP 10/10"
	"time" // used for simple delays between poses

	"gt" // the small API exposed by the client
)

// PluginName must be unique across all loaded plugins.
var PluginName = "Examples"

// Init runs once when the plugin is loaded and sets up everything below.
func Init() {
	// Print a message so the user knows the plugin is active.
	gt.Logf("[examples] running on client %d", gt.ClientVersion)
	gt.Console("[examples] hello, " + gt.PlayerName())

	// Watch chat. If someone types our name we show a popup.
	gt.RegisterChatHandler(func(msg string) {
		if gt.Includes(gt.Lower(msg), gt.Lower(gt.PlayerName())) {
			gt.ShowNotification("Someone said your name!")
		}
	})

	// Watch player updates and print each player we see.
	gt.RegisterPlayerHandler(func(p gt.Player) {
		gt.Console("[examples] saw player: " + p.Name)
	})

	// Bind Ctrl+D to call our "dance" subcommand.
	gt.AddHotkey("ctrl+d", "/rad dance")

	// Bind Ctrl+N to run a slash command immediately.
	gt.AddHotkey("ctrl+n", "/rad notify")

	// Register the command "/rad" with many subcommands handled below.
	gt.RegisterCommand("rad", handleRad)
}

// Dance cycles through a few pose commands and plays a short sound.
func Dance() {
	poses := []string{"celebrate", "leanleft", "leanright", "celebrate"}
	for _, p := range poses {
		gt.RunCommand("/pose " + p)
		time.Sleep(250 * time.Millisecond)
	}
	gt.PlaySound([]uint16{315})
}

// handleRad runs whenever the player types "/rad ...".
// args is everything after "/rad".
func handleRad(args string) {
	// Split the text into words.
	fields := gt.Words(args)
	// If no words were given, show help text.
	if len(fields) == 0 {
		gt.Console("/rad [notify|stats|players|input <t>|equip <n>|unequip <n>|toggle <n>|hotkeys|rmhotkey <c>|click|frame|keys|say <t>|gear|has <n>|echoinput]")
		return
	}
	// Look at the first word to decide what to do.
	switch fields[0] {
	case "notify":
		// Show a simple popup.
		gt.ShowNotification("Rad!")
	case "stats":
		// Print our HP and SP values.
		s := gt.PlayerStats()
		gt.Console(fmt.Sprintf("HP %d/%d SP %d/%d", s.HP, s.HPMax, s.SP, s.SPMax))
	case "players":
		// List the names of players we know about.
		ps := gt.Players()
		names := make([]string, 0, len(ps))
		for _, p := range ps {
			names = append(names, p.Name)
		}
		gt.Console("players: " + gt.Join(names, ", "))
	case "input":
		// Set the chat box text.
		gt.SetInputText(gt.Join(fields[1:], " "))
	case "echoinput":
		// Print whatever is in the chat box right now.
		gt.Console("input: " + gt.InputText())
	case "equip":
		// Equip an item by name.
		name := gt.Join(fields[1:], " ")
		for _, it := range gt.Inventory() {
			if gt.IgnoreCase(it.Name, name) {
				gt.Equip(it.ID)
				return
			}
		}
		gt.Console("item not found")
	case "unequip":
		// Unequip an item by name.
		name := gt.Join(fields[1:], " ")
		for _, it := range gt.Inventory() {
			if gt.IgnoreCase(it.Name, name) {
				gt.Unequip(it.ID)
				return
			}
		}
		gt.Console("item not equipped")
	case "toggle":
		// Toggle equip state for an item.
		name := gt.Join(fields[1:], " ")
		for _, it := range gt.Inventory() {
			if gt.IgnoreCase(it.Name, name) {
				gt.ToggleEquip(it.ID)
				return
			}
		}
		gt.Console("item not found")
	case "hotkeys":
		// List hotkeys that come from plugins, capped at 15 lines.
		count := 0
		for _, hk := range gt.Hotkeys() {
			if hk.Disabled {
				continue
			}
			for _, cmd := range hk.Commands {
				if cmd.Command != "" {
					gt.Console(fmt.Sprintf("%s -> %s", hk.Combo, cmd.Command))
					count++
					if count >= 15 {
						break
					}
				}
			}
			if count >= 15 {
				break
			}
		}
	case "rmhotkey":
		// Remove a hotkey by its combo string.
		if len(fields) > 1 {
			gt.RemoveHotkey(fields[1])
		}
	case "click":
		// Tell us where we last clicked.
		c := gt.LastClick()
		if c.OnMobile {
			gt.Console(fmt.Sprintf("clicked %s at %d,%d", c.Mobile.Name, c.X, c.Y))
		} else {
			gt.Console(fmt.Sprintf("last click at %d,%d", c.X, c.Y))
		}
	case "frame":
		// Show the current game frame number.
		gt.Console(fmt.Sprintf("frame %d", gt.FrameNumber()))
	case "keys":
		// Show the state of some keys and mouse buttons.
		gt.Console(fmt.Sprintf("space=%v just=%v mouse0=%v", gt.KeyPressed("space"), gt.KeyJustPressed("space"), gt.MousePressed("mouse0")))
	case "say":
		// Send a /say command with the rest of the text.
		msg := gt.Join(fields[1:], " ")
		gt.EnqueueCommand("/say " + msg)
	case "gear":
		// List names of currently equipped items.
		names := []string{}
		for _, it := range gt.EquippedItems() {
			names = append(names, it.Name)
		}
		gt.Console("equipped: " + gt.Join(names, ", "))
	case "has":
		// Tell us if we have a specific item.
		name := gt.Join(fields[1:], " ")
		gt.Console(fmt.Sprintf("have %s: %v", name, gt.HasItem(name)))
	case "dance":
		// Run the dance action.
		Dance()
	default:
		gt.Console("unknown subcommand")
	}
}
