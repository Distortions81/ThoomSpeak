package main

import (
	"gothoom/eui"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// whiteImage is a reusable 1x1 white pixel used across the UI for drawing
// solid rectangles and lines without creating multiple images.
var whiteImage *ebiten.Image
var blackImage *ebiten.Image
var grayImage *ebiten.Image

func init() {
	whiteImage = newImage(1, 1)
	whiteImage.Fill(color.White)

	blackImage = newImage(1, 1)
	blackImage.Fill(color.Black)

	grayImage = newImage(1, 1)
	grayImage.Fill(eui.Color{R: 128, G: 128, B: 128})
}

// adjustBubbleRect calculates the on-screen rectangle for a bubble and clamps
// it to the visible area. The tail tip coordinates remain unchanged and must
// be handled by the caller if needed.
func adjustBubbleRect(x, y, width, height, tailHeight, sw, sh int, far bool) (left, top, right, bottom int) {
	bottom = y
	if !far {
		bottom = y - tailHeight
	}
	left = x - width/2
	top = bottom - height

	if left < 0 {
		left = 0
	}
	if left+width > sw {
		left = sw - width
	}
	if top < 0 {
		top = 0
	}
	if top+height > sh {
		top = sh - height
	}

	right = left + width
	bottom = top + height
	return
}

// bubbleColors selects the border, background, and text colors for a bubble
// based on its type. Alpha values are premultiplied to match Ebiten's color
// expectations.

func bubbleColors(typ int) (border, bg, text color.Color) {
	alpha := uint8(gs.BubbleOpacity * 255)
	switch typ & kBubbleTypeMask {
	case kBubbleWhisper:
		border = color.NRGBA{0x80, 0x80, 0x80, 0xff}
		bg = color.NRGBA{0x33, 0x33, 0x33, alpha}
		text = color.White
	case kBubbleYell:
		border = color.NRGBA{0xff, 0xff, 0x00, 0xff}
		bg = color.NRGBA{0xff, 0xff, 0xff, alpha}
		text = color.Black
	case kBubbleThought:
		border = color.NRGBA{0x00, 0x00, 0x00, 0x00}
		bg = color.NRGBA{0x80, 0x80, 0x80, alpha}
		text = color.Black
	case kBubblePonder:
		border = color.NRGBA{0x80, 0x80, 0x80, 0xff}
		bg = color.NRGBA{0x80, 0x80, 0x80, alpha}
		text = color.Black
	case kBubbleRealAction:
		border = color.NRGBA{0x00, 0x00, 0x80, 0xff}
		bg = color.NRGBA{0xff, 0xff, 0xff, alpha}
		text = color.Black
	case kBubblePlayerAction:
		border = color.NRGBA{0x80, 0x00, 0x00, 0xff}
		bg = color.NRGBA{0xff, 0xff, 0xff, alpha}
		text = color.Black
	case kBubbleNarrate:
		border = color.NRGBA{0x00, 0x80, 0x00, 0xff}
		bg = color.NRGBA{0xff, 0xff, 0xff, alpha}
		text = color.Black
	case kBubbleMonster:
		border = color.NRGBA{0xd6, 0xd6, 0xd6, 0xff}
		bg = color.NRGBA{0x47, 0x47, 0x47, alpha}
		text = color.White
	default:
		border = color.White
		bg = color.NRGBA{0xff, 0xff, 0xff, alpha}
		text = color.Black
	}
	return
}

// drawBubble renders a text bubble anchored so that (x, y) corresponds to the
// bottom-center point of the balloon tail. If the bubble would extend past the
// screen edges it is clamped while leaving the tail anchored at (x, y). If far
// is true the tail is omitted and (x, y) represents the bottom-center of the
// bubble itself. The tail can also be skipped explicitly via noArrow. The typ
// parameter is currently unused but retained for future compatibility with the
// original bubble images. The colors of the border, background, and text can be
// customized via borderCol, bgCol, and textCol respectively.
func drawBubble(screen *ebiten.Image, txt string, x, y int, typ int, far bool, noArrow bool, borderCol, bgCol, textCol color.Color) {
	if txt == "" {
		return
	}
	tailX, tailY := x, y

	sw := int(float64(gameAreaSizeX) * gs.GameScale)
	sh := int(float64(gameAreaSizeY) * gs.GameScale)
	pad := int((4 + 2) * gs.GameScale)
	tailHeight := int(10 * gs.GameScale)
	tailHalf := int(6 * gs.GameScale)
	bubbleType := typ & kBubbleTypeMask

	maxLineWidth := sw/4 - 2*pad
	width, lines := wrapText(txt, bubbleFont, float64(maxLineWidth))
	metrics := bubbleFont.Metrics()
	lineHeight := int(math.Ceil(metrics.HAscent) + math.Ceil(metrics.HDescent) + math.Ceil(metrics.HLineGap))
	width += 2 * pad
	height := lineHeight*len(lines) + 2*pad

	left, top, right, bottom := adjustBubbleRect(x, y, width, height, tailHeight, sw, sh, far)
	baseX := left + width/2

	bgR, bgG, bgB, bgA := bgCol.RGBA()

	radius := float32(4 * gs.GameScale)
	if bubbleType == kBubblePonder {
		radius = float32(8 * gs.GameScale)
	}

	var body vector.Path
	body.MoveTo(float32(left)+radius, float32(top))
	body.LineTo(float32(right)-radius, float32(top))
	body.Arc(float32(right)-radius, float32(top)+radius, radius, -math.Pi/2, 0, vector.Clockwise)
	body.LineTo(float32(right), float32(bottom)-radius)
	body.Arc(float32(right)-radius, float32(bottom)-radius, radius, 0, math.Pi/2, vector.Clockwise)
	body.LineTo(float32(left)+radius, float32(bottom))
	body.Arc(float32(left)+radius, float32(bottom)-radius, radius, math.Pi/2, math.Pi, vector.Clockwise)
	body.LineTo(float32(left), float32(top)+radius)
	body.Arc(float32(left)+radius, float32(top)+radius, radius, math.Pi, 3*math.Pi/2, vector.Clockwise)
	body.Close()

	var tail vector.Path
	if !far && !noArrow {
		if bubbleType == kBubblePonder {
			r1 := float32(tailHalf)
			cx1 := float32(baseX - tailHalf)
			cy1 := float32(bottom) + r1
			tail.MoveTo(cx1+r1, cy1)
			tail.Arc(cx1, cy1, r1, 0, 2*math.Pi, vector.Clockwise)
			tail.Close()
			r2 := float32(tailHalf) / 2
			cx2 := float32(tailX)
			cy2 := float32(tailY)
			tail.MoveTo(cx2+r2, cy2)
			tail.Arc(cx2, cy2, r2, 0, 2*math.Pi, vector.Clockwise)
			tail.Close()
		} else {
			tail.MoveTo(float32(baseX-tailHalf), float32(bottom))
			tail.LineTo(float32(tailX), float32(tailY))
			tail.LineTo(float32(baseX+tailHalf), float32(bottom))
			tail.Close()
		}
	}

	vs, is := body.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].SrcX = 0
		vs[i].SrcY = 0
		vs[i].ColorR = float32(bgR) / 0xffff
		vs[i].ColorG = float32(bgG) / 0xffff
		vs[i].ColorB = float32(bgB) / 0xffff
		vs[i].ColorA = float32(bgA) / 0xffff
	}
	op := &ebiten.DrawTrianglesOptions{ColorScaleMode: ebiten.ColorScaleModePremultipliedAlpha, AntiAlias: true}
	screen.DrawTriangles(vs, is, whiteImage, op)

	if !far && !noArrow {
		vs, is = tail.AppendVerticesAndIndicesForFilling(vs[:0], is[:0])
		for i := range vs {
			vs[i].SrcX = 0
			vs[i].SrcY = 0
			vs[i].ColorR = float32(bgR) / 0xffff
			vs[i].ColorG = float32(bgG) / 0xffff
			vs[i].ColorB = float32(bgB) / 0xffff
			vs[i].ColorA = float32(bgA) / 0xffff
		}
		screen.DrawTriangles(vs, is, whiteImage, op)
	}

	bdR, bdG, bdB, bdA := borderCol.RGBA()
	var outline vector.Path
	outline.MoveTo(float32(left)+radius, float32(top))
	outline.LineTo(float32(right)-radius, float32(top))
	outline.Arc(float32(right)-radius, float32(top)+radius, radius, -math.Pi/2, 0, vector.Clockwise)
	outline.LineTo(float32(right), float32(bottom)-radius)
	outline.Arc(float32(right)-radius, float32(bottom)-radius, radius, 0, math.Pi/2, vector.Clockwise)
	if !far && !noArrow && bubbleType != kBubblePonder {
		outline.LineTo(float32(baseX+tailHalf), float32(bottom))
		outline.LineTo(float32(tailX), float32(tailY))
		outline.LineTo(float32(baseX-tailHalf), float32(bottom))
	}
	outline.LineTo(float32(left)+radius, float32(bottom))
	outline.Arc(float32(left)+radius, float32(bottom)-radius, radius, math.Pi/2, math.Pi, vector.Clockwise)
	outline.LineTo(float32(left), float32(top)+radius)
	outline.Arc(float32(left)+radius, float32(top)+radius, radius, math.Pi, 3*math.Pi/2, vector.Clockwise)
	outline.Close()

	vs, is = outline.AppendVerticesAndIndicesForStroke(nil, nil, &vector.StrokeOptions{Width: float32(gs.GameScale)})
	for i := range vs {
		vs[i].SrcX = 0
		vs[i].SrcY = 0
		vs[i].ColorR = float32(bdR) / 0xffff
		vs[i].ColorG = float32(bdG) / 0xffff
		vs[i].ColorB = float32(bdB) / 0xffff
		vs[i].ColorA = float32(bdA) / 0xffff
	}
	screen.DrawTriangles(vs, is, whiteImage, op)

	if bubbleType == kBubblePonder && !far && !noArrow {
		var tailOutline vector.Path
		r1 := float32(tailHalf)
		cx1 := float32(baseX - tailHalf)
		cy1 := float32(bottom) + r1
		tailOutline.MoveTo(cx1+r1, cy1)
		tailOutline.Arc(cx1, cy1, r1, 0, 2*math.Pi, vector.Clockwise)
		tailOutline.Close()
		r2 := float32(tailHalf) / 2
		cx2 := float32(tailX)
		cy2 := float32(tailY)
		tailOutline.MoveTo(cx2+r2, cy2)
		tailOutline.Arc(cx2, cy2, r2, 0, 2*math.Pi, vector.Clockwise)
		tailOutline.Close()
		vs, is = tailOutline.AppendVerticesAndIndicesForStroke(vs[:0], is[:0], &vector.StrokeOptions{Width: float32(gs.GameScale)})
		for i := range vs {
			vs[i].SrcX = 0
			vs[i].SrcY = 0
			vs[i].ColorR = float32(bdR) / 0xffff
			vs[i].ColorG = float32(bdG) / 0xffff
			vs[i].ColorB = float32(bdB) / 0xffff
			vs[i].ColorA = float32(bdA) / 0xffff
		}
		screen.DrawTriangles(vs, is, whiteImage, op)
	}

	if bubbleType == kBubbleYell {
		drawSpikes(screen, float32(left), float32(top), float32(right), float32(bottom), radius, float32(gs.GameScale*3), borderCol)
	} else if bubbleType == kBubbleMonster {
		drawJagged(screen, float32(left), float32(top), float32(right), float32(bottom), float32(gs.GameScale*3), borderCol)
	}

	textTop := top + pad
	textLeft := left + pad
	for i, line := range lines {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(textLeft), float64(textTop+i*lineHeight))
		op.ColorScale.ScaleWithColor(textCol)
		text.Draw(screen, line, bubbleFont, op)
	}
}

