package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gothoom/eui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

const hotkeysFile = "global-hotkeys.json"

type HotkeyCommand struct {
	Command  string `json:"command,omitempty"`
	Function string `json:"function,omitempty"`
	Plugin   string `json:"plugin,omitempty"`
}

type Hotkey struct {
	Name     string          `json:"name,omitempty"`
	Combo    string          `json:"combo"`
	Commands []HotkeyCommand `json:"commands"`
	Plugin   string          `json:"plugin,omitempty"`
	Disabled bool            `json:"disabled,omitempty"`
}

type hotkeyFuncRef struct {
	Name   string
	Plugin string
}

var (
	hotkeys          []Hotkey
	hotkeysMu        sync.RWMutex
	hotkeysWin       *eui.WindowData
	hotkeysList      *eui.ItemData
	hotkeyEditWin    *eui.WindowData
	hotkeyComboText  *eui.ItemData
	hotkeyNameInput  *eui.ItemData
	hotkeyCmdSection *eui.ItemData
	hotkeyCmdInputs  []*eui.ItemData
	hotkeyCmdFuncs   []hotkeyFuncRef
	hotkeyFnLabel    *eui.ItemData
	editingHotkey    int = -1

	recording     bool
	recordStart   time.Time
	recordTarget  *eui.ItemData
	recordedCombo string
)

func loadHotkeys() {
	path := filepath.Join(dataDirPath, hotkeysFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	type hotkeyJSON struct {
		Combo    string          `json:"combo"`
		Name     string          `json:"name,omitempty"`
		Commands []HotkeyCommand `json:"commands"`
		Command  string          `json:"command"`
		Text     string          `json:"text,omitempty"`
		Plugin   string          `json:"plugin,omitempty"`
		Disabled bool            `json:"disabled,omitempty"`
	}
	var raw []hotkeyJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}
	var newList []Hotkey
	for _, r := range raw {
		hk := Hotkey{Combo: r.Combo, Name: r.Name, Plugin: r.Plugin, Disabled: r.Disabled}
		if len(r.Commands) > 0 {
			hk.Commands = make([]HotkeyCommand, len(r.Commands))
			copy(hk.Commands, r.Commands)
			for i := range hk.Commands {
				c := &hk.Commands[i]
				if c.Function == "" && strings.HasPrefix(strings.ToLower(c.Command), "plugin:") {
					c.Function = strings.TrimSpace(strings.TrimPrefix(c.Command, "plugin:"))
					c.Command = ""
				}
			}
		} else if r.Command != "" {
			cmd := strings.TrimSpace(r.Command + " " + r.Text)
			if strings.HasPrefix(strings.ToLower(cmd), "plugin:") {
				fn := strings.TrimSpace(strings.TrimPrefix(cmd, "plugin:"))
				hk.Commands = []HotkeyCommand{{Function: fn}}
			} else if cmd != "" {
				hk.Commands = []HotkeyCommand{{Command: cmd}}
			}
		}
		newList = append(newList, hk)
	}
	hotkeysMu.Lock()
	hotkeys = newList
	hotkeysMu.Unlock()
	refreshHotkeysList()
}

