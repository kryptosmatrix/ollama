package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ollama/ollama/version"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Discovery finds MCP servers that already exist on this machine, so a user
// does not have to copy configuration between applications by hand.
//
// There are two quite different things it can mean by "a server on this
// machine", and they are kept apart rather than merged into one list:
//
//   - Configured: something another MCP client has been told about. It may
//     never have been run. Reading these files is how every other client
//     offers to import, and it is a read: nothing here writes to them.
//   - Answering: something listening on the loopback interface right now that
//     replies to an MCP handshake. It may be configured nowhere at all.
//
// A server found either way is only ever offered. Nothing is added, enabled or
// approved by discovery.

// DiscoveredServer is one MCP server found on this machine.
type DiscoveredServer struct {
	// Name is a suggested name, valid for Ollama's configuration. It may
	// differ from the name the other application used; Notes says so when it
	// does.
	Name string
	// Spec is what would be added, after sanitising. It is nil when Problem
	// is set.
	Spec *ServerSpec
	// Sources name where this came from, in words a user can act on: the
	// applications configured with it, or the process and port that answered.
	// It is a list because one server is commonly registered in several
	// applications, and that is one server, not several.
	Sources []string
	// Paths are the files it was read from, in the same order as Sources.
	// Empty for something that answered on the network.
	Paths []string
	// Runs is the command line or URL, for the user to read before agreeing.
	Runs string
	// Origin is "config" or "listening".
	Origin string
	// Notes record what discovery changed or noticed. A credential rewritten
	// into an environment reference is a note, not a silent edit.
	Notes []string
	// Problem is why this entry cannot be added as it stands. It is carried
	// rather than dropped: an entry that vanishes without explanation reads as
	// a discovery that missed it.
	Problem string
}

// OriginConfig and OriginListening are the two kinds of discovery.
const (
	OriginConfig    = "config"
	OriginListening = "listening"
)

// configSource is one place another MCP client keeps its servers.
type configSource struct {
	// Label is the application's name, as a person would say it.
	Label string
	// Path is the file to read.
	Path string
	// Keys are the top-level keys that may hold a map of servers. Different
	// clients chose different names for the same object.
	Keys []string
}

// configSources lists the files discovery reads, in a fixed order.
//
// The list is deliberately explicit rather than a search of the home
// directory: reading a named file that an MCP client documents is a different
// act from trawling someone's disk for anything that looks like configuration.
// Every path is reported back to the caller whether or not it existed, so the
// coverage of this list is visible rather than implied.
func configSources(home string) []configSource {
	join := func(parts ...string) string {
		return filepath.Join(append([]string{home}, parts...)...)
	}

	// The keys in use: "mcpServers" is the shape Claude Desktop introduced and
	// most clients copied; VS Code calls the same object "servers".
	const (
		claudeKey = "mcpServers"
		codeKey   = "servers"
	)

	var sources []configSource
	switch runtime.GOOS {
	case "darwin":
		sources = []configSource{
			{"Claude Desktop", join("Library", "Application Support", "Claude", "claude_desktop_config.json"), []string{claudeKey}},
			{"VS Code", join("Library", "Application Support", "Code", "User", "mcp.json"), []string{codeKey, claudeKey}},
		}
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = join("AppData", "Roaming")
		}
		sources = []configSource{
			{"Claude Desktop", filepath.Join(appData, "Claude", "claude_desktop_config.json"), []string{claudeKey}},
			{"VS Code", filepath.Join(appData, "Code", "User", "mcp.json"), []string{codeKey, claudeKey}},
		}
	default:
		sources = []configSource{
			{"Claude Desktop", join(".config", "Claude", "claude_desktop_config.json"), []string{claudeKey}},
			{"VS Code", join(".config", "Code", "User", "mcp.json"), []string{codeKey, claudeKey}},
		}
	}

	// Same paths on every platform.
	return append(sources,
		configSource{"Claude Code", join(".claude.json"), []string{claudeKey}},
		configSource{"Cursor", join(".cursor", "mcp.json"), []string{claudeKey, codeKey}},
		configSource{"Windsurf", join(".codeium", "windsurf", "mcp_config.json"), []string{claudeKey, codeKey}},
	)
}

// DiscoverConfigured reads the MCP servers other applications on this machine
// have been configured with.
//
// The second return value is every path that was looked at, existing or not,
// so a caller can say what was searched rather than leaving the user to guess
// why their server did not appear.
func DiscoverConfigured(home string) ([]DiscoveredServer, []string) {
	if strings.TrimSpace(home) == "" {
		if resolved, err := os.UserHomeDir(); err == nil {
			home = resolved
		}
	}

	var found []DiscoveredServer
	var searched []string
	for _, source := range configSources(home) {
		searched = append(searched, source.Path)
		found = append(found, readConfigSource(source)...)
	}

	found = mergeSameServer(found)
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].Name != found[j].Name {
			return found[i].Name < found[j].Name
		}
		return firstOf(found[i].Sources) < firstOf(found[j].Sources)
	})
	return found, searched
}

