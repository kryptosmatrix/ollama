//go:build darwin

// OQ-3 failure-number instrument for G1 (persisted instructions reaching the actual model request).
//
// This file is design evidence, not part of the default suite. It is compiled into package cmd by
// `go test -overlay` (see run.sh beside it), so it can drive the real terminal entry
// (launchInteractiveModel -> prepareAgentModel -> GenerateAgentTUI -> agentchat.Run) and the real
// desktop HTTP server (app/ui.Server.Handler) from one test binary, each in its own OS process.
//
// Declared simulations (TECHNE Method 04 T-07): the inference daemon at OLLAMA_HOST is a fake that
// records every request and answers with short scripted replies. Declared omissions from production
// assembly: the desktop's embedded daemon, MCP manager, updater and webview (app/cmd/app/app.go:208-287);
// the terminal launcher's heartbeat, menu, selection save and model resolution (cmd/cmd.go:2161-2260),
// which choose the model and do not touch instructions.
//
// Author: Letterlock (Claude Opus 5.5), continuation 14, 26 September 2026.

package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"

	"github.com/ollama/ollama/app/store"
	apptools "github.com/ollama/ollama/app/tools"
	"github.com/ollama/ollama/app/ui"
)

const (
	oq3HelperModeEnv = "OQ3_HELPER_MODE"
	oq3Model         = "oq3-model"
	// Copied from agent/compactor.go:31 (unexported there); used only to recognise the summariser's request.
	oq3CompactionSystemPrefix = "Summarize the archived part of an Ollama agent conversation."
	oq3ToolTrigger            = "[[OQ3-TOOLCALL]]"
	oq3FixtureFile            = "oq3-fixture.txt"
)

// ---------------------------------------------------------------------------------------------
// Helper processes. The parent test re-executes its own binary with OQ3_HELPER_MODE set.

func TestMain(m *testing.M) {
	if mode := os.Getenv(oq3HelperModeEnv); mode != "" {
		os.Exit(oq3RunHelper(mode))
	}
	os.Exit(m.Run())
}

func oq3RunHelper(mode string) int {
	switch mode {
	case "desktop":
		return oq3HelperDesktop()
	case "terminal":
		return oq3HelperTerminal()
	case "cli":
		return oq3HelperCLI()
	case "legacy":
		return oq3HelperLegacy()
	}
	fmt.Fprintf(os.Stderr, "oq3: unknown helper mode %q\n", mode)
	return 2
}

// oq3HelperDesktop serves the real desktop handler the way app/cmd/app/app.go:208-287 assembles it:
// the default store path under HOME, a tool registry, approvals, Dev false, a random token.
func oq3HelperDesktop() int {
	logPath := os.Getenv("OQ3_LOG_FILE")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		fmt.Fprintln(os.Stderr, "oq3 desktop: log:", err)
		return 2
	}
	defer logFile.Close()
	st := &store.Store{}
	srv := ui.Server{
		Token:        os.Getenv("OQ3_TOKEN"),
		Store:        st,
		ToolRegistry: apptools.NewRegistry(),
		Approvals:    apptools.NewApprovals(),
		Dev:          false,
		Logger:       slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, "oq3 desktop: listen:", err)
		return 2
	}
	hs := &http.Server{Handler: srv.Handler()}
	go func() { _ = hs.Serve(ln) }()
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	portFile := os.Getenv("OQ3_PORT_FILE")
	if err := os.WriteFile(portFile+".tmp", []byte(port), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "oq3 desktop: port file:", err)
		return 2
	}
	if err := os.Rename(portFile+".tmp", portFile); err != nil {
		fmt.Fprintln(os.Stderr, "oq3 desktop: port file:", err)
		return 2
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = hs.Shutdown(ctx)
	if err := st.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "oq3 desktop: store close:", err)
		return 1
	}
	return 0
}

// oq3HelperTerminal enters the agent chat exactly where the launcher's "run a model" action does
// (cmd/cmd.go:2242 calls deps.runModel = launchInteractiveModel), on the real root command.
func oq3HelperTerminal() int {
	root := NewCLI()
	root.SetContext(context.Background())
	if err := launchInteractiveModel(root, os.Getenv("OQ3_MODEL")); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}
	return 0
}

