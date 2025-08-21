package climg

import (
	"image"
	"math"

	"github.com/mjibson/go-dsp/fft"
)

// denoiseImage applies a frequency-domain low-pass filter to soften dithered
// indexed images. sharpness controls the roll-off of the Gaussian filter while
// maxPercent sets the cutoff radius as a fraction of the image size.
func denoiseImage(img *image.RGBA, sharpness, maxPercent float64) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return
	}

	// Extract colour channels.
	r := make([][]float64, h)
	g := make([][]float64, h)
	b := make([][]float64, h)
	for y := 0; y < h; y++ {
		r[y] = make([]float64, w)
		g[y] = make([]float64, w)
		b[y] = make([]float64, w)
		off := y * img.Stride
		for x := 0; x < w; x++ {
			r[y][x] = float64(img.Pix[off+0])
			g[y][x] = float64(img.Pix[off+1])
			b[y][x] = float64(img.Pix[off+2])
			off += 4
		}
	}

	radius := maxPercent * float64(max(w, h)) / 2
	if radius <= 0 {
		return
	}

	// Apply low-pass filter in frequency domain.
	apply := func(ch [][]float64) {
		freq := fft.FFT2Real(ch)
		r2 := radius * radius
		for y := 0; y < h; y++ {
			dy := float64(y)
			if y > h/2 {
				dy = float64(y - h)
			}
			for x := 0; x < w; x++ {
				dx := float64(x)
				if x > w/2 {
					dx = float64(x - w)
				}
				mask := math.Exp(-(dx*dx + dy*dy) / (2 * r2))
				if sharpness != 0 {
					mask = math.Pow(mask, sharpness)
				}
				freq[y][x] *= complex(mask, 0)
			}
		}
		spatial := fft.IFFT2(freq)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				ch[y][x] = real(spatial[y][x])
			}
		}
	}

	apply(r)
	apply(g)
	apply(b)

	// Write the filtered channels back to the image.
	for y := 0; y < h; y++ {
		off := y * img.Stride
		for x := 0; x < w; x++ {
			img.Pix[off+0] = clampByte(r[y][x])
			img.Pix[off+1] = clampByte(g[y][x])
			img.Pix[off+2] = clampByte(b[y][x])
			off += 4
		}
	}
}

func clampByte(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
