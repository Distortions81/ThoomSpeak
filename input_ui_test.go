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

func TestPointInGameWindow(t *testing.T) {
	gameWin = eui.NewWindow()
	gameWin.MarkOpen()
	_ = gameWin.SetPos(eui.Point{X: 10, Y: 10})
	_ = gameWin.SetSize(eui.Point{X: 100, Y: 100})
	gameWin.Margin = 0
	gameWin.Border = 0
	gameWin.BorderPad = 0
	gameWin.Padding = 0
	gameWin.TitleHeight = 0

	if !pointInGameWindow(50, 50) {
		t.Fatalf("pointInGameWindow should detect interior point")
	}
	if pointInGameWindow(5, 5) {
		t.Fatalf("pointInGameWindow should ignore exterior point")
	}
}
