package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	defaultInstrument    = 10
	durationBlack        = 2.0
	durationWhite        = 4.0
	defaultChordDuration = 4.0
)

// instruments holds the instrument table extracted from the classic client.
// Only the program number and octave offset are currently used.
var instruments = []instrument{
	{47, 1},   // 0 Lucky Lyra
	{73, 1},   // 1 Bone Flute
	{47, 0},   // 2 Starbuck Harp
	{106, 0},  // 3 Torjo
	{13, 0},   // 4 Xylo
	{25, 0},   // 5 Gitor
	{76, 1},   // 6 Reed Flute
	{17, -1},  // 7 Temple Organ
	{94, -1},  // 8 Conch
	{80, 1},   // 9 Ocarina
	{77, 1},   // 10 Centaur Organ
	{12, 0},   // 11 Vibra
	{59, -1},  // 12 Tuborn
	{110, 0},  // 13 Bagpipe
	{117, -1}, // 14 Orga Drum
	{115, 0},  // 15 Casserole
	{41, 1},   // 16 Violène
	{78, 1},   // 17 Pine Flute
	{22, -1},  // 18 Groanbox
	{108, -1}, // 19 Gho-To
	{44, -2},  // 20 Mammoth Violène
	{33, -2},  // 21 Gutbucket Bass
	{77, 0},   // 22 Glass Jug
}

// instrument describes a playable instrument mapping Clan Lord's instrument
// index to a General MIDI program number and an octave offset.
type instrument struct {
	program int
	octave  int
}

// queue sequentializes tune playback so overlapping /play commands do not
// render concurrently. Each tune section is played to completion before the
// next begins.
type tuneJob struct {
	program int
	notes   []Note
	who     int
}

var (
	tuneOnce   sync.Once
	tuneQueue  chan tuneJob
	currentMu  sync.Mutex
	currentWho int
)

func startTuneWorker() {
	tuneQueue = make(chan tuneJob, 128)
	go func() {
		for job := range tuneQueue {
			if audioContext == nil {
				continue
			}
			currentMu.Lock()
			currentWho = job.who
			currentMu.Unlock()
			if err := Play(audioContext, job.program, job.notes); err != nil {
				log.Printf("play tune worker: %v", err)
				if musicDebug {
					consoleMessage("play tune: " + err.Error())
					chatMessage("play tune: " + err.Error())
				}
			}
			currentMu.Lock()
			currentWho = 0
			currentMu.Unlock()
		}
	}()
}

// noteEvent represents a parsed tune event. A single event may contain multiple
// simultaneous notes (a chord).
type noteEvent struct {
	keys  []int
	durMS int
}

// playClanLordTune decodes a Clan Lord music string and plays it using the
// music package. The tune may optionally begin with an instrument index.
// For example: "3 cde" plays on instrument #3. It returns any playback error.
func playClanLordTune(tune string) error {
	if audioContext == nil {
		return fmt.Errorf("audio disabled")
	}
	if gs.Mute || gs.MusicVolume <= 0 {
		return fmt.Errorf("music muted")
	}

	inst := defaultInstrument
	fields := strings.Fields(tune)
	if len(fields) > 1 {
		if n, err := strconv.Atoi(fields[0]); err == nil && n >= 0 && n < len(instruments) {
			inst = n
			tune = strings.Join(fields[1:], " ")
		}
	}

	events := parseClanLordTuneWithTempo(tune, 120)
	if len(events) == 0 {
		return fmt.Errorf("empty tune")
	}

	prog := instruments[inst].program
	oct := instruments[inst].octave

	notes := eventsToNotes(events, oct, 100)

	// Enqueue for sequential playback and return immediately.
	tuneOnce.Do(startTuneWorker)
	select {
	case tuneQueue <- tuneJob{program: prog, notes: notes}:
	default:
		// If the queue is full, drop the oldest by draining one then enqueue.
		// This prevents unbounded growth during bursts.
		select {
		case <-tuneQueue:
		default:
		}
		tuneQueue <- tuneJob{program: prog, notes: notes}
	}
	return nil
}

// eventsToNotes converts parsed note events into synth notes with explicit start
// times. All notes in the same event (a chord) share the same start time.
func eventsToNotes(events []noteEvent, oct int, velocity int) []Note {
	var notes []Note
	startMS := 0
	for _, ev := range events {
		for _, k := range ev.keys {
			key := k + oct*12
			if key < 0 || key > 127 {
				continue
			}
			notes = append(notes, Note{
				Key:      key,
				Velocity: velocity,
				Start:    time.Duration(startMS) * time.Millisecond,
				Duration: time.Duration(ev.durMS) * time.Millisecond,
			})
		}
		startMS += ev.durMS
	}
	return notes
}

// parseClanLordTune converts Clan Lord music notation into a slice of note
// events at the default tempo of 120 BPM.
func parseClanLordTune(s string) []noteEvent {
	return parseClanLordTuneWithTempo(s, 120)
}

