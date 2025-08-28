//go:build plugin

package main

import (
	"time"

	"gt"
)

// PluginName shows up in the plugin list.
var PluginName = "Dance Macros"

// Init adds the /dance command and a handy hotkey.
func Init() {
	gt.RegisterCommand("dance", func(args string) {
		// A tiny routine of poses played in sequence.
		poses := []string{"celebrate", "leanleft", "leanright", "celebrate"}
		for _, p := range poses {
			gt.RunCommand("/pose " + p)
			time.Sleep(250 * time.Millisecond)
		}
	})
	// Press Shift+D to start dancing.
	gt.AddHotkey("Shift-D", "/dance")
}
