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

// errSuperseded reports that a connection attempt finished after something
// newer had already decided this server's fate.
var errSuperseded = errors.New("this connection attempt has been superseded")

// Status is where a configured server currently stands.
type Status string

const (
	// StatusInvalid means the configuration entry itself is unusable; the
	// server was never contacted. See Config.Problems.
	StatusInvalid Status = "invalid"
	// StatusDisabled means the user has switched the server off.
	StatusDisabled Status = "disabled"
	// StatusNeedsApproval means the server is configured and enabled, but the
	// exact command it would run has not been approved — either it has never
	// been approved, or it has changed since it was. Its tools are not offered
	// and it is never contacted.
	StatusNeedsApproval Status = "needs-approval"
	// StatusConnecting means a connection attempt is in flight.
	StatusConnecting Status = "connecting"
	// StatusConnected means the session is live and its tools are usable.
	StatusConnected Status = "connected"
	// StatusNeedsSignIn means the server answered that it wants an
	// authorization the user has not given. It is kept apart from
	// StatusFailed because the two ask for different things: a failure sends
	// the user looking for a network or configuration problem, where this one
	// is answered by signing in and by nothing else.
	StatusNeedsSignIn Status = "needs-sign-in"
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
	// Instructions is the server's own description of how to use it, returned
	// from initialize. The protocol intends it for the model's system prompt:
	// tool definitions say what each call does, and nothing else says what the
	// server is FOR or when to reach for it.
	//
	// It is third-party text bound for the system prompt, so it is sanitised
	// and capped exactly as a tool description is, and the caller frames it as
	// the server's own words rather than Ollama's.
	Instructions string
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

	// Approvals decides which servers may actually be run. A nil policy denies
	// everything: a caller that has not thought about approval gets a manager
	// that connects to nothing, rather than one that quietly runs whatever the
	// configuration names.
	Approvals ApprovalPolicy

	// Tokens is where OAuth tokens for remote servers are kept. A nil store
	// means no authorization support: a server that answers 401 fails to
	// connect rather than offering a sign-in.
	Tokens TokenStore

	// OpenBrowser launches the user's browser at an authorization URL. Nil
	// means the operating system's opener. It is a seam, like newTransport
	// below: the full sign-in flow is driven through it in tests, and a
	// headless caller can print the URL rather than launch anything.
	OpenBrowser func(string) error

	// newTransport is a seam for tests, which connect an in-process server over
	// the SDK's in-memory transport. Production leaves it nil and gets the real
	// subprocess and HTTP transports.
	newTransport func(context.Context, *ServerSpec, transportOptions) (sdk.Transport, func(), error)
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
	// releases holds what each live connection's transport is holding beyond
	// the session — today an OAuth redirect listener. Ending a session without
	// running its release leaves a loopback port bound for the life of the
	// process.
	releases map[string]func()
	// epochs supersede in-flight work. Connect returns as soon as its dials
	// finish, but a dial it started can still be running when the next Connect
	// arrives — and the second one cannot cancel the first. Without a marker
	// the older attempt writes its session and its state over the newer
	// decision, and a server the user has just switched off comes back
	// connected with its tools on offer.
	epochs map[string]uint64
	// signingIn holds the servers with a sign-in in flight. A second sign-in
	// for the same server would open a second browser window and a second
	// redirect listener, and whichever callback arrived second would be
	// answering a request that no longer exists.
	signingIn map[string]struct{}
	closed    bool
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
		releases:  map[string]func(){},
		epochs:    map[string]uint64{},
		signingIn: map[string]struct{}{},
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
	// A closed manager connects to nothing. Without this it would set states,
	// spawn goroutines that all fail inside dial, and leave a shut-down manager
	// reporting a list of failures to anyone who asked.
	m.mu.RLock()
	closed := m.closed
	m.mu.RUnlock()
	if closed {
		return
	}
	problems := cfg.Problems()

	// A server the configuration no longer mentions is not merely switched off
	// — it is gone, and reconciling only the servers cfg names would never
	// touch it. Its process would keep running, its state would stay whatever
	// it last was, and Tools and Schemas would keep offering it to the model.
	// Deleting a server has to actually delete it.
	m.forgetServersOutside(cfg.Names())

	type job struct {
		name string
		spec *ServerSpec
	}
	var jobs []job

	for _, name := range cfg.Names() {
		spec, _ := cfg.Get(name)
		switch {
		case problems[name] != nil:
			m.closeSession(name)
			m.setState(&ServerState{Name: name, Spec: spec, Status: StatusInvalid, Err: problems[name]})
		case spec.Disabled:
			m.closeSession(name)
			m.setState(&ServerState{Name: name, Spec: spec, Status: StatusDisabled})
		case !m.approves(spec):
			m.closeSession(name)
			m.setState(&ServerState{
				Name:   name,
				Spec:   spec,
				Status: StatusNeedsApproval,
				Err:    fmt.Errorf("%s has not been approved to run: %s", name, spec.Summary()),
			})
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
			m.connectOne(ctx, j.spec, signInDisallowed, m.beginAttempt(j.name))
		}()
	}
	wg.Wait()
}