func saveHotkeys() {
	path := filepath.Join(dataDirPath, hotkeysFile)
	_ = os.MkdirAll(dataDirPath, 0o755)
	// snapshot under read lock
	hotkeysMu.RLock()
	snap := append([]Hotkey(nil), hotkeys...)
	hotkeysMu.RUnlock()
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

func pluginHotkeys(owner string) []Hotkey {
	hotkeysMu.RLock()
	defer hotkeysMu.RUnlock()
	var list []Hotkey
	for _, hk := range hotkeys {
		if hk.Plugin == owner {
			list = append(list, hk)
		}
	}
	return list
}

func pluginRemoveHotkey(owner, combo string) {
	hotkeysMu.Lock()
	for i := 0; i < len(hotkeys); i++ {
		hk := hotkeys[i]
		if hk.Plugin == owner && hk.Combo == combo {
			hotkeys = append(hotkeys[:i], hotkeys[i+1:]...)
			i--
		}
	}
	hotkeysMu.Unlock()
	refreshHotkeysList()
	saveHotkeys()
}

func makeHotkeysWindow() {
	if hotkeysWin != nil {
		return
	}
	hotkeysWin = eui.NewWindow()
	hotkeysWin.Title = "Hotkeys"
	hotkeysWin.Size = eui.Point{X: 260, Y: 300}
	hotkeysWin.Closable = true
	hotkeysWin.Movable = true
	hotkeysWin.Resizable = true
	hotkeysWin.NoScroll = true
	hotkeysWin.SetZone(eui.HZoneCenter, eui.VZoneMiddleTop)

	flow := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Fixed: true}
	hotkeysWin.AddItem(flow)

	help := &eui.ItemData{ItemType: eui.ITEM_TEXT, Text: "@/@clicked -> last clicked, @hovered -> last hovered", Fixed: true}
	help.Size = eui.Point{X: hotkeysWin.Size.X, Y: 20}
	help.FontSize = 10
	flow.AddItem(help)

	btnRow := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true}
	addBtn, addEvents := eui.NewButton()
	addBtn.Text = "+"
	addBtn.Size = eui.Point{X: 20, Y: 20}
	addBtn.FontSize = 14
	addEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			openHotkeyEditor(-1)
		}
	}
	btnRow.AddItem(addBtn)
	btnRow.Size = eui.Point{X: hotkeysWin.Size.X, Y: addBtn.Size.Y}
	flow.AddItem(btnRow)

	hotkeysList = &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Scrollable: true, Fixed: true}
	hotkeysList.Size = eui.Point{X: hotkeysWin.Size.X, Y: hotkeysWin.Size.Y - btnRow.Size.Y}
	flow.AddItem(hotkeysList)

	hotkeysWin.AddWindow(false)
	refreshHotkeysList()
}

func refreshHotkeysList() {
	if hotkeysList == nil {
		return
	}
	hotkeysList.Contents = hotkeysList.Contents[:0]
	// snapshot to avoid concurrent mutation during UI build
	hotkeysMu.RLock()
	list := append([]Hotkey(nil), hotkeys...)
	hotkeysMu.RUnlock()

	// global hotkeys
	for i, hk := range list {
		if hk.Plugin != "" {
			continue
		}
		idx := i
		row := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true}
		row.Size = eui.Point{X: 220, Y: 20}
		btn, events := eui.NewButton()
		btnText := hk.Combo
		if hk.Name != "" {
			btnText = hk.Name + " : " + hk.Combo
		}
		if len(hk.Commands) > 0 {
			text := hk.Commands[0].Command
			if text == "" && hk.Commands[0].Function != "" {
				text = "plugin:" + hk.Commands[0].Function
			}
			if len(hk.Commands) > 1 {
				text += " ..."
			}
			btnText += " -> " + text
		}
		btn.Text = btnText
		btn.Size = eui.Point{X: 200, Y: 20}
		btn.FontSize = 10
		events.Handle = func(ev eui.UIEvent) {
			if ev.Type == eui.EventClick {
				openHotkeyEditor(idx)
			}
		}
		row.AddItem(btn)
		delBtn, delEvents := eui.NewButton()
		delBtn.Text = "x"
		delBtn.Size = eui.Point{X: 20, Y: 20}
		delBtn.FontSize = 10
		delEvents.Handle = func(ev eui.UIEvent) {
			if ev.Type == eui.EventClick {
				confirmRemoveHotkey(idx)
			}
		}
		row.AddItem(delBtn)
		hotkeysList.AddItem(row)
	}

	// plugin hotkeys header and list
	headerAdded := false
	for i, hk := range list {
		if hk.Plugin == "" {
			continue
		}
		if !headerAdded {
			label := &eui.ItemData{ItemType: eui.ITEM_TEXT, Text: "Plugin Hotkeys", Fixed: true}
			label.Size = eui.Point{X: 220, Y: 20}
			hotkeysList.AddItem(label)
			headerAdded = true
		}
		idx := i
		row := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true}
		row.Size = eui.Point{X: 220, Y: 20}
		cb, cbEvents := eui.NewCheckbox()
		cb.Checked = !hk.Disabled
		cbEvents.Handle = func(ev eui.UIEvent) {
			if ev.Type == eui.EventClick {
				hotkeysMu.Lock()
				if idx >= 0 && idx < len(hotkeys) {
					hotkeys[idx].Disabled = !ev.Checked
				}
				hotkeysMu.Unlock()
				saveHotkeys()
			}
		}
		row.AddItem(cb)
		text := hk.Combo
		if hk.Name != "" {
			text = hk.Name + " : " + hk.Combo
		}
		disp := pluginDisplayNames[hk.Plugin]
		if disp == "" {
			disp = hk.Plugin
		}
		lbl := &eui.ItemData{ItemType: eui.ITEM_TEXT, Text: disp + " -> " + text, Fixed: true}
		lbl.Size = eui.Point{X: 200, Y: 20}
		row.AddItem(lbl)
		hotkeysList.AddItem(row)
	}

	hotkeysList.Dirty = true
	if hotkeysWin != nil {
		hotkeysWin.Refresh()
	}
}

