package music

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"time"

	"github.com/ebitengine/oto/v3"
	meltysynth "github.com/sinshu/go-meltysynth/meltysynth"
)

// Note represents a single MIDI note with a duration.
type Note struct {
	// Key is the MIDI note number (e.g. 60 = middle C).
	Key int
	// Velocity is the MIDI velocity 1..127.
	Velocity int
	// Duration specifies how long the note should sound.
	Duration time.Duration
}

// Play renders the provided notes using the given SoundFont and plays them.
// The reader must point to a SoundFont2 (sf2) file. The function blocks until
// playback has finished.
func Play(sf io.ReadSeeker, program int, notes []Note) error {
	const (
		sampleRate = 44100
		channels   = 2
		block      = 512
	)

	sfnt, err := meltysynth.NewSoundFont(sf)
	if err != nil {
		return err
	}
	settings := meltysynth.NewSynthesizerSettings(sampleRate)
	synth, err := meltysynth.NewSynthesizer(sfnt, settings)
	if err != nil {
		return err
	}

	const ch = 0
	synth.ProcessMidiMessage(ch, 0xC0, int32(program), 0) // program change

	type event struct {
		key, vel   int
		start, end int
	}
	cursor := 0
	var events []event
	for _, n := range notes {
		durSamples := int(float64(sampleRate) * n.Duration.Seconds())
		if durSamples <= 0 {
			continue
		}
		ev := event{key: n.Key, vel: n.Velocity, start: cursor, end: cursor + durSamples}
		events = append(events, ev)
		cursor += durSamples
	}
	totalSamples := cursor

	leftAll := make([]float32, 0, totalSamples)
	rightAll := make([]float32, 0, totalSamples)
	active := map[int]bool{}

	trigger := func(start, count int) {
		end := start + count
		for _, ev := range events {
			if ev.start >= start && ev.start < end && !active[ev.key] {
				synth.NoteOn(ch, int32(ev.key), int32(ev.vel))
				active[ev.key] = true
			}
			if ev.end >= start && ev.end < end && active[ev.key] {
				synth.NoteOff(ch, int32(ev.key))
				active[ev.key] = false
			}
		}
	}

	for pos := 0; pos < totalSamples; pos += block {
		n := block
		if pos+n > totalSamples {
			n = totalSamples - pos
		}
		trigger(pos, n)
		left := make([]float32, n)
		right := make([]float32, n)
		synth.Render(left, right)
		leftAll = append(leftAll, left...)
		rightAll = append(rightAll, right...)
	}

	// Normalize to avoid clipping and boost quiet audio
	var peak float32
	for i := range leftAll {
		if v := float32(math.Abs(float64(leftAll[i]))); v > peak {
			peak = v
		}
		if v := float32(math.Abs(float64(rightAll[i]))); v > peak {
			peak = v
		}
	}
	if peak > 0 {
		g := float32(0.99) / peak
		if g != 1 {
			for i := range leftAll {
				leftAll[i] *= g
				rightAll[i] *= g
			}
		}
	}

	// Interleave samples for oto
	inter := make([]float32, 2*len(leftAll))
	for i := range leftAll {
		inter[2*i] = leftAll[i]
		inter[2*i+1] = rightAll[i]
	}

	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: channels,
		Format:       oto.FormatFloat32LE,
	})
	if err != nil {
		return err
	}
	<-ready

	var pcm bytes.Buffer
	if err := binary.Write(&pcm, binary.LittleEndian, inter); err != nil {
		return err
	}
	player := ctx.NewPlayer(bytes.NewReader(pcm.Bytes()))
	player.Play()

	dur := time.Duration(float64(totalSamples) / sampleRate * float64(time.Second))
	time.Sleep(dur)
	return player.Close()
}
