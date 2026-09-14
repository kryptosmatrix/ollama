package tts

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type countingSynth struct {
	hits  int
	audio []byte
	fail  error
}

func (c *countingSynth) Synthesize(ctx context.Context, req SynthesisRequest, apiKey string) ([]byte, error) {
	c.hits++
	if c.fail != nil {
		return nil, c.fail
	}
	if len(c.audio) == 0 {
		return []byte("ID3fake-mpeg"), nil
	}
	return c.audio, nil
}

func (c *countingSynth) ListVoices(ctx context.Context, apiKey string) ([]Voice, error) {
	c.hits++
	return []Voice{{ID: "v1", Name: "Ada"}}, nil
}

func readyService(t *testing.T) (*Service, *countingSynth) {
	t.Helper()
	svc := Isolated(t.TempDir())
	synth := &countingSynth{}
	svc.Client = synth
	if err := svc.PutKey("sk-test"); err != nil {
		t.Fatal(err)
	}
	voice := "voice_one"
	model := ModelFlash
	speed := 1.0
	if err := svc.PatchSettings(SettingsPatch{VoiceID: &voice, ModelID: &model, Speed: &speed}); err != nil {
		t.Fatal(err)
	}
	return svc, synth
}

func TestSpeakWithoutCommitDoesNotWriteCache(t *testing.T) {
	svc, synth := readyService(t)
	on := true
	if err := svc.PatchSettings(SettingsPatch{CacheEnabled: &on}); err != nil {
		t.Fatal(err)
	}
	chunk, err := svc.Speak(context.Background(), "Hello there.", 0)
	if err != nil {
		t.Fatal(err)
	}
	if synth.hits != 1 {
		t.Fatalf("hits %d", synth.hits)
	}
	if chunk.CacheHit {
		t.Fatal("first speak must miss")
	}
	root := svc.Cache.Root
	if files := cacheAudioFiles(t, root); len(files) != 0 {
		t.Fatalf("speak wrote cache files: %v", files)
	}
	chunk2, err := svc.Speak(context.Background(), "Hello there.", 0)
	if err != nil {
		t.Fatal(err)
	}
	if synth.hits != 2 {
		t.Fatalf("second speak without commit must hit the network, hits=%d cacheHit=%v", synth.hits, chunk2.CacheHit)
	}
}

func TestCommitThenSecondSpeakIsCacheHit(t *testing.T) {
	svc, synth := readyService(t)
	on := true
	if err := svc.PatchSettings(SettingsPatch{CacheEnabled: &on}); err != nil {
		t.Fatal(err)
	}
	chunk, err := svc.Speak(context.Background(), "Hello there.", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Commit(chunk.Fingerprint); err != nil {
		t.Fatal(err)
	}
	if files := cacheAudioFiles(t, svc.Cache.Root); len(files) != 1 {
		t.Fatalf("expected one audio.enc after commit, got %v", files)
	}
	chunk2, err := svc.Speak(context.Background(), "Hello there.", 0)
	if err != nil {
		t.Fatal(err)
	}
	if synth.hits != 1 {
		t.Fatalf("cache hit still called synthesizer: %d", synth.hits)
	}
	if !chunk2.CacheHit {
		t.Fatal("expected cache hit")
	}
}

func TestJSONMIMEIsNotCached(t *testing.T) {
	svc, synth := readyService(t)
	synth.fail = httpErrorf(502, "ElevenLabs returned data that was not audio.")
	on := true
	if err := svc.PatchSettings(SettingsPatch{CacheEnabled: &on}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Speak(context.Background(), "Hello there.", 0)
	if err == nil {
		t.Fatal("expected MIME failure")
	}
	if err := svc.Commit("anything"); err != nil {
		t.Fatal(err)
	}
	if files := cacheAudioFiles(t, svc.Cache.Root); len(files) != 0 {
		t.Fatalf("cached a MIME failure: %v", files)
	}
}

func TestKeyReplacementSetsClearPendingBeforeMutation(t *testing.T) {
	svc, _ := readyService(t)
	on := true
	if err := svc.PatchSettings(SettingsPatch{CacheEnabled: &on}); err != nil {
		t.Fatal(err)
	}
	chunk, err := svc.Speak(context.Background(), "Hello there.", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Commit(chunk.Fingerprint); err != nil {
		t.Fatal(err)
	}
	if err := svc.PutKey("sk-other"); err != nil {
		t.Fatal(err)
	}
	if files := cacheAudioFiles(t, svc.Cache.Root); len(files) != 0 {
		t.Fatalf("cache survived key replacement: %v", files)
	}
	prefs, err := svc.Prefs.Load()
	if err != nil {
		t.Fatal(err)
	}
	if prefs.CacheClearPending {
		t.Fatal("barrier should clear after a successful purge")
	}
}

func TestDisabledCacheNeverWrites(t *testing.T) {
	svc, _ := readyService(t)
	chunk, err := svc.Speak(context.Background(), "Hello there.", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Commit(chunk.Fingerprint); err != nil {
		t.Fatal(err)
	}
	if files := cacheAudioFiles(t, svc.Cache.Root); len(files) != 0 {
		t.Fatalf("disabled cache wrote: %v", files)
	}
}

func cacheAudioFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasSuffix(path, audioFileName) {
			files = append(files, path)
		}
		return nil
	})
	return files
}