// parseClanLordTuneWithTempo converts Clan Lord music notation into a slice of
// note events using the provided tempo in BPM.
func parseClanLordTuneWithTempo(s string, tempo int) []noteEvent {
	if tempo <= 0 {
		tempo = 120
	}
	quarter := 60000 / tempo // ms
	octave := 4
	i := 0
	var events []noteEvent
	for i < len(s) {
		c := s[i]
		switch c {
		case ' ', '\n', '\r', '\t':
			i++
		case '<': // comment
			for i < len(s) && s[i] != '>' {
				i++
			}
			if i < len(s) {
				i++
			}
		case '+':
			octave++
			i++
		case '-':
			octave--
			i++
		case '=':
			octave = 4
			i++
		case '/':
			octave = 5
			i++
		case '\\':
			octave = 3
			i++
		case 'p': // rest
			i++
			durBeats := durationBlack
			if i < len(s) && s[i] >= '1' && s[i] <= '9' {
				durBeats = float64(s[i] - '0')
				i++
			}
			events = append(events, noteEvent{nil, int(durBeats * float64(quarter))})
		case '[': // chord
			i++
			var keys []int
			for i < len(s) && s[i] != ']' {
				if handleOctave(&octave, s[i]) {
					i++
					continue
				}
				if isNoteLetter(s[i]) {
					k, _ := parseNoteCL(s, &i, &octave)
					if k != 0 {
						keys = append(keys, k)
					}
					continue
				}
				i++
			}
			if i < len(s) && s[i] == ']' {
				i++
			}
			durBeats := defaultChordDuration
			if i < len(s) && s[i] >= '1' && s[i] <= '9' {
				durBeats = float64(s[i] - '0')
				i++
			}
			if len(keys) > 0 {
				events = append(events, noteEvent{keys, int(durBeats * float64(quarter))})
			}
		default:
			if isNoteLetter(c) {
				k, beats := parseNoteCL(s, &i, &octave)
				if k != 0 {
					events = append(events, noteEvent{[]int{k}, int(beats * float64(quarter))})
				}
			} else {
				i++
			}
		}
	}
	return events
}

func handleOctave(oct *int, c byte) bool {
	switch c {
	case '+':
		*oct = *oct + 1
		return true
	case '-':
		*oct = *oct - 1
		return true
	case '=':
		*oct = 4
		return true
	case '/':
		*oct = 5
		return true
	case '\\':
		*oct = 3
		return true
	}
	return false
}

// parseNoteCL parses a single note and returns its MIDI key and beat length.
func parseNoteCL(s string, i *int, octave *int) (int, float64) {
	c := s[*i]
	isUpper := unicode.IsUpper(rune(c))
	base := noteOffset(unicode.ToLower(rune(c)))
	if base < 0 {
		*i++
		return 0, 0
	}
	(*i)++
	pitch := base + ((*octave)+1)*12
	beats := durationBlack
	if isUpper {
		beats = durationWhite
	}
	for *i < len(s) {
		ch := s[*i]
		switch {
		case ch == '#':
			pitch++
			(*i)++
		case ch == '.':
			pitch--
			(*i)++
		case ch >= '1' && ch <= '9':
			beats = float64(ch - '0')
			(*i)++
		case ch == '_':
			(*i)++ // ignore ties
		default:
			return pitch, beats
		}
	}
	return pitch, beats
}

func noteOffset(r rune) int {
	switch r {
	case 'c':
		return 0
	case 'd':
		return 2
	case 'e':
		return 4
	case 'f':
		return 5
	case 'g':
		return 7
	case 'a':
		return 9
	case 'b':
		return 11
	}
	return -1
}

func isNoteLetter(b byte) bool {
	return (b >= 'a' && b <= 'g') || (b >= 'A' && b <= 'G')
}

// Extended support for /music commands
type MusicParams struct {
	Inst   int
	Notes  string
	Tempo  int // BPM 60..180
	VolPct int // 0..100
	Part   bool
	Stop   bool
	Who    int
	With   []int
	Me     bool
}

// Internal state for assembling multipart songs.
type pendingSong struct {
	inst    int
	tempo   int
	volPct  int
	notes   []string
	withIDs []int
}

var (
	pendingMu   sync.Mutex
	pendingByID = make(map[int]*pendingSong)
)

