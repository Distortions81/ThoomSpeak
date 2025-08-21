package climg

import (
	"image"
	"image/color"
	"testing"
)

func TestFixAlphaDither(t *testing.T) {
	cases := []struct {
		name   string
		alphas [16]uint8 // 4x4 block in row-major order
		want   uint8     // expected R and A values
	}{
		{
			name: "25%",
			alphas: [16]uint8{
				255, 255, 255, 255,
				0, 0, 0, 0,
				0, 0, 0, 0,
				0, 0, 0, 0,
			},
			want: 64,
		},
		{
			name: "50%",
			alphas: [16]uint8{
				255, 255, 255, 255,
				255, 255, 255, 255,
				0, 0, 0, 0,
				0, 0, 0, 0,
			},
			want: 128,
		},
		{
			name: "75%",
			alphas: [16]uint8{
				255, 255, 255, 255,
				255, 255, 255, 255,
				255, 255, 255, 255,
				0, 0, 0, 0,
			},
			want: 191,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, 4, 4))
			for i, a := range tc.alphas {
				x, y := i%4, i/4
				img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: a})
			}

			fixAlphaDither(img)

			want := color.RGBA{R: tc.want, G: 0, B: 0, A: tc.want}
			for y := 0; y < 4; y++ {
				for x := 0; x < 4; x++ {
					if c := img.RGBAAt(x, y); c != want {
						t.Fatalf("pixel (%d,%d) = %v, want %v", x, y, c, want)
					}
				}
			}
		})
	}
}