// mergeSameServer collapses one server registered in several applications into
// one entry naming all of them.
//
// Registering the same server in Claude Code and in Cursor is the ordinary
// thing to do, and it is still one server: listing it twice invites a user to
// add it twice, where the second add collides with the first on its name. Two
// entries are the same server when the specification Ollama would write is
// identical, which is what Fingerprint answers — so two registrations that
// differ in arguments or environment stay apart, because those really are
// different servers.
//
// An entry that could not be read has no specification to compare, so it is
// never merged with anything.
func mergeSameServer(found []DiscoveredServer) []DiscoveredServer {
	merged := make([]DiscoveredServer, 0, len(found))
	at := map[string]int{}

	for _, server := range found {
		if server.Spec == nil {
			merged = append(merged, server)
			continue
		}
		key := server.Name + "\x00" + server.Spec.Fingerprint()
		if index, seen := at[key]; seen {
			merged[index].Sources = append(merged[index].Sources, server.Sources...)
			merged[index].Paths = append(merged[index].Paths, server.Paths...)
			for _, note := range server.Notes {
				if !slices.Contains(merged[index].Notes, note) {
					merged[index].Notes = append(merged[index].Notes, note)
				}
			}
			continue
		}
		at[key] = len(merged)
		merged = append(merged, server)
	}
	return merged
}

