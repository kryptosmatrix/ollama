//go:build windows || darwin

package store

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ollama/ollama/app/types/not"
)

// R-01/R-06: absence retains its established sentinel for both read modes.
func TestChatReadMissingHeaderIsNotFound(t *testing.T) {
	s, cleanup := setupTestStore(t)
	t.Cleanup(cleanup)
	for _, attachments := range []bool{false, true} {
		chat, err := s.ChatWithOptions("absent-chat", attachments)
		if chat != nil || !errors.Is(err, not.Found) {
			t.Errorf("attachments=%t: chat=%#v error=%v; want absence", attachments, chat, err)
		}
	}
}

// R-01/R-05: missing child rows are empty collections, not missing chats.
func TestChatReadValidEmptyAndPopulated(t *testing.T) {
	s, cleanup := setupTestStore(t)
	t.Cleanup(cleanup)
	for _, populated := range []bool{false, true} {
		want := NewChat("valid-chat")
		want.Title = "retained title"
		if populated {
			want.Messages = []Message{
				NewMessage("user", "first original body", nil),
				NewMessage("assistant", "second original body", nil),
			}
		}
		if err := s.SetChat(*want); err != nil {
			t.Fatal(err)
		}
		for _, attachments := range []bool{false, true} {
			got, err := s.ChatWithOptions(want.ID, attachments)
			if err != nil || got == nil {
				t.Fatalf("populated=%t attachments=%t: valid read failed: %v", populated, attachments, err)
			}
			if got.ID != want.ID || got.Title != want.Title || len(got.Messages) != len(want.Messages) {
				t.Fatalf("R-05: valid chat changed: got=%#v want=%#v", got, want)
			}
			for i, msg := range want.Messages {
				if got.Messages[i].Content != msg.Content || got.Messages[i].Role != msg.Role {
					t.Errorf("R-05: message %d changed: %#v", i, got.Messages[i])
				}
			}
		}
	}
}

// R-01/R-02: the header is present and the recorded message reaches Scan.
func TestChatReadRejectsMalformedMessage(t *testing.T) {
	s, cleanup := setupTestStore(t)
	t.Cleanup(cleanup)
	want := NewChat("malformed-chat")
	want.Messages = []Message{NewMessage("user", "original message", nil)}
	if err := s.SetChat(*want); err != nil {
		t.Fatal(err)
	}
	result, err := s.db.conn.Exec("UPDATE messages SET stream = 'invalid_boolean' WHERE chat_id = ?", want.ID)
	if err != nil {
		t.Fatal(err)
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		t.Fatalf("fault affected %d rows: %v", n, err)
	}
	var headerID, rawValue string
	if err := s.db.conn.QueryRow("SELECT id FROM chats WHERE id = ?", want.ID).Scan(&headerID); err != nil || headerID != want.ID {
		t.Fatalf("original header missing: id=%q err=%v", headerID, err)
	}
	if err := s.db.conn.QueryRow("SELECT stream FROM messages WHERE chat_id = ?", want.ID).Scan(&rawValue); err != nil || rawValue != "invalid_boolean" {
		t.Fatalf("fault value not delivered: value=%q err=%v", rawValue, err)
	}
	for _, attachments := range []bool{false, true} {
		got, err := s.ChatWithOptions(want.ID, attachments)
		if got != nil || err == nil {
			t.Fatalf("R-01: malformed message returned success: %#v", got)
		}
		if errors.Is(err, not.Found) || errors.Is(err, sql.ErrNoRows) {
			t.Errorf("R-01: scan failure became absence: %v", err)
		}
		if errors.Unwrap(err) == nil || !strings.Contains(err.Error(), "scan message:") {
			t.Errorf("R-02: scan failure lost its cause/context: %v", err)
		}
	}
}

// R-02: failure before the query must not authorise creation either.
func TestChatReadInitialisationFailureIsNotAbsence(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "regular-file")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Store{DBPath: filepath.Join(blocker, "db.sqlite")}
	t.Cleanup(func() { s.Close() })
	for _, attachments := range []bool{false, true} {
		chat, err := s.ChatWithOptions("any-chat", attachments)
		var cause *os.PathError
		if chat != nil || err == nil || errors.Is(err, not.Found) || !errors.As(err, &cause) {
			t.Errorf("R-02: initialisation failure lost its path cause: chat=%#v err=%v", chat, err)
		}
	}
}

// R-02: a native database failure keeps the original error identity, rather
// than merely acquiring a different error message containing the same words.
func TestChatReadPreservesDatabaseErrorCause(t *testing.T) {
	s, cleanup := setupTestStore(t)
	t.Cleanup(cleanup)
	if err := s.SetChat(*NewChat("existing-chat")); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	cause := s.db.conn.Ping()
	if cause == nil {
		t.Fatal("closing the real database did not produce a database error")
	}
	for _, attachments := range []bool{false, true} {
		chat, err := s.ChatWithOptions("existing-chat", attachments)
		if chat != nil || err == nil {
			t.Fatalf("attachments=%t: closed database returned chat=%#v error=%v", attachments, chat, err)
		}
		if errors.Is(err, not.Found) {
			t.Errorf("R-02: database failure became absence: %v", err)
		}
		if !errors.Is(err, cause) {
			t.Errorf("R-02: database error identity lost: got=%v cause=%v", err, cause)
		}
	}
}
