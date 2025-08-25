//go:build plugin

package main

import "gt"

// PluginName is shown in the client's list of plugins.
var PluginName = "Default Macros"

// Init registers a bunch of handy shortcuts for common commands.
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
