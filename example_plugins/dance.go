//go:build plugin

package main

import "gt"

var PluginName = "dance_macros"

func Init() {
	gt.RegisterCommand("dance", func(args string) {
		gt.RunCommand("/dance")
	})
	gt.AddHotkey("Shift-D", "/dance")
}
