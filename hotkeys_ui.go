package main

import "gothoom/eui"

var (
	hotkeysWin            *eui.WindowData
	hotkeysGlobalList     *eui.ItemData
	hotkeysProfessionList *eui.ItemData
	hotkeysCharacterList  *eui.ItemData
	hotkeysProfDD         *eui.ItemData
	hotkeysCharDD         *eui.ItemData
)

func makeHotkeysWindow() {
	if hotkeysWin != nil {
		return
	}
	hotkeysWin = eui.NewWindow()
	hotkeysWin.Title = "Hotkeys"
	hotkeysWin.Size = eui.Point{X: 600, Y: 400}
	hotkeysWin.Closable = true
	hotkeysWin.Resizable = true
	hotkeysWin.Movable = true
	hotkeysWin.NoScroll = true
	hotkeysWin.SetZone(eui.HZoneCenter, eui.VZoneMiddleTop)

	row := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true}
	hotkeysWin.AddItem(row)

	// Global pane
	gPane := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Fixed: true}
	gPane.Size = eui.Point{X: 200, Y: 10}
	gHead := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true}
	gLabel, _ := eui.NewText()
	gLabel.Text = "Global"
	gHead.AddItem(gLabel)
	gAdd, _ := eui.NewButton()
	gAdd.Text = "+"
	gAdd.Size = eui.Point{X: 24, Y: 24}
	gHead.AddItem(gAdd)
	gPane.AddItem(gHead)
	hotkeysGlobalList = &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Scrollable: true, Fixed: true}
	gPane.AddItem(hotkeysGlobalList)
	row.AddItem(gPane)

	// Profession pane
	pPane := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Fixed: true}
	pPane.Size = eui.Point{X: 200, Y: 10}
	pHead := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true}
	hotkeysProfDD, _ = eui.NewDropdown()
	hotkeysProfDD.Options = []string{"fighter", "healer"}
	hotkeysProfDD.Size = eui.Point{X: 140, Y: 24}
	pHead.AddItem(hotkeysProfDD)
	pAdd, _ := eui.NewButton()
	pAdd.Text = "+"
	pAdd.Size = eui.Point{X: 24, Y: 24}
	pHead.AddItem(pAdd)
	pPane.AddItem(pHead)
	hotkeysProfessionList = &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Scrollable: true, Fixed: true}
	pPane.AddItem(hotkeysProfessionList)
	row.AddItem(pPane)

	// Character pane
	cPane := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Fixed: true}
	cPane.Size = eui.Point{X: 200, Y: 10}
	cHead := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true}
	hotkeysCharDD, _ = eui.NewDropdown()
	hotkeysCharDD.Options = characterNames()
	hotkeysCharDD.Size = eui.Point{X: 140, Y: 24}
	cHead.AddItem(hotkeysCharDD)
	cAdd, _ := eui.NewButton()
	cAdd.Text = "+"
	cAdd.Size = eui.Point{X: 24, Y: 24}
	cHead.AddItem(cAdd)
	cPane.AddItem(cHead)
	hotkeysCharacterList = &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Scrollable: true, Fixed: true}
	cPane.AddItem(hotkeysCharacterList)
	row.AddItem(cPane)

	hotkeysWin.AddWindow(false)
}
