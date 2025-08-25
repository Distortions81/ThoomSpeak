package main

import (
	"math"
	"testing"
	"time"
)

// classicNoteTiming computes expected note start/duration in ms for a parsed
// tune using the classic client's timing model derived from old_mac_client:
//   - Base unit (sixteenth note) length in ms: 15000/tempo
//   - For a note of duration d units: if tied, use d*unit; else use
//     ((d-1)*unit + noteLen) where noteLen = 90% of unit
//   - For rests of duration d units: duration = d*unit
func classicNoteTiming(pt parsedTune) []Note {
	var out []Note
	tempo := pt.tempo
	tempoIdx := 0
	startMS := 0

	unit := func(t int) int { // sixteenth-note unit length in ms
		return int(math.Round(15000.0 / float64(t)))
	}

	for i := 0; i < len(pt.events); i++ {
		for tempoIdx < len(pt.tempos) && pt.tempos[tempoIdx].index == i {
			tempo = pt.tempos[tempoIdx].tempo
			tempoIdx++
		}
		ev := pt.events[i]
		d := ev.beats
		u := unit(tempo)
		if len(ev.keys) == 0 { // rest
			startMS += int(d) * u
			continue
		}
		// note
		dur := 0
		if ev.nogap {
			dur = int(d) * u
		} else {
			// (d-1)*unit + 0.9*unit (truncate like classic client)
			dur = (int(d)-1)*u + int(0.9*float64(u))
		}
		for _, k := range ev.keys {
			out = append(out, Note{
				Key:      k,
				Velocity: 100,
				Start:    time.Duration(startMS) * time.Millisecond,
				Duration: time.Duration(dur) * time.Millisecond,
			})
		}
		startMS += int(d) * u
	}
	return out
}

func TestClassicTimingMatchesReference(t *testing.T) {
	cases := []string{
		"c",
		"C",
		"c1",
		"p",
		"[ce]",
		"[ce]3",
		"c_p_d",
		"c_1c1",
		"(cd)2@+60e%5f",
	}
	inst := instrument{program: 0, octave: 0, chord: 100, melody: 100}
	for _, s := range cases {
		pt := parseClanLordTuneWithTempo(s, 120)
		ref := classicNoteTiming(pt)
		got := eventsToNotes(pt, inst, 100)
		if len(ref) != len(got) {
			t.Fatalf("%q: expected %d notes, got %d", s, len(ref), len(got))
		}
		for i := range ref {
			if ref[i].Start != got[i].Start || ref[i].Duration != got[i].Duration {
				t.Fatalf("%q note %d: start/dur mismatch: got (%v,%v) want (%v,%v)", s, i, got[i].Start, got[i].Duration, ref[i].Start, ref[i].Duration)
			}
		}
	}
}
