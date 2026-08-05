//go:build windows || darwin

package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DefaultApprovalTimeout bounds how long a tool call waits for an answer. A
// prompt nobody ever answers must not hold a chat open for ever; when it
// expires the call is refused, which is the safe direction.
const DefaultApprovalTimeout = 10 * time.Minute

// ApprovalRequired is implemented by tools that must not run until the user
// says so. A tool that does not implement it runs as before, so the existing
// first-party tools are unaffected.
type ApprovalRequired interface {
	RequiresApproval(args map[string]any) bool
}

// ScopedTool is implemented by tools that want an approval to cover something
// narrower or wider than their name — for example one MCP server's tool rather
// than every tool called "read".
type ScopedTool interface {
	ApprovalScope(args map[string]any) string
}

// ToolRequiresApproval reports whether this call needs the user's agreement.
func ToolRequiresApproval(tool Tool, args map[string]any) bool {
	if tool == nil {
		return false
	}
	if required, ok := tool.(ApprovalRequired); ok {
		return required.RequiresApproval(args)
	}
	return false
}

// ToolApprovalScope returns the key an approval is remembered under.
func ToolApprovalScope(tool Tool, args map[string]any) string {
	if scoped, ok := tool.(ScopedTool); ok {
		if scope := strings.TrimSpace(scoped.ApprovalScope(args)); scope != "" {
			return scope
		}
	}
	if tool == nil {
		return ""
	}
	return strings.TrimSpace(tool.Name())
}

// ApprovalState is what a chat has already agreed to.
type ApprovalState struct {
	mu       sync.RWMutex
	allowAll bool
	scopes   map[string]bool
}

// Allows reports whether scope has already been granted.
func (s *ApprovalState) Allows(scope string) bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.allowAll || s.scopes[scope]
}

// Grant remembers a scope for the rest of this chat.
func (s *ApprovalState) Grant(scope string) {
	if s == nil {
		return
	}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scopes == nil {
		s.scopes = map[string]bool{}
	}
	s.scopes[scope] = true
}

// GrantAll remembers a blanket agreement for the rest of this chat. It is only
// ever set by an explicit user act.
func (s *ApprovalState) GrantAll() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allowAll = true
}

// ApprovalRequest is one pending question to the user.
type ApprovalRequest struct {
	ID       string         `json:"id"`
	ChatID   string         `json:"chatId"`
	ToolName string         `json:"toolName"`
	Scope    string         `json:"scope"`
	Args     map[string]any `json:"args,omitempty"`
}

// ApprovalDecision is the user's answer.
type ApprovalDecision struct {
	// Allow runs the call once.
	Allow bool `json:"allow"`
	// Remember additionally grants the scope for the rest of the chat, so the
	// same tool is not asked about again.
	Remember bool `json:"remember,omitempty"`
	// RememberAll grants everything for the rest of the chat. It exists so the
	// user can say it deliberately, and nothing else ever sets it.
	RememberAll bool `json:"rememberAll,omitempty"`
}

// ErrNotPending is returned when an answer arrives for a request that is no
// longer waiting — because it was already answered, timed out, or its chat
// ended. It is reported rather than ignored so a stale answer cannot look like
// it took effect.
var ErrNotPending = errors.New("no tool call is waiting for that approval")

// Approvals is the rendezvous between a chat's streaming response, which is
// blocked waiting for an answer, and the separate request that carries the
// answer back. It also holds what each chat has already agreed to.
//
// It is safe for concurrent use and is owned by the app server for the life of
// the process.
type Approvals struct {
	// Timeout bounds one wait. Zero means DefaultApprovalTimeout.
	Timeout time.Duration

	mu      sync.Mutex
	pending map[string]chan ApprovalDecision
	states  map[string]*ApprovalState
}

// NewApprovals returns an empty registry.
func NewApprovals() *Approvals {
	return &Approvals{
		pending: map[string]chan ApprovalDecision{},
		states:  map[string]*ApprovalState{},
	}
}

// State returns what the given chat has already agreed to, creating it on first
// use. Approvals are per chat: agreeing to something in one conversation must
// not silently agree to it in another.
func (a *Approvals) State(chatID string) *ApprovalState {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.states == nil {
		a.states = map[string]*ApprovalState{}
	}
	state, ok := a.states[chatID]
	if !ok {
		state = &ApprovalState{}
		a.states[chatID] = state
	}
	return state
}

// Await registers a pending approval, hands it to notify so the user can be
// asked, and blocks until an answer arrives, the caller's context ends, or the
// timeout expires.
//
// Every exit removes the pending entry, so a chat that is cancelled or times
// out cannot leave a request behind that a later answer would resolve.
func (a *Approvals) Await(ctx context.Context, req ApprovalRequest, notify func(ApprovalRequest) error) (ApprovalDecision, error) {
	if req.ID == "" {
		req.ID = uuid.NewString()
	}

	answers := make(chan ApprovalDecision, 1)

	a.mu.Lock()
	if a.pending == nil {
		a.pending = map[string]chan ApprovalDecision{}
	}
	a.pending[req.ID] = answers
	a.mu.Unlock()

	defer a.forget(req.ID)

	if notify != nil {
		if err := notify(req); err != nil {
			return ApprovalDecision{}, err
		}
	}

	timeout := a.Timeout
	if timeout <= 0 {
		timeout = DefaultApprovalTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case decision := <-answers:
		if decision.Allow {
			state := a.State(req.ChatID)
			if decision.RememberAll {
				state.GrantAll()
			} else if decision.Remember {
				state.Grant(req.Scope)
			}
		}
		return decision, nil
	case <-ctx.Done():
		return ApprovalDecision{}, ctx.Err()
	case <-timer.C:
		return ApprovalDecision{}, fmt.Errorf("no answer within %s", timeout)
	}
}

// Resolve delivers the user's answer to the waiting tool call.
func (a *Approvals) Resolve(id string, decision ApprovalDecision) error {
	a.mu.Lock()
	answers, ok := a.pending[id]
	if ok {
		// Remove it here rather than in the waiter, so a second answer for the
		// same request is refused instead of racing the first.
		delete(a.pending, id)
	}
	a.mu.Unlock()

	if !ok {
		return ErrNotPending
	}
	// The channel is buffered, so this never blocks even if the waiter has
	// already given up between the lookup above and now.
	answers <- decision
	return nil
}

// CancelChat refuses every approval still pending for a chat. It is called when
// a chat is deleted, so nothing is left waiting on an answer that can no longer
// be given.
func (a *Approvals) CancelChat(chatID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for id, answers := range a.pending {
		if strings.HasPrefix(id, chatID+":") {
			delete(a.pending, id)
			answers <- ApprovalDecision{}
		}
	}
	delete(a.states, chatID)
}

// NewRequestID builds an identifier that carries its chat, so CancelChat can
// find every request belonging to a chat without a second index.
func NewRequestID(chatID string) string {
	return chatID + ":" + uuid.NewString()
}

// Pending reports how many approvals are waiting. It exists for tests and
// diagnostics; a number that never returns to zero means requests are leaking.
func (a *Approvals) Pending() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.pending)
}

func (a *Approvals) forget(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.pending, id)
}
