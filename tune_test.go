package main

import (
	"testing"
	"time"
)

// Stubs to satisfy references in tune.go when building this test.
type Note struct {
	Key      int
	Velocity int
	Start    time.Duration
	Duration time.Duration
}

var audioContext interface{}
var gs struct {
	Mute        bool
	MusicVolume int
}
var musicDebug bool

func Play(interface{}, int, []Note) error { return nil }
func consoleMessage(string)               {}
func chatMessage(string)                  {}
func stopAllMusic()                       {}

func TestParseClanLordTuneDurations(t *testing.T) {
	tests := []struct {
		input string
		want  []int
	}{
		{"c", []int{1000}},     // lowercase uses durationBlack=2 beats
		{"C", []int{2000}},     // uppercase uses durationWhite=4 beats
		{"c1", []int{500}},     // explicit duration 1 beat
		{"p", []int{1000}},     // rest defaults to durationBlack
		{"[ce]", []int{2000}},  // chord defaults to defaultChordDuration
		{"[ce]3", []int{1500}}, // chord with explicit duration
	}
	for _, tt := range tests {
		events := parseClanLordTuneWithTempo(tt.input, 120)
		if len(events) != len(tt.want) {
			t.Fatalf("%q parsed to %d events, want %d", tt.input, len(events), len(tt.want))
		}
		for i, ev := range events {
			if ev.durMS != tt.want[i] {
				t.Errorf("%q event %d duration = %d, want %d", tt.input, i, ev.durMS, tt.want[i])
			}
		}
	}
}
