package tts

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const (
	SecretsPathEnv = "OLLAMA_TTS_SECRETS"
	PrefsPathEnv   = "OLLAMA_TTS_PREFS"
	CachePathEnv   = "OLLAMA_TTS_CACHE"

	keychainService    = "Ollama TTS"
	apiKeyAccount      = "elevenlabs-api-key"
	cacheMasterAccount = "speech-cache-master-secret-v1"
	maximumAPIKeyBytes = 8 * 1024
	masterSecretBytes  = 32
	secretsFilename    = "tts-secret.json"
	prefsFilename      = "tts-prefs.json"
)

// Protector encrypts a file at rest. Same split as MCP: location vs protection.
type Protector interface {
	Protect(plaintext []byte) ([]byte, error)
	Unprotect(ciphertext []byte) ([]byte, error)
	Describe() string
}

// SecretStore holds the ElevenLabs API key and the cache master secret.
type SecretStore interface {
	LoadAPIKey() (string, error)
	SaveAPIKey(key string) error
	DeleteAPIKey() error
	LoadCacheMaster() ([]byte, error)
	SaveCacheMaster(secret []byte) error
	DeleteCacheMaster() error
	Description() string
}

type protectedFile struct {
	Protected []byte `json:"protected"`
}

type secretPayload struct {
	APIKey      string `json:"api_key,omitempty"`
	CacheMaster []byte `json:"cache_master,omitempty"`
}

var fileLocks sync.Map

func lockPath(path string) func() {
	value, _ := fileLocks.LoadOrStore(path, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// FileSecretStore is a 0600 JSON file, optionally wrapped by a Protector.
type FileSecretStore struct {
	Path    string
	Protect Protector
}

func (s *FileSecretStore) path() (string, error) {
	if strings.TrimSpace(s.Path) != "" {
		return s.Path, nil
	}
	return defaultSecretsPath()
}

func (s *FileSecretStore) Description() string {
	path, err := s.path()
	if err != nil {
		path = "a file in your Ollama configuration directory"
	}
	if s.Protect != nil {
		return path + ", " + s.Protect.Describe()
	}
	return path + ", readable by any program running as you"
}

func (s *FileSecretStore) LoadAPIKey() (string, error) {
	payload, err := s.read()
	if err != nil {
		return "", err
	}
	return payload.APIKey, nil
}

func (s *FileSecretStore) SaveAPIKey(key string) error {
	if err := validateAPIKey(key); err != nil {
		return err
	}
	return s.update(func(p *secretPayload) error {
		p.APIKey = key
		return nil
	})
}

func (s *FileSecretStore) DeleteAPIKey() error {
	return s.update(func(p *secretPayload) error {
		p.APIKey = ""
		return nil
	})
}

func (s *FileSecretStore) LoadCacheMaster() ([]byte, error) {
	payload, err := s.read()
	if err != nil {
		return nil, err
	}
	if len(payload.CacheMaster) == 0 {
		return nil, nil
	}
	out := make([]byte, len(payload.CacheMaster))
	copy(out, payload.CacheMaster)
	return out, nil
}

func (s *FileSecretStore) SaveCacheMaster(secret []byte) error {
	if len(secret) != masterSecretBytes {
		return errors.New("cache master secret must be 32 bytes")
	}
	dup := make([]byte, len(secret))
	copy(dup, secret)
	return s.update(func(p *secretPayload) error {
		p.CacheMaster = dup
		return nil
	})
}

func (s *FileSecretStore) DeleteCacheMaster() error {
	return s.update(func(p *secretPayload) error {
		p.CacheMaster = nil
		return nil
	})
}

func (s *FileSecretStore) read() (secretPayload, error) {
	path, err := s.path()
	if err != nil {
		return secretPayload{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return secretPayload{}, nil
		}
		return secretPayload{}, fmt.Errorf("read tts secrets %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return secretPayload{}, nil
	}
	var envelope protectedFile
	if json.Unmarshal(data, &envelope) == nil && len(envelope.Protected) > 0 {
		if s.Protect == nil {
			return secretPayload{}, fmt.Errorf("tts secrets %s are encrypted and this build cannot read them; save the API key again", path)
		}
		plaintext, err := s.Protect.Unprotect(envelope.Protected)
		if err != nil {
			return secretPayload{}, fmt.Errorf("decrypt tts secrets %s: %w", path, err)
		}
		data = plaintext
	}
	var payload secretPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return secretPayload{}, fmt.Errorf("parse tts secrets %s: %w", path, err)
	}
	return payload, nil
}

func (s *FileSecretStore) update(mutate func(*secretPayload) error) error {
	path, err := s.path()
	if err != nil {
		return err
	}
	defer lockPath(path)()
	payload, err := s.read()
	if err != nil {
		return err
	}
	if err := mutate(&payload); err != nil {
		return err
	}
	return s.write(payload, path)
}

func (s *FileSecretStore) write(payload secretPayload, path string) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tts secrets: %w", err)
	}
	if s.Protect != nil {
		ciphertext, err := s.Protect.Protect(data)
		if err != nil {
			return fmt.Errorf("encrypt tts secrets: %w", err)
		}
		if data, err = json.MarshalIndent(protectedFile{Protected: ciphertext}, "", "  "); err != nil {
			return fmt.Errorf("marshal tts secrets envelope: %w", err)
		}
	}
	return writeFilePrivate(path, append(data, '\n'))
}

func validateAPIKey(key string) error {
	if key == "" || !utf8OK(key) {
		return httpErrorf(400, "The ElevenLabs API key is empty or not valid UTF-8.")
	}
	if len(key) > maximumAPIKeyBytes {
		return httpErrorf(400, "The ElevenLabs API key is too long.")
	}
	return nil
}

func utf8OK(s string) bool {
	return strings.ToValidUTF8(s, "\uFFFD") == s
}

func defaultSecretsPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv(SecretsPathEnv)); p != "" {
		return filepath.Abs(p)
	}
	return ollamaFile(secretsFilename)
}

func defaultPrefsPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv(PrefsPathEnv)); p != "" {
		return filepath.Abs(p)
	}
	return ollamaFile(prefsFilename)
}

func defaultCacheRoot() (string, error) {
	if p := strings.TrimSpace(os.Getenv(CachePathEnv)); p != "" {
		return filepath.Abs(p)
	}
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Caches", "Ollama", "speech-cache", "v1"), nil
	case "windows":
		local := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
		if local == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			local = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(local, "Ollama", "speech-cache", "v1"), nil
	default:
		return ollamaFile(filepath.Join("speech-cache", "v1"))
	}
}

func ollamaFile(name string) (string, error) {
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return filepath.Join(xdg, "ollama", name), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ollama", name), nil
}

func DefaultSecretStore() SecretStore {
	if strings.TrimSpace(os.Getenv(SecretsPathEnv)) != "" {
		return &FileSecretStore{Protect: platformProtector()}
	}
	return platformSecretStore()
}
