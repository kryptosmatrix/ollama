package tts

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
)

// Synthesis constants match VOX SpeechSynthesisRequest.
const (
	RequestSchemaVersion = 1
	OutputFormat         = "mp3_44100_128"
	Stability            = 0.5
	SimilarityBoost      = 0.8
	Style                = 0.0
	UseSpeakerBoost      = true

	ModelFlash        = "eleven_flash_v2_5"
	ModelMultilingual = "eleven_multilingual_v2"

	MinSpeed = 0.70
	MaxSpeed = 1.20
)

// SynthesisRequest is one chunk sent to ElevenLabs and one cache identity.
type SynthesisRequest struct {
	Text         string
	VoiceID      string
	ModelID      string
	Speed        float64
	AccountScope []byte
}

// AccountScope is SHA-256 of the API key, never the key itself.
func AccountScope(apiKey string) []byte {
	sum := sha256.Sum256([]byte(apiKey))
	return sum[:]
}

// IdentityMaterial is the length-delimited cache lookup input. Selected text
// is never used as a path.
func (r SynthesisRequest) IdentityMaterial() []byte {
	var b []byte
	b = appendU64(b, RequestSchemaVersion)
	b = appendLen(b, []byte(OutputFormat))
	b = appendLen(b, []byte(r.Text))
	b = appendLen(b, []byte(r.VoiceID))
	b = appendLen(b, []byte(r.ModelID))
	b = appendLen(b, r.AccountScope)
	b = appendU64(b, math.Float64bits(r.Speed))
	b = appendU64(b, math.Float64bits(Stability))
	b = appendU64(b, math.Float64bits(SimilarityBoost))
	b = appendU64(b, math.Float64bits(Style))
	if UseSpeakerBoost {
		b = append(b, 1)
	} else {
		b = append(b, 0)
	}
	return b
}

// Fingerprint is a commit handle the renderer may see. It is a hash of the
// identity, not an HMAC, so it does not require the cache master secret.
func Fingerprint(r SynthesisRequest) string {
	sum := sha256.Sum256(r.IdentityMaterial())
	return hex.EncodeToString(sum[:])
}

func SupportedModel(id string) bool {
	return id == ModelFlash || id == ModelMultilingual
}

func appendU64(b []byte, v uint64) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], v)
	return append(b, buf[:]...)
}

func appendLen(b, payload []byte) []byte {
	b = appendU64(b, uint64(len(payload)))
	return append(b, payload...)
}