func confirmRemoveHotkey(idx int) {
	hotkeysMu.RLock()
	if idx < 0 || idx >= len(hotkeys) {
		hotkeysMu.RUnlock()
		return
	}
	hk := hotkeys[idx]
	hotkeysMu.RUnlock()
	showPopup(
		"Remove Hotkey",
		fmt.Sprintf("Remove hotkey %s : %s?", hk.Name, hk.Combo),
		[]popupButton{
			{Text: "Cancel"},
			{Text: "Remove", Color: &eui.ColorDarkRed, HoverColor: &eui.ColorRed, Action: func() {
				hotkeysMu.Lock()
				if idx >= 0 && idx < len(hotkeys) {
					hotkeys = append(hotkeys[:idx], hotkeys[idx+1:]...)
				}
				hotkeysMu.Unlock()
				saveHotkeys()
				refreshHotkeysList()
			}},
		},
	)
}

func openHotkeyEditor(idx int) {
	if hotkeyEditWin != nil {
		return
	}
	editingHotkey = idx
	hotkeyEditWin = eui.NewWindow()
	hotkeyEditWin.OnClose = func() { hotkeyEditWin = nil }
	hotkeyEditWin.Title = "Hotkey"
	hotkeyEditWin.Size = eui.Point{X: 400, Y: 160}
	hotkeyEditWin.AutoSize = true
	hotkeyEditWin.Closable = true
	hotkeyEditWin.Movable = true
	hotkeyEditWin.Resizable = false
	hotkeyEditWin.NoScroll = true
	hotkeyEditWin.SetZone(eui.HZoneCenter, eui.VZoneMiddleTop)

	flow := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Fixed: true}
	hotkeyEditWin.AddItem(flow)

	row := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true}
	label, _ := eui.NewText()
	label.Text = "Keys:"
	label.Size = eui.Point{X: 40, Y: 20}
	label.FontSize = 12
	row.AddItem(label)
	hotkeyComboText, _ = eui.NewText()
	hotkeyComboText.Text = ""
	hotkeyComboText.Size = eui.Point{X: 200, Y: 20}
	hotkeyComboText.FontSize = 12
	row.AddItem(hotkeyComboText)
	recordBtn, recordEvents := eui.NewButton()
	recordBtn.Text = "Record"
	recordBtn.Size = eui.Point{X: 60, Y: 20}
	recordBtn.FontSize = 12
	recordEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			startRecording(hotkeyComboText)
		}
	}
	row.AddItem(recordBtn)
	flow.AddItem(row)

	nameRow := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true}
	nameLabel, _ := eui.NewText()
	nameLabel.Text = "Name:"
	nameLabel.Size = eui.Point{X: 40, Y: 20}
	nameLabel.FontSize = 12
	nameRow.AddItem(nameLabel)
	hotkeyNameInput, _ = eui.NewInput()
	hotkeyNameInput.Size = eui.Point{X: hotkeyEditWin.Size.X - 40, Y: 20}
	hotkeyNameInput.FontSize = 12
	nameRow.AddItem(hotkeyNameInput)
	flow.AddItem(nameRow)

	hotkeyCmdSection = &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Fixed: true}
	flow.AddItem(hotkeyCmdSection)
	hotkeyCmdInputs = nil
	hotkeyCmdFuncs = nil

	// Row to add a plain command input
	addCmdRow, addCmdEvents := eui.NewButton()
	addCmdRow.Text = "+"
	addCmdRow.Size = eui.Point{X: 20, Y: 20}
	addCmdRow.FontSize = 14
	addCmdEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			addHotkeyCommand("", "", "")
		}
	}
	flow.AddItem(addCmdRow)

	// Row to add a plugin function via dropdown
	fnRow := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true}
	fnLabel, _ := eui.NewText()
	fnLabel.Text = "Function:"
	fnLabel.Size = eui.Point{X: 64, Y: 20}
	fnLabel.FontSize = 12
	fnRow.AddItem(fnLabel)

	fnDD, fnDDEvents := eui.NewDropdown()
	fnDD.Size = eui.Point{X: hotkeyEditWin.Size.X - 120, Y: 20}
	infos := pluginFunctionInfos()
	fnOpts := make([]string, len(infos))
	fnKeys := make([]string, len(infos))
	for i, inf := range infos {
		fnOpts[i] = inf.Name
		fnKeys[i] = inf.Plugin
	}
	fnDD.Options = append([]string{"none"}, fnOpts...)
	fnDD.Selected = 0
	fnSel := ""
	fnSelKey := ""
	fnDDEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventDropdownSelected {
			fnDD.Selected = ev.Index
			if ev.Index > 0 && ev.Index <= len(fnOpts) {
				fnSel = fnOpts[ev.Index-1]
				fnSelKey = fnKeys[ev.Index-1]
			} else {
				fnSel = ""
				fnSelKey = ""
			}
			selectHotkeyFunction(fnSel, fnSelKey)
		}
	}
	fnRow.AddItem(fnDD)

	addFnBtn, addFnEv := eui.NewButton()
	addFnBtn.Text = "+"
	addFnBtn.Size = eui.Point{X: 20, Y: 20}
	addFnBtn.FontSize = 14
	addFnBtn.Disabled = len(fnOpts) == 0
	addFnEv.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			if fnSel == "" {
				return
			}
			addHotkeyCommand("", fnSel, fnSelKey)
		}
	}
	fnRow.AddItem(addFnBtn)
	flow.AddItem(fnRow)

	btnRow := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true}
	okBtn, okEvents := eui.NewButton()
	okBtn.Text = "OK"
	okBtn.Size = eui.Point{X: 80, Y: 20}
	okBtn.FontSize = 12
	okEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			finishHotkeyEdit(true)
		}
	}
	btnRow.AddItem(okBtn)

	cancelBtn, cancelEvents := eui.NewButton()
	cancelBtn.Text = "Cancel"
	cancelBtn.Size = eui.Point{X: 80, Y: 20}
	cancelBtn.FontSize = 12
	cancelEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			finishHotkeyEdit(false)
		}
	}
	btnRow.AddItem(cancelBtn)

	flow.AddItem(btnRow)

	hotkeysMu.RLock()
	curLen := len(hotkeys)
	if idx >= 0 && idx < curLen {
		hk := hotkeys[idx]
		hotkeysMu.RUnlock()
		hotkeyComboText.Text = hk.Combo
		hotkeyNameInput.Text = hk.Name
		if len(hk.Commands) > 0 {
			for _, c := range hk.Commands {
				addHotkeyCommand(c.Command, c.Function, c.Plugin)
			}
		} else {
			addHotkeyCommand("", "", "")
		}
	} else {
		hotkeysMu.RUnlock()
		addHotkeyCommand("", "", "")
	}

	hotkeyEditWin.AddWindow(true)
	hotkeyEditWin.MarkOpen()
	wrapHotkeyInputs()
}

