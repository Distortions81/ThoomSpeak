package main

import "testing"

func TestMacRomanEncodeDecode(t *testing.T) {
	s := "Méme"
	b := encodeMacRoman(s)
	got := decodeMacRoman(b)
	if got != s {
		t.Fatalf("round-trip = %q, want %q", got, s)
	}
}
