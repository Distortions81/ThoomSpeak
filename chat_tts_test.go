package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPreparePiperMissingArchive(t *testing.T) {
	origSilent := silent
	silent = true
	defer func() { silent = origSilent }()

	supported := false
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64", "arm64", "arm":
			supported = true
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64", "arm64":
			supported = true
		}
	case "windows":
		if runtime.GOARCH == "amd64" {
			supported = true
		}
	}
	if !supported {
		t.Skipf("unsupported platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	dataDir := t.TempDir()
	binDir := filepath.Join(dataDir, "piper", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	binName := "piper"
	if runtime.GOOS == "windows" {
		binName = "piper.exe"
	}
	if err := os.WriteFile(filepath.Join(binDir, binName), []byte{}, 0o755); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, _, _, err := preparePiper(dataDir)
	if err == nil {
		t.Fatal("expected error for missing voice archive")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected missing file error, got %v", err)
	}
}