func addHotkeyCommand(cmd, fn, plugin string) {
	if hotkeyCmdSection == nil {
		return
	}
	if fn != "" {
		fnLabel, _ := eui.NewText()
		fnLabel.Text = "Function: " + fn
		fnLabel.Size = eui.Point{X: hotkeyEditWin.Size.X - 40, Y: 20}
		fnLabel.FontSize = 12
		hotkeyCmdSection.AddItem(fnLabel)
		if len(hotkeyCmdInputs) == 0 {
			hotkeyFnLabel = fnLabel
		}
	}
	cmdLabel, _ := eui.NewText()
	cmdLabel.Text = "Command:"
	cmdLabel.Size = eui.Point{X: hotkeyEditWin.Size.X - 40, Y: 20}
	cmdLabel.FontSize = 12
	hotkeyCmdSection.AddItem(cmdLabel)

	var cmdEvents *eui.EventHandler
	cmdInput, cmdEvents := eui.NewInput()
	cmdInput.Size = eui.Point{X: hotkeyEditWin.Size.X - 40, Y: 20}
	cmdInput.FontSize = 12
	cmdInput.Scrollable = true
	cmdInput.Text = cmd
	cmdEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventInputChanged {
			wrapHotkeyInputs()
		}
	}
	hotkeyCmdSection.AddItem(cmdInput)
	hotkeyCmdInputs = append(hotkeyCmdInputs, cmdInput)
	hotkeyCmdFuncs = append(hotkeyCmdFuncs, hotkeyFuncRef{Name: fn, Plugin: plugin})

	hotkeyEditWin.Refresh()
	wrapHotkeyInputs()
}

