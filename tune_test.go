package main

import (
	"testing"
	"time"
)

func TestParseClanLordTuneDurations(t *testing.T) {
	tests := []struct {
		input string
		want  []int
	}{
		{"c", []int{250}},     // lowercase uses durationBlack=2 half-beats
		{"C", []int{500}},     // uppercase uses durationWhite=4 half-beats
		{"c1", []int{125}},    // explicit duration 1 half-beat
		{"p", []int{250}},     // rest defaults to durationBlack
		{"[ce]", []int{250}},  // chord defaults to defaultChordDuration
		{"[ce]3", []int{375}}, // chord with explicit duration
	}
	for _, tt := range tests {
		pt := parseClanLordTuneWithTempo(tt.input, 120)
		if len(pt.events) != len(tt.want) {
			t.Fatalf("%q parsed to %d events, want %d", tt.input, len(pt.events), len(tt.want))
		}
		quarter := 60000 / 120
		for i, ev := range pt.events {
			got := int((ev.beats / 4) * float64(quarter))
			if got != tt.want[i] {
				t.Errorf("%q event %d duration = %d, want %d", tt.input, i, got, tt.want[i])
			}
		}
	}
}

func TestEventsToNotesNoGap(t *testing.T) {
	pt := parseClanLordTune("cd")
	inst := instrument{program: 0, octave: 0, chord: 100, melody: 100}
	notes := eventsToNotes(pt, inst, 100)
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	if notes[0].Duration != 250*time.Millisecond {
		t.Fatalf("first note duration = %v, want 250ms", notes[0].Duration)
	}
	if notes[1].Start != 250*time.Millisecond {
		t.Fatalf("second note start = %v, want 250ms", notes[1].Start)
	}
	gap := notes[1].Start - notes[0].Start - notes[0].Duration
	if gap != 0 {
		t.Fatalf("gap = %v, want 0", gap)
	}
}

func TestRestDuration(t *testing.T) {
	pt := parseClanLordTune("cpd")
	inst := instrument{program: 0, octave: 0, chord: 100, melody: 100}
	notes := eventsToNotes(pt, inst, 100)
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	if notes[1].Start != 500*time.Millisecond {
		t.Fatalf("second note start = %v, want 500ms", notes[1].Start)
	}
	gap := notes[1].Start - notes[0].Start - notes[0].Duration
	if gap != 250*time.Millisecond {
		t.Fatalf("gap = %v, want 250ms", gap)
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
	pt := parsedTune{
		events: []noteEvent{
			{keys: []int{60, 64}, beats: 2, volume: 10},
			{keys: []int{67}, beats: 2, volume: 10},
		},
		tempo: 120,
	}
	notes := eventsToNotes(pt, inst, 100)
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

func TestLoopAndTempoAndVolume(t *testing.T) {
	// Loop: (cd)2 should produce 4 notes, then tempo change and volume change.
	pt := parseClanLordTuneWithTempo("(cd)2@+60e%5f", 120)
	inst := instrument{program: 0, octave: 0, chord: 100, melody: 100}
	notes := eventsToNotes(pt, inst, 100)
	if len(notes) != 6 {
		t.Fatalf("expected 6 notes, got %d", len(notes))
	}
	// After tempo change to 180 BPM, note 'e' should have shorter duration.
	if notes[4].Duration != 166*time.Millisecond {
		t.Fatalf("tempo change not applied, got %v", notes[4].Duration)
	}
	// volume change should reduce velocity per square-root scaling (volume set to 5)
	if notes[5].Velocity != 71 {
		t.Fatalf("volume change not applied, got %d", notes[5].Velocity)
	}
}

// TestEventsToNotesLoop verifies that repeated loops terminate properly and
// produce the expected number of notes without getting stuck.
func TestEventsToNotesLoop(t *testing.T) {
	pt := parseClanLordTune("(c)2")
	inst := instrument{chord: 100, melody: 100}
	notes := eventsToNotes(pt, inst, 100)
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
}

// TestLoopSeamlessRepeat ensures that looping a sequence starting with a note
// does not introduce an extra rest between iterations when no explicit rest is
// present at the loop boundary.
func TestLoopSeamlessRepeat(t *testing.T) {
	pt := parseClanLordTune("(c)2")
	inst := instrument{program: 0, octave: 0, chord: 100, melody: 100}
	notes := eventsToNotes(pt, inst, 100)
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	gap := notes[1].Start - notes[0].Start - notes[0].Duration
	if gap != 0 {
		t.Fatalf("gap between loop iterations = %v, want 0", gap)
	}
}

func TestNoteDurationsWithTempoChange(t *testing.T) {
	tune := "c d1 @+60 E g2"
	pt := parseClanLordTuneWithTempo(tune, 120)
	inst := instrument{program: 0, octave: 0, chord: 100, melody: 100}
	notes := eventsToNotes(pt, inst, 100)
	if len(notes) != 4 {
		t.Fatalf("expected 4 notes, got %d", len(notes))
	}
	want := []time.Duration{
		250 * time.Millisecond, // c: 1 beat at 120 BPM
		125 * time.Millisecond, // d1: half beat at 120 BPM
		333 * time.Millisecond, // E: 2 beats at 180 BPM
		166 * time.Millisecond, // g2: 1 beat at 180 BPM
	}
	for i, n := range notes {
		if n.Duration != want[i] {
			t.Errorf("note %d duration = %v, want %v", i, n.Duration, want[i])
		}
	}
}

func TestNoteDurationsUncommonTempos(t *testing.T) {
	inst := instrument{program: 0, octave: 0, chord: 100, melody: 100}
	cases := []struct {
		tempo int
		want  time.Duration
	}{
		{95, 473 * time.Millisecond},
		{177, 254 * time.Millisecond},
	}
	for _, c := range cases {
		pt := parseClanLordTuneWithTempo("c3", c.tempo)
		notes := eventsToNotes(pt, inst, 100)
		if len(notes) != 1 {
			t.Fatalf("tempo %d: expected 1 note, got %d", c.tempo, len(notes))
		}
		if notes[0].Duration != c.want {
			t.Errorf("tempo %d: duration = %v, want %v", c.tempo, notes[0].Duration, c.want)
		}
	}
}

func TestParseNoteLowestCPreserved(t *testing.T) {
	pt := parseClanLordTune("\\----c")
	if len(pt.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(pt.events))
	}
	if len(pt.events[0].keys) != 1 || pt.events[0].keys[0] != 0 {
		t.Fatalf("expected low C (0), got %+v", pt.events[0].keys)
	}
}