func firstOf(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func readConfigSource(source configSource) []DiscoveredServer {
	data, err := os.ReadFile(source.Path)
	if err != nil {
		// A missing file is the ordinary case: most people do not have every
		// one of these applications. An unreadable one is reported, because
		// silently skipping it would look identical to it being empty.
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return []DiscoveredServer{{
			Sources: []string{source.Label},
			Paths:   []string{source.Path},
			Origin:  OriginConfig,
			Problem: fmt.Sprintf("could not be read: %v", err),
		}}
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return []DiscoveredServer{{
			Sources: []string{source.Label},
			Paths:   []string{source.Path},
			Origin:  OriginConfig,
			Problem: fmt.Sprintf("is not valid JSON: %v", err),
		}}
	}

	var found []DiscoveredServer
	for _, key := range source.Keys {
		raw, ok := top[key]
		if !ok {
			continue
		}
		// The same type Ollama's own configuration uses, so a foreign file is
		// read by the parser that already knows this shape rather than by a
		// second one written for the occasion.
		servers := map[string]*ServerSpec{}
		if err := json.Unmarshal(raw, &servers); err != nil {
			found = append(found, DiscoveredServer{
				Sources: []string{source.Label},
				Paths:   []string{source.Path},
				Origin:  OriginConfig,
				Problem: fmt.Sprintf("%s could not be read: %v", key, err),
			})
			continue
		}
		for _, name := range slices.Sorted(maps.Keys(servers)) {
			spec := servers[name]
			if spec == nil {
				continue
			}
			found = append(found, describeDiscovered(name, spec, source.Label, source.Path, OriginConfig))
		}
		// One key per file is enough; the second is only a fallback for
		// clients that renamed it.
		break
	}
	return found
}

// describeDiscovered turns a foreign server into something Ollama could add,
// or explains why it cannot.
func describeDiscovered(name string, spec *ServerSpec, source, path, origin string) DiscoveredServer {
	found := DiscoveredServer{Sources: []string{source}, Paths: []string{path}, Origin: origin}

	suggested, renamed := sanitiseName(name)
	found.Name = suggested
	if renamed {
		found.Notes = append(found.Notes, fmt.Sprintf("named %q there; Ollama needs %q", name, suggested))
	}

	spec.Name = suggested
	found.Notes = append(found.Notes, sanitiseDiscoveredSpec(spec)...)
	found.Spec = spec
	found.Runs = spec.Summary()

	// The same validation an added server faces. Reporting it here rather than
	// at the moment of the click means the user is told before they try.
	if err := validateServer(suggested, spec); err != nil {
		found.Problem = err.Error()
		found.Spec = nil
	}
	return found
}

// serverNameReplacement is anything Ollama's server names do not allow.
var serverNameReplacement = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

// sanitiseName makes a foreign name into one Ollama accepts, reporting whether
// it had to change it.
func sanitiseName(name string) (string, bool) {
	cleaned := serverNameReplacement.ReplaceAllString(strings.TrimSpace(name), "-")
	cleaned = strings.Trim(cleaned, "-_")
	cleaned = strings.ReplaceAll(cleaned, "__", "_")
	if len(cleaned) > 64 {
		cleaned = strings.Trim(cleaned[:64], "-_")
	}
	if cleaned == "" {
		cleaned = "server"
	}
	return cleaned, cleaned != name
}

// sanitiseDiscoveredSpec removes what must not be carried across, and says what
// it removed.
//
// A credential written literally in another application's configuration cannot
// be copied into Ollama's: mcp.json is a file other programs running as this
// user can read, and Ollama's own validation refuses a literal secret there.
// Rewriting it as an environment reference keeps the server importable and
// leaves the secret where it already is, rather than making a second copy of it
// in a second file.
func sanitiseDiscoveredSpec(spec *ServerSpec) []string {
	var notes []string

	for _, key := range slices.Sorted(maps.Keys(spec.Env)) {
		value := spec.Env[key]
		if value == "" || isEnvRef(value) || !secretishEnvKey.MatchString(key) {
			continue
		}
		spec.Env[key] = fmt.Sprintf("${env:%s}", envVarName(key))
		notes = append(notes, fmt.Sprintf("%s held a value that looks like a credential; it was not copied — set %s in your environment", key, envVarName(key)))
	}

	for _, key := range slices.Sorted(maps.Keys(spec.Headers)) {
		if !sensitiveHeaders[strings.ToLower(strings.TrimSpace(key))] || isEnvRef(spec.Headers[key]) {
			continue
		}
		reference := envVarName(key)
		spec.Headers[key] = fmt.Sprintf("${env:%s}", reference)
		notes = append(notes, fmt.Sprintf("the %s header held a literal value; it was not copied — set %s in your environment", key, reference))
	}

	// A server another client had switched off arrives switched off here too.
	// Turning someone's disabled server on because it was imported would be
	// deciding something they had already decided.
	if spec.Disabled {
		notes = append(notes, "it is switched off in that application, and stays switched off here")
	}
	return notes
}

// envVarName turns a key or header name into a plausible environment variable.
func envVarName(name string) string {
	upper := strings.ToUpper(strings.TrimSpace(name))
	upper = serverNameReplacement.ReplaceAllString(upper, "_")
	upper = strings.Trim(upper, "_")
	if upper == "" || (upper[0] >= '0' && upper[0] <= '9') {
		upper = "MCP_" + upper
	}
	return upper
}

// --- servers answering on this machine ------------------------------------

const (
	// probeTimeout bounds one request to a local port. Everything here is on
	// this machine, where a server that is going to answer answers in
	// milliseconds — measured at three for a local service and under one for a
	// closed port. A second is generous, and it is the difference between a
	// button that takes six seconds and one that takes twenty-four.
	probeTimeout = time.Second
	// probeBudget bounds the whole sweep, so the button cannot hang on a
	// machine with an unusual number of listening ports.
	probeBudget = 30 * time.Second
	// probeConcurrency bounds how many ports are contacted at once. These are
	// connections to this machine, so the limit is about not flooding the
	// user's own services rather than about bandwidth.
	probeConcurrency = 16
)

// probePaths are the paths tried on a listening port. Streamable HTTP servers
// almost always mount at one of these, and trying more would mean sending
// requests to a longer list of things that are not MCP servers at all.
var probePaths = []string{"/mcp", "/"}

// ProbeLoopback finds MCP servers answering on this machine right now.
//
// It contacts only the loopback interface, only ports that are already
// listening, and only with the MCP handshake — the same handshake an ordinary
// connection makes, through the same transport, so what it proves is that a
// real MCP server is there rather than that a port is open.
//
// It is a network act on the user's own machine, which is why it is never run
// as part of loading a page: something has to ask for it.
func ProbeLoopback(ctx context.Context) ([]DiscoveredServer, error) {
	ctx, cancel := context.WithTimeout(ctx, probeBudget)
	defer cancel()

	ports, err := listeningPorts(ctx)
	if err != nil {
		return nil, err
	}

	var (
		mu    sync.Mutex
		found []DiscoveredServer
		wg    sync.WaitGroup
	)
	slots := make(chan struct{}, probeConcurrency)

	for _, listener := range ports {
		wg.Add(1)
		go func(listener loopbackListener) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()

			for _, path := range probePaths {
				url := fmt.Sprintf("http://127.0.0.1:%d%s", listener.Port, path)
				tools, ok := mcpHandshake(ctx, url)
				if !ok {
					continue
				}
				server := DiscoveredServer{
					Name:    sanitiseNameOnly(fmt.Sprintf("%s-%d", listener.Process, listener.Port)),
					Spec:    &ServerSpec{Type: TransportHTTP, URL: url},
					Sources: []string{fmt.Sprintf("%s, listening on port %d", listener.Process, listener.Port)},
					Origin:  OriginListening,
					Runs:    url,
				}
				if tools > 0 {
					server.Notes = append(server.Notes, fmt.Sprintf("it answered the handshake and offers %d tools", tools))
				} else {
					server.Notes = append(server.Notes, "it answered the handshake and offers no tools")
				}
				mu.Lock()
				found = append(found, server)
				mu.Unlock()
				return
			}
		}(listener)
	}
	wg.Wait()

	sort.SliceStable(found, func(i, j int) bool { return found[i].Runs < found[j].Runs })
	return found, nil
}