// forgetServersOutside drops every server this manager is holding that the
// configuration no longer names: the session ends, the transport gives back
// what it held, and the state disappears rather than lingering as a status for
// something that is not configured.
func (m *Manager) forgetServersOutside(configured []string) {
	m.mu.RLock()
	var removed []string
	for name := range m.states {
		if !slices.Contains(configured, name) {
			removed = append(removed, name)
		}
	}
	for name := range m.clients {
		if !slices.Contains(configured, name) && !slices.Contains(removed, name) {
			removed = append(removed, name)
		}
	}
	m.mu.RUnlock()

	for _, name := range removed {
		// closeSession does the session and the release, outside the lock.
		m.closeSession(name)
		m.mu.Lock()
		delete(m.states, name)
		m.mu.Unlock()
	}
}

// beginAttempt marks the start of a connection attempt and returns the epoch it
// belongs to. Anything that supersedes the attempt moves the epoch on, and the
// attempt then declines to install its results.
func (m *Manager) beginAttempt(name string) uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.epochs[name]++
	return m.epochs[name]
}

// closeSession ends any live session for a server that should no longer be
// running. Connect is called again whenever the configuration changes, so a
// server the user has just switched off, invalidated or un-approved must lose
// its process here — dropping it from the tool list while leaving it running
// would be a switch that does not switch anything off.
func (m *Manager) closeSession(name string) {
	m.mu.Lock()
	// Ending a session supersedes any attempt still dialling for this server:
	// whatever the user just did outranks a decision taken before they did it.
	m.epochs[name]++
	session := m.clients[name]
	delete(m.clients, name)
	release := m.releases[name]
	delete(m.releases, name)
	m.mu.Unlock()

	if release != nil {
		release()
	}
	if session != nil {
		if err := session.Close(); err != nil {
			m.opts.Logger.Debug("mcp server closed on reconfigure", "server", name, "reason", err)
		}
	}
}

// approves reports whether the configured approval policy permits this exact
// spec. A manager with no policy approves nothing.
func (m *Manager) approves(spec *ServerSpec) bool {
	if m.opts.Approvals == nil {
		return false
	}
	return m.opts.Approvals.Allows(spec)
}

func (m *Manager) connectOne(ctx context.Context, spec *ServerSpec, mode signInMode, epoch uint64) {
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
		if err = m.dial(ctx, spec, mode, epoch); err == nil {
			return
		}
		if errors.Is(err, errSuperseded) {
			return
		}
		// A server asking to be signed in will ask again on every attempt.
		// Retrying turns one honest answer into three, and delays the state
		// the user is waiting to see by the whole backoff.
		if SignInRequired(err) {
			break
		}
	}

	if SignInRequired(err) {
		m.setStateIfCurrent(epoch, &ServerState{Name: spec.Name, Spec: spec, Status: StatusNeedsSignIn, Err: err})
		m.opts.Logger.Info("mcp server needs a sign-in", "server", spec.Name)
		return
	}
	m.setStateIfCurrent(epoch, &ServerState{Name: spec.Name, Spec: spec, Status: StatusFailed, Err: err})
	m.opts.Logger.Warn("mcp server unavailable", "server", spec.Name, "error", err)
}