// drawSpikes renders spiky triangles around the bubble rectangle to emphasize
// a shouted yell. Triangles are drawn pointing outward along each edge using
// the given border color. Spikes are positioned to avoid overlapping the
// rounded bubble corners.
func drawSpikes(screen *ebiten.Image, left, top, right, bottom, radius, size float32, col color.Color) {
	bdR, bdG, bdB, bdA := col.RGBA()
	step := size * 2
	op := &ebiten.DrawTrianglesOptions{ColorScaleMode: ebiten.ColorScaleModePremultipliedAlpha, AntiAlias: true}

	startX := left + radius
	endX := right - radius
	width := endX - startX
	count := int(width / step)
	offsetX := (width - float32(count)*step) / 2
	// top and bottom edges
	for x := startX + offsetX; x+step <= endX; x += step {
		var p vector.Path
		p.MoveTo(x, top)
		p.LineTo(x+size, top-size)
		p.LineTo(x+step, top)
		p.Close()
		vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
		for i := range vs {
			vs[i].SrcX = 0
			vs[i].SrcY = 0
			vs[i].ColorR = float32(bdR) / 0xffff
			vs[i].ColorG = float32(bdG) / 0xffff
			vs[i].ColorB = float32(bdB) / 0xffff
			vs[i].ColorA = float32(bdA) / 0xffff
		}
		screen.DrawTriangles(vs, is, whiteImage, op)

		p.Reset()
		p.MoveTo(x, bottom)
		p.LineTo(x+size, bottom+size)
		p.LineTo(x+step, bottom)
		p.Close()
		vs, is = p.AppendVerticesAndIndicesForFilling(nil, nil)
		for i := range vs {
			vs[i].SrcX = 0
			vs[i].SrcY = 0
			vs[i].ColorR = float32(bdR) / 0xffff
			vs[i].ColorG = float32(bdG) / 0xffff
			vs[i].ColorB = float32(bdB) / 0xffff
			vs[i].ColorA = float32(bdA) / 0xffff
		}
		screen.DrawTriangles(vs, is, whiteImage, op)
	}

	startY := top + radius
	endY := bottom - radius
	height := endY - startY
	count = int(height / step)
	offsetY := (height - float32(count)*step) / 2
	// left and right edges
	for y := startY + offsetY; y+step <= endY; y += step {
		var p vector.Path
		p.MoveTo(left, y)
		p.LineTo(left-size, y+size)
		p.LineTo(left, y+step)
		p.Close()
		vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
		for i := range vs {
			vs[i].SrcX = 0
			vs[i].SrcY = 0
			vs[i].ColorR = float32(bdR) / 0xffff
			vs[i].ColorG = float32(bdG) / 0xffff
			vs[i].ColorB = float32(bdB) / 0xffff
			vs[i].ColorA = float32(bdA) / 0xffff
		}
		screen.DrawTriangles(vs, is, whiteImage, op)

		p.Reset()
		p.MoveTo(right, y)
		p.LineTo(right+size, y+size)
		p.LineTo(right, y+step)
		p.Close()
		vs, is = p.AppendVerticesAndIndicesForFilling(nil, nil)
		for i := range vs {
			vs[i].SrcX = 0
			vs[i].SrcY = 0
			vs[i].ColorR = float32(bdR) / 0xffff
			vs[i].ColorG = float32(bdG) / 0xffff
			vs[i].ColorB = float32(bdB) / 0xffff
			vs[i].ColorA = float32(bdA) / 0xffff
		}
		screen.DrawTriangles(vs, is, whiteImage, op)
	}
}

