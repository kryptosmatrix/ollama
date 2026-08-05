package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/version"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Status is where a configured server currently stands.
type Status string

const (
	// StatusInvalid means the configuration entry itself is unusable; the
	// server was never contacted. See Config.Problems.
	StatusInvalid Status = "invalid"
	// StatusDisabled means the user has switched the server off.
	StatusDisabled Status = "disabled"
	// StatusConnecting means a connection attempt is in flight.
	StatusConnecting Status = "connecting"
	// StatusConnected means the session is live and its tools are usable.
	StatusConnected Status = "connected"
	// StatusFailed means the last connection attempt did not succeed. Err
	// carries the reason.
	StatusFailed Status = "failed"
)

const (
	// DefaultConnectTimeout bounds one connection attempt, including the
	// server's own start-up.
	DefaultConnectTimeout = 30 * time.Second
	// DefaultCallTimeout bounds one tool call.
	DefaultCallTimeout = 2 * time.Minute
	// DefaultResultLimit caps how much text one tool result may contribute to
	// the model's context. The agent harness truncates again at its own limit;
	// this one exists so a hostile or broken server cannot force that work.
	DefaultResultLimit = 60000
	// httpConnectAttempts is how many times an http connection is retried.
	// Stdio is deliberately not retried: a failed exec is deterministic, and
	// retrying it only spawns more processes.
	httpConnectAttempts = 3
	// httpRetryBackoff is the pause before the first http retry; it doubles.
	httpRetryBackoff = 500 * time.Millisecond
)

// SkippedTool records a tool Ollama refused to offer to the model, and why.
// These are surfaced rather than swallowed: a user whose tool is missing needs
// to know it was rejected rather than absent.
type SkippedTool struct {
	Name   string
	Reason string
}

// ServerState is a snapshot of one configured server.
type ServerState struct {
	Name    string
	Spec    *ServerSpec
	Status  Status
	Err     error
	Tools   []Tool
	Skipped []SkippedTool

	// ToolsDigest is a stable hash over the tools this server advertised,
	// including their descriptions and schemas. A change between sessions means
	// the server is offering the model something different from what the user
	// approved, which is the "rug pull" case the UI must surface.
	ToolsDigest string

	// ServerInfo is the peer's self-description from initialize.
	ServerInfo string
	// ProtocolVersion is the version negotiated with the peer.
	ProtocolVersion string
}

// CallResult is the outcome of one tool call.
type CallResult struct {
	// Content is the text handed to the model.
	Content string
	// Structured is the server's structured result, if it sent one. It is kept
	// for persistence and display; the model sees Content.
	Structured any
	// IsError reports that the server itself reported the call as failed. This
	// is not a transport error: the model is meant to see it and react.
	IsError bool
}

// Options configure a Manager.
type Options struct {
	Logger         *slog.Logger
	ConnectTimeout time.Duration
	CallTimeout    time.Duration
	ResultLimit    int

	// newTransport is a seam for tests, which connect an in-process server over
	// the SDK's in-memory transport. Production leaves it nil and gets the real
	// subprocess and HTTP transports.
	newTransport func(context.Context, *ServerSpec) (sdk.Transport, error)
}

// Manager owns the live connections to configured MCP servers. It is safe for
// concurrent use.
type Manager struct {
	opts Options

	// lifetime bounds every server process this manager starts. It is
	// deliberately not the caller's context: a connect timeout must bound the
	// handshake, not the life of the process, and Close must be able to reap
	// children whether or not the caller's context is still live.
	lifetime  context.Context
	endOfLife context.CancelFunc

	mu      sync.RWMutex
	states  map[string]*ServerState
	clients map[string]*sdk.ClientSession
	closed  bool
}

// NewManager returns a manager with no connections.
func NewManager(opts Options) *Manager {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.ConnectTimeout <= 0 {
		opts.ConnectTimeout = DefaultConnectTimeout
	}
	if opts.CallTimeout <= 0 {
		opts.CallTimeout = DefaultCallTimeout
	}
	if opts.ResultLimit <= 0 {
		opts.ResultLimit = DefaultResultLimit
	}
	if opts.newTransport == nil {
		opts.newTransport = newTransport
	}
	lifetime, endOfLife := context.WithCancel(context.Background())
	return &Manager{
		opts:      opts,
		lifetime:  lifetime,
		endOfLife: endOfLife,
		states:    map[string]*ServerState{},
		clients:   map[string]*sdk.ClientSession{},
	}
}

