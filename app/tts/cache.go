package tts

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	cacheRetention     = 24 * time.Hour
	cacheMaxEntries    = 256
	cacheMaxBytes      = 512 * 1024 * 1024
	manifestFileName   = "manifest.enc"
	audioFileName      = "audio.enc"
	stagingPrefix      = ".staging-"
	manifestMaxBytes   = 64 * 1024
	encryptionOverhead = 1 * 1024 * 1024
	hkdfSalt           = "Ollama speech cache v1"
	lookupKeyInfo      = "lookup HMAC key"
	encryptionKeyInfo  = "AES-GCM encryption key"
	manifestAAD        = "Ollama encrypted cache manifest v1"
	audioAADPrefix     = "Ollama encrypted cache audio v1"
	entryTimeLayout    = "20060102T150405Z"
)

var errCacheUnavailable = errors.New("the private audio cache is temporarily unavailable")

type cacheManifest struct {
	Version                 int       `json:"version"`
	LookupCode              []byte    `json:"lookup_code"`
	CreatedAt               time.Time `json:"created_at"`
	PlaintextAudioByteCount int       `json:"plaintext_audio_byte_count"`
}

type derivedKeys struct {
	lookup     []byte
	encryption []byte
}

type cacheEntry struct {
	dir      string
	manifest cacheManifest
	bytes    int64
}

// Cache is an optional encrypted exact-match store. Writes happen only through
// Store, which the service must not call until the client commits playback.
type Cache struct {
	Root    string
	Secrets SecretStore
	Now     func() time.Time
	Limits  struct {
		MaxEntries int
		MaxBytes   int64
	}

	mu   sync.Mutex
	keys *derivedKeys
}

func (c *Cache) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now().UTC()
}

func (c *Cache) limits() (int, int64) {
	entries := c.Limits.MaxEntries
	if entries <= 0 {
		entries = cacheMaxEntries
	}
	bytes := c.Limits.MaxBytes
	if bytes <= 0 {
		bytes = cacheMaxBytes
	}
	return entries, bytes
}

func (c *Cache) root() (string, error) {
	if strings.TrimSpace(c.Root) != "" {
		return c.Root, nil
	}
	return defaultCacheRoot()
}

func (c *Cache) Lookup(req SynthesisRequest) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	root, err := c.root()
	if err != nil {
		return nil, err
	}
	if !dirReady(root) {
		return nil, nil
	}
	keys, err := c.keysLocked()
	if err != nil {
		return nil, err
	}
	code := lookupCode(req, keys.lookup)
	entries, err := c.validatedLocked(root, keys)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if hmac.Equal(e.manifest.LookupCode, code) {
			audio, err := c.decryptAudio(e, keys.encryption)
			if err != nil || audio == nil {
				_ = os.RemoveAll(e.dir)
				continue
			}
			return audio, nil
		}
	}
	return nil, nil
}

func (c *Cache) Store(req SynthesisRequest, audio []byte) error {
	if len(audio) == 0 {
		return errCacheUnavailable
	}
	_, maxBytes := c.limits()
	if int64(len(audio)) > maxBytes {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	root, err := c.root()
	if err != nil {
		return err
	}
	if err := ensurePrivateDir(root); err != nil {
		return err
	}
	keys, err := c.keysLocked()
	if err != nil {
		return err
	}
	now := c.now()
	code := lookupCode(req, keys.lookup)
	entries, err := c.validatedLocked(root, keys)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if hmac.Equal(e.manifest.LookupCode, code) {
			_ = os.RemoveAll(e.dir)
		}
	}
	name := now.UTC().Format(entryTimeLayout) + "_" + uuid.NewString()
	finalDir := filepath.Join(root, name)
	staging := filepath.Join(root, stagingPrefix+uuid.NewString())
	if err := ensurePrivateDir(staging); err != nil {
		return err
	}
	manifest := cacheManifest{
		Version:                 1,
		LookupCode:              code,
		CreatedAt:               now.UTC(),
		PlaintextAudioByteCount: len(audio),
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		os.RemoveAll(staging)
		return err
	}
	encManifest, err := seal(keys.encryption, manifestJSON, []byte(manifestAAD))
	if err != nil {
		os.RemoveAll(staging)
		return err
	}
	encAudio, err := seal(keys.encryption, audio, audioAAD(manifest))
	if err != nil {
		os.RemoveAll(staging)
		return err
	}
	if err := os.WriteFile(filepath.Join(staging, manifestFileName), encManifest, 0o600); err != nil {
		os.RemoveAll(staging)
		return err
	}
	if err := os.WriteFile(filepath.Join(staging, audioFileName), encAudio, 0o600); err != nil {
		os.RemoveAll(staging)
		return err
	}
	if err := os.Rename(staging, finalDir); err != nil {
		os.RemoveAll(staging)
		return err
	}
	_ = os.Chmod(finalDir, 0o700)
	return c.enforceLocked(root, keys)
}

func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	root, err := c.root()
	if err != nil {
		return err
	}
	if err := os.RemoveAll(root); err != nil {
		return err
	}
	c.keys = nil
	if c.Secrets != nil {
		_ = c.Secrets.DeleteCacheMaster()
	}
	return nil
}

