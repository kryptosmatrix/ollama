package tts

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const pendingTTL = 10 * time.Minute
const maxPending = 8

type pendingAudio struct {
	req    SynthesisRequest
	audio  []byte
	expiry time.Time
}

// Chunk is one synthesised piece plus the headers the client needs to continue.
type Chunk struct {
	Audio       []byte
	Index       int
	Count       int
	Fingerprint string
	CacheHit    bool
}

// Status is what GET /api/v1/tts/status returns. It must never include the key.
type Status struct {
	HasAPIKey         bool    `json:"has_api_key"`
	VoiceID           string  `json:"voice_id"`
	ModelID           string  `json:"model_id"`
	Speed             float64 `json:"speed"`
	CacheEnabled      bool    `json:"cache_enabled"`
	CacheClearPending bool    `json:"cache_clear_pending"`
	SecretStore       string  `json:"secret_store"`
}

// SettingsPatch is the non-secret write from Settings.
type SettingsPatch struct {
	VoiceID      *string  `json:"voice_id"`
	ModelID      *string  `json:"model_id"`
	Speed        *float64 `json:"speed"`
	CacheEnabled *bool    `json:"cache_enabled"`
}

// Service is the production TTS stack used by the desktop UI server.
type Service struct {
	Secrets SecretStore
	Prefs   *PrefStore
	Cache   *Cache
	Client  Synthesizer
	Now     func() time.Time

	mu      sync.Mutex
	pending map[string]pendingAudio
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Default constructs the production stores. Tests must not call this without
// OLLAMA_TTS_SECRETS / PREFS / CACHE pointing at a temp directory.
func Default() *Service {
	secrets := DefaultSecretStore()
	prefs := &PrefStore{}
	cacheRoot, _ := defaultCacheRoot()
	return &Service{
		Secrets: secrets,
		Prefs:   prefs,
		Cache:   &Cache{Root: cacheRoot, Secrets: secrets},
		Client:  &Client{},
		pending: map[string]pendingAudio{},
	}
}

func Isolated(dir string) *Service {
	secrets := &FileSecretStore{Path: filepath.Join(dir, "secrets.json")}
	prefs := &PrefStore{Path: filepath.Join(dir, "prefs.json")}
	cache := &Cache{Root: filepath.Join(dir, "cache"), Secrets: secrets}
	return &Service{
		Secrets: secrets,
		Prefs:   prefs,
		Cache:   cache,
		Client:  &Client{},
		pending: map[string]pendingAudio{},
	}
}

func (s *Service) Status() (Status, error) {
	key, err := s.Secrets.LoadAPIKey()
	if err != nil {
		return Status{}, err
	}
	prefs, err := s.Prefs.Load()
	if err != nil {
		return Status{}, err
	}
	return Status{
		HasAPIKey:         key != "",
		VoiceID:           prefs.VoiceID,
		ModelID:           prefs.ModelID,
		Speed:             prefs.Speed,
		CacheEnabled:      prefs.CacheEnabled,
		CacheClearPending: prefs.CacheClearPending,
		SecretStore:       s.Secrets.Description(),
	}, nil
}

func (s *Service) PutKey(key string) error {
	if err := s.markClearPending(); err != nil {
		return err
	}
	if err := s.Secrets.SaveAPIKey(strings.TrimSpace(key)); err != nil {
		return err
	}
	s.dropPending()
	if err := s.Cache.Clear(); err != nil {
		return err
	}
	return s.clearClearPending()
}

func (s *Service) DeleteKey() error {
	if err := s.markClearPending(); err != nil {
		return err
	}
	if err := s.Secrets.DeleteAPIKey(); err != nil {
		return err
	}
	s.dropPending()
	if err := s.Cache.Clear(); err != nil {
		return err
	}
	return s.clearClearPending()
}

func (s *Service) PatchSettings(patch SettingsPatch) error {
	err := s.Prefs.Update(func(p *Prefs) error {
		if patch.VoiceID != nil {
			p.VoiceID = strings.TrimSpace(*patch.VoiceID)
		}
		if patch.ModelID != nil {
			if !SupportedModel(*patch.ModelID) {
				return httpErrorf(http.StatusBadRequest, "Choose Flash v2.5 or Multilingual v2.")
			}
			p.ModelID = *patch.ModelID
		}
		if patch.Speed != nil {
			if *patch.Speed != *patch.Speed || *patch.Speed < MinSpeed || *patch.Speed > MaxSpeed {
				return httpErrorf(http.StatusBadRequest, "Speech speed must be between 0.70 and 1.20.")
			}
			p.Speed = *patch.Speed
		}
		if patch.CacheEnabled != nil {
			was := p.CacheEnabled
			p.CacheEnabled = *patch.CacheEnabled
			if was && !*patch.CacheEnabled {
				p.CacheClearPending = true
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if patch.CacheEnabled != nil && !*patch.CacheEnabled {
		s.dropPending()
		if err := s.Cache.Clear(); err != nil {
			return err
		}
		return s.clearClearPending()
	}
	return nil
}

func (s *Service) ClearCache() error {
	if err := s.markClearPending(); err != nil {
		return err
	}
	s.dropPending()
	if err := s.Cache.Clear(); err != nil {
		return err
	}
	return s.clearClearPending()
}

func (s *Service) ListVoices(ctx context.Context) ([]Voice, error) {
	key, err := s.requireKey()
	if err != nil {
		return nil, err
	}
	return s.Client.ListVoices(ctx, key)
}

func (s *Service) Speak(ctx context.Context, text string, chunkIndex int) (*Chunk, error) {
	key, err := s.requireKey()
	if err != nil {
		return nil, err
	}
	prefs, err := s.Prefs.Load()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(prefs.VoiceID) == "" {
		return nil, httpErrorf(http.StatusBadRequest, "Choose an ElevenLabs voice in Settings.")
	}
	if !SupportedModel(prefs.ModelID) {
		return nil, httpErrorf(http.StatusBadRequest, "Choose Flash v2.5 or Multilingual v2.")
	}
	if prefs.Speed < MinSpeed || prefs.Speed > MaxSpeed {
		return nil, httpErrorf(http.StatusBadRequest, "Speech speed must be between 0.70 and 1.20.")
	}
	chunks := Chunks(text, MaxChunkRunes(prefs.ModelID))
	if len(chunks) == 0 {
		return nil, httpErrorf(http.StatusBadRequest, "Nothing to speak.")
	}
	if chunkIndex < 0 || chunkIndex >= len(chunks) {
		return nil, httpErrorf(http.StatusBadRequest, "That speech chunk does not exist.")
	}
	req := SynthesisRequest{
		Text:         chunks[chunkIndex],
		VoiceID:      strings.TrimSpace(prefs.VoiceID),
		ModelID:      prefs.ModelID,
		Speed:        prefs.Speed,
		AccountScope: AccountScope(key),
	}
	fp := Fingerprint(req)
	cacheOK := prefs.CacheEnabled && !prefs.CacheClearPending
	var audio []byte
	hit := false
	if cacheOK {
		audio, err = s.Cache.Lookup(req)
		if err != nil {
			audio = nil
		} else if audio != nil {
			hit = true
		}
	}
	if !hit {
		audio, err = s.Client.Synthesize(ctx, req, key)
		if err != nil {
			return nil, err
		}
		if cacheOK {
			s.holdPending(fp, req, audio)
		}
	}
	return &Chunk{
		Audio:       audio,
		Index:       chunkIndex,
		Count:       len(chunks),
		Fingerprint: fp,
		CacheHit:    hit,
	}, nil
}

func (s *Service) Commit(fingerprint string) error {
	prefs, err := s.Prefs.Load()
	if err != nil {
		return err
	}
	if !prefs.CacheEnabled || prefs.CacheClearPending {
		return nil
	}
	s.mu.Lock()
	pending, ok := s.pending[fingerprint]
	if ok {
		delete(s.pending, fingerprint)
	}
	s.mu.Unlock()
	if !ok {
		return nil
	}
	if s.now().After(pending.expiry) {
		return nil
	}
	return s.Cache.Store(pending.req, pending.audio)
}

func (s *Service) requireKey() (string, error) {
	key, err := s.Secrets.LoadAPIKey()
	if err != nil {
		return "", err
	}
	if key == "" {
		return "", httpErrorf(http.StatusBadRequest, "Add an ElevenLabs API key in Settings.")
	}
	return key, nil
}

func (s *Service) markClearPending() error {
	return s.Prefs.Update(func(p *Prefs) error {
		p.CacheClearPending = true
		return nil
	})
}

func (s *Service) clearClearPending() error {
	return s.Prefs.Update(func(p *Prefs) error {
		p.CacheClearPending = false
		return nil
	})
}

func (s *Service) holdPending(fp string, req SynthesisRequest, audio []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil {
		s.pending = map[string]pendingAudio{}
	}
	s.sweepPendingLocked()
	if len(s.pending) >= maxPending {
		var oldest string
		var when time.Time
		for k, v := range s.pending {
			if oldest == "" || v.expiry.Before(when) {
				oldest = k
				when = v.expiry
			}
		}
		delete(s.pending, oldest)
	}
	s.pending[fp] = pendingAudio{req: req, audio: audio, expiry: s.now().Add(pendingTTL)}
}

func (s *Service) dropPending() {
	s.mu.Lock()
	s.pending = map[string]pendingAudio{}
	s.mu.Unlock()
}

func (s *Service) sweepPendingLocked() {
	now := s.now()
	for k, v := range s.pending {
		if now.After(v.expiry) {
			delete(s.pending, k)
		}
	}
}
