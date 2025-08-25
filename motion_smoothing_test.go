package main

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestMotionSmoothingFailure(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	moviePath := filepath.Join(filepath.Dir(file), "clmovFiles", "2025a.clMov")
	resetState()
	initFont()
	frames, err := parseMovie(moviePath, 200)
	if err != nil {
		t.Fatalf("parseMovie: %v", err)
	}
	parsed := 0
	for _, f := range frames {
		if err := parseDrawState(f.data, false); err != nil {
			continue
		}
		parsed++
		if parsed == 2 {
			break
		}
	}
	if parsed < 2 {
		t.Fatalf("parsed %d frames", parsed)
	}
	if dx, dy, _, ok := pictureShift(state.prevPictures, state.pictures, maxInterpPixels); ok {
		t.Fatalf("pictureShift succeeded unexpectedly: (%d,%d)", dx, dy)
	}
}
