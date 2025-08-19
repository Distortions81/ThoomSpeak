package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCharactersRequiresVersion(t *testing.T) {
	dir := t.TempDir()
	oldDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	if err := os.Mkdir("data", 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data := []byte(`[{"name":"foo","passHash":"bar"}]`)
	if err := os.WriteFile(filepath.Join("data", charsFilePath), data, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	characters = nil
	loadCharacters()
	if len(characters) != 0 {
		t.Fatalf("expected 0 characters, got %d", len(characters))
	}
}

func TestLoadCharactersWithVersion(t *testing.T) {
	dir := t.TempDir()
	oldDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	if err := os.Mkdir("data", 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cf := charactersFile{
		Version:    charsFileVersion,
		Characters: []Character{{Name: "foo", PassHash: "bar"}},
	}
	data, err := json.Marshal(cf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join("data", charsFilePath), data, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	characters = nil
	loadCharacters()
	if len(characters) != 1 || characters[0].Name != "foo" {
		t.Fatalf("characters not loaded: %+v", characters)
	}
}
