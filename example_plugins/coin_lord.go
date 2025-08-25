//go:build plugin

package main

import (
	"fmt"
	"gt"
	"strconv"
	"strings"
	"time"
)

var PluginName = "coin_lord"

var (
	clRunning bool
	clTotal   int
	clStart   time.Time
)

func Init() {
	gt.RegisterCommand("cw", func(args string) {
		clRunning = !clRunning
		if clRunning {
			clStart = time.Now()
			clTotal = 0
			gt.Console("Coin Lord started")
		} else {
			gt.Console("Coin Lord stopped")
		}
	})
	gt.RegisterCommand("cwnew", func(args string) {
		clStart = time.Now()
		clTotal = 0
		gt.Console("Coin data reset")
	})
	gt.RegisterCommand("cwdata", func(args string) {
		hours := time.Since(clStart).Hours()
		rate := 0.0
		if hours > 0 {
			rate = float64(clTotal) / hours
		}
		gt.Console(fmt.Sprintf("Coins: %d (%.0f/hr)", clTotal, rate))
	})
	gt.RegisterChatHandler(clHandle)
	gt.AddHotkey("Shift-C", "/cwdata")
}

func clHandle(msg string) {
	if !clRunning {
		return
	}
	if !strings.HasPrefix(msg, "You get ") || !strings.Contains(msg, " coin") {
		return
	}
	fields := strings.Fields(msg)
	if len(fields) < 3 {
		return
	}
	n, err := strconv.Atoi(fields[2])
	if err != nil {
		return
	}
	clTotal += n
}
