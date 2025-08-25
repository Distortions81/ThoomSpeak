package main

import (
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

var pluginExports = interp.Exports{
	"pluginapi": {
		"Logf":          reflect.ValueOf(pluginLogf),
		"AddHotkey":     reflect.ValueOf(pluginAddHotkey),
		"ClientVersion": reflect.ValueOf(&clientVersion).Elem(),
	},
}

func pluginLogf(format string, args ...interface{}) {
	log.Printf("[plugin] "+format, args...)
}

func pluginAddHotkey(combo, command string) {
	hk := Hotkey{Combo: combo, Commands: []HotkeyCommand{{Command: command}}}
	hotkeys = append(hotkeys, hk)
	refreshHotkeysList()
	saveHotkeys()
}

func loadPlugins() {
	dir := "plugins"
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("read plugin dir: %v", err)
		}
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			log.Printf("read plugin %s: %v", path, err)
			continue
		}
		i := interp.New(interp.Options{})
		i.Use(stdlib.Symbols)
		i.Use(pluginExports)
		if _, err := i.Eval(string(src)); err != nil {
			log.Printf("plugin %s: %v", e.Name(), err)
			continue
		}
		if v, err := i.Eval("Init"); err == nil {
			if fn, ok := v.Interface().(func()); ok {
				fn()
			}
		}
		log.Printf("loaded plugin %s", e.Name())
	}
}
