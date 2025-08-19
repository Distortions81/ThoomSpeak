package main

import "testing"

func TestScrambleHashBlank(t *testing.T) {
	if s := scrambleHash("name", ""); s != "" {
		t.Fatalf("scrambleHash on blank hash = %q, want empty", s)
	}
	if s := unscrambleHash("name", ""); s != "" {
		t.Fatalf("unscrambleHash on blank hash = %q, want empty", s)
	}
}

func TestScrambleHashBlankString(t *testing.T) {
	if s := scrambleHash("", ""); s != "" {
		t.Fatalf("scrambleHash on blank name and hash = %q, want empty", s)
	}
	if s := unscrambleHash("", ""); s != "" {
		t.Fatalf("unscrambleHash on blank name and hash = %q, want empty", s)
	}
}

func TestScrambleHashRoundTrip(t *testing.T) {
	const name = "char"
	const hash = "0123456789abcdef0123456789abcdef"
	enc := scrambleHash(name, hash)
	if enc == hash {
		t.Fatalf("scrambleHash(%q, %q) = %q, expected different", name, hash, enc)
	}
	dec := unscrambleHash(name, enc)
	if dec != hash {
		t.Fatalf("unscrambleHash returned %q, want %q", dec, hash)
	}
}
