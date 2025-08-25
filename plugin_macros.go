package main

import (
	"strings"
	"sync"
)

var (
	macroMu   sync.RWMutex
	macroMaps = map[string]map[string]string{}
)

func pluginAddMacro(owner, short, full string) {
	short = strings.ToLower(short)
	macroMu.Lock()
	m := macroMaps[owner]
	if m == nil {
		m = map[string]string{}
		macroMaps[owner] = m
		pluginRegisterInputHandler(func(txt string) string {
			macroMu.RLock()
			local := macroMaps[owner]
			macroMu.RUnlock()
			lower := strings.ToLower(txt)
			for k, v := range local {
				if strings.HasPrefix(lower, k) {
					return v + txt[len(k):]
				}
			}
			return txt
		})
	}
	m[short] = full
	macroMu.Unlock()
}

func pluginAddMacros(owner string, macros map[string]string) {
	for k, v := range macros {
		pluginAddMacro(owner, k, v)
	}
}

func pluginAutoReply(owner, trigger, cmd string) {
	trig := strings.ToLower(trigger)
	pluginRegisterChatHandler(func(msg string) {
		if strings.HasPrefix(strings.ToLower(msg), trig) {
			pluginRunCommand(owner, cmd)
		}
	})
}
