//go:build windows || darwin

package ui

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/ollama/ollama/app/tts"
)

func (s *Server) ttsService() *tts.Service {
	if s.TTS != nil {
		return s.TTS
	}
	s.ttsOnce.Do(func() {
		if s.TTS == nil {
			s.TTS = tts.Default()
		}
	})
	return s.TTS
}

type ttsSpeakBody struct {
	Text       string `json:"text"`
	ChunkIndex int    `json:"chunk_index"`
}

type ttsKeyBody struct {
	APIKey string `json:"api_key"`
}

type ttsCommitBody struct {
	Fingerprint string `json:"fingerprint"`
}

func (s *Server) ttsSpeak(w http.ResponseWriter, r *http.Request) error {
	var body ttsSpeakBody
	if err := json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(&body); err != nil {
		return ttsHTTP(http.StatusBadRequest, "The speech request was not valid JSON.")
	}
	chunk, err := s.ttsService().Speak(r.Context(), body.Text, body.ChunkIndex)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("X-Ollama-TTS-Chunk-Index", strconv.Itoa(chunk.Index))
	w.Header().Set("X-Ollama-TTS-Chunk-Count", strconv.Itoa(chunk.Count))
	w.Header().Set("X-Ollama-TTS-Fingerprint", chunk.Fingerprint)
	if chunk.CacheHit {
		w.Header().Set("X-Ollama-TTS-Cache", "hit")
	} else {
		w.Header().Set("X-Ollama-TTS-Cache", "miss")
	}
	w.Header().Set("Access-Control-Expose-Headers", "X-Ollama-TTS-Chunk-Index, X-Ollama-TTS-Chunk-Count, X-Ollama-TTS-Fingerprint, X-Ollama-TTS-Cache")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(chunk.Audio)
	return err
}

func (s *Server) ttsVoices(w http.ResponseWriter, r *http.Request) error {
	voices, err := s.ttsService().ListVoices(r.Context())
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]any{"voices": voices})
}

func (s *Server) ttsStatus(w http.ResponseWriter, r *http.Request) error {
	st, err := s.ttsService().Status()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(st)
}

func (s *Server) ttsPutKey(w http.ResponseWriter, r *http.Request) error {
	var body ttsKeyBody
	if err := json.NewDecoder(io.LimitReader(r.Body, maximumAPIKeyJSON)).Decode(&body); err != nil {
		return ttsHTTP(http.StatusBadRequest, "The key request was not valid JSON.")
	}
	if err := s.ttsService().PutKey(body.APIKey); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) ttsDeleteKey(w http.ResponseWriter, r *http.Request) error {
	if err := s.ttsService().DeleteKey(); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) ttsSettings(w http.ResponseWriter, r *http.Request) error {
	var patch tts.SettingsPatch
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&patch); err != nil {
		return ttsHTTP(http.StatusBadRequest, "The speech settings were not valid JSON.")
	}
	if err := s.ttsService().PatchSettings(patch); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) ttsCacheClear(w http.ResponseWriter, r *http.Request) error {
	if err := s.ttsService().ClearCache(); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) ttsCacheCommit(w http.ResponseWriter, r *http.Request) error {
	var body ttsCommitBody
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&body); err != nil {
		return ttsHTTP(http.StatusBadRequest, "The cache commit was not valid JSON.")
	}
	if err := s.ttsService().Commit(body.Fingerprint); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

const maximumAPIKeyJSON = 16 << 10

func ttsHTTP(code int, msg string) error {
	return &tts.HTTPError{Code: code, Message: msg}
}