func (c *Cache) keysLocked() (*derivedKeys, error) {
	if c.keys != nil {
		return c.keys, nil
	}
	if c.Secrets == nil {
		return nil, errCacheUnavailable
	}
	master, err := c.Secrets.LoadCacheMaster()
	if err != nil {
		return nil, err
	}
	if len(master) != masterSecretBytes {
		master = make([]byte, masterSecretBytes)
		if _, err := io.ReadFull(rand.Reader, master); err != nil {
			return nil, err
		}
		if err := c.Secrets.SaveCacheMaster(master); err != nil {
			return nil, err
		}
	}
	lookup, err := hkdf.Key(sha256.New, master, []byte(hkdfSalt), lookupKeyInfo, 32)
	if err != nil {
		return nil, err
	}
	enc, err := hkdf.Key(sha256.New, master, []byte(hkdfSalt), encryptionKeyInfo, 32)
	if err != nil {
		return nil, err
	}
	c.keys = &derivedKeys{lookup: lookup, encryption: enc}
	return c.keys, nil
}

func (c *Cache) validatedLocked(root string, keys *derivedKeys) ([]cacheEntry, error) {
	entries := []cacheEntry{}
	children, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	now := c.now()
	for _, child := range children {
		name := child.Name()
		if strings.HasPrefix(name, stagingPrefix) {
			_ = os.RemoveAll(filepath.Join(root, name))
			continue
		}
		path := filepath.Join(root, name)
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		entry, ok := c.validateOne(path, keys, now)
		if !ok {
			_ = os.RemoveAll(path)
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (c *Cache) validateOne(dir string, keys *derivedKeys, now time.Time) (cacheEntry, bool) {
	manifestPath := filepath.Join(dir, manifestFileName)
	audioPath := filepath.Join(dir, audioFileName)
	encManifest, err := readPrivateRegular(manifestPath, manifestMaxBytes)
	if err != nil {
		return cacheEntry{}, false
	}
	plain, err := open(keys.encryption, encManifest, []byte(manifestAAD))
	if err != nil {
		return cacheEntry{}, false
	}
	var manifest cacheManifest
	if json.Unmarshal(plain, &manifest) != nil || manifest.Version != 1 {
		return cacheEntry{}, false
	}
	if len(manifest.LookupCode) != sha256.Size || manifest.PlaintextAudioByteCount <= 0 {
		return cacheEntry{}, false
	}
	if now.Sub(manifest.CreatedAt) >= cacheRetention || manifest.CreatedAt.After(now) {
		return cacheEntry{}, false
	}
	base := filepath.Base(dir)
	if len(base) < 16 || base[:16] != manifest.CreatedAt.UTC().Format(entryTimeLayout) {
		return cacheEntry{}, false
	}
	st, err := os.Lstat(audioPath)
	if err != nil || !st.Mode().IsRegular() {
		return cacheEntry{}, false
	}
	mst, _ := os.Lstat(manifestPath)
	stored := st.Size()
	if mst != nil {
		stored += mst.Size()
	}
	return cacheEntry{dir: dir, manifest: manifest, bytes: stored}, true
}

func (c *Cache) decryptAudio(e cacheEntry, key []byte) ([]byte, error) {
	_, maxBytes := c.limits()
	enc, err := readPrivateRegular(filepath.Join(e.dir, audioFileName), maxBytes+encryptionOverhead)
	if err != nil {
		return nil, err
	}
	plain, err := open(key, enc, audioAAD(e.manifest))
	if err != nil {
		return nil, nil
	}
	if len(plain) != e.manifest.PlaintextAudioByteCount {
		return nil, nil
	}
	return plain, nil
}

func (c *Cache) enforceLocked(root string, keys *derivedKeys) error {
	maxEntries, maxBytes := c.limits()
	entries, err := c.validatedLocked(root, keys)
	if err != nil {
		return err
	}
	// oldest first
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].manifest.CreatedAt.Before(entries[i].manifest.CreatedAt) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	var total int64
	for _, e := range entries {
		total += e.bytes
	}
	for len(entries) > maxEntries || total > maxBytes {
		oldest := entries[0]
		_ = os.RemoveAll(oldest.dir)
		total -= oldest.bytes
		entries = entries[1:]
	}
	return nil
}

func lookupCode(req SynthesisRequest, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(req.IdentityMaterial())
	return mac.Sum(nil)
}

func seal(key, plaintext, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, aad), nil
}

func open(key, combined, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(combined) < ns {
		return nil, errCacheUnavailable
	}
	return gcm.Open(nil, combined[:ns], combined[ns:], aad)
}

func audioAAD(m cacheManifest) []byte {
	var b []byte
	b = append(b, audioAADPrefix...)
	b = append(b, m.LookupCode...)
	b = append(b, []byte(fmt.Sprintf("%v", m.CreatedAt.UTC().Unix()))...)
	return b
}

func dirReady(root string) bool {
	info, err := os.Lstat(root)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func readPrivateRegular(path string, max int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errCacheUnavailable
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		return nil, errCacheUnavailable
	}
	if info.Size() <= 0 || info.Size() > max {
		return nil, errCacheUnavailable
	}
	return os.ReadFile(path)
}
