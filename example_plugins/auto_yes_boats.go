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

	// Watch every chat line the client receives.
	gt.RegisterChatHandler(func(msg string) {
		lower := gt.Lower(msg)

		// Boat ferrymen say "My fine boats" when offering a ride.
		// If the message also mentions our name we whisper "yes".
		if gt.Includes(lower, "my fine boats") && gt.Includes(lower, nameLower) {
			gt.RunCommand("/whisper yes")
		}
	})
}
