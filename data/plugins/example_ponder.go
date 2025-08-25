//go:build plugin

package main

import (
	"gt"
)

var PluginName = "Example"

// Highly recommend vscodium or vscode with the official golang plugin!
func Init() {
	// This registers a slash command '/example'
	gt.RegisterCommand("example", func(args string) {
		if args == "" {
			args = "hello world"
		}
		// This enters this command and sends it
		gt.RunCommand("/ponder " + args)
	})
}
