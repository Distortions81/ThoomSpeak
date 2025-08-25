package main

import "gt"

// Init registers a simple hotkey example. Pressing the '1' key
// (aka Ebiten's "Digit1") will run "/ponder hello world".
func Init() {
	gt.AddHotkey("Digit1", "/ponder hello world")
}
