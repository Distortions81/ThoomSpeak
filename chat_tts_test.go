package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestListPiperVoices(t *testing.T) {
	dir := t.TempDir()
	orig := dataDirPath
	dataDirPath = dir
	defer func() { dataDirPath = orig }()

	voicesDir := filepath.Join(dir, "piper", "voices")
	if err := os.MkdirAll(voicesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// voice stored directly in voices directory
	if err := os.WriteFile(filepath.Join(voicesDir, "rootvoice.onnx"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(voicesDir, "rootvoice.onnx.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	// voice stored inside a matching subdirectory
	sub := filepath.Join(voicesDir, "dirvoice")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "dirvoice.onnx"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "dirvoice.onnx.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	// voice stored inside a mismatching subdirectory
	mis := filepath.Join(voicesDir, "mismatch")
	if err := os.MkdirAll(mis, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mis, "othervoice.onnx"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mis, "othervoice.onnx.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	voices, err := listPiperVoices()
	if err != nil {
		t.Fatalf("listPiperVoices: %v", err)
	}
	want := []string{"dirvoice", "othervoice", "rootvoice"}
	if !reflect.DeepEqual(voices, want) {
		t.Fatalf("voices = %v, want %v", voices, want)
	}
}
