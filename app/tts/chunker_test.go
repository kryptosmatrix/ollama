package tts

import "testing"

func TestChunksMatchesSelectAloudASCIICases(t *testing.T) {
	if got := Chunks("", 10); len(got) != 0 {
		t.Fatalf("empty: %v", got)
	}
	if got := Chunks(" \n\t ", 10); len(got) != 0 {
		t.Fatalf("whitespace: %v", got)
	}
	got := Chunks("  First line.\r\nSecond line.  ", 100)
	if len(got) != 1 || got[0] != "First line.\nSecond line." {
		t.Fatalf("trim/crlf: %v", got)
	}
	text := "First paragraph.\r\n\r\nSecond paragraph."
	chunks := Chunks(text, 20)
	if chunks[0] != "First paragraph.\n\n" {
		t.Fatalf("paragraph: %q", chunks[0])
	}
	if joined := join(chunks); joined != "First paragraph.\n\nSecond paragraph." {
		t.Fatalf("joined: %q", joined)
	}
	for _, c := range chunks {
		if len([]rune(c)) > 20 {
			t.Fatalf("oversize %q", c)
		}
	}
	sentence := Chunks("First sentence. Second sentence.", 18)
	if sentence[0] != "First sentence. " {
		t.Fatalf("sentence: %q", sentence[0])
	}
	hard := Chunks("abcdefghi", 4)
	if len(hard) != 3 || hard[0] != "abcd" || hard[1] != "efgh" || hard[2] != "i" {
		t.Fatalf("hard: %v", hard)
	}
	if Chunks("text", 0) != nil || Chunks("text", -1) != nil {
		t.Fatal("nonpositive must yield nothing")
	}
}

func TestMaxChunkRunesByModel(t *testing.T) {
	if MaxChunkRunes(ModelMultilingual) != 9_000 {
		t.Fatal("multilingual")
	}
	if MaxChunkRunes(ModelFlash) != 12_000 {
		t.Fatal("flash")
	}
}

func join(s []string) string {
	out := ""
	for _, p := range s {
		out += p
	}
	return out
}
