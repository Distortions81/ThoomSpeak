//go:build !test

package main

import "gothoom/eui"

var consoleWin *eui.WindowData
var messagesFlow *eui.ItemData
var inputFlow *eui.ItemData
var consolePrevCount int

func updateConsoleWindow() {
	if consoleWin == nil {
		return
	}
	inputMsg := "[Press Enter To Type]"
	if inputActive {
		inputMsg = string(inputText)
	}
	lines := getConsoleMessages()
	updateTextWindow(consoleWin, messagesFlow, inputFlow, lines, gs.ConsoleFontSize, inputMsg)
	if messagesFlow != nil && len(lines) > consolePrevCount {
		// Scroll to bottom on new text; clamp occurs on Refresh.
		messagesFlow.Scroll.Y = 1e9
		if consoleWin != nil {
			consoleWin.Refresh()
		}
	}
	consolePrevCount = len(lines)
}

func makeConsoleWindow() {
	if consoleWin != nil {
		return
	}
	consoleWin, messagesFlow, inputFlow = makeTextWindow("Console", eui.HZoneLeft, eui.VZoneBottom, true)
	// Rewrap and refresh on window resize
	consoleWin.OnResize = func() {
		updateConsoleWindow()
		if consoleWin != nil {
			consoleWin.Refresh()
		}
	}
	consoleMessage("Starting...")
	updateConsoleWindow()
}
