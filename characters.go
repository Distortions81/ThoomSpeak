package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"

	keyring "github.com/tmc/keyring"
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

// loadCharacters reads the characters.json file if it exists and
// populates the in-memory list, preferring password hashes from the OS
// keyring when available.
func loadCharacters() {

	data, err := os.ReadFile(filepath.Join(dataDirPath, charsFilePath))
	if err == nil {
		_ = json.Unmarshal(data, &characters)
	}

	for i := range characters {
		if pw, err := keyring.Get(keyringService, characters[i].Name); err == nil && pw != "" {
			characters[i].PassHash = pw
		} else if errors.Is(err, keyring.ErrNotFound) && characters[i].PassHash != "" {
			_ = keyring.Set(keyringService, characters[i].Name, characters[i].PassHash)
		}
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

// rememberCharacter stores the given name and password hash.
func rememberCharacter(name, pass string) {
	h := md5.Sum([]byte(pass))
	hash := hex.EncodeToString(h[:])
	for i := range characters {
		if characters[i].Name == name {
			characters[i].PassHash = hash
			_ = keyring.Set(keyringService, name, hash)
			saveCharacters()
			gs.LastCharacter = name
			saveSettings()
			return
		}
	}
	characters = append(characters, Character{Name: name, PassHash: hash})
	_ = keyring.Set(keyringService, name, hash)
	saveCharacters()
	gs.LastCharacter = name
	saveSettings()
}

// removeCharacter deletes a stored character by name.
func removeCharacter(name string) {
	for i, c := range characters {
		if c.Name == name {
			characters = append(characters[:i], characters[i+1:]...)
			_ = keyring.Set(keyringService, name, "")
			saveCharacters()
			if gs.LastCharacter == name {
				gs.LastCharacter = ""
				saveSettings()
			}
			return
		}
	}
}
