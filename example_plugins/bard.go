//go:build plugin

package main

import (
	"gt"
	"strings"
)

var PluginName = "bard_macros"

func Init() {
	gt.RegisterCommand("playsong", func(args string) {
		parts := strings.Fields(args)
		if len(parts) < 2 {
			return
		}
		inst := parts[0]
		notes := strings.Join(parts[1:], " ")
		gt.RunCommand("/equip instrument case")
		gt.RunCommand("/useitem instrument case /remove " + inst)
		gt.RunCommand("/equip " + inst)
		gt.RunCommand("/useitem " + inst + " " + notes)
		gt.RunCommand("/useitem instrument case /add " + inst)
	})
	gt.AddHotkey("Shift-B", "/playsong pine_flute cfedcgdec")
}
