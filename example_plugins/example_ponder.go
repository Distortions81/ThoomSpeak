package main

import (
	"gt"
)

// Highly recommend vscodium or vscode with the official golang plugin!
func Wave() {
	gt.RunCommand("/ponder hello world")
}

func Init() {
	//This makes the function available in the hotkey editor
	gt.RegisterFunc("wave", Wave)

	//This registers a slash command '/example'
	gt.RegisterCommand("example", func(args string) {
		if args == "" {
			args = "hello world"
		}
		//This enters this command and sends it
		gt.RunCommand("/ponder " + args)
	})
}