// oq3HelperCLI runs the root command as main.go does, with the arguments in OQ3_ARGS (JSON array).
func oq3HelperCLI() int {
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("OQ3_ARGS")), &args); err != nil {
		fmt.Fprintln(os.Stderr, "oq3 cli: args:", err)
		return 2
	}
	root := NewCLI()
	root.SetArgs(args)
	if err := root.ExecuteContext(context.Background()); err != nil {
		// cobra.CheckErr, as main.go uses it: print and exit 1.
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}
	return 0
}

var _ = cobra.CheckErr

// oq3HelperLegacy writes a saved desktop conversation through the desktop store's own API before
// any desktop lifetime starts: the state an existing installation has before G1 is installed (a
// transcript with no instruction binding). Declared precondition, identical before and after G1.
func oq3HelperLegacy() int {
	st := &store.Store{}
	chat := store.NewChat(os.Getenv("OQ3_CHAT_ID"))
	chat.Messages = append(chat.Messages,
		store.NewMessage("user", "OQ3 legacy question asked before instructions existed.", nil),
		store.NewMessage("assistant", "OQ3 legacy answer.", &store.MessageOptions{Model: oq3Model}))
	if err := st.SetChat(*chat); err != nil {
		fmt.Fprintln(os.Stderr, "oq3 legacy: set chat:", err)
		return 1
	}
	if err := st.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "oq3 legacy: close:", err)
		return 1
	}
	return 0
}

// ---------------------------------------------------------------------------------------------
// Pseudo-terminal (darwin). /dev/ptmx, grant, unlock, name.

func oq3OpenPTY() (master *os.File, slave *os.File, err error) {
	fd, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("open ptmx: %w", err)
	}
	if err := unix.IoctlSetInt(fd, unix.TIOCPTYGRANT, 0); err != nil {
		unix.Close(fd)
		return nil, nil, fmt.Errorf("grantpt: %w", err)
	}
	if err := unix.IoctlSetInt(fd, unix.TIOCPTYUNLK, 0); err != nil {
		unix.Close(fd)
		return nil, nil, fmt.Errorf("unlockpt: %w", err)
	}
	buf := make([]byte, 128)
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(unix.TIOCPTYGNAME), uintptr(unsafe.Pointer(&buf[0]))); errno != 0 {
		unix.Close(fd)
		return nil, nil, fmt.Errorf("ptsname: %w", errno)
	}
	name := string(buf[:bytes.IndexByte(buf, 0)])
	sfd, err := unix.Open(name, unix.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		unix.Close(fd)
		return nil, nil, fmt.Errorf("open %s: %w", name, err)
	}
	if err := unix.IoctlSetWinsize(sfd, unix.TIOCSWINSZ, &unix.Winsize{Row: 50, Col: 160}); err != nil {
		unix.Close(fd)
		unix.Close(sfd)
		return nil, nil, fmt.Errorf("winsize: %w", err)
	}
	return os.NewFile(uintptr(fd), "/dev/ptmx"), os.NewFile(uintptr(sfd), name), nil
}

// ---------------------------------------------------------------------------------------------
// Fake daemon: records every request; answers the endpoints the two entry points call.

type oq3Capture struct {
	Seq      int             `json:"seq"`
	At       string          `json:"at"`
	Method   string          `json:"method"`
	Path     string          `json:"path"`
	Label    string          `json:"label"`
	Body     json.RawMessage `json:"body,omitempty"`
	RawBody  string          `json:"raw_body,omitempty"`
	Response string          `json:"response_kind"`
}

type oq3Daemon struct {
	t   *testing.T
	srv *httptest.Server
	// modelSystems maps each served model to its model-system text (the Modelfile SYSTEM).
	modelSystems map[string]string

	mu       sync.Mutex
	label    string
	seq      int
	captures []oq3Capture
	replySeq int
	// Per-label behaviour, set by the driver before the operation it labels.
	holds      map[string]chan struct{} // the /api/chat response waits until the channel is closed
	fails      map[string]int           // the /api/chat request is answered with this HTTP status
	promptEval map[string]int           // prompt_eval_count reported in the final record
	toolRounds map[string]int           // number of consecutive tool calls to emit for the label
	toolsDone  map[string]int
}

