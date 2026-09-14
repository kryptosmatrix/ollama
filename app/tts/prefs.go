package tts

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Prefs are the non-secret speech settings. They must never contain an API key.
type Prefs struct {
	VoiceID           string  `json:"voice_id"`
	ModelID           string  `json:"model_id"`
	Speed             float64 `json:"speed"`
	CacheEnabled      bool    `json:"cache_enabled"`
	CacheClearPending bool    `json:"cache_clear_pending"`
}

func DefaultPrefs() Prefs {
	return Prefs{
		ModelID: ModelFlash,
		Speed:   1.0,
	}
}

type PrefStore struct {
	Path string
	mu   sync.Mutex
}

func (s *PrefStore) path() (string, error) {
	if strings.TrimSpace(s.Path) != "" {
		return s.Path, nil
	}
	return defaultPrefsPath()
}

func (s *PrefStore) Load() (Prefs, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *PrefStore) loadLocked() (Prefs, error) {
	path, err := s.path()
	if err != nil {
		return Prefs{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultPrefs(), nil
		}
		return Prefs{}, fmt.Errorf("read tts prefs %s: %w", path, err)
	}
	prefs := DefaultPrefs()
	if len(strings.TrimSpace(string(data))) == 0 {
		return prefs, nil
	}
	if err := json.Unmarshal(data, &prefs); err != nil {
		return Prefs{}, fmt.Errorf("parse tts prefs %s: %w", path, err)
	}
	if !SupportedModel(prefs.ModelID) {
		prefs.ModelID = ModelFlash
	}
	if prefs.Speed < MinSpeed || prefs.Speed > MaxSpeed || prefs.Speed != prefs.Speed {
		prefs.Speed = 1.0
	}
	return prefs, nil
}

func (s *PrefStore) Save(prefs Prefs) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(prefs)
}

func (s *PrefStore) saveLocked(prefs Prefs) error {
	path, err := s.path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	return writeFilePrivate(path, append(data, '\n'))
}

func (s *PrefStore) Update(mutate func(*Prefs) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prefs, err := s.loadLocked()
	if err != nil {
		return err
	}
	if err := mutate(&prefs); err != nil {
		return err
	}
	return s.saveLocked(prefs)
}
