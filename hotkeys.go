package main

import (
    "encoding/json"
    "fmt"
    "math"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "time"
    "sync"

	"gothoom/eui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

const hotkeysFile = "global-hotkeys.json"

type HotkeyCommand struct {
	Command string `json:"command"`
}

type Hotkey struct {
	Name     string          `json:"name,omitempty"`
	Combo    string          `json:"combo"`
	Commands []HotkeyCommand `json:"commands"`
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
	}
	var raw []hotkeyJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}
    var newList []Hotkey
    for _, r := range raw {
        hk := Hotkey{Combo: r.Combo, Name: r.Name}
        if len(r.Commands) > 0 {
            hk.Commands = r.Commands
        } else if r.Command != "" {
            cmd := strings.TrimSpace(r.Command + " " + r.Text)
            hk.Commands = []HotkeyCommand{{Command: cmd}}
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
    // snapshot to avoid concurrent mutation during UI build
    hotkeysMu.RLock()
    list := append([]Hotkey(nil), hotkeys...)
    hotkeysMu.RUnlock()
    for i, hk := range list {
        idx := i
        row := &eui.ItemData{ItemType: eui.ITEM_FLOW, FlowType: eui.FLOW_HORIZONTAL, Fixed: true}
        row.Size = eui.Point{X: 220, Y: 20}
		btn, events := eui.NewButton()
		text := ""
		if len(hk.Commands) > 0 {
			text = hk.Commands[0].Command
			if len(hk.Commands) > 1 {
				text += " ..."
			}
		}
		btn.Text = hk.Combo
		if hk.Name != "" {
			btn.Text += " [" + hk.Name + "]"
		}
		btn.Text += " -> " + text
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
    hotkeysMu.RLock()
    if idx < 0 || idx >= len(hotkeys) {
        hotkeysMu.RUnlock()
        return
    }
    hk := hotkeys[idx]
    hotkeysMu.RUnlock()
    showPopup(
        "Remove Hotkey",
        fmt.Sprintf("Remove hotkey %s?", hk.Combo),
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
    addHotkeyCommand("")

    // Row to add a plain command input
    addCmdRow, addCmdEvents := eui.NewButton()
    addCmdRow.Text = "+"
    addCmdRow.Size = eui.Point{X: 20, Y: 20}
    addCmdRow.FontSize = 14
	addCmdEvents.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			addHotkeyCommand("")
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
    fnDD.Options = pluginFunctionNames()
    if len(fnDD.Options) == 0 {
        fnDD.Options = []string{"(no plugin functions)"}
        fnDD.Disabled = true
    }
    fnDDEvents.Handle = func(ev eui.UIEvent) {
        if ev.Type == eui.EventDropdownSelected {
            // no-op; selection used when Add clicked
        }
    }
    fnRow.AddItem(fnDD)

    addFnBtn, addFnEv := eui.NewButton()
    addFnBtn.Text = "+"
    addFnBtn.Size = eui.Point{X: 20, Y: 20}
    addFnBtn.FontSize = 14
    addFnBtn.Disabled = fnDD.Disabled
    addFnEv.Handle = func(ev eui.UIEvent) {
        if ev.Type == eui.EventClick {
            opts := pluginFunctionNames()
            if len(opts) == 0 {
                return
            }
            sel := fnDD.Selected
            if sel < 0 || sel >= len(opts) {
                sel = 0
            }
            addHotkeyCommand("plugin:" + opts[sel])
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
            hotkeyCmdInputs[0].Text = hk.Commands[0].Command
            for _, c := range hk.Commands[1:] {
                addHotkeyCommand(c.Command)
            }
        }
    } else {
        hotkeysMu.RUnlock()
    }

	hotkeyEditWin.AddWindow(true)
	hotkeyEditWin.MarkOpen()
	wrapHotkeyInputs()
}

func addHotkeyCommand(cmd string) {
	if hotkeyCmdSection == nil {
		return
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

	hotkeyEditWin.Refresh()
	wrapHotkeyInputs()
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
	var goFace *text.GoTextFace
	if src := eui.FontSource(); src != nil {
		goFace = &text.GoTextFace{Source: src, Size: facePx}
	} else {
		goFace = &text.GoTextFace{Size: facePx}
	}
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
		for i := range hotkeyCmdInputs {
			cmd := strings.ReplaceAll(hotkeyCmdInputs[i].Text, "\n", " ")
			if cmd != "" {
				cmds = append(cmds, HotkeyCommand{Command: cmd})
			}
		}
        if combo != "" && len(cmds) > 0 {
            hk := Hotkey{Name: name, Combo: combo, Commands: cmds}
            hotkeysMu.Lock()
            if editingHotkey >= 0 && editingHotkey < len(hotkeys) {
                hotkeys[editingHotkey] = hk
            } else {
                hotkeys = append(hotkeys, hk)
            }
            hotkeysMu.Unlock()
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
        hotkeysMu.RLock()
        list := append([]Hotkey(nil), hotkeys...)
        hotkeysMu.RUnlock()
        for _, hk := range list {
            if hk.Combo == combo {
                for _, c := range hk.Commands {
                    cmd := strings.TrimSpace(c.Command)
                    if strings.HasPrefix(strings.ToLower(cmd), "plugin:") {
                        name := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(cmd, "plugin:")))
                        pluginMu.RLock()
                        fn, ok := pluginFuncs[name]
                        pluginMu.RUnlock()
                        if ok && fn != nil {
                            // Surface plugin invocation as console input for visibility
                            consoleMessage("> [plugin] " + name)
                            go fn()
                        } else {
                            consoleMessage("[plugin] function not found: " + name)
                        }
                        continue
                    }
                    // Show hotkey-triggered command as if it were typed
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

// pluginFunctionNames returns a sorted snapshot of registered plugin function names.
func pluginFunctionNames() []string {
    pluginMu.RLock()
    names := make([]string, 0, len(pluginFuncs))
    for k := range pluginFuncs {
        names = append(names, k)
    }
    pluginMu.RUnlock()
    sort.Strings(names)
    return names
}
