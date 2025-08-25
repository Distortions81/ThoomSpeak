//go:build plugin

package main

import "gt"

var PluginName = "Numpad Poser"

func Init() {
	gt.AddHotkey("Numpad1", "/pose leanleft")
	gt.AddHotkey("Numpad2", "/pose akimbo")
	gt.AddHotkey("Numpad3", "/pose leanright")
	gt.AddHotkey("Numpad4", "/pose kneel")
	gt.AddHotkey("Numpad5", "/pose sit")
	gt.AddHotkey("Numpad6", "/pose angry")
	gt.AddHotkey("Numpad7", "/pose lie")
	gt.AddHotkey("Numpad8", "/pose seated")
	gt.AddHotkey("Numpad9", "/pose celebrate")
}
