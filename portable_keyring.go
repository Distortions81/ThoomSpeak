package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	keyring "github.com/99designs/keyring"
)

// openPortableRing tries to find a working keyring backend by actually storing
// and retrieving a test value. The detected backend is saved in the settings
// for future runs. KEYRING_FORCE_FILE=1 forces the encrypted file backend.
func openPortableRing() (keyring.Keyring, error) {
	_ = os.MkdirAll(dataDirPath, 0o755)
	if os.Getenv("KEYRING_FORCE_FILE") == "1" {
		ring, err := tryBackend(keyring.FileBackend)
		if err == nil {
			gs.KeyringBackend = string(keyring.FileBackend)
			saveSettings()
		}
		return ring, err
	}

	var backends []keyring.BackendType
	switch runtime.GOOS {
	case "darwin":
		backends = []keyring.BackendType{
			keyring.KeychainBackend,
			keyring.FileBackend,
		}
	case "linux":
		backends = []keyring.BackendType{
			keyring.SecretServiceBackend,
			keyring.KWalletBackend,
			keyring.PassBackend,
			keyring.FileBackend,
		}
	case "windows":
		backends = []keyring.BackendType{
			keyring.WinCredBackend,
			keyring.FileBackend,
		}
	default:
		backends = []keyring.BackendType{keyring.FileBackend}
	}

	if gs.KeyringBackend != "" {
		if ring, err := tryBackend(keyring.BackendType(gs.KeyringBackend)); err == nil {
			return ring, nil
		}
	}

	for _, b := range backends {
		ring, err := tryBackend(b)
		if err == nil {
			if gs.KeyringBackend != string(b) {
				gs.KeyringBackend = string(b)
				saveSettings()
			}
			return ring, nil
		}
	}

	return nil, errors.New("no usable keyring backend found")
}

func tryBackend(b keyring.BackendType) (keyring.Keyring, error) {
	var (
		ring keyring.Keyring
		err  error
	)
	if b == keyring.FileBackend {
		ring, err = openFileRing()
	} else {
		cfg := keyring.Config{
			ServiceName:     keyringService,
			AllowedBackends: []keyring.BackendType{b},
			FileDir:         defaultFileDir(),
			FilePasswordFunc: keyring.FixedStringPrompt(
				envOr("KEYRING_FILE_PASSPHRASE", "test-passphrase"),
			),
			PassPrefix: keyringService,
		}
		ring, err = keyring.Open(cfg)
	}
	if err != nil {
		return nil, err
	}
	const testKey = "__kr_test__"
	const testVal = "ok"
	if err := ring.Set(keyring.Item{Key: testKey, Data: []byte(testVal)}); err != nil {
		return nil, err
	}
	item, err := ring.Get(testKey)
	if err != nil {
		return nil, err
	}
	if string(item.Data) != testVal {
		return nil, errors.New("keyring test mismatch")
	}
	_ = ring.Remove(testKey)
	return ring, nil
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