// selectHotkeyFunction applies a function selection to the first blank
// hotkey command slot. It updates the stored function reference and shows a
// label in the editor so users don't need to press the add button for the
// initial function choice.
func selectHotkeyFunction(fn, plugin string) {
	if hotkeyCmdSection == nil {
		return
	}
	// Only apply to a single empty command slot.
	if len(hotkeyCmdInputs) != 1 || hotkeyCmdInputs[0].Text != "" {
		return
	}
	hotkeyCmdFuncs[0] = hotkeyFuncRef{Name: fn, Plugin: plugin}
	if fn == "" {
		if hotkeyFnLabel != nil {
			hotkeyCmdSection.Contents = hotkeyCmdSection.Contents[1:]
			hotkeyFnLabel = nil
			hotkeyEditWin.Refresh()
		}
		return
	}
	if hotkeyFnLabel == nil {
		hotkeyFnLabel, _ = eui.NewText()
		hotkeyFnLabel.Size = eui.Point{X: hotkeyEditWin.Size.X - 40, Y: 20}
		hotkeyFnLabel.FontSize = 12
		hotkeyCmdSection.Contents = append([]*eui.ItemData{hotkeyFnLabel}, hotkeyCmdSection.Contents...)
	}
	hotkeyFnLabel.Text = "Function: " + fn
	hotkeyEditWin.Refresh()
}

func wrapHotkeyInputs() {
	if hotkeyEditWin == nil {
		return
	}
	ui := eui.UIScale()
	fs := float32(12)
	if len(hotkeyCmdInputs) > 0 {
		fs = hotkeyCmdInputs[0].FontSize
	}
	facePx := float64(fs * ui)
	src := eui.FontSource()
	if src == nil {
		if s, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF)); err == nil {
			src = s
		} else {
			return
		}
	}
	goFace := &text.GoTextFace{Source: src, Size: facePx}
	metrics := goFace.Metrics()
	linePx := math.Ceil(metrics.HAscent + metrics.HDescent + 2)
	rowUnits := float32(linePx) / ui
	padPx := float64(6) * float64(ui)

	resize := func(it *eui.ItemData) {
		if it == nil {
			return
		}
		raw := strings.ReplaceAll(it.Text, "\n", " ")
		_, lines := wrapText(raw, goFace, float64(it.Size.X*ui)-padPx)
		if len(lines) == 0 {
			lines = []string{""}
		}
		if n := len(raw) - len(strings.TrimRight(raw, " ")); n > 0 {
			lines[len(lines)-1] += strings.Repeat(" ", n)
		}
		it.Text = strings.Join(lines, "\n")
		if it.TextPtr != nil {
			*it.TextPtr = it.Text
		}
		it.Size.Y = rowUnits * float32(len(lines))
	}

	for _, it := range hotkeyCmdInputs {
		resize(it)
	}
	hotkeyEditWin.Refresh()
}

