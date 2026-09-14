package tts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "ollama-tts-pkg-")
	if err != nil {
		os.Exit(1)
	}
	os.Setenv(SecretsPathEnv, filepath.Join(dir, "secrets.json"))
	os.Setenv(PrefsPathEnv, filepath.Join(dir, "prefs.json"))
	os.Setenv(CachePathEnv, filepath.Join(dir, "cache"))
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
