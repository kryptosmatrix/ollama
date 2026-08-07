//go:build darwin && cgo

package mcp

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// keychainStore returns a store filed under a service name that belongs to this
// test alone, and clears it out both before and after.
//
// These tests write to the real login keychain, because a keychain that is
// stood in for proves nothing about the keychain. The scoping is what keeps
// that honest: the service name carries the test's own name, nothing else can
// collide with it, no real sign-in is ever touched, and a run that crashed
// halfway is cleaned up by the next one before it starts.
func keychainStore(t *testing.T) *KeychainStore {
	t.Helper()
	store := &KeychainStore{Service: "Ollama MCP (test " + t.Name() + ")"}
	clear := func() {
		if servers, err := store.Servers(); err == nil {
			for _, server := range servers {
				store.Delete(server)
			}
		}
		sweepKeychainService(t, store.Service)
	}
	clear()
	t.Cleanup(clear)
	return store
}

// sweepKeychainService removes every item under a test's service name, using a
// path that shares no code with the store being tested.
//
// This exists because of something a falsification pass did: a sabotage that
// dropped the account from the query created items the store's own cleanup
// could no longer find, and two were left in the real login keychain. A
// cleanup that depends on the code under test is not a cleanup. Deleting by
// service name passes no secret on the command line — the objection that rules
// this tool out for storing a token does not apply to removing one.
func sweepKeychainService(t *testing.T, service string) {
	t.Helper()
	for range 32 {
		cmd := exec.Command("security", "delete-generic-password", "-s", service)
		if err := cmd.Run(); err != nil {
			return // nothing left under this service
		}
	}
	t.Errorf("could not clear the test keychain service %q", service)
}

func TestKeychainRoundTrip(t *testing.T) {
	store := keychainStore(t)
	want := &SignInRecord{
		Token: &oauth2.Token{
			AccessToken:  "access-abc123",
			TokenType:    "Bearer",
			RefreshToken: "refresh-def456",
			Expiry:       time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC),
		},
		ClientID: "client-xyz789",
	}

	if err := store.Save("hosted", want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load("hosted")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Token.AccessToken != want.Token.AccessToken {
		t.Errorf("AccessToken = %q", got.Token.AccessToken)
	}
	if got.Token.RefreshToken != want.Token.RefreshToken {
		t.Errorf("the refresh token must survive, or the user is signed out at the first expiry: %q", got.Token.RefreshToken)
	}
	if got.Token.TokenType != want.Token.TokenType {
		t.Errorf("TokenType = %q", got.Token.TokenType)
	}
	if !got.Token.Expiry.Equal(want.Token.Expiry) {
		t.Errorf("Expiry = %v, want %v; a lost expiry means the token is never refreshed", got.Token.Expiry, want.Token.Expiry)
	}
	if got.ClientID != want.ClientID {
		t.Errorf("ClientID = %q, want %q; without it the sign-in cannot be revoked", got.ClientID, want.ClientID)
	}
}

// TestKeychainSaveReplacesRatherThanDuplicates guards the update path. A
// keychain refuses a duplicate item, so a save that only ever added would fail
// on the second sign-in to the same server — and one that deleted first would
// leave the user signed out if the add then failed.
func TestKeychainSaveReplacesRatherThanDuplicates(t *testing.T) {
	store := keychainStore(t)

	for _, token := range []string{"first", "second", "third"} {
		if err := store.Save("hosted", &SignInRecord{
			Token:    &oauth2.Token{AccessToken: token},
			ClientID: "client-1",
		}); err != nil {
			t.Fatalf("Save %q: %v", token, err)
		}
	}

	got, err := store.Load("hosted")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Token.AccessToken != "third" {
		t.Errorf("AccessToken = %q, want the most recent", got.Token.AccessToken)
	}
	servers, err := store.Servers()
	if err != nil {
		t.Fatalf("Servers: %v", err)
	}
	if len(servers) != 1 {
		t.Errorf("servers = %v, want one item rather than one per save", servers)
	}
}

