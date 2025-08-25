package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

var (
	chatTTSMu    sync.Mutex
	ttsPlayers   = make(map[*audio.Player]struct{})
	ttsPlayersMu sync.Mutex
	chatTTSQueue = make(chan string, 16)

	piperPath   string
	piperModel  string
	piperConfig string
)

func init() {
	var err error
	piperPath, piperModel, piperConfig, err = preparePiper(dataDirPath)
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
	if piperPath == "" {
		logError("chat tts: piper not initialized")
		return
	}

	wavData, err := synthesizeWithPiper(text)
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

func preparePiper(dataDir string) (string, string, string, error) {
	piperDir := filepath.Join(dataDir, "piper")
	binDir := filepath.Join(piperDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", "", "", err
	}

	var archiveName, binName string
	switch runtime.GOOS {
	case "linux":
		binName = "piper"
		switch runtime.GOARCH {
		case "amd64":
			archiveName = "piper_linux_x86_64.tar.gz"
		case "arm64":
			archiveName = "piper_linux_aarch64.tar.gz"
		case "arm":
			archiveName = "piper_linux_armv7l.tar.gz"
		default:
			return "", "", "", fmt.Errorf("unsupported arch %s", runtime.GOARCH)
		}
	case "darwin":
		binName = "piper"
		switch runtime.GOARCH {
		case "amd64":
			archiveName = "piper_macos_x64.tar.gz"
		case "arm64":
			archiveName = "piper_macos_aarch64.tar.gz"
		default:
			return "", "", "", fmt.Errorf("unsupported arch %s", runtime.GOARCH)
		}
	case "windows":
		binName = "piper.exe"
		archiveName = "piper_windows_amd64.zip"
	default:
		return "", "", "", fmt.Errorf("unsupported OS %s", runtime.GOOS)
	}

	binPath := filepath.Join(binDir, binName)
	if _, err := os.Stat(binPath); err != nil {
		archivePath := filepath.Join(piperDir, archiveName)
		if err := extractArchive(archivePath, binDir); err != nil {
			return "", "", "", err
		}
		_ = os.Chmod(binPath, 0o755)
	}

	voice := "en_US-hfc_female-medium"
	model := filepath.Join(piperDir, "voices", voice, voice+".onnx")
	cfg := filepath.Join(piperDir, "voices", voice, voice+".onnx.json")
	return binPath, model, cfg, nil
}

func extractArchive(src, dst string) error {
	if strings.HasSuffix(src, ".tar.gz") {
		f, err := os.Open(src)
		if err != nil {
			return err
		}
		defer f.Close()
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		tr := tar.NewReader(gz)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			target := filepath.Join(dst, hdr.Name)
			if hdr.Typeflag == tar.TypeDir {
				if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
					return err
				}
				continue
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
			_ = os.Chmod(target, os.FileMode(hdr.Mode))
		}
		return nil
	}
	if strings.HasSuffix(src, ".zip") {
		zr, err := zip.OpenReader(src)
		if err != nil {
			return err
		}
		defer zr.Close()
		for _, f := range zr.File {
			target := filepath.Join(dst, f.Name)
			if f.FileInfo().IsDir() {
				if err := os.MkdirAll(target, 0o755); err != nil {
					return err
				}
				continue
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			rc, err := f.Open()
			if err != nil {
				return err
			}
			out, err := os.Create(target)
			if err != nil {
				rc.Close()
				return err
			}
			if _, err := io.Copy(out, rc); err != nil {
				rc.Close()
				out.Close()
				return err
			}
			rc.Close()
			out.Close()
			_ = os.Chmod(target, f.Mode())
		}
		return nil
	}
	return fmt.Errorf("unknown archive format: %s", src)
}

func synthesizeWithPiper(text string) ([]byte, error) {
	if piperPath == "" || piperModel == "" {
		return nil, fmt.Errorf("piper not initialized")
	}
	var out bytes.Buffer
	cmd := exec.Command(piperPath, "--model", piperModel, "--config", piperConfig, "--output_file", "-")
	cmd.Stdin = strings.NewReader(text)
	cmd.Stdout = &out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("piper run: %v: %s", err, stderr.String())
	}
	return out.Bytes(), nil
}
