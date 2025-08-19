package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// Character holds a saved character name and password hash.
type Character struct {
	Name     string `json:"name"`
	PassHash string `json:"passHash"`
}

var characters []Character

const (
	charsFilePath    = "characters.json"
	charsFileVersion = 1
)

type charactersFile struct {
	Version    int         `json:"version"`
	Characters []Character `json:"characters"`
}

func loadCharacters() {
	data, err := os.ReadFile(filepath.Join(dataDirPath, charsFilePath))
	if err != nil {
		return
	}
	var cf charactersFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return
	}
	if cf.Version != charsFileVersion {
		return
	}
	characters = cf.Characters
}

func saveCharacters() {
	cf := charactersFile{Version: charsFileVersion, Characters: characters}
	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		log.Printf("save characters: %v", err)
		return
	}
	if err := os.WriteFile(filepath.Join(dataDirPath, charsFilePath), data, 0644); err != nil {
		log.Printf("save characters: %v", err)
	}
}

// removeCharacter deletes a stored character by name.
func removeCharacter(name string) {
	for i, c := range characters {
		if c.Name == name {
			characters = append(characters[:i], characters[i+1:]...)
			saveCharacters()
			if gs.LastCharacter == name {
				gs.LastCharacter = ""
				saveSettings()
			}
			return
		}
	}
}
