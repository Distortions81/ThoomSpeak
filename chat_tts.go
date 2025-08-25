package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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
	if err := os.MkdirAll(chatSpeech.Folder, 0o755); err != nil {
		return nil, fmt.Errorf("create audio dir: %w", err)
	}

	// Build request to Google Translate TTS endpoint.
	q := url.QueryEscape(text)
	ttsURL := fmt.Sprintf("https://translate.googleapis.com/translate_tts?ie=UTF-8&client=tw-ob&tl=%s&q=%s", chatSpeech.Language, q)
	req, err := http.NewRequest(http.MethodGet, ttsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("tts request: %w", err)
	}
	// Set a User-Agent so the request is not rejected.
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tts fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("tts fetch: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tts read: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("tts: empty response")
	}

	return io.NopCloser(bytes.NewReader(data)), nil
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
