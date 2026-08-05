//go:build windows || darwin

package tools

import (
	"context"
	"errors"
	"testing"
	"time"
)

// gatedTool requires approval and scopes it to a server, the way an MCP tool
// will. It also records whether it was ever executed, which is the assertion
// that matters most: a declined call must not run.
type gatedTool struct {
	name     string
	scope    string
	required bool
	executed int
}

func (g *gatedTool) Name() string           { return g.name }
func (g *gatedTool) Description() string    { return "a gated tool" }
func (g *gatedTool) Schema() map[string]any { return map[string]any{"type": "object"} }
func (g *gatedTool) Prompt() string         { return "" }

func (g *gatedTool) Execute(context.Context, map[string]any) (any, string, error) {
	g.executed++
	return nil, "ran", nil
}

func (g *gatedTool) RequiresApproval(map[string]any) bool { return g.required }

func (g *gatedTool) ApprovalScope(map[string]any) string { return g.scope }

// plainTool implements only the base interface, like the existing first-party
// tools, and must never be gated.
type plainTool struct{ name string }

func (p *plainTool) Name() string           { return p.name }
func (p *plainTool) Description() string    { return "a plain tool" }
func (p *plainTool) Schema() map[string]any { return map[string]any{"type": "object"} }
func (p *plainTool) Prompt() string         { return "" }
func (p *plainTool) Execute(context.Context, map[string]any) (any, string, error) {
	return nil, "ran", nil
}

func TestToolRequiresApproval(t *testing.T) {
	if ToolRequiresApproval(&plainTool{name: "web_search"}, nil) {
		t.Error("a tool that does not opt in must not be gated; the existing tools would all break")
	}
	if !ToolRequiresApproval(&gatedTool{name: "files__read", required: true}, nil) {
		t.Error("a tool that requires approval must be gated")
	}
	if ToolRequiresApproval(nil, nil) {
		t.Error("a nil tool must not be gated")
	}
}

func TestToolApprovalScope(t *testing.T) {
	if got := ToolApprovalScope(&gatedTool{name: "files__read", scope: "files__read"}, nil); got != "files__read" {
		t.Errorf("scope = %q", got)
	}
	if got := ToolApprovalScope(&plainTool{name: "web_search"}, nil); got != "web_search" {
		t.Errorf("a tool with no scope falls back to its name, got %q", got)
	}
	if got := ToolApprovalScope(&gatedTool{name: "files__read", scope: "   "}, nil); got != "files__read" {
		t.Errorf("a blank scope must fall back to the name rather than becoming a shared bucket, got %q", got)
	}
}

func awaitAsync(t *testing.T, a *Approvals, req ApprovalRequest) (<-chan ApprovalDecision, <-chan error) {
	t.Helper()
	decisions := make(chan ApprovalDecision, 1)
	errs := make(chan error, 1)
	asked := make(chan struct{})

	go func() {
		decision, err := a.Await(t.Context(), req, func(ApprovalRequest) error {
			close(asked)
			return nil
		})
		decisions <- decision
		errs <- err
	}()

	select {
	case <-asked:
	case <-time.After(5 * time.Second):
		t.Fatal("the user was never asked")
	}
	return decisions, errs
}

