//go:build !test

package main

import (
	clipboard "golang.design/x/clipboard"
	"gothoom/eui"
	"time"
)

var chatWin *eui.WindowData
var chatList *eui.ItemData
var chatHighlighted *eui.ItemData

func updateChatWindow() {
	if chatWin == nil || !chatWin.IsOpen() {
		return
	}

	scrollit := chatList.ScrollAtBottom()

	msgs := getChatMessages()
	updateTextWindow(chatWin, chatList, nil, msgs, gs.ChatFontSize, "", nil)
	if chatList != nil {
		// Auto-scroll list to bottom on new messages
		if scrollit {
			chatList.Scroll.Y = 1e9
		}
		chatWin.Refresh()
	}
}

func makeChatWindow() error {
	if gs.MessagesToConsole {
		return nil
	}
	if chatWin != nil {
		return nil
	}
	chatWin, chatList, _ = newTextWindow("Chat", eui.HZoneRight, eui.VZoneBottom, false, updateChatWindow)
	updateChatWindow()
	chatWin.Refresh()
	return nil
}

// handleChatCopyRightClick copies the clicked chat line to the clipboard,
// highlights it, and optionally shows a notification. Returns true if a line
// was found under the cursor.
func handleChatCopyRightClick(mx, my int) bool {
	if chatWin == nil || chatList == nil || !chatWin.IsOpen() {
		return false
	}
	pos := eui.Point{X: float32(mx), Y: float32(my)}
	for _, row := range chatList.Contents {
		r := row.DrawRect
		if pos.X >= r.X0 && pos.X <= r.X1 && pos.Y >= r.Y0 && pos.Y <= r.Y1 {
			// Clear previous highlights in chat list.
			for _, it := range chatList.Contents {
				it.Filled = false
				it.Focused = false
			}
			// Highlight selected line briefly and copy the text.
			row.Filled = true
			row.Focused = true
			chatHighlighted = row
			chatWin.Refresh()
			scheduleChatUnhighlight(row)
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

func scheduleChatUnhighlight(row *eui.ItemData) {
	go func(target *eui.ItemData) {
		time.Sleep(1200 * time.Millisecond)
		if chatHighlighted == target {
			target.Filled = false
			target.Focused = false
			if chatWin != nil {
				chatWin.Refresh()
			}
			chatHighlighted = nil
		}
	}(row)
}
