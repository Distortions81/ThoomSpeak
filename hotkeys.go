package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gothoom/eui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

const hotkeysFile = "global-hotkeys.json"

type HotkeyCommand struct {
	Command string `json:"command"`
	Text    string `json:"text,omitempty"`
}

type Hotkey struct {
	Combo    string          `json:"combo"`
	Commands []HotkeyCommand `json:"commands"`
}

var (
	hotkeys          []Hotkey
	hotkeysWin       *eui.WindowData
	hotkeysList      *eui.ItemData
	hotkeyEditWin    *eui.WindowData
	hotkeyComboText  *eui.ItemData
	hotkeyCmdSection *eui.ItemData
	hotkeyCmdInputs  []*eui.ItemData
	hotkeyTextInputs []*eui.ItemData
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
		Commands []HotkeyCommand `json:"commands"`
		Command  string          `json:"command"`
		Text     string          `json:"text,omitempty"`
	}
	var raw []hotkeyJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}
	hotkeys = hotkeys[:0]
	for _, r := range raw {
		hk := Hotkey{Combo: r.Combo}
		if len(r.Commands) > 0 {
			hk.Commands = r.Commands
		} else if r.Command != "" {
			hk.Commands = []HotkeyCommand{{Command: r.Command, Text: r.Text}}
		}
		hotkeys = append(hotkeys, hk)
	}
	refreshHotkeysList()
}

