package tts

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type reversingProtector struct{ fail bool }

func (reversingProtector) Describe() string {
	return "reversed, which protects nothing and is for tests"
}

func (p reversingProtector) Protect(plaintext []byte) ([]byte, error) {
	if p.fail {
		return nil, errors.New("cannot encrypt")
	}
	return reverseBytes(plaintext), nil
}

func (p reversingProtector) Unprotect(ciphertext []byte) ([]byte, error) {
	if p.fail {
		return nil, errors.New("cannot decrypt")
	}
	return reverseBytes(ciphertext), nil
}

func reverseBytes(b []byte) []byte {
	out := make([]byte, len(b))
	for i := range b {
		out[len(b)-1-i] = b[i]
	}
	return out
}

func TestAPIKeyRoundTripAndNeverEchoedInPrefs(t *testing.T) {
	dir := t.TempDir()
	store := &FileSecretStore{Path: filepath.Join(dir, "secrets.json")}
	prefs := &PrefStore{Path: filepath.Join(dir, "prefs.json")}
	if err := store.SaveAPIKey("sk-secret-value"); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadAPIKey()
	if err != nil || got != "sk-secret-value" {
		t.Fatalf("load %q %v", got, err)
	}
	if err := prefs.Save(DefaultPrefs()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(prefs.Path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "sk-secret-value") {
		t.Fatal("prefs contained the API key")
	}
}

func TestEmptyAndOverlongKeysAreRejected(t *testing.T) {
	store := &FileSecretStore{Path: filepath.Join(t.TempDir(), "s.json")}
	if err := store.SaveAPIKey(""); err == nil {
		t.Fatal("empty")
	}
	if err := store.SaveAPIKey(strings.Repeat("a", maximumAPIKeyBytes+1)); err == nil {
		t.Fatal("overlong")
	}
}

func TestProtectedStoreCannotBeReadWithoutTheProtector(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tts-secret.json")
	if err := (&FileSecretStore{Path: path, Protect: reversingProtector{}}).SaveAPIKey("sk-secret-value"); err != nil {
		t.Fatal(err)
	}
	_, err := (&FileSecretStore{Path: path}).LoadAPIKey()
	if err == nil {
		t.Fatal("an encrypted store was read by a build with no protector")
	}
	if !strings.Contains(err.Error(), "save the API key again") {
		t.Errorf("error %v", err)
	}
}

func TestFailingProtectorNeverWritesCleartext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tts-secret.json")
	store := &FileSecretStore{Path: path, Protect: reversingProtector{fail: true}}
	if err := store.SaveAPIKey("sk-secret-value"); err == nil {
		t.Fatal("save must fail")
	}
	if data, err := os.ReadFile(path); err == nil && strings.Contains(string(data), "sk-secret-value") {
		t.Fatalf("cleartext written:\n%s", data)
	}
}

func TestDescriptionSaysWhatProtectsIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tts-secret.json")
	plain := (&FileSecretStore{Path: path}).Description()
	if !strings.Contains(plain, "readable by any program running as you") {
		t.Errorf("plain %q", plain)
	}
	protected := (&FileSecretStore{Path: path, Protect: reversingProtector{}}).Description()
	if strings.Contains(protected, "readable by any program running as you") {
		t.Errorf("protected still claims unprotected: %q", protected)
	}
	if !strings.Contains(protected, "protects nothing") {
		t.Errorf("missing protector words: %q", protected)
	}
}

func TestStatusJSONHasNoAPIKeyField(t *testing.T) {
	svc := Isolated(t.TempDir())
	if err := svc.PutKey("sk-secret-value"); err != nil {
		t.Fatal(err)
	}
	st, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	if !st.HasAPIKey {
		t.Fatal("has_api_key")
	}
	raw, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "sk-secret-value") {
		t.Fatalf("status leaked the key: %s", raw)
	}
	if strings.Contains(string(raw), `"api_key"`) {
		t.Fatalf("status has api_key field: %s", raw)
	}
}

func TestDefaultSecretStoreHonoursEnvAndDoesNotTouchLoginKeychain(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.json")
	t.Setenv(SecretsPathEnv, path)
	store := DefaultSecretStore()
	file, ok := store.(*FileSecretStore)
	if !ok {
		t.Fatalf("env override must be a file store, got %T", store)
	}
	if err := file.SaveAPIKey("sk-from-env"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(got), "sk-from-env") {
		t.Fatalf("wrote elsewhere: %s %v", got, err)
	}
}
