package tts

import (
	"strings"
	"unicode/utf8"
)

// Chunks splits text the way SelectAloud's TextChunker does: CRLF to LF, trim,
// then break at the latest preferred delimiter that still fits, else a hard
// cut. The unit is a Unicode code point (Go rune). Swift counts extended
// grapheme clusters; emoji ZWJ sequences may therefore split here where they
// would not in VOX. ASCII chat text matches the donor tests.
func Chunks(text string, maximum int) []string {
	if maximum <= 0 {
		return nil
	}
	normalised := strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if normalised == "" {
		return nil
	}
	runes := []rune(normalised)
	if len(runes) <= maximum {
		return []string{normalised}
	}

	preferred := []string{"\n\n", ".\n", ". ", "? ", "! ", "\n", "; ", ", ", " "}
	var chunks []string
	remaining := runes
	for len(remaining) > 0 {
		if len(remaining) <= maximum {
			chunks = append(chunks, string(remaining))
			break
		}
		window := string(remaining[:maximum])
		end := maximum
		for _, delimiter := range preferred {
			if i := strings.LastIndex(window, delimiter); i >= 0 {
				end = utf8.RuneCountInString(window[:i+len(delimiter)])
				break
			}
		}
		if end <= 0 {
			end = maximum
		}
		chunk := string(remaining[:end])
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		remaining = remaining[end:]
	}
	return chunks
}

// MaxChunkRunes is the donor per-model limit.
func MaxChunkRunes(modelID string) int {
	if modelID == ModelMultilingual {
		return 9_000
	}
	return 12_000
}
