package main

import (
	"fmt"
	"sync"
	"time"
)

const (
	maxMessages = 1000
)

var (
	messageMu sync.Mutex
	messages  []timedMessage
)

func consoleMessage(msg string) {
	if msg == "" {
		return
	}

	messageMu.Lock()
	messages = append(messages, timedMessage{Text: msg, Time: time.Now()})

	//Remove oldest message if full
	if len(messages) > maxMessages {
		messages = messages[len(messages)-maxMessages:]
	}
	messageMu.Unlock()

	updateConsoleWindow()
}

func getConsoleMessages() []string {
	messageMu.Lock()
	defer messageMu.Unlock()

	out := make([]string, len(messages))
	for i, msg := range messages {
		if gs.ConsoleTimestamps {
			out[i] = fmt.Sprintf("[%s] %s", msg.Time.Format("15:04"), msg.Text)
		} else {
			out[i] = msg.Text
		}
	}
	return out
}
