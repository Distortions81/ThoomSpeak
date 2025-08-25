package main

import (
	"gt"
)

// Wave demonstrates an exported plugin function that sends a command.
func Wave() {
	gt.RunCommand("/ponder hello world")
}

// Init sets up example hotkeys and commands. Pressing the '1' key
// (aka Ebiten's "Digit1") will call the Wave function above.
func Init() {
	// Bind a hotkey to the Wave function. This uses the special
	// "plugin:<name>" command format under the hood.
	gt.RegisterFunc("wave", Wave)
	gt.AddHotkeyFunc("Digit1", "wave")

	// Slash command `/example` handled locally
	gt.RegisterCommand("example", func(args string) {
		if args == "" {
			args = "hello world"
		}
		gt.Logf("/example invoked with: %s", args)
		// Demonstrate sending a server command immediately
		gt.RunCommand("/ponder " + args)
	})
}
