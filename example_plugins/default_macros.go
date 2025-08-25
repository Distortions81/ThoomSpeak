//go:build plugin

package main

import (
	"gt"      // small API exposed to plugins
	"strings" // helpers for working with text
)

// PluginName shows in the plugin list. It must be unique so the client
// knows which plugin this is.
var PluginName = "Default Macros"

// macroMap links short text to longer slash commands. When you type the short
// version in the chat box the plugin swaps it for the full command.
var macroMap = map[string]string{
	// "??" becomes "/help "
	"??": "/help ",
	// "aa" becomes "/action "
	"aa": "/action ",
	"gg": "/give ",
	"ii": "/info ",
	"kk": "/karma ",
	"mm": "/money",
	"nn": "/news",
	"pp": "/ponder ",
	"sh": "/share ",
	"sl": "/sleep",
	// single letter macros work too
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

// Init runs once when the plugin is loaded by the client.
func Init() {
	// Register a small function that runs every time you hit enter in the
	// chat box. It can change what gets sent to the server.
	gt.RegisterInputHandler(func(txt string) string {
		// work with a lowercase copy so matching is easy
		lower := strings.ToLower(txt)
		// look through all known macros
		for short, full := range macroMap {
			// if the text starts with a macro
			if strings.HasPrefix(lower, short) {
				// replace the macro with the full command and
				// keep the rest of what the user typed
				return full + txt[len(short):]
			}
		}
		// no macro matched; return the original text untouched
		return txt
	})
}