const (
	oq3ModelSystem  = "OQ3 model-system text for oq3-model: SENTINEL-MS1-7f3a."
	oq3Model2       = "oq3-model-2"
	oq3Model2System = "OQ3 model-system text for oq3-model-2: SENTINEL-MS2-c41e."
)

func oq3StartDaemon(t *testing.T) *oq3Daemon {
	d := &oq3Daemon{t: t, modelSystems: map[string]string{oq3Model: oq3ModelSystem, oq3Model2: oq3Model2System},
		holds: map[string]chan struct{}{}, fails: map[string]int{}, promptEval: map[string]int{}, toolRounds: map[string]int{}, toolsDone: map[string]int{}}
	d.srv = httptest.NewServer(http.HandlerFunc(d.serve))
	t.Cleanup(func() {
		d.mu.Lock()
		for _, ch := range d.holds {
			select {
			case <-ch:
			default:
				close(ch)
			}
		}
		d.mu.Unlock()
		d.srv.Close()
	})
	return d
}

func (d *oq3Daemon) hold(label string) {
	d.mu.Lock()
	d.holds[label] = make(chan struct{})
	d.mu.Unlock()
}

func (d *oq3Daemon) release(label string) {
	d.mu.Lock()
	ch := d.holds[label]
	d.mu.Unlock()
	if ch != nil {
		select {
		case <-ch:
		default:
			close(ch)
		}
	}
}

func (d *oq3Daemon) failNext(label string, status int) {
	d.mu.Lock()
	d.fails[label] = status
	d.mu.Unlock()
}

func (d *oq3Daemon) reportPromptEval(label string, n int) {
	d.mu.Lock()
	d.promptEval[label] = n
	d.mu.Unlock()
}

func (d *oq3Daemon) setToolRounds(label string, n int) {
	d.mu.Lock()
	d.toolRounds[label] = n
	d.mu.Unlock()
}

func oq3ModelBase(name string) string {
	name = strings.TrimSuffix(name, ":latest")
	return name
}

func (d *oq3Daemon) setLabel(label string) {
	d.mu.Lock()
	d.label = label
	d.mu.Unlock()
}

func (d *oq3Daemon) snapshot() []oq3Capture {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]oq3Capture(nil), d.captures...)
}

func (d *oq3Daemon) count(method, path string) int {
	n := 0
	for _, c := range d.snapshot() {
		if c.Method == method && c.Path == path {
			n++
		}
	}
	return n
}

// waitFor polls until cond is true or the deadline passes; it never sleeps a fixed interval blindly.
func oq3WaitFor(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cond()
}

func (d *oq3Daemon) record(r *http.Request, body []byte, kind string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seq++
	c := oq3Capture{Seq: d.seq, At: time.Now().Format(time.RFC3339Nano), Method: r.Method, Path: r.URL.Path, Label: d.label, Response: kind}
	if len(body) > 0 {
		if json.Valid(body) {
			c.Body = append(json.RawMessage(nil), body...)
		} else {
			c.RawBody = string(body)
		}
	}
	d.captures = append(d.captures, c)
}

