package eui

import "github.com/hajimehoshi/ebiten/v2/text/v2"

var faceCache = map[float64]text.Face{}

func textFace(size float32) text.Face {
	if len(mplusFaceSources) == 0 {
		return &text.GoTextFace{Size: float64(size)}
	}
	s := float64(size)
	if f, ok := faceCache[s]; ok {
		return f
	}
	faces := make([]text.Face, len(mplusFaceSources))
	for i, src := range mplusFaceSources {
		faces[i] = &text.GoTextFace{Source: src, Size: s}
	}
	if len(faces) == 1 {
		faceCache[s] = faces[0]
		return faces[0]
	}
	mf, err := text.NewMultiFace(faces...)
	if err != nil {
		faceCache[s] = faces[0]
		return faces[0]
	}
	faceCache[s] = mf
	return mf
}
