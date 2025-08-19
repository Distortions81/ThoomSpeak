package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"math"
	"os"
	"path"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	meltysynth "github.com/sinshu/go-meltysynth/meltysynth"
)

const (
	sampleRate = 44100
	block      = 512

	// tailMultiplier extends the rendered length to allow reverb to decay.
	tailMultiplier = 4
)

// Note represents a single MIDI note with a duration and start time.
type Note struct {
	// Key is the MIDI note number (e.g. 60 = middle C).
	Key int
	// Velocity is the MIDI velocity 1..127.
	Velocity int
	// Start is the time offset from the beginning when the note starts.
	Start time.Duration
	// Duration specifies how long the note should sound.
	Duration time.Duration
}

// synthesizer abstracts the subset of meltysynth.Synthesizer used by Play.
type synthesizer interface {
	ProcessMidiMessage(channel int32, command int32, data1, data2 int32)
	NoteOn(channel, key, vel int32)
	NoteOff(channel, key int32)
	Render(left, right []float32)
}

var synth synthesizer
var setupSynthOnce sync.Once

func setupSynth() {
	var err error

	sfPath := path.Join(dataDirPath, "soundfont.sf2")

	var sfData []byte
	sfData, err = os.ReadFile(sfPath)
	if err != nil {
		log.Printf("soundfont missing: %v", err)
		return
	}
	rs := bytes.NewReader(sfData)
	sfnt, err := meltysynth.NewSoundFont(rs)
	if err != nil {
		return
	}
	settings := meltysynth.NewSynthesizerSettings(sampleRate)
	synth, err = meltysynth.NewSynthesizer(sfnt, settings)
	if err != nil {
		return
	}
}

// Play renders the provided notes using the given SoundFont and plays them
// through the provided audio context. The reader must point to a SoundFont2
// (sf2) file. The function blocks until playback has finished.
func Play(ctx *audio.Context, program int, notes []Note) error {

	if ctx == nil {
		return errors.New("nil audio context")
	}
	setupSynthOnce.Do(setupSynth)
	if synth == nil {
		return errors.New("synth not initialized")
	}

	const ch = 0
	synth.ProcessMidiMessage(ch, 0xC0, int32(program), 0) // program change

	type event struct {
		key, vel   int
		start, end int
	}
	var events []event
	var maxEnd int
	for _, n := range notes {
		durSamples := int(n.Duration.Nanoseconds() * sampleRate / int64(time.Second))
		if durSamples <= 0 {
			continue
		}
		startSamples := int(n.Start.Nanoseconds() * sampleRate / int64(time.Second))
		ev := event{key: n.Key, vel: n.Velocity, start: startSamples, end: startSamples + durSamples}
		events = append(events, ev)
		if ev.end > maxEnd {
			maxEnd = ev.end
		}
	}
	totalSamples := maxEnd * tailMultiplier

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

	// Interleave and convert to 16-bit PCM for the audio context.
	pcm := make([]byte, len(leftAll)*4)
	for i := range leftAll {
		l := int16(leftAll[i] * 32767)
		r := int16(rightAll[i] * 32767)
		binary.LittleEndian.PutUint16(pcm[4*i:], uint16(l))
		binary.LittleEndian.PutUint16(pcm[4*i+2:], uint16(r))
	}

	player := ctx.NewPlayerFromBytes(pcm)
	player.Play()

	dur := time.Duration(float64(totalSamples) / sampleRate * float64(time.Second))
	time.Sleep(dur)
	return player.Close()
}
