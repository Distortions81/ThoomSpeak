package main

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// Test that preparePiper extracts voice archives and removes them.
func TestPreparePiperVoiceArchive(t *testing.T) {
	dataDir := t.TempDir()
	piperDir := filepath.Join(dataDir, "piper")
	binDir := filepath.Join(piperDir, "bin")
	if err := os.MkdirAll(filepath.Join(binDir, "piper"), 0o755); err != nil {
		t.Fatal(err)
	}
	// create dummy piper binary inside nested directory to skip download
	binPath := filepath.Join(binDir, "piper", "piper")
	if err := os.WriteFile(binPath, []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	voicesDir := filepath.Join(piperDir, "voices")
	if err := os.MkdirAll(voicesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// create voice tarball
	tarPath := filepath.Join(voicesDir, piperFemaleVoice+".tar.gz")
	f, err := os.Create(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	files := map[string]string{
		filepath.Join(piperFemaleVoice, piperFemaleVoice+".onnx"):      "",
		filepath.Join(piperFemaleVoice, piperFemaleVoice+".onnx.json"): "{}",
	}
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0644, Size: int64(len(content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	// run preparePiper which should extract and remove tarball
	path, model, cfg, err := preparePiper(dataDir)
	if err != nil {
		t.Fatalf("preparePiper: %v", err)
	}
	if path != binPath {
		t.Fatalf("bin path = %v, want %v", path, binPath)
	}
	if _, err := os.Stat(tarPath); !os.IsNotExist(err) {
		t.Fatalf("archive not removed")
	}
	if _, err := os.Stat(model); err != nil {
		t.Fatalf("model missing: %v", err)
	}
	if _, err := os.Stat(cfg); err != nil {
		t.Fatalf("config missing: %v", err)
	}
}
