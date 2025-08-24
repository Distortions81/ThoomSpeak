package main

import (
	"bytes"
	_ "embed"
	"log"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"gothoom/eui"
)

//go:embed data/font/NotoSans/NotoSans-Regular.ttf
var notoSansRegular []byte

//go:embed data/font/NotoSans/NotoSans-Bold.ttf
var notoSansBold []byte

//go:embed data/font/NotoSans/NotoSans-Italic.ttf
var notoSansItalic []byte

//go:embed data/font/NotoSans/NotoSans-BoldItalic.ttf
var notoSansBoldItalic []byte

//go:embed data/font/NotoSans/NotoColorEmoji.ttf
var notoColorEmoji []byte

//go:embed data/font/NotoSans/NotoSansSymbols-Regular.ttf
var notoSansSymbols []byte

//go:embed data/font/NotoSans/NotoSansSymbols2-Regular.ttf
var notoSansSymbols2 []byte

var mainFont, mainFontBold, mainFontItalic, mainFontBoldItalic, bubbleFont text.Face
var fontGen uint32

func initFont() {
	fontGen++

	regular, err := text.NewGoTextFaceSource(bytes.NewReader(notoSansRegular))
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}

	bold, err := text.NewGoTextFaceSource(bytes.NewReader(notoSansBold))
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}

	italic, err := text.NewGoTextFaceSource(bytes.NewReader(notoSansItalic))
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}

	boldItalic, err := text.NewGoTextFaceSource(bytes.NewReader(notoSansBoldItalic))
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}

	emoji, err := text.NewGoTextFaceSource(bytes.NewReader(notoColorEmoji))
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}

	symbols, err := text.NewGoTextFaceSource(bytes.NewReader(notoSansSymbols))
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}

	symbols2, err := text.NewGoTextFaceSource(bytes.NewReader(notoSansSymbols2))
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}

	eui.SetFontSources(regular, symbols, symbols2, emoji)

	makeFace := func(src *text.GoTextFaceSource, size float64) text.Face {
		faces := []text.Face{
			&text.GoTextFace{Source: src, Size: size},
			&text.GoTextFace{Source: symbols, Size: size},
			&text.GoTextFace{Source: symbols2, Size: size},
			&text.GoTextFace{Source: emoji, Size: size},
		}
		mf, err := text.NewMultiFace(faces...)
		if err != nil {
			log.Fatalf("failed to create font face: %v", err)
		}
		return mf
	}

	size := gs.MainFontSize * gs.GameScale
	mainFont = makeFace(regular, size)
	mainFontBold = makeFace(bold, size)
	mainFontItalic = makeFace(italic, size)
	mainFontBoldItalic = makeFace(boldItalic, size)

	//Bubble
	bubbleFont = makeFace(bold, gs.BubbleFontSize*gs.GameScale)
}