func (d *oq3Daemon) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	switch {
	case r.URL.Path == "/" || r.URL.Path == "/api/version":
		// Advertises the G1 admission capability (blueprint §6.3 A12) so that a G1 client is expected
		// to send carrier-bearing requests; the unchanged candidate ignores the field.
		d.record(r, body, "version")
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"version":"0.0.0-oq3-fake","capabilities":["chat.admission.v1"]}`)
	case r.URL.Path == "/api/show":
		d.record(r, body, "show")
		var req struct {
			Model string `json:"model"`
			Name  string `json:"name"`
		}
		_ = json.Unmarshal(body, &req)
		name := req.Model
		if name == "" {
			name = req.Name
		}
		system, ok := d.modelSystems[oq3ModelBase(name)]
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `{"error":"model not found"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"system":       system,
			"capabilities": []string{"completion", "tools"},
			"details":      map[string]any{"format": "gguf", "family": "oq3", "parameter_size": "1B", "quantization_level": "Q4_0"},
			"model_info":   map[string]any{"general.architecture": "oq3", "general.context_length": 32768},
			"modified_at":  "2026-09-26T00:00:00Z",
		})
	case r.URL.Path == "/api/tags":
		d.record(r, body, "tags")
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"models":[`+
			`{"name":"oq3-model:latest","model":"oq3-model:latest","modified_at":"2026-09-26T00:00:00Z","size":1,"digest":"0000000000000000000000000000000000000000000000000000000000000001","details":{"format":"gguf","family":"oq3"}},`+
			`{"name":"oq3-model-2:latest","model":"oq3-model-2:latest","modified_at":"2026-09-26T00:00:00Z","size":1,"digest":"0000000000000000000000000000000000000000000000000000000000000002","details":{"format":"gguf","family":"oq3"}}]}`)
	case r.URL.Path == "/api/ps":
		d.record(r, body, "ps")
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"models":[{"name":"oq3-model:latest","model":"oq3-model:latest","size":1,"digest":"0000000000000000000000000000000000000000000000000000000000000000","context_length":32768}]}`)
	case r.URL.Path == "/api/generate":
		d.record(r, body, "generate-preload")
		w.Header().Set("Content-Type", "application/x-ndjson")
		io.WriteString(w, `{"model":"oq3-model","created_at":"2026-09-26T00:00:00Z","response":"","done":true,"done_reason":"load"}`+"\n")
	case r.URL.Path == "/api/chat":
		d.serveChat(w, r, body)
	default:
		d.record(r, body, "not-found")
		http.Error(w, `{"error":"oq3 fake daemon: unsupported endpoint"}`, http.StatusNotFound)
	}
}

func (d *oq3Daemon) nextReply() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.replySeq++
	return fmt.Sprintf("OQ3-REPLY-%03d", d.replySeq)
}

func (d *oq3Daemon) serveChat(w http.ResponseWriter, r *http.Request, body []byte) {
	var req struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Admission json.RawMessage `json:"admission"`
	}
	_ = json.Unmarshal(body, &req)
	d.mu.Lock()
	label := d.label
	hold := d.holds[label]
	failStatus := d.fails[label]
	promptEval := d.promptEval[label]
	rounds, done := d.toolRounds[label], d.toolsDone[label]
	d.mu.Unlock()

	// The response is chosen from the request's shape and the label's declared behaviour, never
	// from whether the request carries instructions.
	kind := "reply"
	n := len(req.Messages)
	switch {
	case n > 0 && req.Messages[0].Role == "system" && strings.HasPrefix(req.Messages[0].Content, oq3CompactionSystemPrefix):
		kind = "summary"
	case failStatus != 0:
		kind = "injected-failure"
	case n > 0 && req.Messages[n-1].Role == "user" && strings.Contains(req.Messages[n-1].Content, oq3ToolTrigger) && rounds > 0 && done == 0:
		kind = "toolcall"
	case n > 0 && req.Messages[n-1].Role == "tool" && done > 0 && done < rounds:
		kind = "toolcall"
	}
	d.record(r, body, kind)
	if kind == "toolcall" {
		d.mu.Lock()
		d.toolsDone[label]++
		call := d.toolsDone[label]
		d.mu.Unlock()
		kind = fmt.Sprintf("toolcall-%d", call)
	}
	if hold != nil {
		select {
		case <-hold:
		case <-r.Context().Done():
			return
		}
	}
	if failStatus != 0 {
		d.mu.Lock()
		delete(d.fails, label)
		d.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(failStatus)
		io.WriteString(w, `{"error":"oq3 fake daemon: injected failure"}`)
		return
	}
	model := req.Model
	if model == "" {
		model = oq3Model
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	enc := json.NewEncoder(w)
	now := "2026-09-26T00:00:00Z"
	switch {
	case kind == "summary":
		enc.Encode(map[string]any{"model": model, "created_at": now, "message": map[string]any{"role": "assistant", "content": "OQ3-SUMMARY: earlier turns were exchanged."}, "done": false})
	case strings.HasPrefix(kind, "toolcall-"):
		enc.Encode(map[string]any{"model": model, "created_at": now, "message": map[string]any{"role": "assistant", "content": "", "tool_calls": []any{map[string]any{"id": "call_oq3_" + strings.TrimPrefix(kind, "toolcall-"), "function": map[string]any{"name": "read", "arguments": map[string]any{"path": oq3FixtureFile}}}}}, "done": false})
	default:
		enc.Encode(map[string]any{"model": model, "created_at": now, "message": map[string]any{"role": "assistant", "content": d.nextReply()}, "done": false})
	}
	final := map[string]any{"model": model, "created_at": now, "message": map[string]any{"role": "assistant", "content": ""}, "done": true, "done_reason": "stop", "prompt_eval_count": 10, "eval_count": 3}
	if promptEval > 0 {
		final["prompt_eval_count"] = promptEval
	}
	if len(req.Admission) > 0 && string(req.Admission) != "null" {
		// Blueprint §6.3 A9: the final record of an admitted request carries its diagnostics.
		final["admission"] = map[string]any{"version": 1, "route": "local", "context_length": 32768, "reserve": 4096, "dropped_messages": 0, "context_exhausted": false}
	}
	enc.Encode(final)
}

