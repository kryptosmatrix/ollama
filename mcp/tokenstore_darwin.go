//go:build darwin && cgo

package mcp

/*
#cgo LDFLAGS: -framework CoreFoundation -framework Security
#include <CoreFoundation/CoreFoundation.h>
#include <Security/Security.h>

// CFTypeRef is already "const void *", so these two do nothing but let Go pass
// a CF reference without converting a uintptr to an unsafe.Pointer. Doing that
// conversion on the Go side is what "go vet" flags as a possible misuse, and it
// is right to: the rule exists because a uintptr is not a reference the garbage
// collector knows about. These objects live outside the Go heap, so the
// conversion is safe in fact — but silencing an analyser by reasoning is worse
// than not needing it.
static void ollamaDictionarySet(CFMutableDictionaryRef dictionary, CFTypeRef key, CFTypeRef value) {
	CFDictionarySetValue(dictionary, key, value);
}

static CFTypeRef ollamaDictionaryGet(CFDictionaryRef dictionary, CFTypeRef key) {
	return CFDictionaryGetValue(dictionary, key);
}
*/
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"unsafe"

	"golang.org/x/oauth2"
)

// keychainService is the service name every Ollama MCP credential is filed
// under. It is what a user sees in Keychain Access, and it is how they delete
// one by hand if they ever want to.
const keychainService = "Ollama MCP"

// errSecItemNotFound is the Security framework's "no such item".
const errSecItemNotFound = C.OSStatus(-25300)

// KeychainStore keeps each server's sign-in in the macOS keychain, one generic
// password item per server.
//
// What this actually buys, stated plainly because the word "keychain" promises
// more than it delivers here: the keychain database is encrypted at rest and
// unlocked with the user's login, so a token no longer sits in cleartext in a
// file that gets copied into backups, sync folders and support bundles. It is
// **not** a guarantee that other programs cannot read it. An item added by an
// unsigned build is readable by another process running as the same user
// without a prompt — this was measured, not assumed, by writing an item from
// one binary and reading it from a different one.
type KeychainStore struct {
	// Service overrides the keychain service name. Empty means
	// keychainService. It exists so tests can file their items somewhere that
	// cannot collide with a real sign-in.
	Service string

	// Fallback is consulted when an item is missing, so a token written by an
	// older build — one without cgo, or on a machine whose ~/.ollama was
	// carried over — is moved into the keychain rather than silently lost.
	// Nil disables the migration.
	Fallback TokenStore
}

func (s *KeychainStore) service() string {
	if strings.TrimSpace(s.Service) != "" {
		return s.Service
	}
	return keychainService
}

// Description tells the user where their tokens are and what protects them.
func (s *KeychainStore) Description() string {
	return "your macOS keychain, encrypted at rest and unlocked with your login"
}

// Load returns the stored sign-in for a server.
//
// A token found in the fallback store is moved into the keychain and removed
// from the file, so upgrading a build takes the credential out of cleartext
// rather than leaving a copy behind.
func (s *KeychainStore) Load(server string) (*SignInRecord, error) {
	data, err := s.read(server)
	if err == nil {
		var stored storedToken
		if err := json.Unmarshal(data, &stored); err != nil {
			return nil, fmt.Errorf("parse the keychain item for %s: %w", server, err)
		}
		// A file copy can survive a migration whose delete failed, and once the
		// item is in the keychain this branch is the only one that ever runs
		// again — so the cleanup is retried here rather than once. Without it a
		// single failed delete leaves the credential in cleartext for ever.
		s.removeFallbackCopy(server)
		return stored.record(), nil
	}
	if !errors.Is(err, ErrNoToken) || s.Fallback == nil {
		return nil, err
	}

	record, fallbackErr := s.Fallback.Load(server)
	if fallbackErr != nil {
		// The keychain's answer is the one that matters; the fallback is only
		// consulted to migrate. Report the original miss.
		return nil, err
	}
	if saveErr := s.Save(server, record); saveErr != nil {
		return nil, saveErr
	}
	// A cleanup that cannot run is not a reason to sign the user out: the token
	// is in the keychain and works. It is retried on every later load instead,
	// so the condition heals as soon as whatever blocked it clears.
	s.removeFallbackCopy(server)
	return record, nil
}

// removeFallbackCopy takes the cleartext copy away, if there is one and if it
// can. Failure is deliberately not reported: the caller wanted a token and has
// one, and this is retried on every load rather than abandoned.
func (s *KeychainStore) removeFallbackCopy(server string) {
	if s.Fallback == nil {
		return
	}
	s.Fallback.Delete(server)
}

