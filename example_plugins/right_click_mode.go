//go:build plugin

package main

import (
	"strings"
	"time"

	"gt"
)

var PluginName = "Right Click Mode"

var (
	rightClick = "pull"
	target     string
	staffName  = "caduceus"
)

func Init() {
	gt.RegisterCommand("trade", func(string) { toggleTrade() })
	gt.RegisterCommand("pushpull", func(string) { togglePushPull() })
	gt.RegisterCommand("healpotion", func(string) { toggleHealPotion() })
	gt.RegisterCommand("cadset", func(string) { toggleCadSet() })
	go watchRightClicks()
}

func watchRightClicks() {
	for {
		if gt.MouseJustPressed("right") {
			handleRightClick()
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func rightClickState() {
	msg := "Right-Click: " + rightClick
	if target != "" {
		msg += " " + target
	}
	gt.Console(msg)
}

func toggleTrade() {
	if rightClick != "sell" {
		rightClick = "sell"
	} else {
		rightClick = "buy"
	}
	rightClickState()
}

func togglePushPull() {
	if rightClick != "pull" {
		rightClick = "pull"
	} else {
		rightClick = "push"
	}
	rightClickState()
}

func toggleHealPotion() {
	if rightClick != "rhs" {
		rightClick = "rhs"
	} else {
		rightClick = "rhp"
	}
	rightClickState()
}

func toggleCadSet() {
	if rightClick != "cad" {
		rightClick = "cad"
	} else {
		rightClick = "set"
	}
	rightClickState()
}

func handleRightClick() {
	c := gt.LastClick()
	if !c.OnMobile {
		return
	}
	name := c.Mobile.Name
	if isHealer() {
		healer(name)
	} else {
		noClass(name)
	}
}

func noClass(name string) {
	switch rightClick {
	case "pull":
		gt.RunCommand("/pull " + name)
	case "push":
		gt.RunCommand("/push " + name)
	case "sell":
		gt.RunCommand("/sell 0 " + name)
	case "buy":
		gt.RunCommand("/buy 0 " + name)
	}
}

func healer(name string) {
	switch rightClick {
	case "pull":
		gt.RunCommand("/pull " + name)
	case "cad":
		ensureEquipped(staffName)
		gt.RunCommand("/use /lock " + name)
	case "push":
		gt.RunCommand("/push " + name)
	case "rhs":
		ensureEquipped("redhealingsalve")
		gt.RunCommand("/useitem redhealingsalve " + name)
		rightClick = "pull"
		rightClickState()
	case "rhp":
		ensureEquipped("redhealingpotion")
		gt.RunCommand("/useitem redhealingpotion " + name)
		rightClick = "pull"
		rightClickState()
	case "set":
		if strings.EqualFold(name, gt.PlayerName()) {
			target = "/pet"
		} else {
			target = name
		}
		rightClickState()
	case "sell":
		gt.RunCommand("/sell 0 " + name)
	case "buy":
		gt.RunCommand("/buy 0 " + name)
	}
}

func ensureEquipped(name string) {
	for _, it := range gt.Inventory() {
		if strings.EqualFold(it.Name, name) {
			if !it.Equipped {
				gt.Equip(it.ID)
			}
			return
		}
	}
}

func isHealer() bool {
	me := gt.PlayerName()
	for _, p := range gt.Players() {
		if strings.EqualFold(p.Name, me) {
			return strings.EqualFold(p.Class, "healer")
		}
	}
	return false
}