// ---------------------------------------------------------------------------------------------
// Sandbox: one fresh HOME per scenario; nothing of the operator's is read or written.

type oq3Sandbox struct {
	root, home, work, tmp, logs, xdg string
}

func oq3NewSandbox(t *testing.T, name string) *oq3Sandbox {
	root := filepath.Join(t.TempDir(), name)
	s := &oq3Sandbox{root: root, home: filepath.Join(root, "home"), work: filepath.Join(root, "work"), tmp: filepath.Join(root, "tmp"), logs: filepath.Join(root, "logs"), xdg: filepath.Join(root, "xdg")}
	for _, p := range []string{s.home, s.work, s.tmp, s.logs, s.xdg} {
		if err := os.MkdirAll(p, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(s.work, oq3FixtureFile), []byte("OQ3 fixture file content.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return s
}

func (s *oq3Sandbox) env(daemonURL string, extra ...string) []string {
	env := []string{
		"HOME=" + s.home,
		"USERPROFILE=" + s.home,
		"TMPDIR=" + s.tmp,
		"PATH=" + os.Getenv("PATH"),
		"TERM=xterm-256color",
		"LANG=en_AU.UTF-8",
		"TZ=Australia/Brisbane",
		"XDG_CONFIG_HOME=" + s.xdg,
		"OLLAMA_HOST=" + daemonURL,
		// Network guard: every non-loopback HTTP(S) request goes to a closed local port and fails at
		// once, so the instrument cannot reach an external service; loopback (the fake daemon) is
		// never proxied by Go's ProxyFromEnvironment.
		"HTTP_PROXY=http://127.0.0.1:9",
		"HTTPS_PROXY=http://127.0.0.1:9",
		"NO_PROXY=127.0.0.1,localhost",
	}
	for _, k := range []string{"USER", "LOGNAME"} {
		if v := os.Getenv(k); v != "" {
			env = append(env, k+"="+v)
		}
	}
	return append(env, extra...)
}

// ---------------------------------------------------------------------------------------------
// Desktop driver: a real OS process per desktop lifetime, driven over authenticated HTTP.

// oq3Fail reports a harness failure: t.Fatalf in the smoke test, a recovered panic in the runner.
type oq3Fail func(format string, a ...any)

func oq3Panic(format string, a ...any) { panic(fmt.Sprintf(format, a...)) }

// oq3HelperCmd prepares a re-executed helper process in the sandbox with a constructed environment.
func oq3HelperCmd(s *oq3Sandbox, d *oq3Daemon, mode string, extraEnv ...string) *exec.Cmd {
	c := exec.Command(os.Args[0], "-test.run=^$")
	c.Dir = s.work
	c.Env = s.env(d.srv.URL, append([]string{oq3HelperModeEnv + "=" + mode}, extraEnv...)...)
	return c
}

type oq3Desktop struct {
	fail   oq3Fail
	cmd    *exec.Cmd
	base   string
	token  string
	stderr *bytes.Buffer
	exited chan error
}

func oq3StartDesktop(t *testing.T, s *oq3Sandbox, d *oq3Daemon, life int) *oq3Desktop {
	return oq3StartDesktopWith(t.Fatalf, s, d, life)
}

func oq3StartDesktopNoFatal(r *oq3Run, life int) *oq3Desktop {
	return oq3StartDesktopWith(oq3Panic, r.s, r.d, life)
}

func oq3StartDesktopWith(fail oq3Fail, s *oq3Sandbox, d *oq3Daemon, life int) *oq3Desktop {
	token := fmt.Sprintf("oq3-token-%d-%d", time.Now().UnixNano(), life)
	portFile := filepath.Join(s.tmp, fmt.Sprintf("desktop-%d.port", life))
	c := oq3HelperCmd(s, d, "desktop", "OQ3_TOKEN="+token, "OQ3_PORT_FILE="+portFile,
		"OQ3_LOG_FILE="+filepath.Join(s.logs, fmt.Sprintf("desktop-%d.log", life)))
	var stderr bytes.Buffer
	c.Stderr = &stderr
	c.Stdout = &stderr
	if err := c.Start(); err != nil {
		fail("start desktop: %v", err)
	}
	p := &oq3Desktop{fail: fail, cmd: c, token: token, stderr: &stderr, exited: make(chan error, 1)}
	go func() { p.exited <- c.Wait() }()
	var port []byte
	if !oq3WaitFor(20*time.Second, func() bool {
		b, err := os.ReadFile(portFile)
		if err == nil && len(b) > 0 {
			port = b
			return true
		}
		return false
	}) {
		_ = c.Process.Kill()
		fail("desktop process %d never published its port; stderr=%s", life, stderr.String())
	}
	p.base = "http://127.0.0.1:" + string(port)
	return p
}

// stop ends the desktop process the way the operator quitting does (SIGTERM) and waits for it, so
// that a later lifetime can only have read what the earlier one persisted.
func (p *oq3Desktop) stop() {
	if err := p.stopNoFatal(); err != nil {
		p.fail("%v", err)
	}
}

func (p *oq3Desktop) stopNoFatal() error {
	_ = p.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case err := <-p.exited:
		if err != nil {
			return fmt.Errorf("desktop process exit: %v; output=%s", err, p.stderr.String())
		}
		return nil
	case <-time.After(15 * time.Second):
		_ = p.cmd.Process.Kill()
		return fmt.Errorf("desktop process did not exit on SIGTERM")
	}
}

type oq3HTTPResult struct {
	Status      int    `json:"status"`
	ContentType string `json:"content_type"`
	Body        string `json:"body"`
	Error       string `json:"transport_error,omitempty"`
}

func (p *oq3Desktop) do(method, path string, body any) oq3HTTPResult {
	return p.doAuth(method, path, body, true)
}

// doAuth never fails the test: a transport error is returned in the result (status -1), so it is
// safe from any goroutine; callers decide whether it is a harness failure.
func (p *oq3Desktop) doAuth(method, path string, body any, auth bool) oq3HTTPResult {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return oq3HTTPResult{Status: -1, Error: err.Error()}
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, p.base+path, rd)
	if err != nil {
		return oq3HTTPResult{Status: -1, Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	if auth {
		req.AddCookie(&http.Cookie{Name: "token", Value: p.token})
	}
	resp, err := (&http.Client{Timeout: 90 * time.Second}).Do(req)
	if err != nil {
		return oq3HTTPResult{Status: -1, Error: err.Error()}
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return oq3HTTPResult{Status: resp.StatusCode, ContentType: resp.Header.Get("Content-Type"), Body: string(data)}
}

// chat sends one plain turn through POST /api/v1/chat/{id}.
func (p *oq3Desktop) chat(id, prompt string) (string, oq3HTTPResult) {
	return p.chatBody(id, map[string]any{"model": oq3Model, "prompt": prompt})
}

// chatBody sends one turn and returns the chat id the stream reports.
func (p *oq3Desktop) chatBody(id string, body map[string]any) (string, oq3HTTPResult) {
	res := p.do(http.MethodPost, "/api/v1/chat/"+id, body)
	chatID := id
	sc := bufio.NewScanner(strings.NewReader(res.Body))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var ev struct {
			EventName string  `json:"eventName"`
			ChatID    *string `json:"chatId"`
		}
		if json.Unmarshal(sc.Bytes(), &ev) == nil && ev.ChatID != nil && *ev.ChatID != "" {
			chatID = *ev.ChatID
		}
	}
	return chatID, res
}

// ---------------------------------------------------------------------------------------------
// Terminal driver: a real OS process per terminal session on a real pseudo-terminal.

type oq3Terminal struct {
	fail   oq3Fail
	cmd    *exec.Cmd
	master *os.File
	mu     sync.Mutex
	gone   bool
	raw    bytes.Buffer
	exited chan error
	done   bool
}

var oq3ANSI = regexp.MustCompile(`\x1b\[[0-9;?:<>=]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(\x07|\x1b\\)|\x1b[()][0-9A-Za-z]|\x1b[=>78DEHMNOPZc]`)

func oq3StartTerminal(t *testing.T, s *oq3Sandbox, d *oq3Daemon, session string) *oq3Terminal {
	return oq3StartTerminalWith(t.Fatalf, s, d, session, oq3Model, nil)
}

func oq3StartTerminalNoFatal(r *oq3Run, session, model string) *oq3Terminal {
	if model == "" {
		model = oq3Model
	}
	return oq3StartTerminalWith(oq3Panic, r.s, r.d, session, model, nil)
}

// oq3StartTerminalWith starts either the launch-function helper (argv nil) or a given program
// (the built binary for the entry arm) on a fresh pseudo-terminal.
func oq3StartTerminalWith(fail oq3Fail, s *oq3Sandbox, d *oq3Daemon, session, model string, argv []string) *oq3Terminal {
	master, slave, err := oq3OpenPTY()
	if err != nil {
		fail("pty: %v", err)
	}
	var c *exec.Cmd
	if argv == nil {
		c = oq3HelperCmd(s, d, "terminal", "OQ3_MODEL="+model)
	} else {
		c = exec.Command(argv[0], argv[1:]...)
		c.Dir = s.work
		c.Env = s.env(d.srv.URL)
	}
	c.Stdin, c.Stdout, c.Stderr = slave, slave, slave
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	if err := c.Start(); err != nil {
		fail("start terminal: %v", err)
	}
	slave.Close()
	p := &oq3Terminal{fail: fail, cmd: c, master: master, exited: make(chan error, 1)}
	go p.readLoop(filepath.Join(s.logs, "terminal-"+session+".raw"))
	go func() {
		err := c.Wait()
		p.mu.Lock()
		p.gone = true
		p.mu.Unlock()
		p.exited <- err
	}()
	return p
}

// hasExited reports, without consuming the exit status, whether the process has ended.
func (p *oq3Terminal) hasExited() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.gone
}

func (p *oq3Terminal) rawSince(mark int) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	raw := p.raw.String()
	if mark > len(raw) {
		mark = len(raw)
	}
	return raw[mark:]
}

func (d *oq3Daemon) countLabel(method, path, label string) int {
	n := 0
	for _, c := range d.snapshot() {
		if c.Method == method && c.Path == path && c.Label == label {
			n++
		}
	}
	return n
}

func (d *oq3Daemon) countSummaries(label string) int {
	n := 0
	for _, c := range d.snapshot() {
		if c.Path == "/api/chat" && c.Label == label && c.Response == "summary" {
			n++
		}
	}
	return n
}

// readLoop keeps the transcript and answers the terminal queries a real emulator would answer
// (background colour, foreground colour, cursor position, device attributes).
func (p *oq3Terminal) readLoop(rawPath string) {
	f, _ := os.Create(rawPath)
	defer func() {
		if f != nil {
			f.Close()
		}
	}()
	buf := make([]byte, 64*1024)
	for {
		n, err := p.master.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if f != nil {
				f.Write(chunk)
			}
			p.mu.Lock()
			p.raw.Write(chunk)
			p.mu.Unlock()
			s := string(chunk)
			if strings.Contains(s, "\x1b]11;?") {
				p.master.Write([]byte("\x1b]11;rgb:1c1c/1c1c/1c1c\x1b\\"))
			}
			if strings.Contains(s, "\x1b]10;?") {
				p.master.Write([]byte("\x1b]10;rgb:e0e0/e0e0/e0e0\x1b\\"))
			}
			if strings.Contains(s, "\x1b[6n") {
				p.master.Write([]byte("\x1b[1;1R"))
			}
			if strings.Contains(s, "\x1b[c") || strings.Contains(s, "\x1b[0c") {
				p.master.Write([]byte("\x1b[?62;22c"))
			}
		}
		if err != nil {
			return
		}
	}
}

