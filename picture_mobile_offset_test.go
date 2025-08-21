package main

import "testing"

func TestPictureMobileOffset(t *testing.T) {
	alpha := 0.5
	p := framePicture{H: 23, V: 26, PrevH: 21, PrevV: 23}
	m := frameMobile{Index: 1, H: 20, V: 30}
	pm := frameMobile{Index: 1, H: 18, V: 27}
	mobiles := []frameMobile{m}
	prev := map[uint8]frameMobile{1: pm}

	dx, dy, ok := pictureMobileOffset(p, mobiles, prev, alpha, 0, 0)
	if !ok {
		t.Fatalf("expected picture to follow mobile")
	}
	wantDX := (float64(pm.H)*(1-alpha) + float64(m.H)*alpha) - float64(m.H)
	wantDY := (float64(pm.V)*(1-alpha) + float64(m.V)*alpha) - float64(m.V)
	if dx != wantDX || dy != wantDY {
		t.Fatalf("unexpected offsets dx=%v dy=%v want=(%v,%v)", dx, dy, wantDX, wantDY)
	}

	// Change previous offsets so they no longer match.
	p.PrevH++
	if _, _, ok := pictureMobileOffset(p, mobiles, prev, alpha, 0, 0); ok {
		t.Fatalf("expected mismatch in offsets to disable following")
	}
}
