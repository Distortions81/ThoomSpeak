//go:build plugin

package main

import "gt"

// PluginName is shown in the plugin list.
var PluginName = "Quick Reply"

var lastThinker string // remembers who last thought to us

// Init watches chat for "thinks to you" messages and adds /r.
func Init() {
	gt.RegisterChatHandler(func(msg string) {
		lower := gt.Lower(msg)
		if gt.Includes(lower, "thinks to you") {
			words := gt.Words(msg)
			if len(words) > 0 {
				lastThinker = words[0]
			}
		}
	})
	gt.RegisterCommand("r", func(args string) {
		if lastThinker != "" {
			gt.RunCommand("/thinkto " + lastThinker + " " + args)
		}
	})
}
