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
// Each instrument defines the General MIDI program number, octave offset, and
// velocity scaling factors for chord and melody notes.
var instruments = []instrument{
	{47, 1, 100, 100},   // 0 Lucky Lyra
	{73, 1, 100, 100},   // 1 Bone Flute
	{47, 0, 100, 100},   // 2 Starbuck Harp
	{106, 0, 100, 100},  // 3 Torjo
	{13, 0, 100, 100},   // 4 Xylo
	{25, 0, 100, 100},   // 5 Gitor
	{76, 1, 100, 100},   // 6 Reed Flute
	{17, -1, 100, 100},  // 7 Temple Organ
	{94, -1, 100, 100},  // 8 Conch
	{80, 1, 100, 100},   // 9 Ocarina
	{77, 1, 100, 100},   // 10 Centaur Organ
	{12, 0, 100, 100},   // 11 Vibra
	{59, -1, 100, 100},  // 12 Tuborn
	{110, 0, 100, 100},  // 13 Bagpipe
	{117, -1, 100, 100}, // 14 Orga Drum
	{115, 0, 100, 100},  // 15 Casserole
	{41, 1, 100, 100},   // 16 Violène
	{78, 1, 100, 100},   // 17 Pine Flute
	{22, -1, 100, 100},  // 18 Groanbox
	{108, -1, 100, 100}, // 19 Gho-To
	{44, -2, 100, 100},  // 20 Mammoth Violène
	{33, -2, 100, 100},  // 21 Gutbucket Bass
	{77, 0, 100, 100},   // 22 Glass Jug
}

