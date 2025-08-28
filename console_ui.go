//go:build !test

package main

import (
    "time"
    clipboard "golang.design/x/clipboard"
    "gothoom/eui"
)

var consoleWin *eui.WindowData
var messagesFlow *eui.ItemData
var inputFlow *eui.ItemData
var consoleHighlighted *eui.ItemData

func updateConsoleWindow() {
	if consoleWin == nil {
		return
	}
	inputMsg := "[Press Enter To Type]"
	if inputActive {
		inputMsg = string(inputText)
	}
	scrollit := messagesFlow.ScrollAtBottom()

	msgs := getConsoleMessages()
	updateTextWindow(consoleWin, messagesFlow, inputFlow, msgs, gs.ConsoleFontSize, inputMsg, nil)
	if messagesFlow != nil {
		// Scroll to bottom on new text; clamp occurs on Refresh.
		if scrollit {
			messagesFlow.Scroll.Y = 1e9
		}
		if consoleWin != nil {
			consoleWin.Refresh()
		}
	}
}

func makeConsoleWindow() {
	if consoleWin != nil {
		return
	}
	consoleWin, messagesFlow, inputFlow = newTextWindow("Console", eui.HZoneLeft, eui.VZoneBottom, true, updateConsoleWindow)
	consoleMessage("Starting...")
	updateConsoleWindow()
}

// handleConsoleInputContext shows a context menu when right-clicking the
// console input bar. Returns true if the click was on the input.
func handleConsoleInputContext(mx, my int) bool {
    if consoleWin == nil || inputFlow == nil || !consoleWin.IsOpen() {
        return false
    }
    pos := eui.Point{X: float32(mx), Y: float32(my)}
    // Identify the input area rect (flow or its first child text).
    r := inputFlow.DrawRect
    if len(inputFlow.Contents) > 0 {
        r = inputFlow.Contents[0].DrawRect
    }
    if !(pos.X >= r.X0 && pos.X <= r.X1 && pos.Y >= r.Y0 && pos.Y <= r.Y1) {
        return false
    }
    // Prepare clipboard preview for Paste action.
    var clip string
    if b := clipboard.Read(clipboard.FmtText); len(b) > 0 {
        clip = string(b)
    }
    preview := func(s string) string {
        rn := []rune(s)
        if len(rn) > 32 {
            rn = rn[:32]
        }
        return string(rn)
    }
    opts := []string{}
    actions := []func(){}
    headerCount := 0
    if clip == "" {
        // Disabled paste shown as header (non-interactive, grayed).
        opts = append(opts, "Paste (empty)")
        headerCount = 1
    } else {
        opts = append(opts, "Paste "+preview(clip)+"…")
        actions = append(actions, func() {
            // Paste into the input and ensure input mode is active
            // and the console updates immediately.
            cur := string(inputText)
            pluginSetInputText(cur + clip)
            updateConsoleWindow()
            if consoleWin != nil {
                consoleWin.Refresh()
            }
        })
    }
    // Copy current line
    opts = append(opts, "Copy Line")
    actions = append(actions, func() {
        cur := string(inputText)
        if cur != "" {
            clipboard.Write(clipboard.FmtText, []byte(cur))
            if gs.NotifyCopyText {
                showNotification("text copied")
            }
        }
    })
    // Clear current line
    opts = append(opts, "Clear Line")
    actions = append(actions, func() {
        // Clear the input and switch to input mode so the empty state is visible.
        pluginSetInputText("")
        updateConsoleWindow()
        if consoleWin != nil {
            consoleWin.Refresh()
        }
    })

    menu := eui.ShowContextMenu(opts, pos.X, pos.Y, func(i int) {
        idx := i
        if headerCount > 0 {
            idx = i - headerCount
        }
        if idx >= 0 && idx < len(actions) {
            actions[idx]()
        }
    })
    if menu != nil {
        menu.HeaderCount = headerCount
    }
    return true
}

// handleConsoleCopyRightClick copies the clicked console line to the clipboard,
// highlights it briefly, and optionally shows a notification. Returns true if a
// line was found under the cursor.
func handleConsoleCopyRightClick(mx, my int) bool {
    if consoleWin == nil || messagesFlow == nil || !consoleWin.IsOpen() {
        return false
    }
    pos := eui.Point{X: float32(mx), Y: float32(my)}
    for _, row := range messagesFlow.Contents {
        r := row.DrawRect
        if pos.X >= r.X0 && pos.X <= r.X1 && pos.Y >= r.Y0 && pos.Y <= r.Y1 {
            // Clear previous highlights in this list.
            for _, it := range messagesFlow.Contents {
                it.Filled = false
                it.Focused = false
            }
            // Highlight selected line briefly.
            row.Filled = true
            row.Focused = true
            consoleHighlighted = row
            consoleWin.Refresh()
            scheduleConsoleUnhighlight(row)
            // Copy its raw text.
            if row.Text != "" {
                clipboard.Write(clipboard.FmtText, []byte(row.Text))
                if gs.NotifyCopyText {
                    showNotification("text copied")
                }
            }
            return true
        }
    }
    return false
}

func scheduleConsoleUnhighlight(row *eui.ItemData) {
    go func(target *eui.ItemData) {
        time.Sleep(1200 * time.Millisecond)
        if consoleHighlighted == target {
            target.Filled = false
            target.Focused = false
            if consoleWin != nil {
                consoleWin.Refresh()
            }
            consoleHighlighted = nil
        }
    }(row)
}
