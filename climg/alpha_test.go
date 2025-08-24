package climg

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestNonTransparentPixelsIgnoresPaletteIndex verifies that transparency
// is determined solely by the original pixel index and not by the mapped
// palette color. A non-zero pixel mapped to palette index 0 should still
// be counted as non-transparent.
func TestNonTransparentPixelsIgnoresPaletteIndex(t *testing.T) {
	// Build minimal image data: 1x1 pixel with value 1.
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint16(1)) // height
	binary.Write(&buf, binary.BigEndian, uint16(1)) // width
	binary.Write(&buf, binary.BigEndian, uint32(0)) // pad
	buf.WriteByte(1)                                // value bits per pixel
	buf.WriteByte(0)                                // block length bits
	buf.Write([]byte{0x40})                         // encoded pixel: t=0, value=1

	imgData := buf.Bytes()

	c := &CLImages{
		data: imgData,
		idrefs: map[uint32]*dataLocation{
			1: {imageID: 1, colorID: 1, flags: pictDefFlagTransparent},
		},
		images: map[uint32]*dataLocation{
			1: {offset: 0},
		},
		colors: map[uint32]*dataLocation{
			// Map color index 1 to palette index 0 to replicate bug case.
			1: {colorBytes: []uint16{0, 0}},
		},
	}

	if n := c.NonTransparentPixels(1); n != 1 {
		t.Fatalf("NonTransparentPixels = %d, want 1", n)
	}
}