func TestAwaitBlocksUntilAnswered(t *testing.T) {
	approvals := NewApprovals()
	req := ApprovalRequest{ID: NewRequestID("chat-1"), ChatID: "chat-1", ToolName: "files__read", Scope: "files__read"}

	decisions, errs := awaitAsync(t, approvals, req)

	if approvals.Pending() != 1 {
		t.Fatalf("pending = %d, want 1 while waiting", approvals.Pending())
	}
	select {
	case decision := <-decisions:
		t.Fatalf("Await returned %+v before anyone answered", decision)
	case <-time.After(50 * time.Millisecond):
	}

	if err := approvals.Resolve(req.ID, ApprovalDecision{Allow: true}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if err := <-errs; err != nil {
		t.Fatalf("Await: %v", err)
	}
	if decision := <-decisions; !decision.Allow {
		t.Error("the answer did not reach the waiting call")
	}
	if approvals.Pending() != 0 {
		t.Errorf("pending = %d after answering; requests are leaking", approvals.Pending())
	}
}

func TestResolveIsOnlyEverHonouredOnce(t *testing.T) {
	approvals := NewApprovals()
	req := ApprovalRequest{ID: NewRequestID("chat-1"), ChatID: "chat-1", Scope: "files__read"}
	_, errs := awaitAsync(t, approvals, req)

	if err := approvals.Resolve(req.ID, ApprovalDecision{Allow: true}); err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	<-errs

	err := approvals.Resolve(req.ID, ApprovalDecision{Allow: true})
	if !errors.Is(err, ErrNotPending) {
		t.Errorf("second Resolve = %v, want ErrNotPending; a repeated answer must not look like it took effect", err)
	}
}

func TestResolveUnknownRequest(t *testing.T) {
	approvals := NewApprovals()
	if err := approvals.Resolve("chat-1:nope", ApprovalDecision{Allow: true}); !errors.Is(err, ErrNotPending) {
		t.Errorf("Resolve = %v, want ErrNotPending", err)
	}
}

func TestAwaitStopsWhenTheChatIsCancelled(t *testing.T) {
	approvals := NewApprovals()
	ctx, cancel := context.WithCancel(t.Context())

	errs := make(chan error, 1)
	asked := make(chan struct{})
	go func() {
		_, err := approvals.Await(ctx, ApprovalRequest{ID: NewRequestID("chat-1"), ChatID: "chat-1"}, func(ApprovalRequest) error {
			close(asked)
			return nil
		})
		errs <- err
	}()
	<-asked

	cancel()
	select {
	case err := <-errs:
		if err == nil {
			t.Error("a cancelled chat must not return an approval")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Await did not return when the chat was cancelled; the response would hang for ever")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if approvals.Pending() == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("pending = %d after cancellation; a later answer could resolve a request nobody is waiting on", approvals.Pending())
}

func TestAwaitTimesOutAndRefuses(t *testing.T) {
	approvals := NewApprovals()
	approvals.Timeout = 150 * time.Millisecond

	decision, err := approvals.Await(t.Context(), ApprovalRequest{ID: NewRequestID("chat-1"), ChatID: "chat-1"}, nil)
	if err == nil {
		t.Fatal("a prompt nobody answers must eventually fail rather than hold the chat open")
	}
	if decision.Allow {
		t.Error("a timeout must never allow the call")
	}
	if approvals.Pending() != 0 {
		t.Errorf("pending = %d after a timeout", approvals.Pending())
	}
}

func TestAwaitReturnsANotifyFailure(t *testing.T) {
	approvals := NewApprovals()
	_, err := approvals.Await(t.Context(), ApprovalRequest{ID: NewRequestID("chat-1")}, func(ApprovalRequest) error {
		return errors.New("the client went away")
	})
	if err == nil {
		t.Fatal("if the user cannot be asked, the call must not proceed")
	}
	if approvals.Pending() != 0 {
		t.Errorf("pending = %d after a failed notify", approvals.Pending())
	}
}

func TestRememberGrantsTheScopeForTheChatOnly(t *testing.T) {
	approvals := NewApprovals()
	req := ApprovalRequest{ID: NewRequestID("chat-1"), ChatID: "chat-1", Scope: "files__read"}
	_, errs := awaitAsync(t, approvals, req)

	if err := approvals.Resolve(req.ID, ApprovalDecision{Allow: true, Remember: true}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	<-errs

	if !approvals.State("chat-1").Allows("files__read") {
		t.Error("remember did not grant the scope")
	}
	if approvals.State("chat-1").Allows("files__write") {
		t.Error("remembering one tool must not grant its siblings")
	}
	if approvals.State("chat-2").Allows("files__read") {
		t.Error("an approval in one chat must not apply to another")
	}
}

func TestRememberAllIsSeparateAndDeliberate(t *testing.T) {
	approvals := NewApprovals()
	req := ApprovalRequest{ID: NewRequestID("chat-1"), ChatID: "chat-1", Scope: "files__read"}
	_, errs := awaitAsync(t, approvals, req)

	if err := approvals.Resolve(req.ID, ApprovalDecision{Allow: true, RememberAll: true}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	<-errs

	if !approvals.State("chat-1").Allows("anything__at__all") {
		t.Error("rememberAll did not grant everything")
	}
}

func TestDecliningNeverGrantsAnything(t *testing.T) {
	approvals := NewApprovals()
	req := ApprovalRequest{ID: NewRequestID("chat-1"), ChatID: "chat-1", Scope: "files__read"}
	decisions, errs := awaitAsync(t, approvals, req)

	// Remember set alongside a refusal must not be read as agreement.
	if err := approvals.Resolve(req.ID, ApprovalDecision{Allow: false, Remember: true, RememberAll: true}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	<-errs

	if decision := <-decisions; decision.Allow {
		t.Fatal("the decision should be a refusal")
	}
	if approvals.State("chat-1").Allows("files__read") {
		t.Error("declining must never grant the scope")
	}
	if approvals.State("chat-1").Allows("anything") {
		t.Error("declining must never grant everything")
	}
}

func TestCancelChatRefusesOnlyThatChat(t *testing.T) {
	approvals := NewApprovals()

	doomed := ApprovalRequest{ID: NewRequestID("chat-1"), ChatID: "chat-1", Scope: "files__read"}
	other := ApprovalRequest{ID: NewRequestID("chat-2"), ChatID: "chat-2", Scope: "files__read"}

	doomedDecisions, doomedErrs := awaitAsync(t, approvals, doomed)
	_, otherErrs := awaitAsync(t, approvals, other)

	approvals.CancelChat("chat-1")

	select {
	case err := <-doomedErrs:
		if err != nil {
			t.Fatalf("Await: %v", err)
		}
		if decision := <-doomedDecisions; decision.Allow {
			t.Error("a cancelled chat's pending call must be refused")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelling the chat did not release its waiting call")
	}

	select {
	case err := <-otherErrs:
		t.Fatalf("the other chat's approval was released too: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	if err := approvals.Resolve(other.ID, ApprovalDecision{Allow: true}); err != nil {
		t.Fatalf("the other chat should still be answerable: %v", err)
	}
	<-otherErrs
}

func TestConcurrentApprovalsDoNotInterfere(t *testing.T) {
	approvals := NewApprovals()

	const count = 20
	type waiter struct {
		req       ApprovalRequest
		decisions <-chan ApprovalDecision
		errs      <-chan error
	}
	waiters := make([]waiter, 0, count)
	for i := range count {
		req := ApprovalRequest{ID: NewRequestID("chat-1"), ChatID: "chat-1", Scope: "scope"}
		decisions, errs := awaitAsync(t, approvals, req)
		waiters = append(waiters, waiter{req: req, decisions: decisions, errs: errs})
		_ = i
	}

	if approvals.Pending() != count {
		t.Fatalf("pending = %d, want %d", approvals.Pending(), count)
	}

	// Answer them out of order; each waiter must get its own answer.
	for i := len(waiters) - 1; i >= 0; i-- {
		allow := i%2 == 0
		if err := approvals.Resolve(waiters[i].req.ID, ApprovalDecision{Allow: allow}); err != nil {
			t.Fatalf("Resolve %d: %v", i, err)
		}
	}
	for i, w := range waiters {
		if err := <-w.errs; err != nil {
			t.Fatalf("waiter %d: %v", i, err)
		}
		if got := (<-w.decisions).Allow; got != (i%2 == 0) {
			t.Errorf("waiter %d got %v; answers were delivered to the wrong caller", i, got)
		}
	}
	if approvals.Pending() != 0 {
		t.Errorf("pending = %d at the end", approvals.Pending())
	}
}
