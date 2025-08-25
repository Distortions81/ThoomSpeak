//go:build !test

package main

import "gothoom/eui"

var chatWin *eui.WindowData
var chatList *eui.ItemData

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
