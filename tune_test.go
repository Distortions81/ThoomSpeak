package main

import (
	"math"
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
		pt, err := parseClanLordTuneWithTempo(tt.input, 120)
		if err != nil {
			t.Fatalf("parse error for %q: %v", tt.input, err)
		}
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

func TestEventsToNotesDefaultGap(t *testing.T) {
	pt := parseClanLordTune("cd")
	inst := instrument{program: 0, octave: 0, chord: 100, melody: 100}
	notes := eventsToNotes(pt, inst, 100)
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	gap := time.Duration(int(math.Round(1500.0/120.0))) * time.Millisecond
	wantDur := 250*time.Millisecond - gap
	if notes[0].Duration != wantDur {
		t.Fatalf("first note duration = %v, want %v", notes[0].Duration, wantDur)
	}
	if notes[1].Start != 250*time.Millisecond {
		t.Fatalf("second note start = %v, want 250ms", notes[1].Start)
	}
	gotGap := notes[1].Start - notes[0].Start - notes[0].Duration
	if gotGap != gap {
		t.Fatalf("gap = %v, want %v", gotGap, gap)
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
	gapUnit := time.Duration(int(math.Round(1500.0/120.0))) * time.Millisecond
	gotGap := notes[1].Start - notes[0].Start - notes[0].Duration
	wantGap := gapUnit + 250*time.Millisecond
	if gotGap != wantGap {
		t.Fatalf("gap = %v, want %v", gotGap, wantGap)
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
	pt, err := parseClanLordTuneWithTempo("(cd)2@+60e%5f", 120)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	inst := instrument{program: 0, octave: 0, chord: 100, melody: 100}
	notes := eventsToNotes(pt, inst, 100)
	if len(notes) != 6 {
		t.Fatalf("expected 6 notes, got %d", len(notes))
	}
	// After tempo change to 180 BPM, note 'e' should have shorter duration with gap applied.
	gap := time.Duration(int(math.Round(1500.0/180.0))) * time.Millisecond
	wantDur := 166*time.Millisecond - gap
	if notes[4].Duration != wantDur {
		t.Fatalf("tempo change not applied, got %v want %v", notes[4].Duration, wantDur)
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
	wantGap := time.Duration(int(math.Round(1500.0/120.0))) * time.Millisecond
	if gap != wantGap {
		t.Fatalf("gap between loop iterations = %v, want %v", gap, wantGap)
	}
}

func TestNoteDurationsWithTempoChange(t *testing.T) {
	tune := "c d1 @+60 E g2"
	pt, err := parseClanLordTuneWithTempo(tune, 120)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	inst := instrument{program: 0, octave: 0, chord: 100, melody: 100}
	notes := eventsToNotes(pt, inst, 100)
	if len(notes) != 4 {
		t.Fatalf("expected 4 notes, got %d", len(notes))
	}
	gap120 := time.Duration(int(math.Round(1500.0/120.0))) * time.Millisecond
	gap180 := time.Duration(int(math.Round(1500.0/180.0))) * time.Millisecond
	want := []time.Duration{
		250*time.Millisecond - gap120, // c: 1 beat at 120 BPM
		125*time.Millisecond - gap120, // d1: half beat at 120 BPM
		333*time.Millisecond - gap180, // E: 2 beats at 180 BPM
		166*time.Millisecond - gap180, // g2: 1 beat at 180 BPM
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
		{95, 457 * time.Millisecond},
		{177, 246 * time.Millisecond},
	}
	for _, c := range cases {
		pt, err := parseClanLordTuneWithTempo("c3", c.tempo)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
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

func TestParseNoteCLInvalid(t *testing.T) {
	i := 0
	octave := 4
	if _, _, err := parseNoteCL("h", &i, &octave); err == nil {
		t.Fatalf("expected error for invalid note letter")
	}
	i = 0
	octave = 4
	if _, _, err := parseNoteCL("c$", &i, &octave); err == nil {
		t.Fatalf("expected error for invalid modifier")
	}
}

func TestParseClanLordTuneWithTempoErrors(t *testing.T) {
	tests := []string{
		"z",           // invalid note
		"c$",          // invalid modifier
		"(c",          // unmatched loop start
		"c)",          // unmatched loop end
		"++++++++++c", // octave out of range
		"@+200c",      // tempo change out of range
	}
	for _, s := range tests {
		if _, err := parseClanLordTuneWithTempo(s, 120); err == nil {
			t.Errorf("%q: expected error", s)
		}
	}
	if _, err := parseClanLordTuneWithTempo("c", 50); err == nil {
		t.Errorf("initial tempo out of range should error")
	}
}
