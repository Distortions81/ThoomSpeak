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

	if name := chatSpeaker(msg); name != "" {
		playersMu.RLock()
		p, ok := players[name]
		blocked := ok && (p.Blocked || p.Ignored)
		playersMu.RUnlock()
		if blocked {
			return
		}
	}

	chatLog.Add(msg)
	appendChatLog(msg)

	updateChatWindow()

	if gs.ChatTTS && !blockTTS && !isSelfChatMessage(msg) {
		speakChatMessage(msg)
	} else if !gs.ChatTTS {
		chatTTSDisabledOnce.Do(func() {
			consoleMessage("Chat TTS is disabled. Enable it in settings to hear messages.")
		})
	}

	runTriggers(msg)
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

// chatSpeaker extracts the leading player name from a chat message, folded to
// canonical form. It returns an empty string if no name could be parsed.
func chatSpeaker(msg string) string {
	m := strings.TrimSpace(msg)
	if strings.HasPrefix(m, "(") {
		m = m[1:]
	}
	if i := strings.IndexByte(m, ' '); i > 0 {
		return utfFold(m[:i])
	}
	return ""
}
