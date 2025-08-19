package main

import (
	"fmt"
	"sync"
	"time"
)

const (
	maxChatMessages = 1000
)

type timedMessage struct {
	Text string
	Time time.Time
}

var (
	chatMsgMu sync.Mutex
	chatMsgs  []timedMessage
)

func chatMessage(msg string) {
	if msg == "" {
		return
	}
	chatMsgMu.Lock()
	chatMsgs = append(chatMsgs, timedMessage{Text: msg, Time: time.Now()})

	//Remove oldest message if full
	if len(chatMsgs) > maxChatMessages {
		chatMsgs = chatMsgs[len(chatMsgs)-maxChatMessages:]
	}

	chatMsgMu.Unlock()

	updateChatWindow()

	if gs.ChatTTS && !blockTTS {
		speakChatMessage(msg)
	}
}

func getChatMessages() []string {
	chatMsgMu.Lock()
	defer chatMsgMu.Unlock()

	out := make([]string, len(chatMsgs))
	for i, msg := range chatMsgs {
		if gs.ChatTimestamps {
			out[i] = fmt.Sprintf("[%s] %s", msg.Time.Format("15:04"), msg.Text)
		} else {
			out[i] = msg.Text
		}
	}
	return out
}
