package climg

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestPaletteRemapToIndexZeroTransparent ensures that a non-zero pixel mapped
// to palette index 0 is treated as transparent, matching old_mac_client
// behavior.
func TestPaletteRemapToIndexZeroTransparent(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint16(1)) // height
	binary.Write(&buf, binary.BigEndian, uint16(1)) // width
	binary.Write(&buf, binary.BigEndian, uint32(0)) // pad
	buf.WriteByte(1)                                // bits per pixel
	buf.WriteByte(0)                                // run length bits
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
			1: {colorBytes: []uint16{0, 0}}, // map index 1 -> palette 0
		},
	}

	if n := c.NonTransparentPixels(1); n != 0 {
		t.Fatalf("NonTransparentPixels = %d, want 0", n)
	}
}
