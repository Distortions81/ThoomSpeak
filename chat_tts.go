package main

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	htgotts "github.com/hegedustibor/htgo-tts"
	"github.com/hegedustibor/htgo-tts/voices"
)

var (
	chatTTSMu    sync.Mutex
	ttsPlayers   = make(map[*audio.Player]struct{})
	ttsPlayersMu sync.Mutex
	chatSpeech   = htgotts.Speech{Folder: "audio", Language: voices.English}
)

func stopAllTTS() {
	ttsPlayersMu.Lock()
	defer ttsPlayersMu.Unlock()
	for p := range ttsPlayers {
		_ = p.Close()
		delete(ttsPlayers, p)
	}
}

func fetchTTS(text string) (io.ReadCloser, error) {
	fileName := fmt.Sprintf("chat_%d", time.Now().UnixNano())
	r, err := chatSpeech.CreateSpeechBuff(text, fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create speech: %w", err)
	}
	return io.NopCloser(r), nil
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
	go func(text string) {
		rc, err := fetchTTS(text)
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
		if closer, ok := any(stream).(io.Closer); ok {
			defer closer.Close()
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
	}(msg)
}