func (p *oq3Terminal) screen() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return oq3ANSI.ReplaceAllString(p.raw.String(), "")
}

func (p *oq3Terminal) rawLen() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.raw.Len()
}

// quiet waits until the program has produced no output for the given interval.
func (p *oq3Terminal) quiet(interval, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	last := p.rawLen()
	lastChange := time.Now()
	for time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
		if n := p.rawLen(); n != last {
			last, lastChange = n, time.Now()
			continue
		}
		if time.Since(lastChange) >= interval {
			return true
		}
	}
	return false
}

func (p *oq3Terminal) screenContainsAfter(mark int, text string) bool {
	p.mu.Lock()
	raw := p.raw.String()
	p.mu.Unlock()
	if mark > len(raw) {
		mark = len(raw)
	}
	return strings.Contains(oq3ANSI.ReplaceAllString(raw[mark:], ""), text)
}

func (p *oq3Terminal) typeLine(text string) {
	if _, err := p.master.Write([]byte(text)); err != nil {
		p.fail("pty write: %v", err)
	}
	time.Sleep(60 * time.Millisecond)
	if _, err := p.master.Write([]byte("\r")); err != nil {
		p.fail("pty write: %v", err)
	}
}

func (p *oq3Terminal) key(s string) {
	if _, err := p.master.Write([]byte(s)); err != nil {
		p.fail("pty write: %v", err)
	}
}

