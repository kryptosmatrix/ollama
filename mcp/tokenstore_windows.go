//go:build windows

package mcp

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows keeps MCP tokens in the same file as every other platform, encrypted
// at rest with DPAPI.
//
// DPAPI is the right fit here because it needs nothing: no daemon, no unlock
// prompt, no user interaction, and no new dependency — the key is derived from
// the user's own logon credentials by the operating system. The Credential
// Manager was the alternative and was not chosen: it is meant for a small
// number of short values, and it would have split the token store's shape
// across platforms for no gain, since the thing being protected here is a file
// that already exists.
//
// What it buys, stated as narrowly as the macOS store states its own case: a
// token stops being readable by anything that can read the file. A backup, a
// synced folder, a support bundle, a stolen disk — none of them yield a
// credential any more, because the key never leaves the machine and belongs to
// this user's logon.
//
// What it does not buy: protection from a program already running as this user.
// Such a program can call CryptUnprotectData itself. That is the same limit the
// macOS keychain has for an unsigned build, and it is said in the same words.

var (
	crypt32                = windows.NewLazySystemDLL("crypt32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	procLocalFree          = kernel32.NewProc("LocalFree")
)

// cryptProtectUIForbidden refuses to show a prompt. Ollama may be saving a
// token from a background reconnect with nobody at the keyboard, and a dialog
// nobody can answer would hang the connection rather than fail it.
const cryptProtectUIForbidden = 0x1

// dataBlob is DPAPI's DATA_BLOB.
type dataBlob struct {
	size uint32
	data *byte
}

func newBlob(b []byte) dataBlob {
	if len(b) == 0 {
		return dataBlob{}
	}
	return dataBlob{size: uint32(len(b)), data: &b[0]}
}

// bytes copies the blob's contents into Go memory. The caller frees the blob.
func (b dataBlob) bytes() []byte {
	if b.size == 0 || b.data == nil {
		return nil
	}
	out := make([]byte, b.size)
	copy(out, unsafe.Slice(b.data, b.size))
	return out
}

func (b dataBlob) free() {
	if b.data != nil {
		procLocalFree.Call(uintptr(unsafe.Pointer(b.data)))
	}
}

// dpapiProtector encrypts the token store with the current user's DPAPI key.
type dpapiProtector struct{}

func (dpapiProtector) Describe() string {
	return "encrypted for your Windows account so a copy of the file is useless elsewhere"
}

func (dpapiProtector) Protect(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("nothing to encrypt")
	}
	in := newBlob(plaintext)
	var out dataBlob

	// The description string is what Windows shows if anything ever surfaces
	// this blob to a user, so it names the product rather than the file.
	description, err := windows.UTF16PtrFromString("Ollama MCP tokens")
	if err != nil {
		return nil, err
	}

	ok, _, callErr := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)),
		uintptr(unsafe.Pointer(description)),
		0, // no additional entropy: the user's logon key is the secret
		0,
		0,
		cryptProtectUIForbidden,
		uintptr(unsafe.Pointer(&out)),
	)
	// in is kept alive across the call: Go may otherwise collect the slice
	// whose first element the blob points at.
	runtime.KeepAlive(plaintext)
	if ok == 0 {
		return nil, fmt.Errorf("windows could not encrypt the token store: %w", callErr)
	}
	defer out.free()
	return out.bytes(), nil
}

func (dpapiProtector) Unprotect(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, errors.New("nothing to decrypt")
	}
	in := newBlob(ciphertext)
	var out dataBlob

	ok, _, callErr := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0, // the description is not wanted back
		0,
		0,
		0,
		cryptProtectUIForbidden,
		uintptr(unsafe.Pointer(&out)),
	)
	runtime.KeepAlive(ciphertext)
	if ok == 0 {
		// The usual cause is a file carried from another machine or another
		// account, which DPAPI cannot read by design. That is the protection
		// working, so the message says so rather than reading as corruption.
		return nil, fmt.Errorf("windows could not decrypt the token store, which happens when it was written by a different account or on a different machine: %w", callErr)
	}
	defer out.free()
	return out.bytes(), nil
}

// DefaultTokenStore returns where this build keeps tokens: the same file as
// everywhere else, encrypted at rest with DPAPI.
//
// Setting OLLAMA_MCP_TOKENS still names the location, and the contents are
// still protected: an explicit path says where a credential lives, not that it
// should stop being encrypted.
func DefaultTokenStore() TokenStore {
	return &FileTokenStore{Protect: dpapiProtector{}}
}
