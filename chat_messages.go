package main

import (
	"strings"
	"sync"
)

const (
	maxChatMessages = 1000
)

var (
	chatLog             = messageLog{max: maxChatMessages}
	chatTTSDisabledOnce sync.Once
)

func chatMessage(msg string) {
	if msg == "" {
		return
	}

	chatLog.Add(msg)

	updateChatWindow()

	if gs.ChatTTS && !blockTTS && !isSelfChatMessage(msg) {
		speakChatMessage(msg)
	} else if !gs.ChatTTS {
		chatTTSDisabledOnce.Do(func() {
			consoleMessage("Chat TTS is disabled. Enable it in settings to hear messages.")
		})
	}

	chatHandlersMu.RLock()
	handlers := append([]func(string){}, chatHandlers...)
	chatHandlersMu.RUnlock()
	for _, h := range handlers {
		go h(msg)
	}
}

func getChatMessages() []string {
	format := gs.TimestampFormat
	if format == "" {
		format = "3:04PM"
	}
	return chatLog.Entries(format, gs.ChatTimestamps)
}

func isSelfChatMessage(msg string) bool {
	if playerName == "" {
		return false
	}
	m := strings.ToLower(strings.TrimSpace(msg))
	name := strings.ToLower(playerName)

	if strings.HasPrefix(m, "("+name+" ") {
		return true
	}
	if strings.HasPrefix(m, name+" ") {
		return true
	}
	return false
}
