//go:build plugin

package main

import "gt"

// PluginName is the label shown in the client's plugin list.
var PluginName = "Auto Yes Boats"

// Init runs once when the plugin is loaded.
// It listens for boat offers and politely replies for us.
func Init() {
	// Grab our own name in lowercase so comparisons are easy.
	nameLower := gt.Lower(gt.PlayerName())

	// Watch for boat offers and respond automatically.
	gt.RegisterTriggers([]string{"my fine boats"}, func(msg string) {
		lower := gt.Lower(msg)
		if gt.Includes(lower, nameLower) {
			gt.RunCommand("/whisper yes")
		}
	})
}
