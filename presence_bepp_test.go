package main

import "testing"

// helper to build BEPP line with lg prefix
func presenceLine(msg []byte) []byte {
	b := []byte{0xC2, 'l', 'g'}
	b = append(b, msg...)
	b = append(b, 0)
	return b
}

func TestDecodeLoginBEPP(t *testing.T) {
	players = make(map[string]*Player)
	players["Bob"] = &Player{Name: "Bob", Offline: true}
	msg := append(pnTag("Bob"), []byte(" has logged on")...)
	raw := presenceLine(msg)
	if got := decodeBEPP(raw); got != "Bob has logged on" {
		t.Fatalf("decodeBEPP returned %q", got)
	}
	playersMu.RLock()
	offline := players["Bob"].Offline
	playersMu.RUnlock()
	if offline {
		t.Errorf("player still offline")
	}
}

func TestDecodeLogoutBEPP(t *testing.T) {
	players = make(map[string]*Player)
	players["Bob"] = &Player{Name: "Bob", Offline: false}
	msg := append(pnTag("Bob"), []byte(" has left the lands")...)
	raw := presenceLine(msg)
	if got := decodeBEPP(raw); got != "Bob has left the lands" {
		t.Fatalf("decodeBEPP returned %q", got)
	}
	playersMu.RLock()
	offline := players["Bob"].Offline
	playersMu.RUnlock()
	if !offline {
		t.Errorf("player not marked offline")
	}
}
