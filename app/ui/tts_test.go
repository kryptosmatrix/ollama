//go:build windows || darwin

package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ollama/ollama/app/tts"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "ollama-ui-tts-")
	if err != nil {
		os.Exit(1)
	}
	os.Setenv(tts.SecretsPathEnv, filepath.Join(dir, "secrets.json"))
	os.Setenv(tts.PrefsPathEnv, filepath.Join(dir, "prefs.json"))
	os.Setenv(tts.CachePathEnv, filepath.Join(dir, "cache"))
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type countingSynth struct {
	hits atomic.Int32
}

func (c *countingSynth) Synthesize(ctx context.Context, req tts.SynthesisRequest, apiKey string) ([]byte, error) {
	c.hits.Add(1)
	return []byte("ID3fake-mpeg"), nil
}

func (c *countingSynth) ListVoices(ctx context.Context, apiKey string) ([]tts.Voice, error) {
	c.hits.Add(1)
	return []tts.Voice{{ID: "v1", Name: "Ada"}}, nil
}

func ttsServer(t *testing.T, synth *countingSynth) *Server {
	t.Helper()
	svc := tts.Isolated(t.TempDir())
	svc.Client = synth
	if err := svc.PutKey("sk-test"); err != nil {
		t.Fatal(err)
	}
	voice := "voice_one"
	if err := svc.PatchSettings(tts.SettingsPatch{VoiceID: &voice}); err != nil {
		t.Fatal(err)
	}
	return &Server{Token: "test-token-123", Dev: false, TTS: svc}
}

func TestSpeakWithoutCookieNeverReachesElevenLabs(t *testing.T) {
	synth := &countingSynth{}
	handler := ttsServer(t, synth).Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tts/speak", strings.NewReader(`{"text":"Hello","chunk_index":0}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Token is required") {
		t.Fatalf("body %s", rec.Body.String())
	}
	if synth.hits.Load() != 0 {
		t.Fatalf("ElevenLabs was contacted %d times without a cookie", synth.hits.Load())
	}
}

func TestSpeakWithCookieReachesSynthesizer(t *testing.T) {
	synth := &countingSynth{}
	server := ttsServer(t, synth)
	handler := server.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tts/speak", strings.NewReader(`{"text":"Hello","chunk_index":0}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "token", Value: server.Token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("content-type %s", rec.Header().Get("Content-Type"))
	}
	if synth.hits.Load() != 1 {
		t.Fatalf("hits %d", synth.hits.Load())
	}
}

func TestStatusNeverContainsTheKey(t *testing.T) {
	synth := &countingSynth{}
	server := ttsServer(t, synth)
	handler := server.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tts/status", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: server.Token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "sk-test") || strings.Contains(body, `"api_key"`) {
		t.Fatalf("leaked key: %s", body)
	}
	var st tts.Status
	if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if !st.HasAPIKey {
		t.Fatal("has_api_key")
	}
}

func TestPutKeyRejectsOverlong(t *testing.T) {
	synth := &countingSynth{}
	server := ttsServer(t, synth)
	handler := server.Handler()
	payload, _ := json.Marshal(map[string]string{"api_key": strings.Repeat("a", 8*1024+1)})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tts/key", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "token", Value: server.Token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
}
