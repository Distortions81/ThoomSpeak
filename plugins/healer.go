package main

import "pluginapi"

func Init() {
	pluginapi.AddHotkey("Mouse 3", "/use")
}
