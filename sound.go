package main

import (
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"gothoom/clsnd"
)

const (
	maxSounds = 64
	dbPad     = -3
)

var (
	soundMu  sync.Mutex
	clSounds *clsnd.CLSounds
	pcmCache = make(map[uint16][]byte)

	audioContext *audio.Context
	soundPlayers = make(map[*audio.Player]struct{})

	sndDumpOnce   sync.Once
	sndDumpMu     sync.Mutex
	dumpedSndIDs  = make(map[uint16]struct{})
	sndMetaWriter *csv.Writer
)

// stopAllSounds halts and disposes all currently playing audio players.
func stopAllSounds() {
	soundMu.Lock()
	defer soundMu.Unlock()
	for sp := range soundPlayers {
		_ = sp.Close()
		delete(soundPlayers, sp)
	}
}

// stopAllAudioPlayers stops and disposes every active audio player type.
func stopAllAudioPlayers() {
	stopAllSounds()
	stopAllTTS()
	stopAllMusic()
}

// playSound mixes the provided sound IDs and plays the result asynchronously.
// Each ID is loaded, mixed with simple clipping and then played at the current
// global volume. The function returns immediately after scheduling playback.
func playSound(ids []uint16) {
	if len(ids) == 0 || gs.Mute {
		return
	}
	go func(ids []uint16) {
		if gs.Mute {
			return
		}
		//logDebug("playSound %v called", ids)
		if blockSound {
			logDebug("playSound blocked by blockSound")
			return
		}
		if audioContext == nil {
			logDebug("playSound no audio context")
			return
		}

		var valid map[uint16]struct{}
		soundMu.Lock()
		c := clSounds
		soundMu.Unlock()
		if c != nil {
			vid := c.IDs()
			valid = make(map[uint16]struct{}, len(vid))
			for _, v := range vid {
				valid[uint16(v)] = struct{}{}
			}
		}

		sounds := make([][]byte, 0, len(ids))
		maxSamples := 0
		for _, id := range ids {
			if valid != nil {
				if _, ok := valid[id]; !ok {
					logDebug("playSound unknown id %d", id)
					continue
				}
			}
			pcm := loadSound(id)
			if pcm == nil {
				continue
			}
			sounds = append(sounds, pcm)
			if n := len(pcm) / 2; n > maxSamples {
				maxSamples = n
			}
		}
		if len(sounds) == 0 {
			logDebug("playSound no pcm returned")
			return
		}

		mixed := make([]int32, maxSamples)

		chunks := runtime.NumCPU()
		if chunks > maxSamples {
			chunks = maxSamples
		}
		chunkSize := (maxSamples + chunks - 1) / chunks

		var wg sync.WaitGroup
		maxCh := make(chan int32, chunks)

		for start := 0; start < maxSamples; start += chunkSize {
			end := start + chunkSize
			if end > maxSamples {
				end = maxSamples
			}
			wg.Add(1)
			go func(start, end int) {
				defer wg.Done()
				localMax := int32(0)
				for i := start; i < end; i++ {
					var sum int32
					for _, pcm := range sounds {
						if n := len(pcm) / 2; i < n {
							sample := int16(binary.LittleEndian.Uint16(pcm[2*i:]))
							sum += int32(sample)
						}
					}
					mixed[i] = sum
					if sum < 0 {
						sum = -sum
					}
					if sum > localMax {
						localMax = sum
					}
				}
				maxCh <- localMax
			}(start, end)
		}
		wg.Wait()
		close(maxCh)

		maxVal := int32(0)
		for v := range maxCh {
			if v > maxVal {
				maxVal = v
			}
		}

		// Apply peak normalization and reduce volume for overlapping sounds
		scale := 1 / float64(len(sounds))
		if maxVal > 0 {
			scale *= math.Min(1, 32767.0/float64(maxVal))
		}

		out := make([]byte, len(mixed)*2)

		wg = sync.WaitGroup{}
		for start := 0; start < len(mixed); start += chunkSize {
			end := start + chunkSize
			if end > len(mixed) {
				end = len(mixed)
			}
			wg.Add(1)
			go func(start, end int) {
				defer wg.Done()
				for i := start; i < end; i++ {
					v := int32(float64(mixed[i]) * scale)
					if v > 32767 {
						v = 32767
					} else if v < -32768 {
						v = -32768
					}
					binary.LittleEndian.PutUint16(out[2*i:], uint16(int16(v)))
				}
			}(start, end)
		}
		wg.Wait()

		p := audioContext.NewPlayerFromBytes(out)
		vol := gs.Volume
		if gs.Mute {
			vol = 0
		}
		p.SetVolume(vol)

		soundMu.Lock()
		for sp := range soundPlayers {
			if !sp.IsPlaying() {
				sp.Close()
				delete(soundPlayers, sp)
			}
		}
		if maxSounds > 0 && len(soundPlayers) >= maxSounds {
			soundMu.Unlock()
			logDebug("playSound too many sound players (%d)", len(soundPlayers))
			p.Close()
			return
		}
		soundPlayers[p] = struct{}{}
		soundMu.Unlock()

		//logDebug("playSound playing")
		p.Play()
	}(ids)
}