// TestKeychainKeepsTheClientIDAcrossARefresh is the same rule the file store
// keeps. The identifier is issued once, at registration; a later save that does
// not carry it must not erase it, or the sign-in becomes unrevocable at the
// first token refresh.
func TestKeychainKeepsTheClientIDAcrossARefresh(t *testing.T) {
	store := keychainStore(t)

	if err := store.Save("hosted", &SignInRecord{
		Token:    &oauth2.Token{AccessToken: "first"},
		ClientID: "client-xyz789",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Save("hosted", &SignInRecord{Token: &oauth2.Token{AccessToken: "refreshed"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Load("hosted")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Token.AccessToken != "refreshed" {
		t.Errorf("AccessToken = %q, want the refreshed one", got.Token.AccessToken)
	}
	if got.ClientID != "client-xyz789" {
		t.Errorf("ClientID = %q, want the one recorded at sign-in to survive a refresh", got.ClientID)
	}
}

func TestKeychainTokensArePerServer(t *testing.T) {
	store := keychainStore(t)

	if err := store.Save("one", &SignInRecord{Token: &oauth2.Token{AccessToken: "first"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Save("two", &SignInRecord{Token: &oauth2.Token{AccessToken: "second"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	first, err := store.Load("one")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if first.Token.AccessToken != "first" {
		t.Errorf("one = %q; one server's credential must never be handed to another", first.Token.AccessToken)
	}
	second, err := store.Load("two")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if second.Token.AccessToken != "second" {
		t.Errorf("two = %q", second.Token.AccessToken)
	}

	servers, err := store.Servers()
	if err != nil {
		t.Fatalf("Servers: %v", err)
	}
	if len(servers) != 2 || servers[0] != "one" || servers[1] != "two" {
		t.Errorf("Servers() = %v, want a stable sorted list", servers)
	}
}

func TestKeychainLoadWithoutAToken(t *testing.T) {
	store := keychainStore(t)

	if _, err := store.Load("hosted"); !errors.Is(err, ErrNoToken) {
		t.Errorf("err = %v, want ErrNoToken; not being signed in is not a failure to read", err)
	}
	if err := store.Save("other", &SignInRecord{Token: &oauth2.Token{AccessToken: "x"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := store.Load("hosted"); !errors.Is(err, ErrNoToken) {
		t.Errorf("err = %v, want ErrNoToken", err)
	}
}

func TestKeychainDelete(t *testing.T) {
	store := keychainStore(t)
	if err := store.Save("hosted", &SignInRecord{Token: &oauth2.Token{AccessToken: "access-abc123"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := store.Delete("hosted"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Load("hosted"); !errors.Is(err, ErrNoToken) {
		t.Errorf("the token survived deletion: %v", err)
	}
	if err := store.Delete("hosted"); err != nil {
		t.Errorf("deleting an absent token is not a failure; the intent is satisfied: %v", err)
	}
}

func TestKeychainRefusesToStoreNothing(t *testing.T) {
	store := keychainStore(t)

	if err := store.Save("", &SignInRecord{Token: &oauth2.Token{AccessToken: "x"}}); err == nil {
		t.Error("a token needs a server name")
	}
	if err := store.Save("hosted", nil); err == nil {
		t.Error("a nil record must be refused rather than written as an empty credential")
	}
	if err := store.Save("hosted", &SignInRecord{}); err == nil {
		t.Error("a record with no token must be refused")
	}
	if err := store.Save("hosted", &SignInRecord{Token: &oauth2.Token{}}); err == nil {
		t.Error("an empty token must be refused; storing it would look like being signed in")
	}
}

// TestKeychainMigratesATokenOutOfTheFile is what an upgrade does. A build
// without cgo, or a machine whose ~/.ollama was carried across, leaves a token
// in cleartext on disk; the first read must move it into the keychain and take
// the cleartext copy away rather than leaving one behind.
func TestKeychainMigratesATokenOutOfTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp-tokens.json")
	file := &FileTokenStore{Path: path}
	if err := file.Save("hosted", &SignInRecord{
		Token:    &oauth2.Token{AccessToken: "access-abc123", RefreshToken: "refresh-def456"},
		ClientID: "client-xyz789",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	store := keychainStore(t)
	store.Fallback = file

	got, err := store.Load("hosted")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Token.AccessToken != "access-abc123" {
		t.Errorf("AccessToken = %q, want the one that was in the file", got.Token.AccessToken)
	}
	if got.ClientID != "client-xyz789" {
		t.Errorf("ClientID = %q, want it carried across; without it the sign-in cannot be revoked", got.ClientID)
	}

	// It is now in the keychain, so a store without the fallback finds it.
	direct := &KeychainStore{Service: store.Service}
	if _, err := direct.Load("hosted"); err != nil {
		t.Errorf("the token was returned but not written to the keychain: %v", err)
	}

	// And the cleartext copy is gone, which is the point of migrating at all.
	if _, err := file.Load("hosted"); !errors.Is(err, ErrNoToken) {
		t.Errorf("the file still holds the token: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	for _, secret := range []string{"access-abc123", "refresh-def456"} {
		if strings.Contains(string(data), secret) {
			t.Errorf("a migrated secret is still in the file: %s", secret)
		}
	}
}

func TestKeychainWithoutAFallbackDoesNotMigrate(t *testing.T) {
	store := keychainStore(t)
	if _, err := store.Load("hosted"); !errors.Is(err, ErrNoToken) {
		t.Errorf("err = %v, want ErrNoToken", err)
	}
}

// TestKeychainDescriptionDoesNotOverclaim. The word "keychain" suggests other
// programs cannot read the item, and for an unsigned build that is not true —
// it was measured by writing an item from one binary and reading it from
// another. What is true is that it is encrypted at rest, and that is what the
// user is told.
func TestKeychainDescriptionDoesNotOverclaim(t *testing.T) {
	description := (&KeychainStore{}).Description()
	if !strings.Contains(description, "keychain") {
		t.Errorf("Description() = %q, want it to name where the token goes", description)
	}
	if !strings.Contains(description, "encrypted at rest") {
		t.Errorf("Description() = %q, want it to say what the protection actually is", description)
	}
	for _, overclaim := range []string{"only Ollama", "no other", "cannot be read", "inaccessible"} {
		if strings.Contains(description, overclaim) {
			t.Errorf("Description() claims %q, which an unsigned build does not deliver", overclaim)
		}
	}
}

// TestDefaultTokenStoreIsTheKeychainOnThisPlatform is the activation evidence.
// A keychain store nothing constructs is a keychain store nobody uses.
func TestDefaultTokenStoreIsTheKeychainOnThisPlatform(t *testing.T) {
	t.Setenv(TokensPathEnv, "")
	store := DefaultTokenStore()
	keychain, ok := store.(*KeychainStore)
	if !ok {
		t.Fatalf("DefaultTokenStore() = %T, want the keychain on a darwin cgo build", store)
	}
	if keychain.Fallback == nil {
		t.Error("without a fallback, a token written by an earlier build is silently lost and the user is signed out")
	}
}

// TestAnExplicitTokensPathIsHonoured keeps OLLAMA_MCP_TOKENS meaning what it
// says. It is also what isolates every other package's tests from the real
// keychain: they set it, so nothing they do can read or delete a credential
// belonging to an actual sign-in.
func TestAnExplicitTokensPathIsHonoured(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp-tokens.json")
	t.Setenv(TokensPathEnv, path)

	store := DefaultTokenStore()
	file, ok := store.(*FileTokenStore)
	if !ok {
		t.Fatalf("DefaultTokenStore() = %T, want the file store when a path is named", store)
	}
	if !strings.Contains(file.Description(), path) {
		t.Errorf("Description() = %q, want it to name the path in use", file.Description())
	}

	if err := store.Save("hosted", &SignInRecord{Token: &oauth2.Token{AccessToken: "in-the-file"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("nothing was written to the named path: %v", err)
	}
	// And nothing reached the keychain under the real service name.
	if _, err := (&KeychainStore{}).Load("hosted"); !errors.Is(err, ErrNoToken) {
		t.Errorf("a token was written to the real keychain despite an explicit path: %v", err)
	}
}
