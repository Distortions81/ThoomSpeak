//go:build plugin

package main

import (
	"gt"
	"time"
)

// PluginName identifies this plugin.
var PluginName = "Sharecads"

var (
	scOn    bool
	scShare = map[string]time.Time{}
)

// Init toggles the feature with /shcads or Shift+S.
func Init() {
	gt.RegisterCommand("shcads", func(args string) {
		scOn = !scOn
		if scOn {
			gt.Console("* Sharecads enabled")
		} else {
			gt.Console("* Sharecads disabled")
		}
	})
	gt.RegisterTriggers("", []string{"You sense healing energy from "}, handleSharecads)
	gt.AddHotkey("Shift-S", "/shcads")
}

// handleSharecads watches for healing energy messages and shares back once.
func handleSharecads(msg string) {
	if !scOn {
		return
	}
	const prefix = "You sense healing energy from "
	if !gt.StartsWith(msg, prefix) {
		return
	}
	name := gt.TrimEnd(gt.TrimStart(msg, prefix), ".")
	now := time.Now()
	if t, ok := scShare[name]; ok && now.Sub(t) < 3*time.Second {
		return
	}
	if len(scShare) >= 5 {
		oldest := now
		oldestName := ""
		for n, ts := range scShare {
			if ts.Before(oldest) {
				oldest = ts
				oldestName = n
			}
		}
		if oldestName != "" {
			delete(scShare, oldestName)
		}
	}
	scShare[name] = now
	gt.EnqueueCommand("/share " + name)
}
