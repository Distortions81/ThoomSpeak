package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"

	keyring "github.com/99designs/keyring"
)

// Character holds a saved character name and password hash.
type Character struct {
	Name     string `json:"name"`
	PassHash string `json:"passHash"`
}

var characters []Character

const (
	charsFilePath  = "characters.json"
	keyringService = "gothoom"
)

// loadCharacters populates the in-memory list of characters. It attempts to
// read from the OS keyring first and falls back to characters.json only if the
// keyring is unavailable or returns an error.
func loadCharacters() {
	if ring != nil {
		keys, err := ring.Keys()
		if err == nil {
			for _, name := range keys {
				item, err := ring.Get(name)
				if err != nil {
					logError("keyring get %q: %v", name, err)
					characters = append(characters, Character{Name: name})
					continue
				}
				characters = append(characters, Character{Name: name, PassHash: string(item.Data)})
			}
			return
		}
		logError("keyring keys: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dataDirPath, charsFilePath))
	if err == nil {
		_ = json.Unmarshal(data, &characters)
	}
}

// saveCharacters writes the current character list to characters.json.
// Password hashes are stored here only as a fallback; the primary
// storage is the OS keyring.
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
			useFile := ring == nil
			if ring != nil {
				if err := ring.Remove(name); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
					logError("keyring remove %q: %v", name, err)
					useFile = true
				} else {
					useFile = false
				}
			}
			if useFile {
				saveCharacters()
			}
			if gs.LastCharacter == name {
				gs.LastCharacter = ""
				saveSettings()
			}
			return
		}
	}
}
