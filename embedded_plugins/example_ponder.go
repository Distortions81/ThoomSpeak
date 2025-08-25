package main

import (
	"pluginapi"
	"strings"
)

// Wave demonstrates an exported plugin function that sends a command.
func Wave() {
	pluginapi.RunCommand("/ponder hello world")
}

// Init sets up example hotkeys and commands. Pressing the '1' key
// (aka Ebiten's "Digit1") will call the Wave function above.
func Init() {
	// Bind a hotkey to the Wave function. This uses the special
	// "plugin:<name>" command format under the hood.
	pluginapi.RegisterFunc("wave", Wave)
	pluginapi.AddHotkeyFunc("Digit1", "wave")

	// Slash command `/example` handled locally
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

	// Another binding example:
	// pluginapi.AddHotkey("Digit2", "/ponder hello world")
}
