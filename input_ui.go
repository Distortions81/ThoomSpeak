package main

import (
	"gothoom/eui"
)

// pointInUI reports whether the given screen coordinate lies within any EUI window or overlay.
func pointInUI(x, y int) bool {
	fx, fy := float32(x), float32(y)

	windows := eui.Windows()
	for _, win := range windows {
		if win == gameWin {
			continue
		}
		if !win.IsOpen() {
			continue
		}
		pos := win.GetPos()
		size := win.GetSize()
		s := eui.UIScale()
		frame := (win.Margin + win.Border + win.BorderPad + win.Padding) * s
		title := win.GetTitleSize()
		x0, y0 := pos.X+1, pos.Y+1
		x1 := x0 + size.X + frame*2
		y1 := y0 + size.Y + frame*2 + title
		if fx >= x0 && fx < x1 && fy >= y0 && fy < y1 {
			return true
		}
	}

	if gameWin != nil && gameWin.IsOpen() {
		if pointInItems(gameWin.Contents, fx, fy) {
			return true
		}
	}

	return false
}

func pointInItems(items []*eui.ItemData, fx, fy float32) bool {
	for _, it := range items {
		if it == nil || it.Invisible {
			continue
		}
		if fx >= it.DrawRect.X0 && fx < it.DrawRect.X1 && fy >= it.DrawRect.Y0 && fy < it.DrawRect.Y1 {
			return true
		}
		if len(it.Contents) > 0 && pointInItems(it.Contents, fx, fy) {
			return true
		}
		if len(it.Tabs) > 0 {
			for _, tab := range it.Tabs {
				if pointInItems(tab.Contents, fx, fy) {
					return true
				}
			}
		}
	}
	return false
}
