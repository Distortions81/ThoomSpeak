//go:build plugin

package main

import "gt"

// PluginName appears in the plugin list.
var PluginName = "Bard Macros"

// Init sets up our commands and hotkeys.
func Init() {
	// /playsong <instrument> <notes>
	gt.RegisterCommand("playsong", func(args string) {
		// Split the arguments into words.
		parts := gt.Words(args)
		if len(parts) < 2 {
			// Need an instrument and at least one note.
			return
		}
		inst := parts[0]
		notes := gt.Join(parts[1:], " ")

		// Pull the instrument from our case, play the notes,
		// then put it back where we found it.
		gt.RunCommand("/equip instrument case")
		gt.RunCommand("/useitem instrument case /remove " + inst)
		gt.RunCommand("/equip " + inst)
		gt.RunCommand("/useitem " + inst + " " + notes)
		gt.RunCommand("/useitem instrument case /add " + inst)
	})

	// A handy hotkey that plays a simple tune.
	gt.AddHotkey("Shift-B", "/playsong pine_flute cfedcgdec")
}
