package main

import (
	"fmt"
	"sync"
	"time"
)

type timedMessage struct {
	Text string
	Time time.Time
}

type messageLog struct {
	mu      sync.Mutex
	entries []timedMessage
	max     int
}

func (l *messageLog) Add(msg string) {
	if msg == "" {
		return
	}
	l.mu.Lock()
	l.entries = append(l.entries, timedMessage{Text: msg, Time: time.Now()})
	if len(l.entries) > l.max {
		l.entries = l.entries[len(l.entries)-l.max:]
	}
	l.mu.Unlock()
}

func (l *messageLog) Entries(format string, useTimestamps bool) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.entries))
	if format == "" {
		format = "3:04PM"
	}
	for i, msg := range l.entries {
		if useTimestamps {
			out[i] = fmt.Sprintf("[%s] %s", msg.Time.Format(format), msg.Text)
		} else {
			out[i] = msg.Text
		}
	}
	return out
}
