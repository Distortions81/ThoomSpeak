package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

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

func fetchTTS(ctx context.Context, text, lang string) (io.ReadCloser, error) {
	u := fmt.Sprintf("http://translate.google.com/translate_tts?ie=UTF-8&textlen=%d&client=tw-ob&q=%s&tl=%s", len(text), url.QueryEscape(text), lang)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query google: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to query google: response status %d: %s", resp.StatusCode, string(body))
	}
	return resp.Body, nil
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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		rc, err := fetchTTS(ctx, text, "en")
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
