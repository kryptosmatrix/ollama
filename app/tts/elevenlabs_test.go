package tts

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestSynthesizePostsDonorShapeAndRejectsJSONMIME(t *testing.T) {
	var hits atomic.Int32
	var sawSpeed bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("xi-api-key") != "sk-test" {
			t.Errorf("key header %q", r.Header.Get("xi-api-key"))
		}
		if !strings.Contains(r.URL.Path, "/v1/text-to-speech/abc") {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("output_format") != OutputFormat {
			t.Errorf("format %s", r.URL.Query().Get("output_format"))
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["model_id"] != ModelFlash {
			t.Errorf("model %v", payload["model_id"])
		}
		settings := payload["voice_settings"].(map[string]any)
		if settings["speed"].(float64) != 1.0 {
			t.Errorf("speed %v", settings["speed"])
		}
		sawSpeed = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"error":"not audio"}`))
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL}
	_, err := client.Synthesize(context.Background(), SynthesisRequest{
		Text:    "hello",
		VoiceID: "abc",
		ModelID: ModelFlash,
		Speed:   1.0,
	}, "sk-test")
	if err == nil {
		t.Fatal("JSON MIME must not be accepted as audio")
	}
	if !strings.Contains(err.Error(), "not audio") {
		t.Errorf("error %v", err)
	}
	if hits.Load() != 1 || !sawSpeed {
		t.Fatal("expected one donor-shaped request")
	}
}

func TestSynthesizeSpeedBelowMinimumMakesNoRequest(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
	}))
	defer server.Close()
	client := &Client{BaseURL: server.URL}
	_, err := client.Synthesize(context.Background(), SynthesisRequest{
		Text: "hello", VoiceID: "abc", ModelID: ModelFlash, Speed: 0.69,
	}, "sk-test")
	if err == nil {
		t.Fatal("speed 0.69 must fail locally")
	}
	if hits.Load() != 0 {
		t.Fatalf("network was used: %d", hits.Load())
	}
}

func TestSameHTTPSOriginRedirects(t *testing.T) {
	from, _ := url.Parse("https://api.elevenlabs.io/v1/x")
	same, _ := url.Parse("https://api.elevenlabs.io/v1/y")
	other, _ := url.Parse("https://evil.example/steal")
	httpURL, _ := url.Parse("http://api.elevenlabs.io/v1/y")
	if !sameHTTPSOrigin(from, same) {
		t.Fatal("same origin")
	}
	if sameHTTPSOrigin(from, other) {
		t.Fatal("cross host")
	}
	if sameHTTPSOrigin(from, httpURL) {
		t.Fatal("http is not https")
	}
}

func TestListVoicesPagesUntilDone(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			w.Write([]byte(`{"voices":[{"voice_id":"a","name":"Ada"}],"has_more":true,"next_page_token":"p2"}`))
			return
		}
		w.Write([]byte(`{"voices":[{"voice_id":"b","name":"Bea"}],"has_more":false}`))
	}))
	defer server.Close()
	voices, err := (&Client{BaseURL: server.URL}).ListVoices(context.Background(), "k")
	if err != nil {
		t.Fatal(err)
	}
	if len(voices) != 2 {
		t.Fatalf("voices %v", voices)
	}
	if hits.Load() != 2 {
		t.Fatalf("pages %d", hits.Load())
	}
}
