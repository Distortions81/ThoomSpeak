package main

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestDrawPictureBypassesSmoothMoving(t *testing.T) {
	origGS := gs
	origCache := imageCache
	defer func() {
		gs = origGS
		imageCache = origCache
	}()

	gs.smoothMoving = false
	gs.MotionSmoothing = true
	gs.GameScale = 1
	gs.hideMoving = false
	gs.dontShiftNewSprites = false

	imageCache = make(map[imageKey]*ebiten.Image)
	img := ebiten.NewImage(1, 1)
	img.Fill(color.White)
	imageCache[makeImageKey(1, 0)] = img

	screen := ebiten.NewImage(600, 600)
	p := framePicture{PictID: 1, H: 23, V: 26, PrevH: 21, PrevV: 23, Moving: true}
	m := frameMobile{Index: 1, H: 20, V: 30}
	pm := frameMobile{Index: 1, H: 18, V: 27}
	mobiles := []frameMobile{m}
	prev := map[uint8]frameMobile{1: pm}

	drawPicture(screen, 0, 0, p, 0.5, 0, mobiles, nil, prev, 0, 0)

	_, _, _, a := screen.At(295, 295).RGBA()
	if a == 0 {
		t.Fatalf("expected pixel at interpolated position")
	}
}