// instrument describes a playable instrument mapping Clan Lord's instrument
// index to a General MIDI program number, octave offset, and velocity scaling
// factors for chords and melodies.
type instrument struct {
	program int
	octave  int
	chord   int // chord velocity factor (0-100)
	melody  int // melody velocity factor (0-100)
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
// simultaneous notes (a chord). Durations are stored in half-beats and converted to
// milliseconds later once tempo and loop processing is applied.
type noteEvent struct {
	keys   []int
	beats  float64
	volume int
}

// tempoEvent notes a tempo change occurring before the event at the given index.
type tempoEvent struct {
	index int
	tempo int
}

// loopMarker describes a looped sequence of events.
type loopMarker struct {
	start  int // index of the first event in the loop
	end    int // index after the last event in the loop
	repeat int // total number of times to play the loop
}

// parsedTune aggregates events with optional loop and tempo metadata.
type parsedTune struct {
	events []noteEvent
	tempos []tempoEvent
	loops  []loopMarker
	tempo  int // initial tempo in BPM
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

	pt := parseClanLordTuneWithTempo(tune, 120)
	if len(pt.events) == 0 {
		return fmt.Errorf("empty tune")
	}

	instData := instruments[inst]
	prog := instData.program

	notes := eventsToNotes(pt, instData, 100)

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
// times. All notes in the same event (a chord) share the same start time. The
// provided instrument's chord or melody velocity factors are applied depending
// on the event type.
func eventsToNotes(pt parsedTune, inst instrument, velocity int) []Note {
	var notes []Note
	tempo := pt.tempo
	tempoIdx := 0
	startMS := 0

	// Build map of loop starts for quick lookup
	loopMap := make(map[int][]loopMarker)
	for _, lp := range pt.loops {
		loopMap[lp.start] = append(loopMap[lp.start], lp)
	}
	type loopState struct {
		start     int
		end       int
		remaining int
	}
	var stack []loopState
	activeLoops := make(map[int]int)

	i := 0
	for i < len(pt.events) {
		// apply tempo changes at this position
		for tempoIdx < len(pt.tempos) && pt.tempos[tempoIdx].index == i {
			tempo = pt.tempos[tempoIdx].tempo
			tempoIdx++
		}

		if lps, ok := loopMap[i]; ok {
			for _, lp := range lps {
				if activeLoops[lp.start] == 0 {
					stack = append(stack, loopState{start: lp.start, end: lp.end, remaining: lp.repeat - 1})
					activeLoops[lp.start] = 1
				}
			}
		}

		ev := pt.events[i]
		durMS := int((ev.beats / 2) * float64(60000/tempo))
		noteMS := durMS * 9 / 10
		restMS := durMS - noteMS

		v := velocity
		if len(ev.keys) > 1 {
			v = v * inst.chord / 100
		} else {
			v = v * inst.melody / 100
		}
		v = v * ev.volume / 10
		if v < 1 {
			v = 1
		} else if v > 127 {
			v = 127
		}
		for _, k := range ev.keys {
			key := k + inst.octave*12
			if key < 0 || key > 127 {
				continue
			}
			notes = append(notes, Note{
				Key:      key,
				Velocity: v,
				Start:    time.Duration(startMS) * time.Millisecond,
				Duration: time.Duration(noteMS) * time.Millisecond,
			})
		}
		startMS += noteMS + restMS
		i++

		for len(stack) > 0 && i == stack[len(stack)-1].end {
			top := &stack[len(stack)-1]
			if top.remaining > 0 {
				top.remaining--
				i = top.start
				// reset tempo to state at loop start
				tempo = pt.tempo
				tempoIdx = 0
				for tempoIdx < len(pt.tempos) && pt.tempos[tempoIdx].index <= i {
					tempo = pt.tempos[tempoIdx].tempo
					tempoIdx++
				}
			} else {
				delete(activeLoops, top.start)
				stack = stack[:len(stack)-1]
			}
		}
	}
	return notes
}

// parseClanLordTune converts Clan Lord music notation into parsed events at
// the default tempo of 120 BPM.
func parseClanLordTune(s string) parsedTune {
	return parseClanLordTuneWithTempo(s, 120)
}

// parseClanLordTuneWithTempo converts Clan Lord music notation into parsed
// events using the provided tempo in BPM. It also records loop markers,
// tempo changes and volume modifiers.
func parseClanLordTuneWithTempo(s string, tempo int) parsedTune {
	if tempo <= 0 {
		tempo = 120
	}
	pt := parsedTune{tempo: tempo}
	octave := 4
	volume := 10
	i := 0
	var loopStarts []int
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
		case '+', '-', '=', '/', '\\':
			handleOctave(&octave, c)
			i++
		case 'p': // rest
			i++
			beats := durationBlack
			if i < len(s) && s[i] >= '1' && s[i] <= '9' {
				beats = float64(s[i] - '0')
				i++
			}
			pt.events = append(pt.events, noteEvent{beats: beats, volume: volume})
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
			beats := defaultChordDuration
			if i < len(s) && s[i] >= '1' && s[i] <= '9' {
				beats = float64(s[i] - '0')
				i++
			}
			if len(keys) > 0 {
				pt.events = append(pt.events, noteEvent{keys: keys, beats: beats, volume: volume})
			}
		case '(':
			i++
			loopStarts = append(loopStarts, len(pt.events))
		case ')':
			i++
			count := 1
			if i < len(s) && s[i] >= '1' && s[i] <= '9' {
				count = int(s[i] - '0')
				i++
			}
			if len(loopStarts) > 0 {
				start := loopStarts[len(loopStarts)-1]
				loopStarts = loopStarts[:len(loopStarts)-1]
				pt.loops = append(pt.loops, loopMarker{start: start, end: len(pt.events), repeat: count})
			}
		case '@':
			i++
			sign := byte(0)
			if i < len(s) && (s[i] == '+' || s[i] == '-' || s[i] == '=') {
				sign = s[i]
				i++
			}
			val := 0
			for i < len(s) && s[i] >= '0' && s[i] <= '9' {
				val = val*10 + int(s[i]-'0')
				i++
			}
			newTempo := 120
			switch sign {
			case '+':
				newTempo = tempo + val
			case '-':
				newTempo = tempo - val
			default:
				if val == 0 {
					newTempo = 120
				} else {
					newTempo = val
				}
			}
			if newTempo < 60 {
				newTempo = 60
			}
			if newTempo > 180 {
				newTempo = 180
			}
			tempo = newTempo
			pt.tempos = append(pt.tempos, tempoEvent{index: len(pt.events), tempo: tempo})
		case '%', '{', '}':
			cmd := c
			i++
			val := 0
			if i < len(s) && s[i] >= '1' && s[i] <= '9' {
				val = int(s[i] - '0')
				i++
			}
			switch cmd {
			case '%':
				if val == 0 {
					volume = 10
				} else {
					volume = val
				}
			case '{':
				if val == 0 {
					val = 1
				}
				volume -= val
			case '}':
				if val == 0 {
					val = 1
				}
				volume += val
			}
			if volume < 1 {
				volume = 1
			}
			if volume > 10 {
				volume = 10
			}
		default:
			if isNoteLetter(c) {
				k, beats := parseNoteCL(s, &i, &octave)
				if k != 0 {
					pt.events = append(pt.events, noteEvent{keys: []int{k}, beats: beats, volume: volume})
				}
			} else {
				i++
			}
		}
	}
	return pt
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
	pt := parseClanLordTuneWithTempo(notes, tempo)
	instData := instruments[inst]
	prog := instData.program
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
	notesOut := eventsToNotes(pt, instData, vel)
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
