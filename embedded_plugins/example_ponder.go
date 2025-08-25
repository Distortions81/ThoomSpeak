package main

import "pluginapi"

// Init registers a simple hotkey example. Pressing the '1' key
// (aka Ebiten's "Digit1") will run "/ponder hello world".
func Init() {
	// Example 1: Hotkey
	pluginapi.AddHotkey("Digit1", "/ponder hello world")

	// Example 2: Slash command `/example` handled locally
    pluginapi.RegisterCommand("example", func(args string) {
        if args == "" {
            args = "hello world"
        }
        pluginapi.Logf("/example invoked with: %s", args)
        // Demonstrate sending a server command immediately
        pluginapi.RunCommand("/ponder "+args)
    })

	// Example 3: Register a function callable from hotkeys
    pluginapi.RegisterFunc("wave", func() {
        pluginapi.RunCommand("/ponder hello world")
    })
	// You can also bind directly to a plugin function:
	// pluginapi.AddHotkeyFunc("Digit2", "wave")
}
