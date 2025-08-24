package main

import (
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"gothoom/eui"
)

type thinkMessage struct {
	item   *eui.ItemData
	img    *ebiten.Image
	text   string
	expiry time.Time
}

var thinkMessages []*thinkMessage

// showThinkMessage displays a temporary think message at the top of the screen.
// msg should already include the sender's name.
func showThinkMessage(msg string) {
	if gameWin == nil {
		return
	}
	btn, events := eui.NewButton()
	btn.Filled = false
	btn.Outlined = false
	btn.Padding = 0
	btn.BorderPad = 0
	btn.Margin = 0
	btn.Text = ""

	sw := int(float64(gameAreaSizeX) * gs.GameScale)
	pad := int((4 + 2) * gs.GameScale)
	tailHeight := int(10 * gs.GameScale)
	maxLineWidth := sw/4 - 2*pad
	font := bubbleFont
	width, lines := wrapText(msg, font, float64(maxLineWidth))
	metrics := font.Metrics()
	lineHeight := int(math.Ceil(metrics.HAscent) + math.Ceil(metrics.HDescent) + math.Ceil(metrics.HLineGap))
	width += 2 * pad
	height := lineHeight*len(lines) + 2*pad

	img := ebiten.NewImage(width, height+tailHeight)
	btn.Image = img
	btn.Size = eui.Point{X: float32(width) / eui.UIScale(), Y: float32(height+tailHeight) / eui.UIScale()}

	events.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			removeThinkMessage(btn)
		}
	}

	dur := time.Duration(gs.NotificationDuration * float64(time.Second))
	if dur <= 0 {
		dur = 6 * time.Second
	}
	m := &thinkMessage{item: btn, img: img, text: msg, expiry: time.Now().Add(dur)}
	thinkMessages = append(thinkMessages, m)
	renderThinkMessage(m)
	gameWin.AddItem(btn)
	layoutThinkMessages()
}

func renderThinkMessage(m *thinkMessage) {
	if m == nil || m.img == nil {
		return
	}
	m.img.Clear()
	borderCol, bgCol, textCol := bubbleColors(kBubbleThought)
	w := m.img.Bounds().Dx()
	drawBubble(m.img, m.text, w/2, 0, kBubbleThought, false, false, borderCol, bgCol, textCol)
}

func removeThinkMessage(item *eui.ItemData) {
	for i, m := range thinkMessages {
		if m.item == item {
			thinkMessages = append(thinkMessages[:i], thinkMessages[i+1:]...)
			break
		}
	}
	if gameWin != nil {
		for i, it := range gameWin.Contents {
			if it == item {
				gameWin.Contents = append(gameWin.Contents[:i], gameWin.Contents[i+1:]...)
				break
			}
		}
		gameWin.Refresh()
	}
}

func layoutThinkMessages() {
	if gameWin == nil {
		return
	}
	margin := float32(8)
	spacer := float32(4)
	x := margin
	y := margin
	rowHeight := float32(0)
	scale := eui.UIScale()
	if gameWin.NoScale {
		scale = 1
	}
	winSize := gameWin.GetSize()
	for _, m := range thinkMessages {
		it := m.item
		sz := it.GetSize()
		if x+sz.X > winSize.X-margin {
			x = margin
			y += rowHeight + spacer
			rowHeight = 0
		}
		it.Position = eui.Point{X: x / scale, Y: y / scale}
		x += sz.X + spacer
		if sz.Y > rowHeight {
			rowHeight = sz.Y
		}
		it.Dirty = true
	}
	gameWin.Refresh()
}

func updateThinkMessages() {
	if len(thinkMessages) == 0 {
		return
	}
	now := time.Now()
	changed := false
	for i := 0; i < len(thinkMessages); {
		m := thinkMessages[i]
		renderThinkMessage(m)
		if m.item != nil {
			m.item.Dirty = true
		}
		if gameWin != nil {
			gameWin.Dirty = true
		}
		if now.After(m.expiry) {
			removeThinkMessage(m.item)
			changed = true
		} else {
			i++
		}
	}
	if changed {
		layoutThinkMessages()
	}
}

func clearThinkMessages() {
	for _, m := range thinkMessages {
		if gameWin != nil {
			for i, it := range gameWin.Contents {
				if it == m.item {
					gameWin.Contents = append(gameWin.Contents[:i], gameWin.Contents[i+1:]...)
					break
				}
			}
		}
	}
	thinkMessages = nil
	if gameWin != nil {
		gameWin.Refresh()
	}
}
