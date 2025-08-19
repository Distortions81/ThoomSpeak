package main

import "testing"

// helper to wrap message into BEPP prefix without pn tag
func fallenLine(prefix, msg string) []byte {
	b := []byte{0xC2, prefix[0], prefix[1]}
	b = append(b, []byte(msg)...)
	b = append(b, 0) // simulate NUL terminator
	return b
}

func TestDecodeFallenWithoutTag(t *testing.T) {
	players = make(map[string]*Player)
	raw := fallenLine("hf", "Bob has fallen")
	if got := decodeBEPP(raw); got != "Bob has fallen" {
		t.Fatalf("decodeBEPP returned %q", got)
	}
	playersMu.RLock()
	dead := players["Bob"].Dead
	playersMu.RUnlock()
	if !dead {
		t.Errorf("player not marked dead")
	}
}

func TestDecodeUnfallenWithoutTag(t *testing.T) {
	players = make(map[string]*Player)
	players["Bob"] = &Player{Name: "Bob", Dead: true}
	raw := fallenLine("nf", "Bob is no longer fallen")
	if got := decodeBEPP(raw); got != "Bob is no longer fallen" {
		t.Fatalf("decodeBEPP returned %q", got)
	}
	playersMu.RLock()
	dead := players["Bob"].Dead
	playersMu.RUnlock()
	if dead {
		t.Errorf("player still marked dead")
	}
}
