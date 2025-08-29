package main

import (
	"bytes"
	_ "embed"
	"log"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"gothoom/eui"
)

//go:embed data/font/NotoSans-Regular.ttf
var notoSansRegular []byte

//go:embed data/font/NotoSans-Bold.ttf
var notoSansBold []byte

//go:embed data/font/NotoSans-Italic.ttf
var notoSansItalic []byte

//go:embed data/font/NotoSans-BoldItalic.ttf
var notoSansBoldItalic []byte

//go:embed data/font/NotoSansMono-Regular.ttf
var notoSansMonoRegular []byte

//go:embed data/font/NotoSansMono-Bold.ttf
var notoSansMonoBold []byte

var mainFont, mainFontBold, mainFontItalic, mainFontBoldItalic, monoFont, monoFontBold, bubbleFont text.Face
var monoFaceSource *text.GoTextFaceSource
var fontGen uint32

func initFont() {
	fontGen++
	regular, err := text.NewGoTextFaceSource(bytes.NewReader(notoSansRegular))
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}
	eui.SetFontSource(regular)
	mainFont = &text.GoTextFace{
		Source: regular,
		Size:   gs.MainFontSize * gs.GameScale,
	}

	bold, err := text.NewGoTextFaceSource(bytes.NewReader(notoSansBold))
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}
	mainFontBold = &text.GoTextFace{
		Source: bold,
		Size:   gs.MainFontSize * gs.GameScale,
	}
	eui.SetBoldFontSource(bold)

	italic, err := text.NewGoTextFaceSource(bytes.NewReader(notoSansItalic))
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}
	mainFontItalic = &text.GoTextFace{
		Source: italic,
		Size:   gs.MainFontSize * gs.GameScale,
	}

	boldItalic, err := text.NewGoTextFaceSource(bytes.NewReader(notoSansBoldItalic))
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}
	mainFontBoldItalic = &text.GoTextFace{
		Source: boldItalic,
		Size:   gs.MainFontSize * gs.GameScale,
	}

	monoRegular, err := text.NewGoTextFaceSource(bytes.NewReader(notoSansMonoRegular))
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}
	monoFaceSource = monoRegular
	monoFont = &text.GoTextFace{
		Source: monoRegular,
		Size:   gs.MainFontSize * gs.GameScale,
	}

	monoBold, err := text.NewGoTextFaceSource(bytes.NewReader(notoSansMonoBold))
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}
	monoFontBold = &text.GoTextFace{
		Source: monoBold,
		Size:   gs.MainFontSize * gs.GameScale,
	}

	//Bubble
	bubbleFont = &text.GoTextFace{
		Source: bold,
		Size:   gs.BubbleFontSize * gs.GameScale,
	}
}
