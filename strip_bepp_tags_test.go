package main

import "testing"

func TestStripBEPPTagsPaired(t *testing.T) {
	raw := []byte{0xC2, 'p', 'n'}
	raw = append(raw, []byte("Bob")...)
	raw = append(raw, 0xC2, 'p', 'n')
	raw = append(raw, []byte(" hi")...)
	got := string(stripBEPPTags(raw))
	want := "Bob hi"
	if got != want {
		t.Fatalf("stripBEPPTags returned %q, want %q", got, want)
	}
}

func TestStripBEPPTagsUnterminated(t *testing.T) {
	raw := []byte{0xC2, 'p', 'n'}
	raw = append(raw, []byte("Bob hi")...)
	got := string(stripBEPPTags(raw))
	want := "Bob hi"
	if got != want {
		t.Fatalf("stripBEPPTags returned %q, want %q", got, want)
	}
}
