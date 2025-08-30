package climg

import (
	"image"
	"image/color"
	"math"
	"sync"
)

// denoiseImage softens pixels by blending with neighbours. Pixels with
// similar hue and brightness are blended more strongly while dissimilar
// pixels are blended less. The sharpness parameter controls how quickly the
// blend amount falls off as colours become more different. Only the immediate
// horizontal and vertical neighbours are considered.
func denoiseImage(img *image.RGBA, sharpness, maxPercent float64) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Work on a copy so neighbour checks aren't affected by in-place writes.
	src := getTempRGBA(bounds)
	copy(src.Pix, img.Pix)

	neighbours := []image.Point{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	for y := 1; y < h-1; y++ {
		yoff := y * src.Stride
		for x := 1; x < w-1; x++ {
			off := yoff + x*4
			c := color.RGBA{src.Pix[off], src.Pix[off+1], src.Pix[off+2], src.Pix[off+3]}

			// If this pixel is opaque and all direct neighbours are
			// non-opaque, blur it unless it is full black.
			if c.A == 0xFF && (c.R != 0 || c.G != 0 || c.B != 0) {
				isolated := true
				for _, n := range neighbours {
					nOff := (y+n.Y)*src.Stride + (x+n.X)*4
					if src.Pix[nOff+3] == 0xFF {
						isolated = false
						break
					}
				}
				if isolated {
					c = mixColour(c, color.RGBA{}, maxPercent)
				}
			}

			for _, n := range neighbours {
				nOff := (y+n.Y)*src.Stride + (x+n.X)*4
				ncol := color.RGBA{src.Pix[nOff], src.Pix[nOff+1], src.Pix[nOff+2], src.Pix[nOff+3]}
				dist := colourDist(c, ncol)
				if dist < 1 {
					blend := maxPercent * math.Pow(1-dist, sharpness)
					if blend > 0 {
						c = mixColour(c, ncol, blend)
					}
				}
			}

			dstOff := y*img.Stride + x*4
			img.Pix[dstOff] = c.R
			img.Pix[dstOff+1] = c.G
			img.Pix[dstOff+2] = c.B
			img.Pix[dstOff+3] = c.A
		}
	}
	putTempRGBA(src)
}

var rgbaPool = sync.Pool{New: func() any { return &image.RGBA{} }}

func getTempRGBA(bounds image.Rectangle) *image.RGBA {
	img := rgbaPool.Get().(*image.RGBA)
	w, h := bounds.Dx(), bounds.Dy()
	need := w * h * 4
	if cap(img.Pix) < need {
		img.Pix = make([]uint8, need)
	}
	img.Pix = img.Pix[:need]
	img.Stride = w * 4
	img.Rect = bounds
	return img
}

func putTempRGBA(img *image.RGBA) { rgbaPool.Put(img) }

// colourDist returns a normalised distance [0,1] based on hue and brightness
// differences between two colours. Values >= 1 indicate colours that should
// not be blended.
const dt = 0

func colourDist(a, b color.RGBA) float64 {
	if a.A < 0xFF || b.A < 0xFF ||
		(a.R == dt && a.G == dt && a.B == dt) ||
		(b.R == dt && b.G == dt && b.B == dt) {
		return 2 // sentinel > 1
	}

	r1, g1, b1 := float64(a.R)/255, float64(a.G)/255, float64(a.B)/255
	r2, g2, b2 := float64(b.R)/255, float64(b.G)/255, float64(b.B)/255

	h1, s1, v1 := rgbToHSV(r1, g1, b1)
	h2, s2, v2 := rgbToHSV(r2, g2, b2)

	dh := math.Abs(h1 - h2)
	if dh > 180 {
		dh = 360 - dh
	}
	dh /= 360
	dv := math.Abs(v1 - v2)
	avgSat := (s1 + s2) / 2

	d := dh*avgSat + dv*(1-avgSat)
	if d > 1 {
		return 1
	}
	return d
}

func rgbToHSV(r, g, b float64) (h, s, v float64) {
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	v = max
	d := max - min
	if max != 0 {
		s = d / max
	} else {
		return 0, 0, 0
	}
	if d == 0 {
		return 0, s, v
	}
	switch {
	case r == max:
		h = (g - b) / d
	case g == max:
		h = 2 + (b-r)/d
	default:
		h = 4 + (r-g)/d
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return
}

// mixColour blends two colours together by the provided percentage.
func mixColour(a, b color.RGBA, p float64) color.RGBA {
	inv := 1 - p
	return color.RGBA{
		R: uint8(float64(a.R)*inv + float64(b.R)*p),
		G: uint8(float64(a.G)*inv + float64(b.G)*p),
		B: uint8(float64(a.B)*inv + float64(b.B)*p),
		A: uint8(float64(a.A)*inv + float64(b.A)*p),
	}
}
