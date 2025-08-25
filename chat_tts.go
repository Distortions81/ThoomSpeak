package main

import (
	"bytes"
	"strings"
	"sync"
	"time"

	jenny "github.com/amitybell/piper-voice-jenny"
	piper "github.com/fresh-cut/piper"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

var (
	chatTTSMu    sync.Mutex
	ttsPlayers   = make(map[*audio.Player]struct{})
	ttsPlayersMu sync.Mutex
	chatTTSQueue = make(chan string, 16)
	ttsEngine    *piper.TTS
)

func init() {
	var err error
	ttsEngine, err = piper.New("", jenny.Asset)
	if err != nil {
		logError("chat tts init: %v", err)
	}
	go chatTTSWorker()
}

func stopAllTTS() {
	ttsPlayersMu.Lock()
	defer ttsPlayersMu.Unlock()
	for p := range ttsPlayers {
		_ = p.Close()
		delete(ttsPlayers, p)
	}
}

func chatTTSWorker() {
	for msg := range chatTTSQueue {
		msgs := []string{msg}
		timer := time.NewTimer(200 * time.Millisecond)
	collect:
		for {
			select {
			case m := <-chatTTSQueue:
				msgs = append(msgs, m)
			case <-timer.C:
				break collect
			}
		}
		timer.Stop()
		go playChatTTS(strings.Join(msgs, ". "))
	}
}

func playChatTTS(text string) {
	if audioContext == nil || blockTTS || gs.Mute {
		return
	}
	if ttsEngine == nil {
		logError("chat tts: tts engine not initialized")
		return
	}

	wavData, err := ttsEngine.Synthesize(text)
	if err != nil {
		logError("chat tts synthesize: %v", err)
		return
	}
	stream, err := wav.DecodeWithSampleRate(audioContext.SampleRate(), bytes.NewReader(wavData))
	if err != nil {
		logError("chat tts decode: %v", err)
		return
	}

	chatTTSMu.Lock()
	defer chatTTSMu.Unlock()

	p, err := audioContext.NewPlayer(stream)
	if err != nil {
		logError("chat tts player: %v", err)
		return
	}

	ttsPlayersMu.Lock()
	ttsPlayers[p] = struct{}{}
	ttsPlayersMu.Unlock()

	vol := gs.MasterVolume * gs.ChatTTSVolume
	if gs.Mute {
		vol = 0
	}
	p.SetVolume(vol)
	p.Play()
	for p.IsPlaying() {
		time.Sleep(100 * time.Millisecond)
	}
	_ = p.Close()

	ttsPlayersMu.Lock()
	delete(ttsPlayers, p)
	ttsPlayersMu.Unlock()
}

func speakChatMessage(msg string) {
	if audioContext == nil || blockTTS || gs.Mute {
		if audioContext == nil {
			logError("chat tts: audio context is nil")
		}
		if blockTTS {
			logDebug("chat tts: tts blocked")
		}
		if gs.Mute {
			logDebug("chat tts: client muted")
		}
		return
	}
	select {
	case chatTTSQueue <- msg:
	default:
		logDebug("chat tts: queue full, dropping message")
	}
}
