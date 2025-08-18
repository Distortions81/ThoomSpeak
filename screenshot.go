package main

import (
	"fmt"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func takeScreenshot() {
	if gameImage == nil {
		return
	}
	dir := filepath.Join(dataDirPath, "Screenshots")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		logError("screenshot: create %v: %v", dir, err)
		return
	}
	ts := time.Now().Format("2006-01-02-15-04-05")
	buf := "clanlord-"
	if clmov != "" {
		clname := path.Base(clmov)
		clname = strings.ToLower(clname)
		clname = strings.TrimSuffix(clname, ".clmov")
		cllen := len(clname)
		if cllen > 16 {
			clname = clname[:16]
		}
		buf = clname
	}
	if gs.LastCharacter != "" {
		buf = gs.LastCharacter
	}

	fn := filepath.Join(dir, fmt.Sprintf("%v__%s.png", buf, ts))
	f, err := os.Create(fn)
	if err != nil {
		logError("screenshot: create %v: %v", fn, err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, gameImage); err != nil {
		logError("screenshot: encode %v: %v", fn, err)
	}
	consoleMessage("Screenshot saved.")
}
