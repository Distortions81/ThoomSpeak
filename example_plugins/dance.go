//go:build plugin

package main

import (
	"time"

	"gt"
)

var PluginName = "dance_macros"

func Init() {
	gt.RegisterCommand("dance", func(args string) {
		poses := []string{"celebrate", "leanleft", "leanright", "celebrate"}
		for _, p := range poses {
			gt.RunCommand("/pose " + p)
			time.Sleep(250 * time.Millisecond)
		}
	})
	gt.AddHotkey("Shift-D", "/dance")
}
