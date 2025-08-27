package main

import (
	"fmt"
	"sort"
	"strings"

	"gothoom/eui"
)

var (
	macrosWin  *eui.WindowData
	macrosList *eui.ItemData
)

func makeMacrosWindow() {
	if macrosWin != nil {
		return
	}
	macrosWin = eui.NewWindow()
	macrosWin.Title = "Macros"
	macrosWin.Size = eui.Point{X: 300, Y: 200}
	macrosWin.Closable = true
	macrosWin.Movable = true
	macrosWin.Resizable = true
	macrosWin.NoScroll = true
	macrosWin.SetZone(eui.HZoneCenter, eui.VZoneMiddleTop)

	flow := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Fixed: true}
	macrosWin.AddItem(flow)

	macrosList = &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Scrollable: true, Fixed: true}
	macrosList.Size = macrosWin.Size
	flow.AddItem(macrosList)
	macrosWin.OnResize = func() { macrosList.Size = macrosWin.Size }
	macrosWin.AddWindow(false)
	refreshMacrosList()
}

func refreshMacrosList() {
	if macrosList == nil {
		return
	}
	macrosList.Contents = macrosList.Contents[:0]
	macroMu.RLock()
	type pair struct{ short, full string }
	var list []pair
	for _, m := range macroMaps {
		for k, v := range m {
			list = append(list, pair{k, v})
		}
	}
	macroMu.RUnlock()
	sort.Slice(list, func(i, j int) bool { return list[i].short < list[j].short })
	for _, p := range list {
		txt := fmt.Sprintf("%s = %s", p.short, strings.TrimSpace(p.full))
		item := &eui.ItemData{ItemType: eui.ITEM_TEXT, Text: txt, Fixed: true}
		macrosList.AddItem(item)
	}
	if macrosWin != nil {
		macrosWin.Refresh()
	}
}
