package main

import "testing"

func TestIsSelfChatMessage(t *testing.T) {
	playerName = "Hero"
	cases := []struct {
		msg  string
		want bool
	}{
		{"Hero says, hello there", true},
		{"(Hero waves)", true},
		{"Hero yells, hey!", true},
		{"Bob says, hi", false},
		{"You are sharing experiences with Bob.", false},
		{"Hero has fallen", false},
	}
	for _, c := range cases {
		if got := isSelfChatMessage(c.msg); got != c.want {
			t.Errorf("isSelfChatMessage(%q) = %v; want %v", c.msg, got, c.want)
		}
	}
}

func TestChatMessageBlocked(t *testing.T) {
	players = make(map[string]*Player)
	chatLog = messageLog{max: maxChatMessages}
	p := getPlayer("Bob")
	playersMu.Lock()
	p.Blocked = true
	playersMu.Unlock()
	chatMessage("Bob says, hi")
	if len(getChatMessages()) != 0 {
		t.Fatalf("expected no messages")
	}
}

func TestChatMessageIgnored(t *testing.T) {
	players = make(map[string]*Player)
	chatLog = messageLog{max: maxChatMessages}
	p := getPlayer("Bob")
	playersMu.Lock()
	p.Ignored = true
	playersMu.Unlock()
	chatMessage("Bob says, hi")
	if len(getChatMessages()) != 0 {
		t.Fatalf("expected no messages")
	}
}