func finishHotkeyEdit(save bool) {
	if save {
		combo := strings.ReplaceAll(hotkeyComboText.Text, "\n", " ")
		name := strings.ReplaceAll(hotkeyNameInput.Text, "\n", " ")
		cmds := []HotkeyCommand{}
		max := len(hotkeyCmdInputs)
		if len(hotkeyCmdFuncs) > max {
			max = len(hotkeyCmdFuncs)
		}
		for i := 0; i < max; i++ {
			cmd := ""
			if i < len(hotkeyCmdInputs) {
				cmd = strings.ReplaceAll(hotkeyCmdInputs[i].Text, "\n", " ")
			}
			fnName := ""
			fnPlugin := ""
			if i < len(hotkeyCmdFuncs) {
				fnName = hotkeyCmdFuncs[i].Name
				fnPlugin = hotkeyCmdFuncs[i].Plugin
			}
			if cmd != "" || fnName != "" {
				cmds = append(cmds, HotkeyCommand{Command: cmd, Function: fnName, Plugin: fnPlugin})
			}
		}
		if combo != "" {
			hk := Hotkey{Name: name, Combo: combo, Commands: cmds}
			hotkeysMu.Lock()
			if editingHotkey >= 0 && editingHotkey < len(hotkeys) {
				hotkeys[editingHotkey] = hk
				hotkeysMu.Unlock()
				saveHotkeys()
				refreshHotkeysList()
			} else {
				hotkeys = append(hotkeys, hk)
				hotkeysMu.Unlock()
				saveHotkeys()
				refreshHotkeysList()
			}
		}
	}
	if hotkeyEditWin != nil {
		hotkeyEditWin.Close()
		hotkeyEditWin = nil
	}
}

func startRecording(target *eui.ItemData) {
	recording = true
	recordStart = time.Now()
	recordTarget = target
	recordedCombo = ""
	if recordTarget != nil {
		recordTarget.Text = "Recording..."
		recordTarget.Dirty = true
		if hotkeyEditWin != nil {
			hotkeyEditWin.Refresh()
		}
	}
}

func finishRecording() {
	recording = false
	if recordTarget != nil {
		if recordedCombo == "" {
			recordTarget.Text = ""
		} else {
			recordTarget.Text = recordedCombo
		}
		recordTarget.Dirty = true
		if hotkeyEditWin != nil {
			hotkeyEditWin.Refresh()
		}
	}
}

func detectCombo() string {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if combo := comboFromMouseWithKey(ebiten.MouseButtonLeft); combo != "" {
			return combo
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		return comboFromMouse(ebiten.MouseButtonRight)
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) {
		return comboFromMouse(ebiten.MouseButtonMiddle)
	}
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		if isModifier(k) {
			continue
		}
		return comboFromKey(k)
	}
	return ""
}

func comboFromKey(k ebiten.Key) string {
	mods := currentMods()
	mods = append(mods, k.String())
	return strings.Join(mods, "-")
}

func comboFromMouse(b ebiten.MouseButton) string {
	mods := currentMods()
	name := mouseButtonName(b)
	mods = append(mods, name)
	return strings.Join(mods, "-")
}

func comboFromMouseWithKey(b ebiten.MouseButton) string {
	mods := currentMods()
	keys := inpututil.AppendPressedKeys(nil)
	keyPart := ""
	for _, k := range keys {
		if isModifier(k) {
			continue
		}
		keyPart = k.String()
		break
	}
	if keyPart == "" && len(mods) == 0 {
		return ""
	}
	if keyPart != "" {
		mods = append(mods, keyPart)
	}
	name := mouseButtonName(b)
	mods = append(mods, name)
	return strings.Join(mods, "-")
}

func currentMods() []string {
	mods := []string{}
	if ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight) {
		mods = append(mods, "Ctrl")
	}
	if ebiten.IsKeyPressed(ebiten.KeyAlt) || ebiten.IsKeyPressed(ebiten.KeyAltLeft) || ebiten.IsKeyPressed(ebiten.KeyAltRight) {
		mods = append(mods, "Alt")
	}
	if ebiten.IsKeyPressed(ebiten.KeyShift) || ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
		mods = append(mods, "Shift")
	}
	return mods
}

