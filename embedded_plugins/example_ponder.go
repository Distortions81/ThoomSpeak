package main

import (
	"pluginapi"
	"strings"
)

// Init registers a simple hotkey example. Pressing the '1' key
// (aka Ebiten's "Digit1") will run an exported plugin function.
func Init() {
	// Example 1: Bind a hotkey directly to a plugin function
	// (this uses the special command format "plugin:<name>" under the hood).
	pluginapi.AddHotkeyFunc("Digit1", "wave")

	// Example 2: Slash command `/example` handled locally
	pluginapi.RegisterCommand("example", func(args string) {
		if args == "" {
			args = "hello world"
		}
		pluginapi.Logf("/example invoked with: %s", args)
		// Demonstrate sending a server command immediately
		pluginapi.RunCommand("/ponder " + args)

		if strings.ToLower(args) == "test" {
			pluginapi.Logf("test")
		}
	})

	// Example 3: Register a function callable from hotkeys
	pluginapi.RegisterFunc("wave", func() {
		pluginapi.RunCommand("/ponder hello world")
	})
	// Another binding example:
	// pluginapi.AddHotkeyFunc("Digit2", "wave")
}