func (m *Manager) dial(ctx context.Context, spec *ServerSpec, mode signInMode, epoch uint64) error {
	// A sign-in is bounded by how long a person takes in a browser, not by how
	// long a server takes to answer. Holding it to the connect timeout would
	// cancel the exchange while the user was still reading the consent screen.
	timeout := m.opts.ConnectTimeout
	if mode == signInAllowed && timeout < DefaultAuthorizationTimeout {
		timeout = DefaultAuthorizationTimeout
	}
	dialCtx, cancelDial := context.WithTimeout(ctx, timeout)
	defer cancelDial()

	// The process is tied to the manager's lifetime, not to dialCtx. Tying it
	// to the dial context would kill the server the instant the handshake
	// finished, because that timeout is cancelled on the way out of this
	// function — a defect this code had, and which
	// TestConnectTimeoutDoesNotKillTheServer exists to catch.
	transport, release, err := m.opts.newTransport(m.lifetime, spec, transportOptions{
		tokens: m.opts.Tokens,
		signIn: mode,
		open:   m.opts.OpenBrowser,
	})
	if err != nil {
		release()
		return err
	}
	// Every failure below this point must release the transport's resources.
	// Only the successful path hands ownership to the manager.
	handedOver := false
	defer func() {
		if !handedOver {
			release()
		}
	}()

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
		state.Instructions = sanitiseText(info.Instructions, maxInstructionRunes)
	}

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		session.Close()
		return errors.New("manager is closed")
	}
	// Something has happened to this server since the attempt began — it was
	// switched off, removed, or another connect started. That decision is newer
	// than this one, so the result is thrown away rather than installed.
	if m.epochs[spec.Name] != epoch {
		m.mu.Unlock()
		session.Close()
		return errSuperseded
	}
	// Whatever was here is taken out under the lock and shut down after it.
	// Closing a session waits for the server process to exit, so doing it here
	// would hold the write lock for as long as an unresponsive server takes to
	// die — and every Call, States and Close for every *other* server would
	// wait behind it. closeSession has always done this work outside the lock;
	// this path did not.
	previousSession := m.clients[spec.Name]
	previousRelease := m.releases[spec.Name]

	m.clients[spec.Name] = session
	m.releases[spec.Name] = release
	handedOver = true
	m.states[spec.Name] = state
	m.mu.Unlock()

	if previousSession != nil {
		previousSession.Close()
	}
	if previousRelease != nil {
		previousRelease()
	}
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
// ServerInstructions is one connected server's own account of how to use it.
type ServerInstructions struct {
	Server string
	Text   string
}

// Instructions returns what each connected server says about itself.
//
// Only connected servers are included: instructions for a server whose tools
// are not on offer would tell a model to use something it cannot call, which is
// worse than silence.
func (m *Manager) Instructions() []ServerInstructions {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var instructions []ServerInstructions
	for _, name := range slices.Sorted(maps.Keys(m.states)) {
		state := m.states[name]
		if state.Status != StatusConnected || strings.TrimSpace(state.Instructions) == "" {
			continue
		}
		instructions = append(instructions, ServerInstructions{Server: name, Text: state.Instructions})
	}
	return instructions
}

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

	// Copied under the lock, not read through the pointer afterwards. A
	// ServerState is replaced wholesale by setState but its fields are also
	// written in place by Close, so holding the pointer and reading Status and
	// Tools after releasing the lock is a data race — one the race detector
	// finds the moment a call and a shutdown overlap.
	m.mu.RLock()
	session := m.clients[server]
	state := m.states[server]
	limit := m.opts.ResultLimit
	timeout := m.opts.CallTimeout
	var status Status
	var stateErr error
	var offersTool bool
	if state != nil {
		status = state.Status
		stateErr = state.Err
		offersTool = slices.ContainsFunc(state.Tools, func(t Tool) bool { return t.Name == name })
	}
	m.mu.RUnlock()

	if state == nil {
		return CallResult{}, fmt.Errorf("no MCP server named %q is configured", server)
	}
	if session == nil || status != StatusConnected {
		if stateErr != nil {
			return CallResult{}, fmt.Errorf("MCP server %q is %s: %w", server, status, stateErr)
		}
		return CallResult{}, fmt.Errorf("MCP server %q is %s", server, status)
	}
	if !offersTool {
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
	releases := m.releases
	m.releases = map[string]func(){}
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
	for _, release := range releases {
		release()
	}

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

// TokenStore returns where this manager keeps its OAuth tokens, or nil when it
// has none. Surfaces read it to tell the user where a credential will end up
// before they create one.
func (m *Manager) TokenStore() TokenStore {
	return m.opts.Tokens
}

// beginSignIn claims the single in-flight sign-in slot for a server.
func (m *Manager) beginSignIn(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, busy := m.signingIn[name]; busy {
		return false
	}
	m.signingIn[name] = struct{}{}
	return true
}

func (m *Manager) endSignIn(name string) {
	m.mu.Lock()
	delete(m.signingIn, name)
	m.mu.Unlock()
}

// SigningIn reports whether a sign-in is in flight for a server.
func (m *Manager) SigningIn(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, busy := m.signingIn[name]
	return busy
}

func (m *Manager) setState(state *ServerState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[state.Name] = state
}

// setStateIfCurrent records a state only when the attempt that produced it has
// not been superseded, so a slow failure cannot overwrite a newer decision.
func (m *Manager) setStateIfCurrent(epoch uint64, state *ServerState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.epochs[state.Name] != epoch {
		return
	}
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
