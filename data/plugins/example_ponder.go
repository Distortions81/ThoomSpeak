package main

import "pluginapi"

// Init registers a simple hotkey example. Pressing the '1' key
// (aka Ebiten's "Digit1") will run "/ponder hello world".
func Init() {
    pluginapi.AddHotkey("Digit1", "/ponder hello world")
}

