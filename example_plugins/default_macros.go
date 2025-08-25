//go:build plugin

package main

import (
	"gt"
	"strings"
)

var PluginName = "Default Macros"

var macroMap = map[string]string{
	"??": "/help ",
	"aa": "/action ",
	"gg": "/give ",
	"ii": "/info ",
	"kk": "/karma ",
	"mm": "/money",
	"nn": "/news",
	"pp": "/ponder ",
	"sh": "/share ",
	"sl": "/sleep",
	"t":  "/think ",
	"tt": "/thinkto ",
	"th": "/thank ",
	"ui": "/useitem ",
	"uu": "/use ",
	"un": "/unshare ",
	"w":  "/who ",
	"wh": "/whisper ",
	"yy": "/yell ",
}

func Init() {
	gt.RegisterInputHandler(func(txt string) string {
		lower := strings.ToLower(txt)
		for k, v := range macroMap {
			if strings.HasPrefix(lower, k) {
				return v + txt[len(k):]
			}
		}
		return txt
	})
}