// initSoundContext initializes the global audio context.
func initSoundContext() {
	rate := 44100
	audioContext = audio.NewContext(rate)
}

func updateSoundVolume() {
	gameVol := gs.Volume
	ttsVol := gs.ChatTTSVolume
	musicVol := gs.MusicVolume
	if gs.Mute {
		gameVol = 0
		ttsVol = 0
		musicVol = 0
	}

	soundMu.Lock()
	players := make([]*audio.Player, 0, len(soundPlayers))
	for sp := range soundPlayers {
		players = append(players, sp)
	}
	soundMu.Unlock()

	ttsPlayersMu.Lock()
	tts := make([]*audio.Player, 0, len(ttsPlayers))
	for p := range ttsPlayers {
		tts = append(tts, p)
	}
	ttsPlayersMu.Unlock()

	musicPlayersMu.Lock()
	music := make([]*audio.Player, 0, len(musicPlayers))
	for p := range musicPlayers {
		music = append(music, p)
	}
	musicPlayersMu.Unlock()

	stopped := make([]*audio.Player, 0)
	for _, sp := range players {
		if sp.IsPlaying() {
			sp.SetVolume(gameVol)
		} else {
			stopped = append(stopped, sp)
		}
	}

	ttsStopped := make([]*audio.Player, 0)
	for _, p := range tts {
		if p.IsPlaying() {
			p.SetVolume(ttsVol)
		} else {
			ttsStopped = append(ttsStopped, p)
		}
	}

	musicStopped := make([]*audio.Player, 0)
	for _, p := range music {
		if p.IsPlaying() {
			p.SetVolume(musicVol)
		} else {
			musicStopped = append(musicStopped, p)
		}
	}

	if len(stopped) > 0 {
		soundMu.Lock()
		for _, sp := range stopped {
			delete(soundPlayers, sp)
			sp.Close()
		}
		soundMu.Unlock()
	}

	if len(ttsStopped) > 0 {
		ttsPlayersMu.Lock()
		for _, p := range ttsStopped {
			delete(ttsPlayers, p)
			p.Close()
		}
		ttsPlayersMu.Unlock()
	}

	if len(musicStopped) > 0 {
		musicPlayersMu.Lock()
		for _, p := range musicStopped {
			delete(musicPlayers, p)
			p.Close()
		}
		musicPlayersMu.Unlock()
	}
}

// fast xorshift32 PRNG
type rnd32 uint32

func (r *rnd32) next() float64 {
	x := uint32(*r)
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	*r = rnd32(x)
	// scale to [0,1)
	return float64(x) * (1.0 / 4294967296.0)
}

// u8 PCM (0..255) -> s16 PCM (-32768..32767) with TPDF dither and 257 scaling
func u8ToS16TPDF(data []byte, seed uint32) []int16 {
	out := make([]int16, len(data))
	r1, r2 := rnd32(seed|1), rnd32(seed*1664525+1013904223)

	for i, b := range data {
		// TPDF dither in [-0.5, +0.5): (rand - rand)
		noise := (r1.next() - r2.next()) * 0.5
		v := float64(b) + noise

		// Map 0..255 -> -32768..32767 using *257 then offset
		// (257 uses full 16-bit span slightly better than <<8)
		s := int32(math.Round(v*257.0)) - 32768
		if s > math.MaxInt16 {
			s = math.MaxInt16
		} else if s < math.MinInt16 {
			s = math.MinInt16
		}
		out[i] = int16(s)
	}
	return out
}

// applyFadeInOut applies a tiny fade to the start and end of the samples
// to avoid clicks when sounds begin or end abruptly. The fade length is
// approximately 5ms of audio.
func applyFadeInOut(samples []int16, rate int) {
	fade := 220
	if fade <= 1 {
		return
	}
	if len(samples) < 2*fade {
		fade = len(samples) / 2
		if fade <= 1 {
			return
		}
	}
	for i := 0; i < fade; i++ {
		inScale := float64(i) / float64(fade)
		samples[i] = int16(float64(samples[i]) * inScale)
		outScale := float64(fade-1-i) / float64(fade)
		idx := len(samples) - fade + i
		samples[idx] = int16(float64(samples[idx]) * outScale)

	}
}

