//go:build plugin

package main

import (
	"fmt"
	"gt"
	"strconv"
	"time"
)

var PluginName = "Coin Lord"

var (
	clRunning bool
	clTotal   int
	clStart   time.Time
)

func Init() {
	// Toggle counting with /cw.
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

	// Reset the totals with /cwnew.
	gt.RegisterCommand("cwnew", func(args string) {
		clStart = time.Now()
		clTotal = 0
		gt.Console("Coin data reset")
	})

	// Show current totals with /cwdata or Shift+C.
	gt.RegisterCommand("cwdata", func(args string) {
		hours := time.Since(clStart).Hours()
		rate := 0.0
		if hours > 0 {
			rate = float64(clTotal) / hours
		}
		gt.Console(fmt.Sprintf("Coins: %d (%.0f/hr)", clTotal, rate))
	})
	gt.RegisterTriggers([]string{"You get "}, clHandle)
	gt.AddHotkey("Shift-C", "/cwdata")
}

// clHandle watches chat for messages like "You get 3 coins" and tallies them.
func clHandle(msg string) {
	if !clRunning {
		return
	}
	if !gt.StartsWith(msg, "You get ") || !gt.Includes(msg, " coin") {
		return
	}
	fields := gt.Words(msg)
	if len(fields) < 3 {
		return
	}
	n, err := strconv.Atoi(fields[2])
	if err != nil {
		return
	}
	clTotal += n
}
