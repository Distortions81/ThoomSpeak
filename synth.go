package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
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
	// Use a small fixed render block that aligns with common synth effect
	// processing sizes to avoid internal ring-buffer edge cases.
	block = 64

	// tailSamples extends the rendered length by 2 seconds to allow reverb to decay.
	tailSamples = sampleRate * 2
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

var (
	setupSynthOnce sync.Once
	sfntCached     *meltysynth.SoundFont
	synthSettings  *meltysynth.SynthesizerSettings

	musicPlayers   = make(map[*audio.Player]struct{})
	musicPlayersMu sync.Mutex
)

func stopAllMusic() {
	musicPlayersMu.Lock()
	defer musicPlayersMu.Unlock()
	for p := range musicPlayers {
		_ = p.Close()
		delete(musicPlayers, p)
	}
}

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
	// Align meltysynth internal block size with our render loop to reduce
	// chances of effect buffers overrunning on odd boundaries.
	settings.BlockSize = block
	sfntCached = sfnt
	synthSettings = settings
}

// renderSong renders the provided notes using the current SoundFont and returns
// the raw left and right channel samples. The caller can further process or mix
// these samples before playback.
func renderSong(program int, notes []Note) ([]float32, []float32, error) {
	setupSynthOnce.Do(setupSynth)
	if sfntCached == nil || synthSettings == nil {
		return nil, nil, errors.New("synth not initialized")
	}

	const ch = 0
	// Build a fresh synth per song to avoid concurrent use of internal state.
	syn, err := meltysynth.NewSynthesizer(sfntCached, synthSettings)
	if err != nil {
		return nil, nil, err
	}
	syn.ProcessMidiMessage(ch, 0xC0, int32(program), 0)

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
	totalSamples := maxEnd + tailSamples

	leftAll := make([]float32, 0, totalSamples)
	rightAll := make([]float32, 0, totalSamples)
	active := map[int]bool{}

	trigger := func(start, count int) {
		end := start + count
		for _, ev := range events {
			if ev.start >= start && ev.start < end && !active[ev.key] {
				syn.NoteOn(ch, int32(ev.key), int32(ev.vel))
				active[ev.key] = true
			}
			if ev.end >= start && ev.end < end && active[ev.key] {
				syn.NoteOff(ch, int32(ev.key))
				active[ev.key] = false
			}
		}
	}

	for pos := 0; pos < totalSamples; pos += block {
		// Render in fixed-size blocks to avoid triggering edge cases in the
		// underlying synth (e.g., effects processing relying on block size).
		n := block
		if pos+n > totalSamples {
			n = totalSamples - pos
		}
		trigger(pos, n)
		// Always ask the synth to render a full block, then trim to the
		// number of remaining samples we actually need to keep timing exact.
		left := make([]float32, block)
		right := make([]float32, block)
		if err := safeRender(syn, left, right); err != nil {
			return nil, nil, fmt.Errorf("synth render: %v", err)
		}
		leftAll = append(leftAll, left[:n]...)
		rightAll = append(rightAll, right[:n]...)
	}

	return leftAll, rightAll, nil
}

// safeRender calls the synthesizer Render method while protecting against
// panics from the underlying synth implementation. Any panic is recovered and
// returned as an error so callers can fail gracefully instead of crashing the
// entire client.
func safeRender(s synthesizer, left, right []float32) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in synth.Render: %v", r)
		}
	}()
	s.Render(left, right)
	return nil
}

// mixPCM normalizes the provided samples and returns interleaved 16-bit PCM
// data suitable for audio playback.
func mixPCM(leftAll, rightAll []float32) []byte {
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

	pcm := make([]byte, len(leftAll)*4)
	for i := range leftAll {
		l := int16(leftAll[i] * 32767)
		r := int16(rightAll[i] * 32767)
		binary.LittleEndian.PutUint16(pcm[4*i:], uint16(l))
		binary.LittleEndian.PutUint16(pcm[4*i+2:], uint16(r))
	}
	return pcm
}

