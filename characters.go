package main

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// Character holds a saved character name and password hash. The hash is stored
// on disk using a reversible scrambling to avoid exposing the raw hash.
type Character struct {
	Name     string `json:"name"`
	PassHash string `json:"passHash"`
}

var characters []Character

const (
	charsFilePath = "characters.json"
	hashKey       = "3k6XsAgldtz1vRw3e9WpfUtXQdKQO4P7a7dxmda4KTNpEJWu0lk08QEcJTbeqisH"
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

	if err := json.Unmarshal(data, &characters); err != nil {
		return
	}
	for i := range characters {
		characters[i].PassHash = unscrambleHash(characters[i].Name, characters[i].PassHash)
	}
	characters = cf.Characters
}

func saveCharacters() {
	toSave := make([]Character, len(characters))
	copy(toSave, characters)
	for i := range toSave {
		toSave[i].PassHash = scrambleHash(toSave[i].Name, toSave[i].PassHash)
	}
	data, err := json.MarshalIndent(toSave, "", "  ")

	if err != nil {
		log.Printf("save characters: %v", err)
		return
	}
	if err := os.WriteFile(filepath.Join(dataDirPath, charsFilePath), data, 0644); err != nil {
		log.Printf("save characters: %v", err)
	}
}

// scrambleHash obscures a hex-encoded hash by XORing with a key derived from a
// hardcoded value and the character name.
func scrambleHash(name, h string) string {
	b, err := hex.DecodeString(h)
	if err != nil {
		return h
	}
	k := []byte(hashKey + name)
	for i := range b {
		b[i] ^= k[i%len(k)]
	}
	return hex.EncodeToString(b)
}

// unscrambleHash reverses the operation of scrambleHash.
func unscrambleHash(name, h string) string { return scrambleHash(name, h) }

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