func saveHotkeys() {
	path := filepath.Join(dataDirPath, hotkeysFile)
	_ = os.MkdirAll(dataDirPath, 0o755)
	data, err := json.MarshalIndent(hotkeys, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
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
	for i, hk := range hotkeys {
		idx := i
		row := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true}
		row.Size = eui.Point{X: 220, Y: 20}
		btn, events := eui.NewButton()
		text := ""
		if len(hk.Commands) > 0 {
			text = hk.Commands[0].Command
			if hk.Commands[0].Text != "" {
				text += " " + hk.Commands[0].Text
			}
			if len(hk.Commands) > 1 {
				text += " ..."
			}
		}
		btn.Text = hk.Combo + " -> " + text
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
	hotkeysList.Dirty = true
	if hotkeysWin != nil {
		hotkeysWin.Refresh()
	}
}

func confirmRemoveHotkey(idx int) {
	if idx < 0 || idx >= len(hotkeys) {
		return
	}
	hk := hotkeys[idx]
	showPopup(
		"Remove Hotkey",
		fmt.Sprintf("Remove hotkey %s?", hk.Combo),
		[]popupButton{
			{Text: "Cancel"},
			{Text: "Remove", Color: &eui.ColorDarkRed, HoverColor: &eui.ColorRed, Action: func() {
				hotkeys = append(hotkeys[:idx], hotkeys[idx+1:]...)
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
	hotkeyEditWin.Size = eui.Point{X: 260, Y: 160}
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
	hotkeyComboText.Size = eui.Point{X: 120, Y: 20}
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

	hotkeyCmdSection = &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_VERTICAL, Fixed: true}
	flow.AddItem(hotkeyCmdSection)
	hotkeyCmdInputs = nil
	hotkeyTextInputs = nil
	addHotkeyCommand("", "")

	addCmdRow, addCmdEvents := eui.NewButton()
	addCmdRow.Text = "+"
	addCmdRow.Size = eui.Point{X: 20, Y: 20}
	addCmdRow.FontSize = 14
	addCmdEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			addHotkeyCommand("", "")
		}
	}
	flow.AddItem(addCmdRow)

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

	if idx >= 0 && idx < len(hotkeys) {
		hotkeyComboText.Text = hotkeys[idx].Combo
		if len(hotkeys[idx].Commands) > 0 {
			hotkeyCmdInputs[0].Text = hotkeys[idx].Commands[0].Command
			hotkeyTextInputs[0].Text = hotkeys[idx].Commands[0].Text
			for _, c := range hotkeys[idx].Commands[1:] {
				addHotkeyCommand(c.Command, c.Text)
			}
		}
	}

	hotkeyEditWin.AddWindow(true)
	hotkeyEditWin.MarkOpen()
	wrapHotkeyInputs()
}

func addHotkeyCommand(cmd, txt string) {
	if hotkeyCmdSection == nil {
		return
	}
	cmdLabel, _ := eui.NewText()
	cmdLabel.Text = "Command:"
	cmdLabel.Size = eui.Point{X: 220, Y: 20}
	cmdLabel.FontSize = 12
	hotkeyCmdSection.AddItem(cmdLabel)

	var cmdEvents *eui.EventHandler
	cmdInput, cmdEvents := eui.NewInput()
	cmdInput.Size = eui.Point{X: 220, Y: 20}
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

	textLabel, _ := eui.NewText()
	textLabel.Text = "Text:"
	textLabel.Size = eui.Point{X: 220, Y: 20}
	textLabel.FontSize = 12
	hotkeyCmdSection.AddItem(textLabel)

	var textEvents *eui.EventHandler
	textInput, textEvents := eui.NewInput()
	textInput.Size = eui.Point{X: 220, Y: 20}
	textInput.FontSize = 12
	textInput.Scrollable = true
	textInput.Text = txt
	textEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventInputChanged {
			wrapHotkeyInputs()
		}
	}
	hotkeyCmdSection.AddItem(textInput)
	hotkeyTextInputs = append(hotkeyTextInputs, textInput)

	hotkeyEditWin.Refresh()
	wrapHotkeyInputs()
}

func wrapHotkeyInputs() {
	if hotkeyEditWin == nil {
		return
	}
	ui := eui.UIScale()
	fs := 12
	if len(hotkeyCmdInputs) > 0 {
		fs = hotkeyCmdInputs[0].FontSize
	}
	facePx := float64(fs * ui)
	var goFace *text.GoTextFace
	if src := eui.FontSource(); src != nil {
		goFace = &text.GoTextFace{Source: src, Size: facePx}
	} else {
		goFace = &text.GoTextFace{Size: facePx}
	}
	metrics := goFace.Metrics()
	linePx := math.Ceil(metrics.HAscent + metrics.HDescent + 2)
	rowUnits := float32(linePx) / ui
	padPx := float64(6 * ui)

	resize := func(it *eui.ItemData) {
		if it == nil {
			return
		}
		raw := strings.ReplaceAll(it.Text, "\n", " ")
		_, lines := wrapText(raw, goFace, float64(it.Size.X*ui)-padPx)
		if len(lines) == 0 {
			lines = []string{""}
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
	for _, it := range hotkeyTextInputs {
		resize(it)
	}
	hotkeyEditWin.Refresh()
}

func finishHotkeyEdit(save bool) {
	if save {
		combo := strings.ReplaceAll(hotkeyComboText.Text, "\n", " ")
		cmds := []HotkeyCommand{}
		for i := range hotkeyCmdInputs {
			cmd := strings.ReplaceAll(hotkeyCmdInputs[i].Text, "\n", " ")
			txt := ""
			if i < len(hotkeyTextInputs) {
				txt = strings.ReplaceAll(hotkeyTextInputs[i].Text, "\n", " ")
			}
			if cmd != "" {
				cmds = append(cmds, HotkeyCommand{Command: cmd, Text: txt})
			}
		}
		if combo != "" && len(cmds) > 0 {
			hk := Hotkey{Combo: combo, Commands: cmds}
			if editingHotkey >= 0 && editingHotkey < len(hotkeys) {
				hotkeys[editingHotkey] = hk
			} else {
				hotkeys = append(hotkeys, hk)
			}
			saveHotkeys()
			refreshHotkeysList()
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
		for _, hk := range hotkeys {
			if hk.Combo == combo {
				for _, c := range hk.Commands {
					enqueueCommand(strings.TrimSpace(c.Command + " " + c.Text))
				}
				nextCommand()
				break
			}
		}
	}
}
