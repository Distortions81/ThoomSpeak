package main

import (
	"fmt"
	"strings"

	"gothoom/eui"
)

var (
	hotbarWin       *eui.WindowData
	hotbarSlots     [10]*eui.ItemData
	hotbarEditLabel *eui.ItemData
	hotbarEdit      bool
	hotbarNormalBG  eui.Color

	hotbarCommands [10][]string
	hotbarCmdQueue []string

	hotbarEditorWin    *eui.WindowData
	hotbarEditorInputs []*eui.ItemData
	hotbarCurrentSlot  int
)

// makeHotbar initializes the hotbar window and slots.
func makeHotbar() {
	hotbarWin = eui.NewWindow()
	hotbarWin.Title = ""
	hotbarWin.Closable = false
	hotbarWin.Resizable = false
	hotbarWin.AutoSize = true
	hotbarWin.NoScroll = true
	hotbarWin.SetZone(eui.HZoneCenter, eui.VZoneBottom)
	hotbarNormalBG = hotbarWin.BGColor

	hotbarEditLabel, _ = eui.NewText()
	hotbarEditLabel.Text = "edit mode on"
	hotbarEditLabel.TextColor = eui.Color{255, 255, 255, 255}
	hotbarEditLabel.Invisible = true
	hotbarWin.AddItem(hotbarEditLabel)

	flow := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL}
	for i := 0; i < 10; i++ {
		btn, ev := eui.NewButton()
		btn.Text = fmt.Sprintf("%d", i)
		idx := i
		ev.Handle = func(ev eui.UIEvent) {
			if ev.Type == eui.EventClick {
				triggerHotbarSlot(idx)
			}
		}
		hotbarSlots[i] = btn
		flow.AddItem(btn)
	}

	gearBtn, gearEv := eui.NewButton()
	gearBtn.Text = "⚙"
	gearEv.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			toggleHotbarEdit()
		}
	}
	flow.AddItem(gearBtn)

	hotbarWin.AddItem(flow)
	hotbarWin.AddWindow(false)
	hotbarWin.MarkOpen()
}

// toggleHotbarEdit toggles the edit mode for the hotbar.
func toggleHotbarEdit() {
	hotbarEdit = !hotbarEdit
	if hotbarEdit {
		hotbarWin.BGColor = eui.Color{255, 0, 0, 255}
		hotbarEditLabel.Invisible = false
	} else {
		hotbarWin.BGColor = hotbarNormalBG
		hotbarEditLabel.Invisible = true
		if hotbarEditorWin != nil {
			hotbarEditorWin.Close()
		}
	}
	hotbarWin.Dirty = true
}

// triggerHotbarSlot executes or edits the specified slot.
func triggerHotbarSlot(idx int) {
	if hotbarEdit {
		openHotbarEditor(idx)
		return
	}
	cmds := hotbarCommands[idx]
	if len(cmds) == 0 {
		return
	}
	if pendingCommand == "" {
		pendingCommand = cmds[0]
		if len(cmds) > 1 {
			hotbarCmdQueue = append(hotbarCmdQueue, cmds[1:]...)
		}
	} else {
		hotbarCmdQueue = append(hotbarCmdQueue, cmds...)
	}
}

// openHotbarEditor opens the editor window for a slot.
func openHotbarEditor(slot int) {
	hotbarCurrentSlot = slot
	if hotbarEditorWin == nil {
		hotbarEditorWin = eui.NewWindow()
		hotbarEditorWin.Resizable = true
		hotbarEditorWin.AutoSize = true
		hotbarEditorWin.SetZone(eui.HZoneCenter, eui.VZoneCenter)
	} else {
		hotbarEditorWin.Contents = nil
	}
	hotbarEditorWin.Title = fmt.Sprintf("Slot %d", slot)

	inputsFlow := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL}
	hotbarEditorInputs = hotbarEditorInputs[:0]
	for _, cmd := range hotbarCommands[slot] {
		in, _ := eui.NewInput()
		in.Text = cmd
		hotbarEditorInputs = append(hotbarEditorInputs, in)
		inputsFlow.AddItem(in)
	}

	addBtn, addEv := eui.NewButton()
	addBtn.Text = "+"
	addEv.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			in, _ := eui.NewInput()
			hotbarEditorInputs = append(hotbarEditorInputs, in)
			inputsFlow.AddItem(in)
			hotbarEditorWin.Refresh()
		}
	}

	container := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL}
	container.AddItem(inputsFlow)
	container.AddItem(addBtn)

	hotbarEditorWin.AddItem(container)
	hotbarEditorWin.OnClose = saveHotbarEditor
	hotbarEditorWin.AddWindow(false)
	hotbarEditorWin.MarkOpen()
}

// saveHotbarEditor persists the edited commands when the editor window closes.
func saveHotbarEditor() {
	cmds := make([]string, 0, len(hotbarEditorInputs))
	for _, in := range hotbarEditorInputs {
		if txt := strings.TrimSpace(in.Text); txt != "" {
			cmds = append(cmds, txt)
		}
	}
	hotbarCommands[hotbarCurrentSlot] = cmds
	hotbarEditorInputs = nil
}
