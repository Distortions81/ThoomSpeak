package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Hotkey struct {
	Key    string `json:"key"`
	Action string `json:"action"`
}

var globalHotkeys []Hotkey
var professionHotkeys map[string][]Hotkey
var characterHotkeys map[string][]Hotkey

func loadHotkeys() {
	dir := filepath.Join(dataDirPath, "hotkeys")
	// global
	if b, err := os.ReadFile(filepath.Join(dir, "global.json")); err == nil {
		var f struct {
			Version int      `json:"version"`
			Hotkeys []Hotkey `json:"hotkeys"`
		}
		if json.Unmarshal(b, &f) == nil {
			globalHotkeys = f.Hotkeys
		}
	}
	// professions
	if b, err := os.ReadFile(filepath.Join(dir, "professions.json")); err == nil {
		var f struct {
			Version int                 `json:"version"`
			Hotkeys map[string][]Hotkey `json:"hotkeys"`
		}
		if json.Unmarshal(b, &f) == nil {
			professionHotkeys = f.Hotkeys
		}
	} else {
		professionHotkeys = make(map[string][]Hotkey)
	}
	// characters
	if b, err := os.ReadFile(filepath.Join(dir, "characters.json")); err == nil {
		var f struct {
			Version int                 `json:"version"`
			Hotkeys map[string][]Hotkey `json:"hotkeys"`
		}
		if json.Unmarshal(b, &f) == nil {
			characterHotkeys = f.Hotkeys
		}
	} else {
		characterHotkeys = make(map[string][]Hotkey)
	}
}

func characterNames() []string {
	names := make([]string, len(characters))
	for i, c := range characters {
		names[i] = c.Name
	}
	return names
}

func init() {
	loadHotkeys()
}