func sanitiseNameOnly(name string) string {
	cleaned, _ := sanitiseName(name)
	return cleaned
}

// mcpHandshake connects as Ollama would and reports whether an MCP server
// answered, and how many tools it offered.
func mcpHandshake(ctx context.Context, url string) (int, bool) {
	dialCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	// The transport is built here rather than through newTransport because the
	// probe needs a bounded HTTP client and an ordinary connection must not
	// have one. Measured, not assumed: a context deadline alone did not stop
	// this — a local service that accepts a connection and never answers held
	// the handshake open for twenty-four seconds, far past the deadline the
	// context carried. A client timeout is the bound that actually holds.
	//
	// It would be wrong for a real session, where a long-lived stream is the
	// point. A probe only sends initialize and tools/list.
	transport := &sdk.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: &http.Client{Timeout: probeTimeout},
	}

	client := sdk.NewClient(&sdk.Implementation{Name: "ollama", Version: version.Version}, nil)
	session, err := client.Connect(dialCtx, transport, nil)
	if err != nil {
		return 0, false
	}
	defer session.Close()

	tools, _, err := listTools(dialCtx, "probe", session)
	if err != nil {
		// It answered the handshake, which is what makes it an MCP server.
		return 0, true
	}
	return len(tools), true
}

// loopbackListener is one process listening on a port reachable over loopback.
type loopbackListener struct {
	Port    int
	Process string
}

// listeningPorts returns the loopback-reachable TCP ports with something
// listening on them.
//
// lsof is used because Go has no portable way to enumerate listening sockets,
// and because it names the process — which is the difference between telling a
// user "something on port 3000" and "node on port 3000".
func listeningPorts(ctx context.Context) ([]loopbackListener, error) {
	path, err := exec.LookPath("lsof")
	if err != nil {
		return nil, fmt.Errorf("looking for servers on this machine needs lsof, which is not installed: %w", err)
	}

	listCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// -F pcn asks for machine-readable output: process id, command name, and
	// the network address, one field per line.
	cmd := exec.CommandContext(listCtx, path, "-nP", "-iTCP", "-sTCP:LISTEN", "-F", "pcn")
	output, err := cmd.Output()
	if err != nil {
		// lsof exits non-zero when it finds nothing, which is not a failure.
		if len(output) == 0 {
			return nil, nil
		}
	}
	return parseLsof(string(output)), nil
}

// parseLsof reads lsof's -F output into the loopback listeners it describes.
func parseLsof(output string) []loopbackListener {
	var (
		process string
		seen    = map[int]string{}
	)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		switch line[0] {
		case 'c':
			process = line[1:]
		case 'n':
			port, ok := loopbackPort(line[1:])
			if !ok {
				continue
			}
			if _, already := seen[port]; !already {
				seen[port] = process
			}
		}
	}

	listeners := make([]loopbackListener, 0, len(seen))
	for port, name := range seen {
		if strings.TrimSpace(name) == "" {
			name = "something"
		}
		listeners = append(listeners, loopbackListener{Port: port, Process: name})
	}
	sort.Slice(listeners, func(i, j int) bool { return listeners[i].Port < listeners[j].Port })
	return listeners
}

// loopbackPort returns the port from an lsof address, when that address is one
// this machine can reach over loopback. A server bound to a specific external
// address is not reachable at 127.0.0.1 and is left alone.
func loopbackPort(address string) (int, bool) {
	address = strings.TrimSpace(address)
	// lsof writes a listening socket as "host:port" or "*:port".
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return 0, false
	}
	number, err := net.LookupPort("tcp", port)
	if err != nil || number == 0 {
		return 0, false
	}

	switch host {
	case "*", "", "127.0.0.1", "::1", "[::1]", "::", "[::]":
		return number, true
	}
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
		if ip.IsLoopback() || ip.IsUnspecified() {
			return number, true
		}
	}
	return 0, false
}
