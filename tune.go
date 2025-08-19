package main

import (
	"math"
	"unicode"
)

type noteEvent struct {
	freqs []float64
	durMS int
}

func dbToGain(db float64) float64 { return math.Pow(10, db/20) }

// playClanLordTune decodes a Clan Lord music string into notes and plays it
// using a simple sine wave synthesizer. The implementation only supports a
// subset of the original format but is sufficient for basic melodies.
func playClanLordTune(tune string) {
	if audioContext == nil {
		return
	}
	events := parseClanLordTune(tune)
	if len(events) == 0 {
		return
	}
	rate := audioContext.SampleRate()
	buf := make([]byte, 0)
	for _, ev := range events {
		samples := synthChord(ev.freqs, rate, ev.durMS)
		for _, v := range samples {
			buf = append(buf, byte(v), byte(v>>8))
		}
	}
	p := audioContext.NewPlayerFromBytes(buf)
	p.SetVolume(dbToGain(-14))
	p.Play()
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
			var freqs []float64
			for i < len(s) && s[i] != ']' {
				if handleOctave(&octave, s[i]) {
					i++
					continue
				}
				if isNoteLetter(s[i]) {
					f, _ := parseNoteCL(s, &i, &octave)
					if f != 0 {
						freqs = append(freqs, f)
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
			if len(freqs) > 0 {
				events = append(events, noteEvent{freqs, int(durBeats * float64(quarter))})
			}
		default:
			if isNoteLetter(c) {
				f, beats := parseNoteCL(s, &i, &octave)
				if f != 0 {
					events = append(events, noteEvent{[]float64{f}, int(beats * float64(quarter))})
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

// parseNoteCL parses a single note and returns its frequency and beat length.
func parseNoteCL(s string, i *int, octave *int) (float64, float64) {
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
			return midiToFreq(pitch), beats
		}
	}
	return midiToFreq(pitch), beats
}

func midiToFreq(m int) float64 { return 440 * math.Pow(2, float64(m-69)/12) }

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

// synthChord mixes one or more sine waves for the given duration.
func synthChord(freqs []float64, rate, durMS int) []int16 {
	n := rate * durMS / 1000
	samples := make([]int16, n)
	if len(freqs) == 0 {
		return samples
	}
	gain := 1.0 / float64(len(freqs))
	for _, f := range freqs {
		for i := 0; i < n; i++ {
			v := math.Sin(2 * math.Pi * f * float64(i) / float64(rate))
			samples[i] += int16(v * gain * math.MaxInt16)
		}
	}
	return samples
}
