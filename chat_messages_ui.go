//go:build !test

package main

import "gothoom/eui"

var chatWin *eui.WindowData
var chatList *eui.ItemData

func updateChatWindow() {
	if chatWin == nil {
		return
	}

	msgs := getChatMessages()
	updateTextWindow(chatWin, chatList, nil, msgs, gs.ChatFontSize, "")
	if chatList != nil {
		// Auto-scroll list to bottom on new messages
		chatList.Scroll.Y = 1e9
		chatWin.Refresh()
	}
}

func makeChatWindow() error {
	if chatWin != nil {
		return nil
	}
	chatWin, chatList, _ = makeTextWindow("Chat", eui.HZoneRight, eui.VZoneBottom, false)
	// Rewrap and refresh on window resize
	chatWin.OnResize = func() {
		updateChatWindow()
		if chatWin != nil {
			chatWin.Refresh()
		}
	}
	updateChatWindow()
	return nil
}