// Play renders the provided notes using the given SoundFont, mixes the entire
// song, and then plays it through the provided audio context. The function
// blocks until playback has finished.
func Play(ctx *audio.Context, program int, notes []Note) error {
	if ctx == nil {
		return errors.New("nil audio context")
	}

	// Stream-render music in small chunks to avoid long upfront rendering.
	stream, err := newMusicStream(program, notes)
	if err != nil {
		return err
	}
	defer stream.Close()
	player, err := ctx.NewPlayer(stream)
	if err != nil {
		return err
	}

	vol := gs.MusicVolume
	if gs.Mute {
		vol = 0
	}
	player.SetVolume(vol)

	musicPlayersMu.Lock()
	musicPlayers[player] = struct{}{}
	musicPlayersMu.Unlock()

	player.Play()

	// Estimate duration from notes for timeout, but exit early when playback completes.
	// Compute max end as in renderSong.
	maxEnd := 0
	for _, n := range notes {
		durSamples := int(n.Duration.Nanoseconds() * sampleRate / int64(time.Second))
		if durSamples <= 0 {
			continue
		}
		startSamples := int(n.Start.Nanoseconds() * sampleRate / int64(time.Second))
		end := startSamples + durSamples
		if end > maxEnd {
			maxEnd = end
		}
	}
	total := maxEnd + tailSamples
	dur := time.Duration(total) * time.Second / sampleRate
	deadline := time.Now().Add(dur)
	for time.Now().Before(deadline) {
		// Exit early if the player has been stopped/closed (e.g., due to mute).
		if !safeIsPlaying(player) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	musicPlayersMu.Lock()
	delete(musicPlayers, player)
	musicPlayersMu.Unlock()

	return player.Close()
}

// safeIsPlaying checks IsPlaying and recovers if the player has been closed.
func safeIsPlaying(p *audio.Player) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	return p.IsPlaying()
}

// musicStream implements audio.ReadSeekCloser and renders PCM on demand
// from meltysynth in small blocks while scheduling note events.
type musicStream struct {
	mu      sync.Mutex
	syn     *meltysynth.Synthesizer
	program int
	events  []struct{ key, vel, start, end int }
	active  map[int]bool
	pos     int // samples rendered so far
	total   int // total samples including tail
	closed  bool
	// buffered PCM to ensure smooth playback
	buf    []byte
	bufPos int
	// scratch render buffers
	left  []float32
	right []float32
}

func newMusicStream(program int, notes []Note) (io.ReadSeekCloser, error) {
	setupSynthOnce.Do(setupSynth)
	if sfntCached == nil || synthSettings == nil {
		return nil, errors.New("synth not initialized")
	}
	syn, err := meltysynth.NewSynthesizer(sfntCached, synthSettings)
	if err != nil {
		return nil, err
	}
	const ch = 0
	syn.ProcessMidiMessage(ch, 0xC0, int32(program), 0)
	// Convert notes to events
	var events []struct{ key, vel, start, end int }
	maxEnd := 0
	for _, n := range notes {
		durSamples := int(n.Duration.Nanoseconds() * sampleRate / int64(time.Second))
		if durSamples <= 0 {
			continue
		}
		startSamples := int(n.Start.Nanoseconds() * sampleRate / int64(time.Second))
		ev := struct{ key, vel, start, end int }{key: n.Key, vel: n.Velocity, start: startSamples, end: startSamples + durSamples}
		events = append(events, ev)
		if ev.end > maxEnd {
			maxEnd = ev.end
		}
	}
	ms := &musicStream{syn: syn, program: program, events: events, active: map[int]bool{}, total: maxEnd + tailSamples, left: make([]float32, block), right: make([]float32, block)}
	// Pre-render roughly one second in quarter-second chunks so playback
	// can start quickly while the background goroutine continues filling.
	ms.mu.Lock()
	for i := 0; i < 4 && ms.pos < ms.total; i++ {
		if err := ms.renderSamplesLocked(sampleRate / 4); err != nil && !errors.Is(err, io.EOF) {
			ms.mu.Unlock()
			return nil, err
		}
	}
	ms.mu.Unlock()
	go ms.fillLoop()
	return ms, nil
}

func (m *musicStream) trigger(start, count int) {
	const ch = 0
	end := start + count
	for _, ev := range m.events {
		if ev.start >= start && ev.start < end && !m.active[ev.key] {
			m.syn.NoteOn(ch, int32(ev.key), int32(ev.vel))
			m.active[ev.key] = true
		}
		if ev.end >= start && ev.end < end && m.active[ev.key] {
			m.syn.NoteOff(ch, int32(ev.key))
			m.active[ev.key] = false
		}
	}
}

func (m *musicStream) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	bytesPerSec := sampleRate * 4
	target := bytesPerSec * 2
	for {
		m.mu.Lock()
		if m.closed {
			m.mu.Unlock()
			return 0, io.EOF
		}
		avail := len(m.buf) - m.bufPos
		if avail > 0 || (m.pos >= m.total && m.bufPos >= len(m.buf)) {
			// either we have data, or everything is rendered and consumed
			if avail > len(p) {
				avail = len(p)
			}
			copy(p[:avail], m.buf[m.bufPos:m.bufPos+avail])
			m.bufPos += avail
			if m.bufPos >= target {
				m.buf = append([]byte(nil), m.buf[m.bufPos:]...)
				m.bufPos = 0
			}
			done := m.pos >= m.total && m.bufPos >= len(m.buf)
			m.mu.Unlock()
			if done {
				if avail == 0 {
					return 0, io.EOF
				}
				return avail, io.EOF
			}
			return avail, nil
		}
		// No data available yet but more to come; wait for renderer.
		m.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
}

