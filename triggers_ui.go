package main

import (
	"sort"

	"gothoom/eui"
)

var (
	triggersWin  *eui.WindowData
	triggersList *eui.ItemData
)

func makeTriggersWindow() {
	if triggersWin != nil {
		return
	}
	triggersWin = eui.NewWindow()
	triggersWin.Title = "Triggers"
	triggersWin.Size = eui.Point{X: 300, Y: 200}
	triggersWin.Closable = true
	triggersWin.Movable = true
	triggersWin.Resizable = true
	triggersWin.NoScroll = true
	triggersWin.SetZone(eui.HZoneCenter, eui.VZoneMiddleTop)

	flow := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Fixed: true}
	triggersWin.AddItem(flow)

	triggersList = &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Scrollable: true, Fixed: true}
	triggersList.Size = triggersWin.Size
	flow.AddItem(triggersList)
	triggersWin.OnResize = func() { triggersList.Size = triggersWin.Size }
	triggersWin.AddWindow(false)
	refreshTriggersList()
}

func refreshTriggersList() {
	if triggersList == nil {
		return
	}
	triggersList.Contents = triggersList.Contents[:0]
	triggerHandlersMu.RLock()
	type entry struct {
		owner    string
		triggers []string
	}
	byOwner := map[string]*entry{}
	for phrase, hs := range pluginTriggers {
		for _, h := range hs {
			e := byOwner[h.owner]
			if e == nil {
				e = &entry{owner: h.owner}
				byOwner[h.owner] = e
			}
			e.triggers = append(e.triggers, phrase)
		}
	}
	triggerHandlersMu.RUnlock()
	var entries []entry
	for _, e := range byOwner {
		entries = append(entries, *e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].owner < entries[j].owner })
	for _, e := range entries {
		disp := pluginDisplayNames[e.owner]
		if disp == "" {
			disp = e.owner
		}
		triggersList.AddItem(&eui.ItemData{ItemType: eui.ITEM_TEXT, Text: disp + ":", Fixed: true})
		sort.Strings(e.triggers)
		for _, t := range e.triggers {
			triggersList.AddItem(&eui.ItemData{ItemType: eui.ITEM_TEXT, Text: "  " + t, Fixed: true})
		}
	}
	if triggersWin != nil {
		triggersWin.Refresh()
	}
}
