package climg

import "image"

// fixAlphaDither scans img for 2x2 blocks that use a diagonal
// 0/255 alpha dithering pattern. When found, the block is replaced
// with a single color at 50% transparency to reduce shimmering
// artifacts during rendering.
func fixAlphaDither(img *image.RGBA) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	pix := img.Pix
	stride := img.Stride

	for y := 0; y < h-1; y++ {
		for x := 0; x < w-1; x++ {
			off00 := y*stride + x*4
			off01 := y*stride + (x+1)*4
			off10 := (y+1)*stride + x*4
			off11 := (y+1)*stride + (x+1)*4

			a00 := pix[off00+3]
			a01 := pix[off01+3]
			a10 := pix[off10+3]
			a11 := pix[off11+3]

			// Detect diagonal dither patterns:
			// [0 255] or [255 0]
			// [255 0]    [0 255]
			if !((a00 == 0 && a11 == 0 && a01 == 255 && a10 == 255) ||
				(a00 == 255 && a11 == 255 && a01 == 0 && a10 == 0)) {
				continue
			}

			var sumR, sumG, sumB, count int
			if a00 == 255 {
				sumR += int(pix[off00])
				sumG += int(pix[off00+1])
				sumB += int(pix[off00+2])
				count++
			}
			if a01 == 255 {
				sumR += int(pix[off01])
				sumG += int(pix[off01+1])
				sumB += int(pix[off01+2])
				count++
			}
			if a10 == 255 {
				sumR += int(pix[off10])
				sumG += int(pix[off10+1])
				sumB += int(pix[off10+2])
				count++
			}
			if a11 == 255 {
				sumR += int(pix[off11])
				sumG += int(pix[off11+1])
				sumB += int(pix[off11+2])
				count++
			}
			if count == 0 {
				continue
			}

			r := uint8(sumR * 128 / (count * 255))
			g := uint8(sumG * 128 / (count * 255))
			b := uint8(sumB * 128 / (count * 255))

			for _, off := range []int{off00, off01, off10, off11} {
				pix[off] = r
				pix[off+1] = g
				pix[off+2] = b
				pix[off+3] = 128
			}
		}
	}
}