// Save stores a server's sign-in, replacing any previous one.
func (s *KeychainStore) Save(server string, record *SignInRecord) error {
	if strings.TrimSpace(server) == "" {
		return errors.New("a token needs a server name")
	}
	if record == nil || record.Token == nil || record.Token.AccessToken == "" {
		return errors.New("refusing to store an empty token")
	}

	// A client identifier recorded at sign-in must survive a refresh that does
	// not carry one, exactly as in the file store: losing it makes the sign-in
	// unrevocable.
	clientID := record.ClientID
	if clientID == "" {
		if previous, err := s.read(server); err == nil {
			var stored storedToken
			if json.Unmarshal(previous, &stored) == nil {
				clientID = stored.ClientID
			}
		}
	}

	data, err := json.Marshal(storedToken{
		AccessToken:  record.Token.AccessToken,
		TokenType:    record.Token.TokenType,
		RefreshToken: record.Token.RefreshToken,
		Expiry:       record.Token.Expiry,
		ClientID:     clientID,
	})
	if err != nil {
		return fmt.Errorf("marshal the keychain item for %s: %w", server, err)
	}
	return s.write(server, data)
}

// Delete removes a server's sign-in. Deleting an absent one is not an error.
func (s *KeychainStore) Delete(server string) error {
	query := s.query(server)
	defer C.CFRelease(C.CFTypeRef(query))

	switch status := C.SecItemDelete(C.CFDictionaryRef(query)); status {
	case 0, errSecItemNotFound:
		return nil
	default:
		return keychainError("delete the keychain item for "+server, status)
	}
}

// Servers lists the servers with a stored sign-in, in a stable order.
func (s *KeychainStore) Servers() ([]string, error) {
	query := newDictionary()
	defer C.CFRelease(C.CFTypeRef(query))

	service := cfString(s.service())
	defer C.CFRelease(C.CFTypeRef(service))

	setValue(query, C.kSecClass, C.CFTypeRef(C.kSecClassGenericPassword))
	setValue(query, C.kSecAttrService, C.CFTypeRef(service))
	setValue(query, C.kSecMatchLimit, C.CFTypeRef(C.kSecMatchLimitAll))
	setValue(query, C.kSecReturnAttributes, C.CFTypeRef(C.kCFBooleanTrue))

	var result C.CFTypeRef
	switch status := C.SecItemCopyMatching(C.CFDictionaryRef(query), &result); status {
	case 0:
	case errSecItemNotFound:
		return nil, nil
	default:
		return nil, keychainError("list keychain items", status)
	}
	defer C.CFRelease(result)

	items := C.CFArrayRef(result)
	count := int(C.CFArrayGetCount(items))
	servers := make([]string, 0, count)
	for i := range count {
		item := C.CFDictionaryRef(C.CFArrayGetValueAtIndex(items, C.CFIndex(i)))
		value := C.ollamaDictionaryGet(item, C.CFTypeRef(C.kSecAttrAccount))
		if value == 0 {
			continue
		}
		name := goString(C.CFStringRef(value))
		if name == "" {
			// goString answers "" when the item's account cannot be read back.
			// An empty name in this list would be a server nobody can act on,
			// so the item is skipped rather than reported as one.
			continue
		}
		servers = append(servers, name)
	}
	// A server whose token has not been migrated yet lives only in the
	// fallback, and a caller listing sign-ins would not see it.
	if lister, ok := s.Fallback.(interface{ Servers() ([]string, error) }); ok {
		pending, err := lister.Servers()
		if err == nil {
			for _, name := range pending {
				if !slices.Contains(servers, name) {
					servers = append(servers, name)
				}
			}
		}
	}
	slices.Sort(servers)
	return servers, nil
}

// read returns the raw item for a server, or ErrNoToken.
func (s *KeychainStore) read(server string) ([]byte, error) {
	query := s.query(server)
	defer C.CFRelease(C.CFTypeRef(query))
	setValue(query, C.kSecReturnData, C.CFTypeRef(C.kCFBooleanTrue))
	setValue(query, C.kSecMatchLimit, C.CFTypeRef(C.kSecMatchLimitOne))

	var result C.CFTypeRef
	switch status := C.SecItemCopyMatching(C.CFDictionaryRef(query), &result); status {
	case 0:
	case errSecItemNotFound:
		return nil, fmt.Errorf("%w: %s", ErrNoToken, server)
	default:
		return nil, keychainError("read the keychain item for "+server, status)
	}
	defer C.CFRelease(result)

	data := C.CFDataRef(result)
	return C.GoBytes(unsafe.Pointer(C.CFDataGetBytePtr(data)), C.int(C.CFDataGetLength(data))), nil
}

