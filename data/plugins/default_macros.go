//go:build plugin

package main

import (
	"gt"
	"strings"
	"time"
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
	go func() {
		for {
			txt := gt.InputText()
			lower := strings.ToLower(txt)
			for k, v := range macroMap {
				if strings.HasPrefix(lower, k) {
					gt.SetInputText(v + txt[len(k):])
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()
}