func (p *oq3Terminal) waitExit(timeout time.Duration) (error, bool) {
	select {
	case err := <-p.exited:
		p.done = true
		return err, true
	case <-time.After(timeout):
		return nil, false
	}
}

func (p *oq3Terminal) kill() {
	if !p.done {
		_ = p.cmd.Process.Kill()
		<-p.exited
		p.done = true
	}
	p.master.Close()
}

// ---------------------------------------------------------------------------------------------
// CLI driver: one OS process per command, as the operator runs it.

type oq3CLIResult struct {
	Args     []string `json:"args"`
	ExitCode int      `json:"exit_code"`
	Output   string   `json:"output"`
}

func oq3RunCLI(t *testing.T, s *oq3Sandbox, d *oq3Daemon, args ...string) oq3CLIResult {
	return oq3RunCLIWith(t.Fatalf, s, d, nil, args...)
}

func oq3RunCLINoFatal(r *oq3Run, args ...string) oq3CLIResult {
	return oq3RunCLIWith(oq3Panic, r.s, r.d, r.builtBinary, args...)
}

// oq3RunCLIWith runs one CLI invocation: the root command in a re-executed helper process, or the
// built binary when one is given (the entry arm).
func oq3RunCLIWith(fail oq3Fail, s *oq3Sandbox, d *oq3Daemon, binary []string, args ...string) oq3CLIResult {
	var c *exec.Cmd
	if binary == nil {
		a, _ := json.Marshal(args)
		c = oq3HelperCmd(s, d, "cli", "OQ3_ARGS="+string(a))
	} else {
		c = exec.Command(binary[0], append(append([]string{}, binary[1:]...), args...)...)
		c.Dir = s.work
		c.Env = s.env(d.srv.URL)
	}
	out, err := c.CombinedOutput()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			fail("cli %v: %v", args, err)
		}
		code = ee.ExitCode()
	}
	return oq3CLIResult{Args: args, ExitCode: code, Output: string(out)}
}