// write adds or replaces a server's item.
func (s *KeychainStore) write(server string, payload []byte) error {
	secret := cfData(payload)
	defer C.CFRelease(C.CFTypeRef(secret))

	query := s.query(server)
	defer C.CFRelease(C.CFTypeRef(query))

	// Update in place when there is already an item, rather than delete and
	// add: a delete that succeeded and an add that then failed would leave the
	// user signed out of a server they were signed in to.
	update := newDictionary()
	defer C.CFRelease(C.CFTypeRef(update))
	setValue(update, C.kSecValueData, C.CFTypeRef(secret))

	switch status := C.SecItemUpdate(C.CFDictionaryRef(query), C.CFDictionaryRef(update)); status {
	case 0:
		return nil
	case errSecItemNotFound:
	default:
		return keychainError("update the keychain item for "+server, status)
	}

	label := cfString(keychainService + ": " + server)
	defer C.CFRelease(C.CFTypeRef(label))
	setValue(query, C.kSecAttrLabel, C.CFTypeRef(label))
	setValue(query, C.kSecValueData, C.CFTypeRef(secret))
	// Available whenever this Mac is unlocked, and only on this Mac: a token
	// for one machine has no business syncing to another through a backup.
	setValue(query, C.kSecAttrAccessible, C.CFTypeRef(C.kSecAttrAccessibleWhenUnlockedThisDeviceOnly))

	if status := C.SecItemAdd(C.CFDictionaryRef(query), nil); status != 0 {
		return keychainError("add the keychain item for "+server, status)
	}
	return nil
}

// query builds the dictionary that identifies exactly one server's item.
func (s *KeychainStore) query(server string) C.CFMutableDictionaryRef {
	query := newDictionary()
	service := cfString(s.service())
	defer C.CFRelease(C.CFTypeRef(service))
	account := cfString(server)
	defer C.CFRelease(C.CFTypeRef(account))

	setValue(query, C.kSecClass, C.CFTypeRef(C.kSecClassGenericPassword))
	setValue(query, C.kSecAttrService, C.CFTypeRef(service))
	setValue(query, C.kSecAttrAccount, C.CFTypeRef(account))
	return query
}

func keychainError(what string, status C.OSStatus) error {
	message := C.SecCopyErrorMessageString(status, nil)
	if message == 0 {
		return fmt.Errorf("%s: keychain error %d", what, int(status))
	}
	defer C.CFRelease(C.CFTypeRef(message))
	return fmt.Errorf("%s: %s", what, goString(message))
}

// record rebuilds a SignInRecord from the persisted shape.
func (t storedToken) record() *SignInRecord {
	return &SignInRecord{
		Token: &oauth2.Token{
			AccessToken:  t.AccessToken,
			TokenType:    t.TokenType,
			RefreshToken: t.RefreshToken,
			Expiry:       t.Expiry,
		},
		ClientID: t.ClientID,
	}
}

// The Core Foundation helpers below own nothing: every reference they return
// must be released by the caller.

func newDictionary() C.CFMutableDictionaryRef {
	return C.CFDictionaryCreateMutable(C.kCFAllocatorDefault, 0,
		&C.kCFTypeDictionaryKeyCallBacks, &C.kCFTypeDictionaryValueCallBacks)
}

func setValue(dictionary C.CFMutableDictionaryRef, key C.CFStringRef, value C.CFTypeRef) {
	C.ollamaDictionarySet(dictionary, C.CFTypeRef(key), value)
}

func cfString(s string) C.CFStringRef {
	b := []byte(s)
	var first *C.UInt8
	if len(b) > 0 {
		first = (*C.UInt8)(unsafe.Pointer(&b[0]))
	}
	return C.CFStringCreateWithBytes(C.kCFAllocatorDefault, first, C.CFIndex(len(b)), C.kCFStringEncodingUTF8, C.false)
}

func cfData(b []byte) C.CFDataRef {
	var first *C.UInt8
	if len(b) > 0 {
		first = (*C.UInt8)(unsafe.Pointer(&b[0]))
	}
	return C.CFDataCreate(C.kCFAllocatorDefault, first, C.CFIndex(len(b)))
}

func goString(ref C.CFStringRef) string {
	length := C.CFStringGetLength(ref)
	if length == 0 {
		return ""
	}
	// CFStringGetCStringPtr may return nil depending on the internal encoding,
	// so the copying form is used rather than the fast path plus a fallback.
	size := C.CFStringGetMaximumSizeForEncoding(length, C.kCFStringEncodingUTF8) + 1
	buffer := make([]byte, size)
	if C.CFStringGetCString(ref, (*C.char)(unsafe.Pointer(&buffer[0])), size, C.kCFStringEncodingUTF8) == C.false {
		return ""
	}
	return string(buffer[:clen(buffer)])
}

func clen(b []byte) int {
	for i, c := range b {
		if c == 0 {
			return i
		}
	}
	return len(b)
}

// DefaultTokenStore returns where this build keeps tokens: the macOS keychain,
// migrating anything left in the file store as it is read.
//
// Setting OLLAMA_MCP_TOKENS overrides that with a file at the named path. It is
// an explicit instruction about where credentials are to live — for a machine
// with no keychain to speak of, or a user who wants them somewhere they can
// see — and it is honoured rather than second-guessed. Every surface shows
// TokenStore.Description(), so what it costs is stated wherever a sign-in is
// offered.
func DefaultTokenStore() TokenStore {
	if strings.TrimSpace(os.Getenv(TokensPathEnv)) != "" {
		return &FileTokenStore{}
	}
	return &KeychainStore{Fallback: &FileTokenStore{}}
}