// drawJagged creates alternating in/out triangles around the bubble rectangle
// to simulate torn fabric edges for monster speech bubbles.
func drawJagged(screen *ebiten.Image, left, top, right, bottom, size float32, col color.Color) {
	bdR, bdG, bdB, bdA := col.RGBA()
	step := size
	op := &ebiten.DrawTrianglesOptions{ColorScaleMode: ebiten.ColorScaleModePremultipliedAlpha, AntiAlias: true}
	toggle := false
	for x := left; x < right-step; x += step {
		var p vector.Path
		p.MoveTo(x, top)
		if toggle {
			p.LineTo(x+step/2, top+size)
		} else {
			p.LineTo(x+step/2, top-size)
		}
		p.LineTo(x+step, top)
		p.Close()
		vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
		for i := range vs {
			vs[i].SrcX = 0
			vs[i].SrcY = 0
			vs[i].ColorR = float32(bdR) / 0xffff
			vs[i].ColorG = float32(bdG) / 0xffff
			vs[i].ColorB = float32(bdB) / 0xffff
			vs[i].ColorA = float32(bdA) / 0xffff
		}
		screen.DrawTriangles(vs, is, whiteImage, op)

		p.Reset()
		p.MoveTo(x, bottom)
		if toggle {
			p.LineTo(x+step/2, bottom-size)
		} else {
			p.LineTo(x+step/2, bottom+size)
		}
		p.LineTo(x+step, bottom)
		p.Close()
		vs, is = p.AppendVerticesAndIndicesForFilling(nil, nil)
		for i := range vs {
			vs[i].SrcX = 0
			vs[i].SrcY = 0
			vs[i].ColorR = float32(bdR) / 0xffff
			vs[i].ColorG = float32(bdG) / 0xffff
			vs[i].ColorB = float32(bdB) / 0xffff
			vs[i].ColorA = float32(bdA) / 0xffff
		}
		screen.DrawTriangles(vs, is, whiteImage, op)

		toggle = !toggle
	}

	toggle = false
	for y := top; y < bottom-step; y += step {
		var p vector.Path
		p.MoveTo(left, y)
		if toggle {
			p.LineTo(left+size, y+step/2)
		} else {
			p.LineTo(left-size, y+step/2)
		}
		p.LineTo(left, y+step)
		p.Close()
		vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
		for i := range vs {
			vs[i].SrcX = 0
			vs[i].SrcY = 0
			vs[i].ColorR = float32(bdR) / 0xffff
			vs[i].ColorG = float32(bdG) / 0xffff
			vs[i].ColorB = float32(bdB) / 0xffff
			vs[i].ColorA = float32(bdA) / 0xffff
		}
		screen.DrawTriangles(vs, is, whiteImage, op)

		p.Reset()
		p.MoveTo(right, y)
		if toggle {
			p.LineTo(right-size, y+step/2)
		} else {
			p.LineTo(right+size, y+step/2)
		}
		p.LineTo(right, y+step)
		p.Close()
		vs, is = p.AppendVerticesAndIndicesForFilling(nil, nil)
		for i := range vs {
			vs[i].SrcX = 0
			vs[i].SrcY = 0
			vs[i].ColorR = float32(bdR) / 0xffff
			vs[i].ColorG = float32(bdG) / 0xffff
			vs[i].ColorB = float32(bdB) / 0xffff
			vs[i].ColorA = float32(bdA) / 0xffff
		}
		screen.DrawTriangles(vs, is, whiteImage, op)

		toggle = !toggle
	}
}
