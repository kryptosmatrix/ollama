//go:build windows

package tts

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	crypt32                = windows.NewLazySystemDLL("crypt32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	procLocalFree          = kernel32.NewProc("LocalFree")
)

const cryptProtectUIForbidden = 0x1

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
	description, err := windows.UTF16PtrFromString("Ollama TTS secrets")
	if err != nil {
		return nil, err
	}
	ok, _, callErr := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)),
		uintptr(unsafe.Pointer(description)),
		0, 0, 0,
		cryptProtectUIForbidden,
		uintptr(unsafe.Pointer(&out)),
	)
	runtime.KeepAlive(plaintext)
	if ok == 0 {
		return nil, fmt.Errorf("windows could not encrypt the TTS secret store: %w", callErr)
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
		0, 0, 0, 0,
		cryptProtectUIForbidden,
		uintptr(unsafe.Pointer(&out)),
	)
	runtime.KeepAlive(ciphertext)
	if ok == 0 {
		return nil, fmt.Errorf("windows could not decrypt the TTS secret store, which happens when it was written by a different account or on a different machine: %w", callErr)
	}
	defer out.free()
	return out.bytes(), nil
}

func platformProtector() Protector { return dpapiProtector{} }
