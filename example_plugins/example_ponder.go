package main

// Example plugin showcasing most of the available scripting features.
// This file is copied into the user's plugin directory on first run and
// can be edited to experiment with the API.

import (
	"fmt"
	"strings"

	"gt"
)

// PluginName must be unique across all loaded plugins.
var PluginName = "Examples"

// Init runs when the plugin is loaded and sets up commands, hotkeys and
// event handlers.
func Init() {
	// Log to the client log and console so the user knows we loaded.
	gt.Logf("[examples] running on client %d", gt.ClientVersion)
	gt.Console("[examples] hello, " + gt.PlayerName())

	// Register a chat handler that notifies us when our name is mentioned.
	gt.RegisterChatHandler(func(msg string) {
		if strings.Contains(strings.ToLower(msg), strings.ToLower(gt.PlayerName())) {
			gt.ShowNotification("Someone said your name!")
		}
	})

	// Player updates stream through this handler.
	gt.RegisterPlayerHandler(func(p gt.Player) {
		gt.Console("[examples] saw player: " + p.Name)
	})

	// Register a callable function and bind it to a hotkey.
	gt.RegisterFunc("dance", Dance)
	gt.AddHotkeyFunc("ctrl+d", "dance")

	// Bind a hotkey directly to a slash command.
	gt.AddHotkey("ctrl+n", "/rad notify")

	// Register the "/rad" command with many subcommands.
	gt.RegisterCommand("rad", handleRad)
}

// Dance sends a fun emote and plays a sound.
func Dance() {
	gt.RunCommand("/me dances")
	gt.PlaySound([]uint16{315})
}

// handleRad demonstrates a variety of features behind the "/rad" command.
func handleRad(args string) {
	fields := strings.Fields(args)
	if len(fields) == 0 {
		gt.Console("/rad [notify|stats|players|input <t>|equip <n>|unequip <n>|toggle <n>|hotkeys|rmhotkey <c>|click|frame|keys|say <t>|gear|has <n>|echoinput]")
		return
	}
	switch fields[0] {
	case "notify":
		gt.ShowNotification("Rad!")
	case "stats":
		s := gt.PlayerStats()
		gt.Console(fmt.Sprintf("HP %d/%d SP %d/%d", s.HP, s.HPMax, s.SP, s.SPMax))
	case "players":
		ps := gt.Players()
		names := make([]string, 0, len(ps))
		for _, p := range ps {
			names = append(names, p.Name)
		}
		gt.Console("players: " + strings.Join(names, ", "))
	case "input":
		gt.SetInputText(strings.Join(fields[1:], " "))
	case "echoinput":
		gt.Console("input: " + gt.InputText())
	case "equip":
		name := strings.Join(fields[1:], " ")
		for _, it := range gt.Inventory() {
			if strings.EqualFold(it.Name, name) {
				gt.Equip(it.ID)
				return
			}
		}
		gt.Console("item not found")
	case "unequip":
		name := strings.Join(fields[1:], " ")
		for _, it := range gt.Inventory() {
			if strings.EqualFold(it.Name, name) {
				gt.Unequip(it.ID)
				return
			}
		}
		gt.Console("item not equipped")
	case "toggle":
		name := strings.Join(fields[1:], " ")
		for _, it := range gt.Inventory() {
			if strings.EqualFold(it.Name, name) {
				gt.ToggleEquip(it.ID)
				return
			}
		}
		gt.Console("item not found")
	case "hotkeys":
		for _, hk := range gt.Hotkeys() {
			if hk.Disabled {
				continue
			}
			for _, cmd := range hk.Commands {
				if cmd.Plugin == "" {
					continue
				}
				if cmd.Command != "" {
					gt.Console(fmt.Sprintf("%s -> %s", hk.Combo, cmd.Command))
				} else if cmd.Function != "" {
					gt.Console(fmt.Sprintf("%s -> func %s", hk.Combo, cmd.Function))
				}
			}
		}
	case "rmhotkey":
		if len(fields) > 1 {
			gt.RemoveHotkey(fields[1])
		}
	case "click":
		c := gt.LastClick()
		if c.OnMobile {
			gt.Console(fmt.Sprintf("clicked %s at %d,%d", c.Mobile.Name, c.X, c.Y))
		} else {
			gt.Console(fmt.Sprintf("last click at %d,%d", c.X, c.Y))
		}
	case "frame":
		gt.Console(fmt.Sprintf("frame %d", gt.FrameNumber()))
	case "keys":
		gt.Console(fmt.Sprintf("space=%v just=%v mouse0=%v", gt.KeyPressed("space"), gt.KeyJustPressed("space"), gt.MousePressed("mouse0")))
	case "say":
		msg := strings.Join(fields[1:], " ")
		gt.EnqueueCommand("/say " + msg)
	case "gear":
		names := []string{}
		for _, it := range gt.EquippedItems() {
			names = append(names, it.Name)
		}
		gt.Console("equipped: " + strings.Join(names, ", "))
	case "has":
		name := strings.Join(fields[1:], " ")
		gt.Console(fmt.Sprintf("have %s: %v", name, gt.HasItem(name)))
	default:
		gt.Console("unknown subcommand")
	}
}
