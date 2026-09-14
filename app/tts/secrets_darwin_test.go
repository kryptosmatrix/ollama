//go:build darwin && cgo

package tts

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestKeychainRoundTripUsesNamedServiceNotMCP(t *testing.T) {
	service := fmt.Sprintf(
		"Ollama TTS test %d %s",
		time.Now().UnixNano(),
		strings.ReplaceAll(t.Name(), "/", "_"),
	)
	store := &KeychainStore{Service: service}
	t.Cleanup(func() {
		_ = store.DeleteAPIKey()
		_ = store.DeleteCacheMaster()
	})
	if strings.Contains(store.Description(), "Ollama MCP") {
		t.Fatal("TTS keychain description must not name the MCP service")
	}
	if err := store.SaveAPIKey("sk-keychain-test"); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	if got != "sk-keychain-test" {
		t.Fatalf("got %q", got)
	}
	if err := store.DeleteAPIKey(); err != nil {
		t.Fatal(err)
	}
	got, err = store.LoadAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("delete left %q", got)
	}
}
