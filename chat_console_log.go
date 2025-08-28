package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	textLogPath string
	textLogOnce sync.Once
)

// appendChatLog appends a chat line to the legacy-style Text Logs file.
func appendChatLog(msg string) { appendTextLog(msg) }

// appendConsoleLog appends a console line to the legacy-style Text Logs file.
func appendConsoleLog(msg string) { appendTextLog(msg) }

func appendTextLog(msg string) {
	if msg == "" {
		return
	}
	// Best-effort; never panic here.
	defer func() { _ = recover() }()

	ensureTextLog()
	if textLogPath == "" {
		return
	}

	// Old client timestamp format: M/D/YY H:MM:SSa (no leading zeros for M/D/H)
	now := time.Now()
	hour := now.Hour()
	ampm := byte('a')
	if hour >= 12 {
		ampm = 'p'
	}
	hour12 := hour % 12
	if hour12 == 0 {
		hour12 = 12
	}
	ts := fmt.Sprintf("%d/%d/%.2d %d:%.2d:%.2d%c ",
		int(now.Month()), now.Day(), now.Year()%100,
		hour12, now.Minute(), now.Second(), ampm,
	)

	// Convert any CR to LF similar to SwapLineEndings before writing.
	line := strings.ReplaceAll(msg, "\r", "\n")
	line = strings.TrimRight(line, "\n")
	// One entry per line
	out := ts + line + "\n"

	// Append
	f, err := os.OpenFile(textLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	_, _ = f.WriteString(out)
	_ = f.Close()
}

// ensureTextLog initializes the legacy Text Log path matching old_mac_client.
// Path: "Text Logs/<CharName>/CL Log YYYY/MM/DD HH.MM.SS.txt"
func ensureTextLog() {
	textLogOnce.Do(func() {
		// Base folder
		base := filepath.Join("Text Logs")

		// Character subfolder (fallback to "Unknown").
		name := playerName
		if strings.TrimSpace(name) == "" {
			name = "Unknown"
		}
		charDir := filepath.Join(base, name)

		// Old filename template: "CL Log %.4d/%.2d/%.2d %.2d.%.2d.%.2d.txt"
		// which effectively creates nested year/month directories.
		now := time.Now()
		year := fmt.Sprintf("%04d", now.Year())
		month := fmt.Sprintf("%02d", int(now.Month()))
		day := fmt.Sprintf("%02d", now.Day())
		timeName := fmt.Sprintf("%s %02d.%02d.%02d.txt", day, now.Hour(), now.Minute(), now.Second())
		yearMonthDir := filepath.Join(charDir, "CL Log "+year, month)

		// Make directories.
		if err := os.MkdirAll(yearMonthDir, 0o755); err != nil {
			return
		}
		textLogPath = filepath.Join(yearMonthDir, timeName)
	})
}
