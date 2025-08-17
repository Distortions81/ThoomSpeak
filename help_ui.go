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
	helpLines = strings.Split(strings.ReplaceAll(helpText, "\r\n", "\n"), "\n")
	helpWin.OnResize = func() { updateTextWindow(helpWin, helpList, nil, helpLines, 15, "") }
	updateTextWindow(helpWin, helpList, nil, helpLines, 15, "")
}