// handleMusicParams translates parsed music params into queued playback. It
// supports /stop, /part accumulation and tempo/volume/instrument parameters.
func handleMusicParams(mp MusicParams) {
	if mp.Stop {
		// Scoped stop: if who provided, clear that pending and stop if playing.
		if mp.Who != 0 {
			pendingMu.Lock()
			delete(pendingByID, mp.Who)
			pendingMu.Unlock()
			currentMu.Lock()
			cw := currentWho
			currentMu.Unlock()
			if cw == mp.Who {
				stopAllMusic()
			}
		} else {
			// Global stop
			pendingMu.Lock()
			pendingByID = make(map[int]*pendingSong)
			pendingMu.Unlock()
			stopAllMusic()
		}
		return
	}
	// Ignore play requests while muted, matching classic behavior when sound
	// is off. Still handled /stop above regardless of mute state.
	if gs.Mute || gs.MusicVolume <= 0 {
		return
	}
	// Validate basics
	if mp.Inst < 0 || mp.Inst >= len(instruments) {
		mp.Inst = defaultInstrument
	}
	if mp.Tempo <= 0 {
		mp.Tempo = 120
	}
	if mp.VolPct <= 0 {
		mp.VolPct = 100
	}
	id := mp.Who // 0 is the system queue

	// Accumulate multipart songs when /part is present.
	if mp.Part {
		pendingMu.Lock()
		ps := pendingByID[id]
		if ps == nil {
			ps = &pendingSong{inst: mp.Inst, tempo: mp.Tempo, volPct: mp.VolPct}
			pendingByID[id] = ps
		} else {
			if mp.Inst != 0 {
				ps.inst = mp.Inst
			}
			if mp.Tempo != 0 {
				ps.tempo = mp.Tempo
			}
			if mp.VolPct != 0 {
				ps.volPct = mp.VolPct
			}
		}
		if n := strings.TrimSpace(mp.Notes); n != "" {
			ps.notes = append(ps.notes, n)
		}
		if len(mp.With) > 0 {
			ps.withIDs = append([]int(nil), mp.With...)
		}
		pendingMu.Unlock()
		return
	}

	// Finalize: merge any pending parts, then queue a single tune.
	inst := mp.Inst
	tempo := mp.Tempo
	vol := mp.VolPct
	notes := strings.TrimSpace(mp.Notes)
	pendingMu.Lock()
	if ps := pendingByID[id]; ps != nil {
		if notes != "" {
			ps.notes = append(ps.notes, notes)
		}
		notes = strings.Join(ps.notes, " ")
		if ps.inst != 0 {
			inst = ps.inst
		}
		if ps.tempo != 0 {
			tempo = ps.tempo
		}
		if ps.volPct != 0 {
			vol = ps.volPct
		}
		if len(mp.With) == 0 && len(ps.withIDs) > 0 {
			mp.With = append([]int(nil), ps.withIDs...)
		}
		delete(pendingByID, id)
	}
	// If sync requested via /with, require that all referenced IDs also have
	// pending content; otherwise, store this song and return until ready.
	if len(mp.With) > 0 {
		// Save current as pending with its group
		p := &pendingSong{inst: inst, tempo: tempo, volPct: vol, notes: []string{notes}, withIDs: append([]int(nil), mp.With...)}
		pendingByID[id] = p
		// Check readiness of group (including self)
		all := append([]int{id}, mp.With...)
		ready := true
		for _, w := range all {
			if _, ok := pendingByID[w]; !ok {
				ready = false
				break
			}
		}
		if !ready {
			pendingMu.Unlock()
			return
		}
		// All parts present: build jobs in sorted order
		// Deduplicate and sort IDs
		idmap := map[int]struct{}{}
		for _, w := range all {
			idmap[w] = struct{}{}
		}
		ids := make([]int, 0, len(idmap))
		for w := range idmap {
			ids = append(ids, w)
		}
		// simple insertion sort
		for i := 1; i < len(ids); i++ {
			j := i
			for j > 0 && ids[j-1] > ids[j] {
				ids[j-1], ids[j] = ids[j], ids[j-1]
				j--
			}
		}
		jobs := make([]tuneJob, 0, len(ids))
		for _, w := range ids {
			ps := pendingByID[w]
			nstr := strings.Join(ps.notes, " ")
			jobs = append(jobs, makeTuneJob(w, ps.inst, ps.tempo, ps.volPct, nstr))
			delete(pendingByID, w)
		}
		pendingMu.Unlock()
		// Enqueue jobs sequentially
		for _, job := range jobs {
			enqueueTune(job)
		}
		return
	}
	pendingMu.Unlock()
	if notes == "" {
		return
	}

	job := makeTuneJob(id, inst, tempo, vol, notes)
	enqueueTune(job)
}

func makeTuneJob(who, inst, tempo, vol int, notes string) tuneJob {
	events := parseClanLordTuneWithTempo(notes, tempo)
	prog := instruments[inst].program
	oct := instruments[inst].octave
	// Scale 0..100 to 1..127 velocity.
	vel := vol
	if vel <= 0 {
		vel = 100
	}
	if vel > 100 {
		vel = 100
	}
	vel = int(float64(vel)*1.27 + 0.5)
	if vel < 1 {
		vel = 1
	} else if vel > 127 {
		vel = 127
	}
	notesOut := eventsToNotes(events, oct, vel)
	return tuneJob{program: prog, notes: notesOut, who: who}
}

func enqueueTune(job tuneJob) {
	tuneOnce.Do(startTuneWorker)
	select {
	case tuneQueue <- job:
	default:
		select {
		case <-tuneQueue:
		default:
		}
		tuneQueue <- job
	}
}

// clearTuneQueue drains any queued tunes so newly queued items can take effect
// immediately after mute/unmute or a stop request.
func clearTuneQueue() {
	if tuneQueue == nil {
		return
	}
	for {
		select {
		case <-tuneQueue:
			// drained one
		default:
			return
		}
	}
}
