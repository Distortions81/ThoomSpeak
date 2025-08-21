package main

import "testing"

func TestParseMusicCommandWithWho(t *testing.T) {
	// Ensure /music commands with a leading /who segment are parsed.
	if !parseMusicCommand("/music/who123/play/inst2/notesabc", nil) {
		t.Fatalf("parseMusicCommand failed to parse /music with /who prefix")
	}
}

func TestParseMusicCommandRawFallback(t *testing.T) {
	if !parseMusicCommand("", []byte("/music/play/inst1/notesabc")) {
		t.Fatalf("parseMusicCommand failed to parse raw payload")
	}
}
