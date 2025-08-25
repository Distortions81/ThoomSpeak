//go:build plugin

package main

import "gt"

var PluginName = "kudzu"

func Init() {
	gt.RegisterCommand("zu", func(args string) {
		gt.RunCommand("/plant kudzu")
	})
	gt.RegisterCommand("zuget", func(args string) {
		gt.RunCommand("/useitem bag of kudzu seedlings /add")
	})
	gt.RegisterCommand("zustore", func(args string) {
		gt.RunCommand("/useitem bag of kudzu seedlings /remove")
	})
	gt.RegisterCommand("zutrans", func(args string) {
		if args != "" {
			gt.RunCommand("/transfer " + args)
		}
	})
	gt.AddHotkey("Shift-K", "/zu")
}
