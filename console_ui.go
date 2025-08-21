//go:build !test

package main

import "gothoom/eui"

var consoleWin *eui.WindowData
var messagesFlow *eui.ItemData
var inputFlow *eui.ItemData

func updateConsoleWindow() {
	if consoleWin == nil {
		return
	}
	inputMsg := "[Press Enter To Type]"
	if inputActive {
		inputMsg = string(inputText)
	}
	scrollit := messagesFlow.ScrollAtBottom()

	msgs := getConsoleMessages()
	updateTextWindow(consoleWin, messagesFlow, inputFlow, msgs, gs.ConsoleFontSize, inputMsg)
	if messagesFlow != nil {
		// Scroll to bottom on new text; clamp occurs on Refresh.
		if scrollit {
			messagesFlow.Scroll.Y = 1e9
		}
		if consoleWin != nil {
			consoleWin.Refresh()
		}
	}
}

func makeConsoleWindow() {
	if consoleWin != nil {
		return
	}
	consoleWin, messagesFlow, inputFlow = newTextWindow("Console", eui.HZoneLeft, eui.VZoneBottom, true, updateConsoleWindow)
	consoleMessage("Starting...")
	updateConsoleWindow()
}