// renderSamplesLocked renders up to the requested number of samples of PCM and
// appends them to m.buf. Caller must hold m.mu.
func (m *musicStream) renderSamplesLocked(samples int) error {
	if m.pos >= m.total {
		return io.EOF
	}
	if samples > m.total-m.pos {
		samples = m.total - m.pos
	}
	for samples > 0 {
		n := block
		if n > samples {
			n = samples
		}
		m.trigger(m.pos, n)
		if err := safeRender(m.syn, m.left, m.right); err != nil {
			return err
		}
		need := n * 4
		off := len(m.buf)
		m.buf = append(m.buf, make([]byte, need)...)
		for i := 0; i < n; i++ {
			l := int16(m.left[i] * 32767)
			r := int16(m.right[i] * 32767)
			binary.LittleEndian.PutUint16(m.buf[off+i*4:], uint16(l))
			binary.LittleEndian.PutUint16(m.buf[off+i*4+2:], uint16(r))
		}
		for i := 0; i < n; i++ {
			m.left[i], m.right[i] = 0, 0
		}
		m.pos += n
		samples -= n
	}
	return nil
}

// renderSecondLocked renders one second of PCM. Caller must hold m.mu.
func (m *musicStream) renderSecondLocked() error {
	return m.renderSamplesLocked(sampleRate)
}

// fillLoop continuously renders small chunks in a background goroutine to keep
// the playback buffer topped up.
func (m *musicStream) fillLoop() {
	bytesPerSec := sampleRate * 4
	target := bytesPerSec * 2
	for {
		m.mu.Lock()
		if m.closed || m.pos >= m.total {
			m.mu.Unlock()
			return
		}
		if len(m.buf)-m.bufPos < target*2 {
			_ = m.renderSamplesLocked(sampleRate / 4)
			m.mu.Unlock()
			continue
		}
		m.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
}

func (m *musicStream) Seek(offset int64, whence int) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Support only restart to beginning for safety.
	switch whence {
	case io.SeekStart:
		if offset == 0 {
			syn, err := meltysynth.NewSynthesizer(sfntCached, synthSettings)
			if err != nil {
				return 0, err
			}
			const ch = 0
			syn.ProcessMidiMessage(ch, 0xC0, int32(m.program), 0)
			m.syn = syn
			m.active = map[int]bool{}
			m.pos = 0
			return 0, nil
		}
		// Only support resetting to start.
		return int64(m.pos * 4), errors.New("unsupported seek")
	case io.SeekCurrent:
		if offset == 0 {
			return int64(m.pos * 4), nil
		}
		return int64(m.pos * 4), errors.New("unsupported seek")
	case io.SeekEnd:
		if offset == 0 {
			return int64(m.total * 4), nil
		}
		return int64(m.pos * 4), errors.New("unsupported seek")
	default:
		return int64(m.pos * 4), errors.New("unsupported seek")
	}
}

func (m *musicStream) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// dumpPCMAsWAV writes the provided 16-bit stereo PCM data to a WAV file when
// the -dumpMusic flag is set. Files are named music_YYYYMMDD_HHMMSS.wav.
func dumpPCMAsWAV(pcm []byte) {
	ts := time.Now().Format("20060102_150405")
	name := "music_" + ts + ".wav"
	f, err := os.Create(name)
	if err != nil {
		log.Printf("dump music: %v", err)
		return
	}
	defer f.Close()

	dataLen := uint32(len(pcm))
	var header [44]byte
	copy(header[0:], []byte("RIFF"))
	binary.LittleEndian.PutUint32(header[4:], 36+dataLen)
	copy(header[8:], []byte("WAVE"))
	copy(header[12:], []byte("fmt "))
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], 2)
	binary.LittleEndian.PutUint32(header[24:], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:], uint32(sampleRate*4))
	binary.LittleEndian.PutUint16(header[32:], 4)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], []byte("data"))
	binary.LittleEndian.PutUint32(header[40:], dataLen)

	if _, err := f.Write(header[:]); err != nil {
		log.Printf("dump music header: %v", err)
		return
	}
	if _, err := f.Write(pcm); err != nil {
		log.Printf("dump music data: %v", err)
		return
	}
	log.Printf("wrote %s", name)
}
