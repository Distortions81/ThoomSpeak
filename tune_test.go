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

func TestEventsToNotesAddsGap(t *testing.T) {
	events := parseClanLordTune("cd")
	inst := instrument{program: 0, octave: 0, chord: 100, melody: 100}
	notes := eventsToNotes(events, inst, 100)
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	if notes[0].Duration != 900*time.Millisecond {
		t.Fatalf("first note duration = %v, want 900ms", notes[0].Duration)
	}
	if notes[1].Start != 1000*time.Millisecond {
		t.Fatalf("second note start = %v, want 1000ms", notes[1].Start)
	}
	gap := notes[1].Start - notes[0].Start - notes[0].Duration
	if gap != 100*time.Millisecond {
		t.Fatalf("gap = %v, want 100ms", gap)
	}
}

func TestInstrumentVelocityFactors(t *testing.T) {
	inst0 := instruments[0]
	if inst0.chord != 100 || inst0.melody != 100 {
		t.Fatalf("instrument 0 velocities = %d,%d; want 100,100", inst0.chord, inst0.melody)
	}
	inst1 := instruments[1]
	if inst1.chord != 100 || inst1.melody != 100 {
		t.Fatalf("instrument 1 velocities = %d,%d; want 100,100", inst1.chord, inst1.melody)
	}
}

func TestEventsToNotesVelocityFactors(t *testing.T) {
	inst := instrument{program: 0, octave: 0, chord: 50, melody: 100}
	events := []noteEvent{
		{keys: []int{60, 64}, durMS: 1000},
		{keys: []int{67}, durMS: 1000},
	}
	notes := eventsToNotes(events, inst, 100)
	if len(notes) != 3 {
		t.Fatalf("expected 3 notes, got %d", len(notes))
	}
	if notes[0].Velocity != 50 || notes[1].Velocity != 50 {
		t.Fatalf("chord note velocities = %d,%d; want 50", notes[0].Velocity, notes[1].Velocity)
	}
	if notes[2].Velocity != 100 {
		t.Fatalf("melody note velocity = %d; want 100", notes[2].Velocity)
	}
}
