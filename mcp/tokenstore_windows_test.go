//go:build windows

package mcp

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// These run only on Windows, and they are the reason this store is not merely
// compiled. Everything else about the protector — the envelope, the migration,
// the refusal to write cleartext when encryption fails — is proven on any
// platform with a stand-in. What a stand-in cannot prove is that DPAPI itself
// was called correctly: the blob layout, the pointer lifetimes, the LocalFree.

func TestDPAPIRoundTrip(t *testing.T) {
	protector := dpapiProtector{}
	plaintext := []byte(`{"tokens":{"hosted":{"accessToken":"access-abc123"}}}`)

	ciphertext, err := protector.Protect(plaintext)
	if err != nil {
		t.Fatalf("Protect: %v", err)
	}
	if len(ciphertext) == 0 {
		t.Fatal("Protect returned nothing")
	}
	if bytes.Contains(ciphertext, []byte("access-abc123")) {
		t.Error("the token is still readable in the encrypted blob")
	}

	recovered, err := protector.Unprotect(ciphertext)
	if err != nil {
		t.Fatalf("Unprotect: %v", err)
	}
	if !bytes.Equal(recovered, plaintext) {
		t.Errorf("round trip changed the contents:\n got %q\nwant %q", recovered, plaintext)
	}
}

// TestDPAPIRefusesADamagedBlob. A store that cannot be decrypted must fail
// loudly. Reading as empty would sign the user out and then overwrite whatever
// was there.
func TestDPAPIRefusesADamagedBlob(t *testing.T) {
	protector := dpapiProtector{}
	ciphertext, err := protector.Protect([]byte("something worth protecting"))
	if err != nil {
		t.Fatalf("Protect: %v", err)
	}

	damaged := append([]byte(nil), ciphertext...)
	damaged[len(damaged)/2] ^= 0xff
	if _, err := protector.Unprotect(damaged); err == nil {
		t.Fatal("a damaged blob was accepted")
	}

	if _, err := protector.Unprotect(nil); err == nil {
		t.Error("an empty blob was accepted")
	}
}

// TestDPAPIProtectsRepeatedly guards the pointer lifetimes. A blob whose
// backing memory was collected or freed early would show up as an intermittent
// failure here rather than a clean one.
func TestDPAPIProtectsRepeatedly(t *testing.T) {
	protector := dpapiProtector{}
	for i := range 200 {
		plaintext := []byte(strings.Repeat("token-", i%64+1))
		ciphertext, err := protector.Protect(plaintext)
		if err != nil {
			t.Fatalf("Protect %d: %v", i, err)
		}
		recovered, err := protector.Unprotect(ciphertext)
		if err != nil {
			t.Fatalf("Unprotect %d: %v", i, err)
		}
		if !bytes.Equal(recovered, plaintext) {
			t.Fatalf("round trip %d lost the contents", i)
		}
	}
}

// TestTheWindowsDefaultStoreEncrypts is the activation evidence. A protector
// nothing constructs is a protector nobody uses.
func TestTheWindowsDefaultStoreEncrypts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp-tokens.json")
	t.Setenv(TokensPathEnv, path)

	store, ok := DefaultTokenStore().(*FileTokenStore)
	if !ok {
		t.Fatalf("DefaultTokenStore() = %T, want the file store", DefaultTokenStore())
	}
	if store.Protect == nil {
		t.Fatal("the Windows default store writes tokens in the clear")
	}
	if !strings.Contains(store.Description(), "Windows account") {
		t.Errorf("Description() = %q, want it to say what protects the file", store.Description())
	}
}
