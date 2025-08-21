package climg

import "image"

// fixAlphaDither scans img for 4x4 blocks that simulate partial
// transparency by dithering 0 and 255 alpha values. When such a
// pattern is found the block is replaced with a uniform color and
// alpha matching the coverage to reduce shimmering artifacts.
func fixAlphaDither(img *image.RGBA) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	pix := img.Pix
	stride := img.Stride

	for y := 0; y <= h-4; y += 4 {
		for x := 0; x <= w-4; x += 4 {
			var sumR, sumG, sumB, count int
			ok := true
			offs := [16]int{}
			idx := 0
			for dy := 0; dy < 4 && ok; dy++ {
				for dx := 0; dx < 4; dx++ {
					off := (y+dy)*stride + (x+dx)*4
					a := pix[off+3]
					if a != 0 && a != 255 {
						ok = false
						break
					}
					if a == 255 {
						sumR += int(pix[off])
						sumG += int(pix[off+1])
						sumB += int(pix[off+2])
						count++
					}
					offs[idx] = off
					idx++
				}
			}
			if !ok || count == 0 || count == 16 {
				continue
			}

			r := uint8((sumR + 8) / 16)
			g := uint8((sumG + 8) / 16)
			b := uint8((sumB + 8) / 16)
			a := uint8((count*255 + 8) / 16)

			for _, off := range offs {
				pix[off] = r
				pix[off+1] = g
				pix[off+2] = b
				pix[off+3] = a
			}
		}
	}
}
