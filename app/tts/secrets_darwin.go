//go:build darwin && cgo

package tts

/*
#cgo LDFLAGS: -framework CoreFoundation -framework Security
#include <CoreFoundation/CoreFoundation.h>
#include <Security/Security.h>

static void ollamaTTSDictionarySet(CFMutableDictionaryRef dictionary, CFTypeRef key, CFTypeRef value) {
	CFDictionarySetValue(dictionary, key, value);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"
)

const errSecItemNotFound = C.OSStatus(-25300)
const errSecInteractionNotAllowed = C.OSStatus(-25308)

// KeychainStore keeps the API key and cache master as two generic passwords
// under service "Ollama TTS". Unsigned-build items are readable by other
// programs running as this user; Description says that.
type KeychainStore struct {
	Service  string
	Fallback SecretStore
}

func (s *KeychainStore) service() string {
	if strings.TrimSpace(s.Service) != "" {
		return s.Service
	}
	return keychainService
}

func (s *KeychainStore) Description() string {
	return "your macOS keychain, encrypted at rest and unlocked with your login. An unsigned build's items can be read by other programs running as you"
}

func (s *KeychainStore) LoadAPIKey() (string, error) {
	data, err := s.copy(apiKeyAccount)
	if err != nil {
		return "", err
	}
	if len(data) == 0 && s.Fallback != nil {
		key, ferr := s.Fallback.LoadAPIKey()
		if ferr != nil {
			return "", ferr
		}
		if key != "" {
			if serr := s.SaveAPIKey(key); serr == nil {
				_ = s.Fallback.DeleteAPIKey()
			}
			return key, nil
		}
	}
	return string(data), nil
}

func (s *KeychainStore) SaveAPIKey(key string) error {
	if err := validateAPIKey(key); err != nil {
		return err
	}
	return s.put(apiKeyAccount, []byte(key))
}

func (s *KeychainStore) DeleteAPIKey() error {
	return s.remove(apiKeyAccount)
}

func (s *KeychainStore) LoadCacheMaster() ([]byte, error) {
	data, err := s.copy(cacheMasterAccount)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 && s.Fallback != nil {
		return s.Fallback.LoadCacheMaster()
	}
	return data, nil
}

func (s *KeychainStore) SaveCacheMaster(secret []byte) error {
	if len(secret) != masterSecretBytes {
		return errors.New("cache master secret must be 32 bytes")
	}
	return s.put(cacheMasterAccount, secret)
}

func (s *KeychainStore) DeleteCacheMaster() error {
	return s.remove(cacheMasterAccount)
}

func (s *KeychainStore) copy(account string) ([]byte, error) {
	query := s.query(account)
	defer C.CFRelease(C.CFTypeRef(query))
	setValue(query, C.kSecReturnData, C.CFTypeRef(C.kCFBooleanTrue))
	setValue(query, C.kSecMatchLimit, C.CFTypeRef(C.kSecMatchLimitOne))

	var result C.CFTypeRef
	status := C.SecItemCopyMatching(C.CFDictionaryRef(query), &result)
	if status == errSecItemNotFound {
		return nil, nil
	}
	if status == errSecInteractionNotAllowed {
		return nil, httpErrorf(503, "The speech secret store cannot be read while the Keychain is locked.")
	}
	if status != 0 {
		return nil, keychainError("read the TTS keychain item", status)
	}
	defer C.CFRelease(result)
	data := C.CFDataRef(result)
	length := C.CFDataGetLength(data)
	if length == 0 {
		return nil, nil
	}
	ptr := C.CFDataGetBytePtr(data)
	out := C.GoBytes(unsafe.Pointer(ptr), C.int(length))
	return out, nil
}

func (s *KeychainStore) put(account string, secret []byte) error {
	query := s.query(account)
	defer C.CFRelease(C.CFTypeRef(query))
	cfSecret := cfData(secret)
	defer C.CFRelease(C.CFTypeRef(cfSecret))

	update := newDictionary()
	defer C.CFRelease(C.CFTypeRef(update))
	setValue(update, C.kSecValueData, C.CFTypeRef(cfSecret))
	status := C.SecItemUpdate(C.CFDictionaryRef(query), C.CFDictionaryRef(update))
	switch status {
	case 0:
		return nil
	case errSecItemNotFound:
	case errSecInteractionNotAllowed:
		return httpErrorf(503, "The speech secret store cannot be written while the Keychain is locked.")
	default:
		return keychainError("update the TTS keychain item", status)
	}

	label := cfString(s.service() + ": " + account)
	defer C.CFRelease(C.CFTypeRef(label))
	setValue(query, C.kSecAttrLabel, C.CFTypeRef(label))
	setValue(query, C.kSecValueData, C.CFTypeRef(cfSecret))
	if status := C.SecItemAdd(C.CFDictionaryRef(query), nil); status != 0 {
		if status == errSecInteractionNotAllowed {
			return httpErrorf(503, "The speech secret store cannot be written while the Keychain is locked.")
		}
		return keychainError("add the TTS keychain item", status)
	}
	return nil
}

func (s *KeychainStore) remove(account string) error {
	query := s.query(account)
	defer C.CFRelease(C.CFTypeRef(query))
	status := C.SecItemDelete(C.CFDictionaryRef(query))
	if status == 0 || status == errSecItemNotFound {
		return nil
	}
	if status == errSecInteractionNotAllowed {
		return httpErrorf(503, "The speech secret store cannot be changed while the Keychain is locked.")
	}
	return keychainError("delete the TTS keychain item", status)
}

func (s *KeychainStore) query(account string) C.CFMutableDictionaryRef {
	query := newDictionary()
	service := cfString(s.service())
	defer C.CFRelease(C.CFTypeRef(service))
	acc := cfString(account)
	defer C.CFRelease(C.CFTypeRef(acc))
	setValue(query, C.kSecClass, C.CFTypeRef(C.kSecClassGenericPassword))
	setValue(query, C.kSecAttrService, C.CFTypeRef(service))
	setValue(query, C.kSecAttrAccount, C.CFTypeRef(acc))
	return query
}

func platformSecretStore() SecretStore {
	return &KeychainStore{Fallback: &FileSecretStore{}}
}

func keychainError(what string, status C.OSStatus) error {
	message := C.SecCopyErrorMessageString(status, nil)
	if message == 0 {
		return fmt.Errorf("%s: keychain error %d", what, int(status))
	}
	defer C.CFRelease(C.CFTypeRef(message))
	return fmt.Errorf("%s: %s", what, goString(message))
}

func newDictionary() C.CFMutableDictionaryRef {
	return C.CFDictionaryCreateMutable(C.kCFAllocatorDefault, 0,
		&C.kCFTypeDictionaryKeyCallBacks, &C.kCFTypeDictionaryValueCallBacks)
}

func setValue(dictionary C.CFMutableDictionaryRef, key C.CFStringRef, value C.CFTypeRef) {
	C.ollamaTTSDictionarySet(dictionary, C.CFTypeRef(key), value)
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
	size := C.CFStringGetMaximumSizeForEncoding(length, C.kCFStringEncodingUTF8) + 1
	buffer := make([]byte, size)
	if C.CFStringGetCString(ref, (*C.char)(unsafe.Pointer(&buffer[0])), size, C.kCFStringEncodingUTF8) == C.false {
		return ""
	}
	for i, c := range buffer {
		if c == 0 {
			return string(buffer[:i])
		}
	}
	return string(buffer)
}
