package main

import (
	"bytes"
	"io"
	"sync"
	"time"

	ttsspeech "github.com/go-tts/tts/pkg/speech"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

var chatTTSMu sync.Mutex

func speakChatMessage(msg string) {
	if audioContext == nil {
		return
	}
	go func(text string) {
		chatTTSMu.Lock()
		defer chatTTSMu.Unlock()

		rc, err := ttsspeech.FromText(text, "en")
		if err != nil {
			logDebug("chat tts: %v", err)
			return
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			logDebug("chat tts read: %v", err)
			return
		}

		stream, err := mp3.Decode(audioContext, bytes.NewReader(data))
		if err != nil {
			logDebug("chat tts decode: %v", err)
			return
		}

		p, err := audioContext.NewPlayer(stream)
		if err != nil {
			logDebug("chat tts player: %v", err)
			return
		}
		vol := gs.ChatTTSVolume * gs.Volume
		if gs.Mute {
			vol = 0
		}
		p.SetVolume(vol)
		p.Play()
		for p.IsPlaying() {
			time.Sleep(100 * time.Millisecond)
		}
		_ = p.Close()
	}(msg)
}