// Connect brings the manager into line with cfg: every valid, enabled server is
// connected concurrently, and every other configured server is recorded with
// the status that explains why it was not.
//
// Connect always returns a state for every configured server. A server that
// fails to connect is not an error for the call as a whole — one unreachable
// server must not stop the others being usable.
func (m *Manager) Connect(ctx context.Context, cfg *Config) {
	if cfg == nil {
		return
	}
	problems := cfg.Problems()

	type job struct {
		name string
		spec *ServerSpec
	}
	var jobs []job

	for _, name := range cfg.Names() {
		spec, _ := cfg.Get(name)
		switch {
		case problems[name] != nil:
			m.setState(&ServerState{Name: name, Spec: spec, Status: StatusInvalid, Err: problems[name]})
		case spec.Disabled:
			m.setState(&ServerState{Name: name, Spec: spec, Status: StatusDisabled})
		default:
			m.setState(&ServerState{Name: name, Spec: spec, Status: StatusConnecting})
			jobs = append(jobs, job{name: name, spec: spec})
		}
	}

	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.connectOne(ctx, j.spec)
		}()
	}
	wg.Wait()
}

func (m *Manager) connectOne(ctx context.Context, spec *ServerSpec) {
	attempts := 1
	if spec.transport() == TransportHTTP {
		attempts = httpConnectAttempts
	}

	backoff := httpRetryBackoff
	var err error
	for attempt := range attempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				err = ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
			if ctx.Err() != nil {
				break
			}
			m.opts.Logger.Debug("retrying mcp connection", "server", spec.Name, "attempt", attempt+1)
		}
		if err = m.dial(ctx, spec); err == nil {
			return
		}
	}

	m.setState(&ServerState{Name: spec.Name, Spec: spec, Status: StatusFailed, Err: err})
	m.opts.Logger.Warn("mcp server unavailable", "server", spec.Name, "error", err)
}

func (m *Manager) dial(ctx context.Context, spec *ServerSpec) error {
	dialCtx, cancelDial := context.WithTimeout(ctx, m.opts.ConnectTimeout)
	defer cancelDial()

	// The process is tied to the manager's lifetime, not to dialCtx. Tying it
	// to the dial context would kill the server the instant the handshake
	// finished, because that timeout is cancelled on the way out of this
	// function — a defect this code had, and which
	// TestConnectTimeoutDoesNotKillTheServer exists to catch.
	transport, err := m.opts.newTransport(m.lifetime, spec)
	if err != nil {
		return err
	}

	client := sdk.NewClient(&sdk.Implementation{Name: "ollama", Version: version.Version}, nil)
	session, err := client.Connect(dialCtx, transport, nil)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	tools, skipped, err := listTools(dialCtx, spec.Name, session)
	if err != nil {
		session.Close()
		return fmt.Errorf("list tools: %w", err)
	}

	state := &ServerState{
		Name:        spec.Name,
		Spec:        spec,
		Status:      StatusConnected,
		Tools:       tools,
		Skipped:     skipped,
		ToolsDigest: toolsDigest(tools),
	}
	if info := session.InitializeResult(); info != nil {
		if info.ServerInfo != nil {
			state.ServerInfo = strings.TrimSpace(info.ServerInfo.Name + " " + info.ServerInfo.Version)
		}
		state.ProtocolVersion = info.ProtocolVersion
	}

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		session.Close()
		return errors.New("manager is closed")
	}
	if previous := m.clients[spec.Name]; previous != nil {
		previous.Close()
	}
	m.clients[spec.Name] = session
	m.states[spec.Name] = state
	m.mu.Unlock()
	return nil
}

// listTools reads the server's tool list and converts each entry, dropping the
// ones Ollama cannot honestly offer and recording why.
func listTools(ctx context.Context, server string, session *sdk.ClientSession) ([]Tool, []SkippedTool, error) {
	result, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, nil, err
	}

	var tools []Tool
	var skipped []SkippedTool
	for _, advertised := range result.Tools {
		if advertised == nil {
			continue
		}
		if err := validateToolName(server, advertised.Name); err != nil {
			skipped = append(skipped, SkippedTool{Name: advertised.Name, Reason: err.Error()})
			continue
		}

		schema, err := marshalSchema(advertised.InputSchema)
		if err != nil {
			skipped = append(skipped, SkippedTool{Name: advertised.Name, Reason: err.Error()})
			continue
		}

		tool := Tool{
			Server:      server,
			Name:        advertised.Name,
			Title:       advertised.Title,
			Description: advertised.Description,
			InputSchema: schema,
		}
		// Convert now rather than at call time so an unusable schema is
		// reported to the user as a skipped tool, not discovered by the model
		// mid-conversation.
		if _, err := tool.Schema(); err != nil {
			skipped = append(skipped, SkippedTool{Name: advertised.Name, Reason: err.Error()})
			continue
		}
		tools = append(tools, tool)
	}

	slices.SortFunc(tools, func(a, b Tool) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(skipped, func(a, b SkippedTool) int { return strings.Compare(a.Name, b.Name) })
	return tools, skipped, nil
}

// States returns a snapshot of every configured server, ordered by name.
func (m *Manager) States() []ServerState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make([]ServerState, 0, len(m.states))
	for _, name := range slices.Sorted(maps.Keys(m.states)) {
		states = append(states, *m.states[name])
	}
	return states
}

// State returns the snapshot for one server.
func (m *Manager) State(name string) (ServerState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, ok := m.states[name]
	if !ok {
		return ServerState{}, false
	}
	return *state, true
}

