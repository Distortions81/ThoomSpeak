package main

import (
	"sync"
	"time"

	ttsspeech "github.com/go-tts/tts/pkg/speech"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

var (
	chatTTSMu    sync.Mutex
	ttsPlayers   = make(map[*audio.Player]struct{})
	ttsPlayersMu sync.Mutex
)

func stopAllTTS() {
	ttsPlayersMu.Lock()
	defer ttsPlayersMu.Unlock()
	for p := range ttsPlayers {
		_ = p.Close()
		delete(ttsPlayers, p)
	}
}

func speakChatMessage(msg string) {
	if audioContext == nil || blockTTS || gs.Mute {
		return
	}
	go func(text string) {
		chatTTSMu.Lock()
		defer chatTTSMu.Unlock()

		rc, err := ttsspeech.FromText(text, "en")
		if err != nil {
			logError("chat tts: %v", err)
			return
		}
		defer rc.Close()

		stream, err := mp3.DecodeWithSampleRate(44100, rc)
		if err != nil {
			logError("chat tts decode: %v", err)
			return
		}

		p, err := audioContext.NewPlayer(stream)
		if err != nil {
			logError("chat tts player: %v", err)
			return
		}

		ttsPlayersMu.Lock()
		ttsPlayers[p] = struct{}{}
		ttsPlayersMu.Unlock()

		vol := gs.ChatTTSVolume
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
	}(msg)
}
