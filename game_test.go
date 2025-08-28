package main

import "testing"

func TestStopWalkIfOutside(t *testing.T) {
	old := gs.ClickToToggle
	gs.ClickToToggle = true
	walkToggled = true
	stopWalkIfOutside(true, false)
	if walkToggled {
		t.Fatalf("walkToggled should be false after outside click")
	}

	walkToggled = true
	stopWalkIfOutside(true, true)
	if !walkToggled {
		t.Fatalf("walkToggled should remain true when clicking inside game")
	}

	walkToggled = true
	stopWalkIfOutside(false, false)
	if !walkToggled {
		t.Fatalf("walkToggled should remain true when not clicking")
	}

	gs.ClickToToggle = old
}

func TestContinueHeldWalk(t *testing.T) {
	prev := inputState{mouseDown: true}
	if !continueHeldWalk(prev, false, true, 0, false) {
		t.Fatalf("walk should continue when mouse is held outside")
	}
	if continueHeldWalk(prev, false, false, 0, false) {
		t.Fatalf("walk should stop when mouse button is released")
	}
	if !continueHeldWalk(inputState{}, true, true, 2, false) {
		t.Fatalf("walk should start when mouse is held inside game")
	}
}
