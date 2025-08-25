//go:build plugin

package main

import "gt"

// PluginName identifies this plugin in the list.
var PluginName = "kudzu"

// Init sets up a few helper commands for planting and moving kudzu seeds.
func Init() {
	gt.RegisterCommand("zu", func(args string) {
		// Quickly plant a seed at your feet.
		gt.RunCommand("/plant kudzu")
	})
	gt.RegisterCommand("zuget", func(args string) {
		// Move a seed from the ground into your bag.
		gt.RunCommand("/useitem bag of kudzu seedlings /add")
	})
	gt.RegisterCommand("zustore", func(args string) {
		// Take a seed out of your bag.
		gt.RunCommand("/useitem bag of kudzu seedlings /remove")
	})
	gt.RegisterCommand("zutrans", func(args string) {
		// Give seeds to another exile if a name is provided.
		if args != "" {
			gt.RunCommand("/transfer " + args)
		}
	})
	// Press Shift+K to plant a seed.
	gt.AddHotkey("Shift-K", "/zu")
}
