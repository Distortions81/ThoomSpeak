//go:build plugin

package main

import (
	"strings"
	"time"

	"gt"
)

var PluginName = "Healer Self-Heal"

func Init() {
	go func() {
		for {
			if gt.MouseJustPressed("right") {
				c := gt.LastClick()
				if c.OnMobile && strings.EqualFold(c.Mobile.Name, gt.PlayerName()) {
					equipMoonstone()
					gt.RunCommand("/use 10")
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()
}

func equipMoonstone() {
	for _, it := range gt.Inventory() {
		if strings.EqualFold(it.Name, "moonstone") {
			if !it.Equipped {
				gt.Equip(it.ID)
			}
			return
		}
	}
}
