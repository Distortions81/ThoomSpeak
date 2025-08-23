package main

import (
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

type noteAction struct {
	key    int
	on     bool
	sample int
}

type mockSynth struct {
	cur    int
	events []noteAction
}

func (m *mockSynth) ProcessMidiMessage(channel int32, command int32, data1, data2 int32) {}

func (m *mockSynth) NoteOn(channel, key, vel int32) {
	m.events = append(m.events, noteAction{int(key), true, m.cur})
}

func (m *mockSynth) NoteOff(channel, key int32) {
	m.events = append(m.events, noteAction{int(key), false, m.cur})
}

func (m *mockSynth) Render(left, right []float32) {
	m.cur += len(left)
}

func durToSamples(d time.Duration) int {
	return int(d.Nanoseconds() * sampleRate / int64(time.Second))
}

func TestPlayOverlappingNotes(t *testing.T) {
	ctx := audio.NewContext(sampleRate)

	ms := &mockSynth{}
	synth = ms
	setupSynthOnce = sync.Once{}
	setupSynthOnce.Do(func() {})

	blockDur := time.Second * time.Duration(block) / sampleRate
	noteDur := 2 * blockDur
	notes := []Note{
		{Key: 60, Velocity: 100, Start: 0, Duration: noteDur},
		{Key: 64, Velocity: 100, Start: blockDur, Duration: noteDur},
	}

	if err := Play(ctx, 0, notes); err != nil {
		t.Fatalf("Play returned error: %v", err)
	}

	if len(ms.events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(ms.events))
	}

	on1 := ms.events[0]
	on2 := ms.events[1]
	off1 := ms.events[2]
	off2 := ms.events[3]

	if on1.key != 60 || !on1.on || on1.sample != 0 {
		t.Fatalf("unexpected first event: %+v", on1)
	}
	expOn2 := durToSamples(blockDur)
	if on2.key != 64 || !on2.on || on2.sample != expOn2 {
		t.Fatalf("unexpected second event: %+v", on2)
	}
	expOff1 := durToSamples(noteDur)
	if off1.key != 60 || off1.on || off1.sample != expOff1 {
		t.Fatalf("unexpected third event: %+v", off1)
	}
	expOff2 := durToSamples(blockDur + noteDur)
	if off2.key != 64 || off2.on || off2.sample != expOff2 {
		t.Fatalf("unexpected fourth event: %+v", off2)
	}
	if !(on2.sample < off1.sample) {
		t.Fatalf("notes did not overlap")
	}
}

func TestEventsToNotesChordStart(t *testing.T) {
	pt := parseClanLordTune("[ce]d")
	inst := instrument{program: 0, octave: 0, chord: 100, melody: 100}
	notes := eventsToNotes(pt, inst, 100)
	if len(notes) != 3 {
		t.Fatalf("expected 3 notes, got %d", len(notes))
	}
	if notes[0].Start != 0 || notes[1].Start != 0 {
		t.Fatalf("chord notes have different start times: %v %v", notes[0].Start, notes[1].Start)
	}
	quarter := 500 * time.Millisecond
	if notes[2].Start != quarter {
		t.Fatalf("third note start = %v; want %v", notes[2].Start, quarter)
	}
}
