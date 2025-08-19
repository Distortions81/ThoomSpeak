package main

import (
	"gothoom/eui"
	"testing"
)

func TestPointInUIOverlay(t *testing.T) {
	gameWin = eui.NewWindow()
	gameWin.MarkOpen()
	btn, _ := eui.NewButton()
	btn.DrawRect.X0 = 10
	btn.DrawRect.Y0 = 10
	btn.DrawRect.X1 = 20
	btn.DrawRect.Y1 = 20
	gameWin.AddItem(btn)
	if !pointInUI(15, 15) {
		t.Fatalf("pointInUI should detect overlay item")
	}
}
