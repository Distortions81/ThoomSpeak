package main

import "testing"

func TestParseThinkText(t *testing.T) {
	tests := []struct {
		name       string
		raw        []byte
		text       string
		wantName   string
		wantTarget thinkTarget
		wantMsg    string
	}{
		{
			name: "BEPP to you",
			raw: func() []byte {
				b := []byte("Torx")
				b = append(b, []byte{0xC2, 't', '_', 't', 't'}...)
				b = append(b, []byte(" to you: hello")...)
				return b
			}(),
			text:       "Torx to you: hello",
			wantName:   "Torx",
			wantTarget: thinkToYou,
			wantMsg:    "hello",
		},
		{
			name:       "suffix to you",
			raw:        []byte("Torx to you: hi"),
			text:       "Torx to you: hi",
			wantName:   "Torx",
			wantTarget: thinkToYou,
			wantMsg:    "hi",
		},
		{
			name:       "suffix to your clan",
			raw:        []byte("Torx to your clan: hi"),
			text:       "Torx to your clan: hi",
			wantName:   "Torx",
			wantTarget: thinkToClan,
			wantMsg:    "hi",
		},
		{
			name:       "suffix to a group",
			raw:        []byte("Torx to a group: hi"),
			text:       "Torx to a group: hi",
			wantName:   "Torx",
			wantTarget: thinkToGroup,
			wantMsg:    "hi",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotTarget, gotMsg := parseThinkText(tt.raw, tt.text)
			if gotName != tt.wantName {
				t.Errorf("name = %q, want %q", gotName, tt.wantName)
			}
			if gotTarget != tt.wantTarget {
				t.Errorf("target = %v, want %v", gotTarget, tt.wantTarget)
			}
			if gotMsg != tt.wantMsg {
				t.Errorf("msg = %q, want %q", gotMsg, tt.wantMsg)
			}
		})
	}
}

func TestParseThinkTextUnknownName(t *testing.T) {
	raw := []byte("someone: hi")
	name, _, _ := parseThinkText(raw, "someone: hi")
	if name != ThinkUnknownName {
		t.Fatalf("name = %q, want %q", name, ThinkUnknownName)
	}
}
