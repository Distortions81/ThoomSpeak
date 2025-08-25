//go:build plugin

package main

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gt"
)

// PluginName identifies this plugin in the UI.
var PluginName = "Dice Roller"

var diceRE = regexp.MustCompile(`(?i)^([0-9]*)d([0-9]+)$`)

// Init registers the /roll command.
func Init() {
	rand.Seed(time.Now().UnixNano())
	gt.RegisterCommand("roll", roll)
}

func roll(args string) {
	args = strings.TrimSpace(args)
	if args == "" {
		gt.Console("usage: /roll NdM, e.g. /roll 2d6")
		return
	}
	m := diceRE.FindStringSubmatch(args)
	if m == nil {
		gt.Console("usage: /roll NdM, e.g. /roll 2d6")
		return
	}
	n := 1
	if m[1] != "" {
		n, _ = strconv.Atoi(m[1])
	}
	sides, _ := strconv.Atoi(m[2])
	if n <= 0 || sides <= 0 {
		gt.Console("invalid dice")
		return
	}

	// try to equip a dice item if present
	for _, it := range gt.Inventory() {
		if strings.Contains(strings.ToLower(it.Name), "dice") {
			if !it.Equipped {
				gt.Equip(it.ID)
			}
			break
		}
	}

	rolls := make([]string, n)
	total := 0
	for i := 0; i < n; i++ {
		r := rand.Intn(sides) + 1
		rolls[i] = strconv.Itoa(r)
		total += r
	}
	gt.RunCommand(fmt.Sprintf("/me rolls %s: %s (total %d)", args, strings.Join(rolls, " "), total))
}
