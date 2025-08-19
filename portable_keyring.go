package main

import (
	"os"
	"path/filepath"
	"runtime"

	keyring "github.com/99designs/keyring"
)

// openPortableRing tries to open a system keyring using native backends and
// falls back to the encrypted file backend when necessary. The file backend can
// also be forced by setting KEYRING_FORCE_FILE=1.
func openPortableRing() (keyring.Keyring, error) {
	if os.Getenv("KEYRING_FORCE_FILE") == "1" {
		return openFileRing()
	}

	cfg := keyring.Config{
		ServiceName: keyringService,
	}

	switch runtime.GOOS {
	case "darwin":
		cfg.AllowedBackends = []keyring.BackendType{
			keyring.KeychainBackend,
			keyring.FileBackend,
		}
	case "linux":
		cfg.AllowedBackends = []keyring.BackendType{
			keyring.SecretServiceBackend,
			keyring.KWalletBackend,
			keyring.PassBackend,
			keyring.FileBackend,
		}
	case "windows":
		cfg.AllowedBackends = []keyring.BackendType{
			keyring.WinCredBackend,
			keyring.FileBackend,
		}
	default:
		return openFileRing()
	}

	cfg.FilePasswordFunc = keyring.FixedStringPrompt(os.Getenv("KEYRING_FILE_PASSPHRASE"))
	if cfg.FilePasswordFunc == nil {
		cfg.FilePasswordFunc = keyring.FixedStringPrompt("test-passphrase")
	}
	cfg.FileDir = defaultFileDir()
	cfg.PassPrefix = keyringService

	return keyring.Open(cfg)
}

// openFileRing opens the encrypted file keyring backend directly.
func openFileRing() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		ServiceName:     keyringService,
		AllowedBackends: []keyring.BackendType{keyring.FileBackend},
		FileDir:         defaultFileDir(),
		FilePasswordFunc: keyring.FixedStringPrompt(
			envOr("KEYRING_FILE_PASSPHRASE", "test-passphrase"),
		),
	})
}

// defaultFileDir returns the directory used by the file backend.
func defaultFileDir() string {
	if d := os.Getenv("KEYRING_FILE_DIR"); d != "" {
		return d
	}
	dir := filepath.Join(dataDirPath, "keyring")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
