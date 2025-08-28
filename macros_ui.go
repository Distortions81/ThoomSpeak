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
	type entry struct {
		owner  string
		macros []pair
	}
	var plugins []entry
	for owner, m := range macroMaps {
		e := entry{owner: owner}
		for k, v := range m {
			e.macros = append(e.macros, pair{k, v})
		}
		plugins = append(plugins, e)
	}
	macroMu.RUnlock()
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].owner < plugins[j].owner })
	for _, p := range plugins {
		disp := pluginDisplayNames[p.owner]
		if disp == "" {
			disp = p.owner
		}
		macrosList.AddItem(&eui.ItemData{ItemType: eui.ITEM_TEXT, Text: disp + ":", Fixed: true})
		sort.Slice(p.macros, func(i, j int) bool { return p.macros[i].short < p.macros[j].short })
		for _, m := range p.macros {
			txt := fmt.Sprintf("  %s = %s", m.short, strings.TrimSpace(m.full))
			item := &eui.ItemData{ItemType: eui.ITEM_TEXT, Text: txt, Fixed: true}
			macrosList.AddItem(item)
		}
	}
	if macrosWin != nil {
		macrosWin.Refresh()
	}
}
