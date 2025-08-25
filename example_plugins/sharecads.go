//go:build plugin

package main

import (
	"gt"
	"strings"
	"time"
)

var PluginName = "sharecads"

var (
	scOn    bool
	scShare = map[string]time.Time{}
)

func Init() {
	gt.RegisterCommand("shcads", func(args string) {
		scOn = !scOn
		if scOn {
			gt.Console("* Sharecads enabled")
		} else {
			gt.Console("* Sharecads disabled")
		}
	})
	gt.RegisterChatHandler(handleSharecads)
	gt.AddHotkey("Shift-S", "/shcads")
}

func handleSharecads(msg string) {
	if !scOn {
		return
	}
	const prefix = "You sense healing energy from "
	if !strings.HasPrefix(msg, prefix) {
		return
	}
	name := strings.TrimSuffix(strings.TrimPrefix(msg, prefix), ".")
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
