//go:build plugin

package main

import "gt"

var PluginName = "Auto Yes Boats"

func Init() {
	nameLower := gt.Lower(gt.PlayerName())
	gt.RegisterChatHandler(func(msg string) {
		lower := gt.Lower(msg)
		if gt.Includes(lower, "my fine boats") && gt.Includes(lower, nameLower) {
			gt.RunCommand("/whisper yes")
		}
	})
}