func mouseButtonName(b ebiten.MouseButton) string {
	switch b {
	case ebiten.MouseButtonLeft:
		return "LeftClick"
	case ebiten.MouseButtonRight:
		return "RightClick"
	case ebiten.MouseButtonMiddle:
		return "MiddleClick"
	default:
		return fmt.Sprintf("Mouse %d", b)
	}
}

func isModifier(k ebiten.Key) bool {
	switch k {
	case ebiten.KeyShift, ebiten.KeyShiftLeft, ebiten.KeyShiftRight,
		ebiten.KeyControl, ebiten.KeyControlLeft, ebiten.KeyControlRight,
		ebiten.KeyAlt, ebiten.KeyAltLeft, ebiten.KeyAltRight:
		return true
	}
	return false
}

func applyHotkeyVars(cmd string) string {
	if strings.Contains(cmd, "@clicked") || strings.Contains(cmd, "@hovered") || strings.Contains(cmd, "@") {
		lastClickMu.Lock()
		clickedName := lastClick.Mobile.Name
		lastClickMu.Unlock()

		lastHoverMu.Lock()
		hoveredName := lastHover.Mobile.Name
		lastHoverMu.Unlock()

		if clickedName != "" {
			cmd = strings.ReplaceAll(cmd, "@clicked", clickedName)
			cmd = strings.ReplaceAll(cmd, "@", clickedName)
		}
		if hoveredName != "" {
			cmd = strings.ReplaceAll(cmd, "@hovered", hoveredName)
		}
	}
	return cmd
}

func updateHotkeyRecording() {
	if !recording {
		return
	}
	if time.Since(recordStart) > 5*time.Second {
		finishRecording()
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		finishRecording()
		return
	}
	if c := detectCombo(); c != "" {
		recordedCombo = c
		finishRecording()
	}
}

func checkHotkeys() {
	if recording || inputActive {
		return
	}
	if combo := detectCombo(); combo != "" {
		hotkeysMu.RLock()
		list := append([]Hotkey(nil), hotkeys...)
		hotkeysMu.RUnlock()
		for _, hk := range list {
			if hk.Combo == combo && !hk.Disabled {
				for _, c := range hk.Commands {
					cmd := strings.TrimSpace(c.Command)
					if c.Function != "" {
						name := strings.ToLower(strings.TrimSpace(c.Function))
						pluginMu.RLock()
						fnMap := pluginFuncs[c.Plugin]
						disabled := pluginDisabled[c.Plugin]
						fn, ok := fnMap[name]
						pluginMu.RUnlock()
						if !disabled && ok && fn != nil {
							consoleMessage("> [plugin] " + name)
							go fn()
						} else {
							consoleMessage("[plugin] function not found: " + name)
						}
						continue
					}
					if strings.HasPrefix(strings.ToLower(cmd), "plugin:") {
						name := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(cmd, "plugin:")))
						pluginMu.RLock()
						var fn PluginFunc
						var found bool
						for owner, m := range pluginFuncs {
							if pluginDisabled[owner] {
								continue
							}
							if f, ok := m[name]; ok {
								fn = f
								found = true
								break
							}
						}
						pluginMu.RUnlock()
						if found && fn != nil {
							consoleMessage("> [plugin] " + name)
							go fn()
						} else {
							consoleMessage("[plugin] function not found: " + name)
						}
						continue
					}
					// Show hotkey-triggered command as if it were typed
					cmd = applyHotkeyVars(cmd)
					if cmd != "" {
						consoleMessage("> " + cmd)
					}
					enqueueCommand(cmd)
				}
				nextCommand()
				break
			}
		}
	}
}

type pluginFuncInfo struct {
	Name   string
	Plugin string
}

// pluginFunctionInfos returns a snapshot of registered plugin functions with their owning plugin.
func pluginFunctionInfos() []pluginFuncInfo {
	pluginMu.RLock()
	var infos []pluginFuncInfo
	for owner, funcs := range pluginFuncs {
		for name := range funcs {
			infos = append(infos, pluginFuncInfo{Name: name, Plugin: owner})
		}
	}
	pluginMu.RUnlock()
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].Name == infos[j].Name {
			return infos[i].Plugin < infos[j].Plugin
		}
		return infos[i].Name < infos[j].Name
	})
	return infos
}
