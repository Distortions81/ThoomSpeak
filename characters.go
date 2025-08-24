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
	Name         string `json:"name"`
	passHash     string `json:"-"`
	Key          string `json:"key"`
	DontRemember bool   `json:"-"`
	PictID       uint16 `json:"pict,omitempty"`
	ColorsHex    string `json:"colors,omitempty"`
	Colors       []byte `json:"-"`
	Profession   string `json:"prof,omitempty"`
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

	var charList charactersFile
	if err := json.Unmarshal(data, &charList); err != nil {
		return
	}
	if charList.Version >= 1 {
		characters = charList.Characters
		for i := range characters {
			characters[i].passHash = unscrambleHash(characters[i].Name, characters[i].Key)
			if charList.Version >= 2 && characters[i].ColorsHex != "" {
				if b, ok := decodeHex(characters[i].ColorsHex); ok && len(b) > 0 {
					cnt := int(b[0])
					if cnt > 0 && 1+cnt <= len(b) {
						characters[i].Colors = append(characters[i].Colors[:0], b[1:1+cnt]...)
					} else {
						characters[i].Colors = append(characters[i].Colors[:0], b...)
					}
				}
			}
		}
	}
}

func saveCharacters() {
	var persisted []Character
	for i := range characters {
		if characters[i].DontRemember {
			continue
		}
		characters[i].Key = scrambleHash(characters[i].Name, characters[i].passHash)
		if len(characters[i].Colors) > 0 {
			buf := make([]byte, 1+len(characters[i].Colors))
			if len(characters[i].Colors) > 255 {
				buf[0] = 255
				copy(buf[1:], characters[i].Colors[:255])
			} else {
				buf[0] = byte(len(characters[i].Colors))
				copy(buf[1:], characters[i].Colors)
			}
			characters[i].ColorsHex = encodeHex(buf)
		} else {
			characters[i].ColorsHex = ""
		}
		persisted = append(persisted, characters[i])
	}

	var charList charactersFile
	charList.Version = 2
	charList.Characters = persisted
	data, err := json.MarshalIndent(charList, "", "  ")

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
