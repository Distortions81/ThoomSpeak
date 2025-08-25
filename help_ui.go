//go:build !test

package main

import (
	_ "embed"
	"strings"

	"gothoom/eui"
)

//go:embed data/help.txt
var helpText string

var helpWin *eui.WindowData
var helpList *eui.ItemData
var helpLines []string

func initHelpUI() {
	if helpWin != nil {
		return
	}
	helpWin, helpList, _ = makeTextWindow("Help", eui.HZoneCenter, eui.VZoneMiddleTop, false)
	helpWin.AutoSize = true
	helpLines = strings.Split(strings.ReplaceAll(helpText, "\r\n", "\n"), "\n")
	helpWin.OnResize = func() { updateTextWindow(helpWin, helpList, nil, helpLines, 15, "", monoFaceSource) }
}

func openHelpWindow(anchor *eui.ItemData) {
	if helpWin == nil {
		return
	}
	updateTextWindow(helpWin, helpList, nil, helpLines, 15, "", monoFaceSource)
	if anchor != nil {
		helpWin.MarkOpenNear(anchor)
	} else {
		helpWin.MarkOpen()
	}
	updateTextWindow(helpWin, helpList, nil, helpLines, 15, "", monoFaceSource)
	helpWin.Refresh()
}

func toggleHelpWindow(anchor *eui.ItemData) {
	if helpWin == nil {
		return
	}
	if helpWin.IsOpen() {
		helpWin.Close()
		return
	}
	openHelpWindow(anchor)
}
