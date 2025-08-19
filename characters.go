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
	charsFilePath = "characters.json"
)

func loadCharacters() {
	data, err := os.ReadFile(filepath.Join(dataDirPath, charsFilePath))
	if err == nil {
		_ = json.Unmarshal(data, &characters)
	}
}

func saveCharacters() {
	data, err := json.MarshalIndent(characters, "", "  ")
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
