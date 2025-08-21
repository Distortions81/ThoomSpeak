package main

import (
	"time"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"gothoom/eui"
)

type thinkMessage struct {
	item   *eui.ItemData
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
	btn.Text = msg
	btn.FontSize = float32(gs.ChatFontSize)
	btn.Filled = true
	btn.Outlined = false
	btn.Color = eui.NewColor(0, 0, 0, 160)
	btn.TextColor = eui.NewColor(255, 255, 255, 255)
	btn.HoverColor = btn.Color
	btn.ClickColor = btn.Color
	btn.Fillet = 6
	btn.Padding = 4
	btn.Margin = 0

	textSize := (btn.FontSize * eui.UIScale()) + 2
	face := &text.GoTextFace{Source: eui.FontSource(), Size: float64(textSize)}
	w, h := text.Measure(msg, face, 0)
	btn.Size = eui.Point{
		X: float32(w)/eui.UIScale() + btn.Padding*2 + btn.BorderPad*2,
		Y: float32(h)/eui.UIScale() + btn.Padding*2 + btn.BorderPad*2,
	}

	events.Handle = func(ev eui.UIEvent) {
		if ev.Type == eui.EventClick {
			removeThinkMessage(btn)
		}
	}

	dur := time.Duration(gs.NotificationDuration * float64(time.Second))
	if dur <= 0 {
		dur = 6 * time.Second
	}
	thinkMessages = append(thinkMessages, &thinkMessage{item: btn, expiry: time.Now().Add(dur)})
	gameWin.AddItem(btn)
	layoutThinkMessages()
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
	scale := eui.UIScale()
	if gameWin.NoScale {
		scale = 1
	}
	for _, m := range thinkMessages {
		it := m.item
		sz := it.GetSize()
		it.Position = eui.Point{X: x / scale, Y: y / scale}
		x += sz.X + spacer
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
		if now.After(thinkMessages[i].expiry) {
			removeThinkMessage(thinkMessages[i].item)
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
