package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"time"
)

func takeScreenshot() {
	if worldRT == nil {
		return
	}
	dir := filepath.Join(dataDirPath, "Screenshots")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		logError("screenshot: create %v: %v", dir, err)
		return
	}
	ts := time.Now().Format("2006-01-02-15-04-05")
	fn := filepath.Join(dir, fmt.Sprintf("clan-lord-%s.png", ts))
	f, err := os.Create(fn)
	if err != nil {
		logError("screenshot: create %v: %v", fn, err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, worldRT); err != nil {
		logError("screenshot: encode %v: %v", fn, err)
	}
}
