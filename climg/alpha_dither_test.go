package climg

import (
	"image"
	"image/color"
	"testing"
)

func TestFixAlphaDither(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	img.Set(1, 0, color.RGBA{R: 0, G: 0, B: 0, A: 0})
	img.Set(0, 1, color.RGBA{R: 0, G: 0, B: 0, A: 0})
	img.Set(1, 1, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	fixAlphaDither(img)

	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			c := img.RGBAAt(x, y)
			if c.A != 128 || c.R != 128 || c.G != 0 || c.B != 0 {
				t.Fatalf("pixel (%d,%d) = %v, want {128,0,0,128}", x, y, c)
			}
		}
	}
}
