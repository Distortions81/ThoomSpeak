package main

import (
	"encoding/binary"
	"reflect"
	"testing"
	"unsafe"

	"gothoom/climg"
)

func mockCLImages(w, h int) *climg.CLImages {
	imgs := &climg.CLImages{}
	v := reflect.ValueOf(imgs).Elem()

	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data[:2], uint16(h))
	binary.BigEndian.PutUint16(data[2:], uint16(w))
	dataField := v.FieldByName("data")
	reflect.NewAt(dataField.Type(), unsafe.Pointer(dataField.UnsafeAddr())).Elem().Set(reflect.ValueOf(data))

	idrefsField := v.FieldByName("idrefs")
	imagesField := v.FieldByName("images")
	idrefsMap := reflect.MakeMap(idrefsField.Type())
	imagesMap := reflect.MakeMap(imagesField.Type())

	dlType := idrefsField.Type().Elem().Elem()
	idref := reflect.New(dlType)
	imageIDField := idref.Elem().FieldByName("imageID")
	reflect.NewAt(imageIDField.Type(), unsafe.Pointer(imageIDField.UnsafeAddr())).Elem().SetUint(1)
	idrefsMap.SetMapIndex(reflect.ValueOf(uint32(1)), idref)

	imgLoc := reflect.New(dlType)
	imagesMap.SetMapIndex(reflect.ValueOf(uint32(1)), imgLoc)

	reflect.NewAt(idrefsField.Type(), unsafe.Pointer(idrefsField.UnsafeAddr())).Elem().Set(idrefsMap)
	reflect.NewAt(imagesField.Type(), unsafe.Pointer(imagesField.UnsafeAddr())).Elem().Set(imagesMap)

	return imgs
}

func TestPictureOnEdge(t *testing.T) {
	halfW := 5
	halfH := 5

	tests := []struct {
		name string
		p    framePicture
		w    int
		h    int
		want bool
	}{
		{"inside", framePicture{PictID: 1, H: 0, V: 0}, 10, 10, false},
		{"left edge", framePicture{PictID: 1, H: int16(-fieldCenterX + halfW), V: 0}, 10, 10, true},
		{"right edge", framePicture{PictID: 1, H: int16(fieldCenterX - halfW), V: 0}, 10, 10, true},
		{"top edge", framePicture{PictID: 1, H: 0, V: int16(-fieldCenterY + halfH)}, 10, 10, true},
		{"bottom edge", framePicture{PictID: 1, H: 0, V: int16(fieldCenterY - halfH)}, 10, 10, true},
		{"outside", framePicture{PictID: 1, H: int16(fieldCenterX + halfW + 1), V: 0}, 10, 10, false},
		{"spanning middle", framePicture{PictID: 1, H: 0, V: 0}, gameAreaSizeX * 2, gameAreaSizeY * 2, false},
		{"wide edge big", framePicture{PictID: 1, H: int16(-fieldCenterX + 150), V: 0}, 300, 10, false},
		{"tall edge big", framePicture{PictID: 1, H: 0, V: int16(-fieldCenterY + 150)}, 10, 300, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clImages = mockCLImages(tt.w, tt.h)
			defer func() { clImages = nil }()
			if got := pictureOnEdge(tt.p); got != tt.want {
				t.Fatalf("pictureOnEdge(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