// loadSound retrieves a sound by ID, resamples it to match the audio context's
// sample rate, and caches the resulting PCM bytes. The CL_Sounds archive is
// opened on first use and individual sounds are parsed lazily.
func loadSound(id uint16) []byte {
	//logDebug("loadSound(%d) called", id)
	if audioContext == nil {
		logDebug("loadSound(%d) no audio context", id)
		return nil
	}

	soundMu.Lock()
	if !gs.NoCaching {
		if pcm, ok := pcmCache[id]; ok {
			soundMu.Unlock()
			if pcm == nil {
				logDebug("loadSound(%d) cached as missing", id)
			} else {
				logDebug("loadSound(%d) cache hit (%d bytes)", id, len(pcm))
			}
			return pcm
		}
	}
	c := clSounds
	soundMu.Unlock()

	if c == nil {
		logDebug("loadSound(%d) CL sounds not loaded", id)
		return nil
	}

	//logDebug("loadSound(%d) fetching from archive", id)
	s, err := c.Get(uint32(id))
	if s == nil {
		if err != nil {
			logError("unable to decode sound %d: %v", id, err)
		} else {
			logError("missing sound %d", id)
		}
		if !gs.NoCaching {
			soundMu.Lock()
			pcmCache[id] = nil
			soundMu.Unlock()
		} else {
			clSounds.ClearCache()
		}
		return nil
	}
	statSoundLoaded(id)
	//logDebug("loadSound(%d) loaded %d Hz %d-bit %d bytes", id, s.SampleRate, s.Bits, len(s.Data))

	srcRate := int(s.SampleRate / 2)
	dstRate := audioContext.SampleRate()

	// Decode the sound data into 16-bit samples.
	var samples []int16
	switch s.Bits {
	case 8:
		if s.Channels > 1 {
			frames := len(s.Data) / int(s.Channels)
			mono := make([]byte, frames)
			for i := 0; i < frames; i++ {
				mono[i] = s.Data[i*int(s.Channels)]
			}
			samples = u8ToS16TPDF(mono, 0xC0FFEE)
		} else {
			samples = u8ToS16TPDF(s.Data, 0xC0FFEE)
		}
	case 16:
		if len(s.Data)%2 != 0 {
			s.Data = append(s.Data, 0x00)
		}
		if s.Channels > 1 {
			frameSize := int(s.Channels) * 2
			frames := len(s.Data) / frameSize
			samples = make([]int16, frames)
			for i := 0; i < frames; i++ {
				off := i * frameSize
				samples[i] = int16(binary.BigEndian.Uint16(s.Data[off : off+2]))
			}
		} else {
			samples = make([]int16, len(s.Data)/2)
			for i := 0; i < len(samples); i++ {
				samples[i] = int16(binary.BigEndian.Uint16(s.Data[2*i : 2*i+2]))
			}
		}
	default:
		log.Fatalf("Invalid number of bits: %v: ID: %v", s.Bits, id)
	}

	if srcRate != dstRate {
		samples = ResampleLanczosInt16PadDB(samples, srcRate, dstRate, dbPad)
	} else {
		samples = PadDB(samples, dbPad)
	}

	applyFadeInOut(samples, dstRate)

	pcm := make([]byte, len(samples)*2)
	for i, v := range samples {
		pcm[2*i] = byte(v)
		pcm[2*i+1] = byte(v >> 8)
	}

	if sndDump {
		dumpSound(id, s, pcm, dstRate)
	}

	if gs.NoCaching {
		clSounds.ClearCache()
	} else {
		soundMu.Lock()
		pcmCache[id] = pcm
		soundMu.Unlock()
		//logDebug("loadSound(%d) cached %d bytes", id, len(pcm))
	}
	return pcm
}

func dumpSound(id uint16, s *clsnd.Sound, pcm []byte, rate int) {
	sndDumpOnce.Do(func() {
		os.MkdirAll(filepath.Join("dump", "snd"), 0755)
		if f, err := os.Create(filepath.Join("dump", "snd", "metadata.csv")); err == nil {
			sndMetaWriter = csv.NewWriter(f)
			sndMetaWriter.Write([]string{"id", "origRate", "origChannels", "origBits", "bytes"})
		}
	})
	sndDumpMu.Lock()
	if _, ok := dumpedSndIDs[id]; ok {
		sndDumpMu.Unlock()
		return
	}
	dumpedSndIDs[id] = struct{}{}
	sndDumpMu.Unlock()

	fn := filepath.Join("dump", "snd", fmt.Sprintf("%d.wav", id))
	f, err := os.Create(fn)
	if err != nil {
		log.Printf("dump sound %d: %v", id, err)
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
	binary.LittleEndian.PutUint16(header[22:], 1)
	binary.LittleEndian.PutUint32(header[24:], uint32(rate))
	binary.LittleEndian.PutUint32(header[28:], uint32(rate*2))
	binary.LittleEndian.PutUint16(header[32:], 2)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], []byte("data"))
	binary.LittleEndian.PutUint32(header[40:], dataLen)

	if _, err := f.Write(header[:]); err != nil {
		log.Printf("dump sound %d header: %v", id, err)
		return
	}
	if _, err := f.Write(pcm); err != nil {
		log.Printf("dump sound %d data: %v", id, err)
		return
	}

	if sndMetaWriter != nil {
		sndMetaWriter.Write([]string{
			strconv.Itoa(int(id)),
			strconv.FormatUint(uint64(s.SampleRate), 10),
			strconv.FormatUint(uint64(s.Channels), 10),
			strconv.FormatUint(uint64(s.Bits), 10),
			strconv.Itoa(len(pcm)),
		})
		sndMetaWriter.Flush()
	}
}

// soundCacheStats returns the number of cached sounds and total bytes used.
func soundCacheStats() (count, bytes int) {
	soundMu.Lock()
	defer soundMu.Unlock()
	for _, pcm := range pcmCache {
		if pcm != nil {
			count++
			bytes += len(pcm)
		}
	}
	return
}
