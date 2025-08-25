//go:build plugin

package main

import "gt"

var PluginName = "Default Macros"

func Init() {
	gt.AddMacros(map[string]string{
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
	})
}
