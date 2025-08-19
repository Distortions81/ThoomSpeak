package main

import (
	"log"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const defaultInstrument = 10

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

// noteEvent represents a parsed tune event. A single event may contain multiple
// simultaneous notes (a chord).
type noteEvent struct {
	keys  []int
	durMS int
}

// playClanLordTune decodes a Clan Lord music string and plays it using the
// music package. The tune may optionally begin with an instrument index.
// For example: "3 cde" plays on instrument #3.
func playClanLordTune(tune string) {
	if audioContext == nil {
		return
	}

	inst := defaultInstrument
	fields := strings.Fields(tune)
	if len(fields) > 1 {
		if n, err := strconv.Atoi(fields[0]); err == nil && n >= 0 && n < len(instruments) {
			inst = n
			tune = strings.Join(fields[1:], " ")
		}
	}

	events := parseClanLordTune(tune)
	if len(events) == 0 {
		return
	}

	prog := instruments[inst].program
	oct := instruments[inst].octave

	notes := eventsToNotes(events, oct)
	if err := Play(audioContext, prog, notes); err != nil {
		log.Printf("play note: %v", err)
	}
}

// eventsToNotes converts parsed note events into synth notes with explicit start
// times. All notes in the same event (a chord) share the same start time.
func eventsToNotes(events []noteEvent, oct int) []Note {
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
				Velocity: 100,
				Start:    time.Duration(startMS) * time.Millisecond,
				Duration: time.Duration(ev.durMS) * time.Millisecond,
			})
		}
		startMS += ev.durMS
	}
	return notes
}

// parseClanLordTune converts Clan Lord music notation into a slice of note
// events. Only a minimal subset (notes, rests, octave changes, chords and
// durations) is implemented.
func parseClanLordTune(s string) []noteEvent {
	const tempo = 120
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
			durBeats := 1.0
			if i < len(s) && s[i] >= '1' && s[i] <= '9' {
				durBeats = 4.0 / float64(s[i]-'0')
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
			durBeats := 1.0
			if i < len(s) && s[i] >= '1' && s[i] <= '9' {
				durBeats = 4.0 / float64(s[i]-'0')
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
	beats := 1.0
	if isUpper {
		beats = 2.0
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
			beats = 4.0 / float64(ch-'0')
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
