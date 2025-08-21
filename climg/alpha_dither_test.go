package climg

import (
	"image"
	"image/color"
	"testing"
)

func TestFixAlphaDither(t *testing.T) {
	cases := []struct {
		name   string
		alphas [4]uint8 // a00, a01, a10, a11
		want   uint8    // expected R and A values
	}{
		{"25%", [4]uint8{255, 0, 0, 0}, 64},
		{"50%", [4]uint8{255, 0, 0, 255}, 128},
		{"75%", [4]uint8{255, 255, 255, 0}, 191},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, 2, 2))
			img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: tc.alphas[0]})
			img.Set(1, 0, color.RGBA{R: 255, G: 0, B: 0, A: tc.alphas[1]})
			img.Set(0, 1, color.RGBA{R: 255, G: 0, B: 0, A: tc.alphas[2]})
			img.Set(1, 1, color.RGBA{R: 255, G: 0, B: 0, A: tc.alphas[3]})

			fixAlphaDither(img)

			want := color.RGBA{R: tc.want, G: 0, B: 0, A: tc.want}
			for y := 0; y < 2; y++ {
				for x := 0; x < 2; x++ {
					if c := img.RGBAAt(x, y); c != want {
						t.Fatalf("pixel (%d,%d) = %v, want %v", x, y, c, want)
					}
				}
			}
		})
	}
}