// Tools returns every tool offered by every connected server, in a stable
// order. Tools from servers that are disabled, invalid or failed are absent.
func (m *Manager) Tools() []Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tools []Tool
	for _, name := range slices.Sorted(maps.Keys(m.states)) {
		state := m.states[name]
		if state.Status != StatusConnected {
			continue
		}
		tools = append(tools, state.Tools...)
	}
	return tools
}

// Schemas returns the Ollama tool definitions for every usable tool. A tool
// whose schema fails to convert here has already been filtered at connect time,
// so a failure at this point is a defect rather than a server's fault and is
// reported instead of silently dropped.
func (m *Manager) Schemas() (api.Tools, error) {
	var schemas api.Tools
	var errs []error
	for _, tool := range m.Tools() {
		fn, err := tool.Schema()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		schemas = append(schemas, api.Tool{Type: "function", Function: fn})
	}
	return schemas, errors.Join(errs...)
}

// Call invokes a namespaced tool. The name is the one the model was given —
// "<server>__<tool>" — and is split back into its halves here, so a caller
// never has to reason about the namespace.
func (m *Manager) Call(ctx context.Context, qualified string, args map[string]any) (CallResult, error) {
	server, name, ok := SplitQualifiedName(qualified)
	if !ok {
		return CallResult{}, fmt.Errorf("%q is not a namespaced MCP tool name", qualified)
	}

	m.mu.RLock()
	session := m.clients[server]
	state := m.states[server]
	limit := m.opts.ResultLimit
	timeout := m.opts.CallTimeout
	m.mu.RUnlock()

	if state == nil {
		return CallResult{}, fmt.Errorf("no MCP server named %q is configured", server)
	}
	if session == nil || state.Status != StatusConnected {
		if state.Err != nil {
			return CallResult{}, fmt.Errorf("MCP server %q is %s: %w", server, state.Status, state.Err)
		}
		return CallResult{}, fmt.Errorf("MCP server %q is %s", server, state.Status)
	}
	if !slices.ContainsFunc(state.Tools, func(t Tool) bool { return t.Name == name }) {
		return CallResult{}, fmt.Errorf("MCP server %q does not offer a tool named %q", server, name)
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := session.CallTool(callCtx, &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return CallResult{}, fmt.Errorf("call %s: %w", qualified, err)
	}

	return CallResult{
		Content:    flattenContent(result.Content, limit),
		Structured: result.StructuredContent,
		IsError:    result.IsError,
	}, nil
}

// Close ends every session and releases every child process. It is safe to call
// more than once.
func (m *Manager) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	sessions := m.clients
	m.clients = map[string]*sdk.ClientSession{}
	for _, state := range m.states {
		if state.Status == StatusConnected {
			state.Status = StatusDisabled
			state.Tools = nil
		}
	}
	m.mu.Unlock()

	// Closing a session shuts its server down: for a stdio server the protocol
	// library closes stdin, then signals, then kills, so no per-server
	// cancellation of our own is needed and none is kept — two attempts to
	// write a test that would fail without it both passed, and unfalsifiable
	// code that looks like a safety measure is worse than none.
	//
	// A process exit reported while closing is the expected outcome, not a
	// failure: shutting the server down is what this method is for, and a
	// server that had to be signalled still shut down. Only errors that mean
	// the cleanup itself went wrong are returned.
	var errs []error
	for name, session := range sessions {
		err := session.Close()
		if err == nil {
			continue
		}
		var exited *exec.ExitError
		if errors.As(err, &exited) {
			m.opts.Logger.Debug("mcp server exited on shutdown", "server", name, "reason", err)
			continue
		}
		errs = append(errs, fmt.Errorf("close %s: %w", name, err))
	}
	// endOfLife cancels the context every server process was started with. It
	// is the backstop that makes the lifetime contract true: no child of this
	// manager outlives it.
	m.endOfLife()
	return errors.Join(errs...)
}

func (m *Manager) setState(state *ServerState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[state.Name] = state
}

// toolsDigest hashes what a server is offering the model: names, descriptions
// and schemas. Descriptions are included deliberately — a server that keeps its
// tool names and rewrites their descriptions has changed what it is telling the
// model, which is precisely the attack the digest exists to reveal.
func toolsDigest(tools []Tool) string {
	if len(tools) == 0 {
		return ""
	}
	sum := sha256.New()
	for _, tool := range tools {
		fmt.Fprintf(sum, "%s\x00%s\x00%s\x00", tool.Name, tool.Description, string(tool.InputSchema))
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// marshalSchema renders the SDK's decoded input schema back to raw JSON. From a
// client the field holds whatever the server sent, decoded into interface
// values; the conversion layer works on the JSON so that nothing is inferred
// from Go's type mapping.
func marshalSchema(schema any) (json.RawMessage, error) {
	if schema == nil {
		return nil, nil
	}
	if raw, ok := schema.(json.RawMessage); ok {
		return raw, nil
	}
	data, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("input schema could not be read: %w", err)
	}
	return data, nil
}
