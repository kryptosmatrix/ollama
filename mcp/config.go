// Package mcp implements Ollama's Model Context Protocol client: the
// configuration describing MCP servers, the connections to them, and the
// conversion of the tools they advertise into Ollama tool definitions.
//
// The package is deliberately independent of both tool stacks. The core agent
// harness (agent/) and the desktop app (app/) each adapt these types into their
// own Tool interface; neither owns the protocol and neither imports the other.
package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// ConfigPathEnv overrides the location of the MCP server configuration file.
const ConfigPathEnv = "OLLAMA_MCP_CONFIG"

const configFilename = "mcp.json"

// Transport identifies how Ollama reaches an MCP server.
type Transport string

const (
	// TransportStdio launches the server as a child process and speaks
	// JSON-RPC over its standard input and output.
	TransportStdio Transport = "stdio"
	// TransportHTTP connects to a remote server over HTTP.
	TransportHTTP Transport = "http"
)

// serverName constrains names that appear in the model's tool list. Tool names
// are namespaced "<server>__<tool>" (see names.go), so a server name may not
// itself contain the "__" separator.
var serverName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// envRef matches a whole-value reference to an environment variable, the only
// form in which credentials may appear in the configuration file.
var envRef = regexp.MustCompile(`^\$\{env:([A-Za-z_][A-Za-z0-9_]*)\}$`)

// sensitiveHeaders are header names whose values must never be literals in the
// configuration file. They are compared case-insensitively.
var sensitiveHeaders = map[string]bool{
	"authorization":       true,
	"proxy-authorization": true,
	"x-api-key":           true,
	"api-key":             true,
	"cookie":              true,
}

// secretishEnvKey matches environment variable names whose values are very
// likely to be credentials. Such values must be environment references rather
// than literals; ordinary configuration values are unaffected.
var secretishEnvKey = regexp.MustCompile(`(?i)(token|secret|password|passwd|credential|api[-_]?key|access[-_]?key|private[-_]?key)`)

// ServerSpec describes one MCP server. It is the persisted form: what to run or
// where to connect, not the live connection.
//
// Unknown fields are preserved across a load/save round trip so that a
// configuration written by a newer Ollama, or by another MCP client, is not
// silently stripped when this one saves.
type ServerSpec struct {
	// Name is the key this spec was stored under. It is not serialised.
	Name string `json:"-"`

	// Type is the transport. Empty means stdio when Command is set and http
	// when URL is set; an explicit value always wins.
	Type Transport `json:"type,omitempty"`

	// Command and Args are the executable and arguments for a stdio server.
	// The command is executed directly and is never passed through a shell.
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`

	// Env is the environment handed to a stdio server. Values may be
	// "${env:NAME}" references, resolved from Ollama's environment at connect
	// time. The child receives only these variables plus a minimal base set.
	Env map[string]string `json:"env,omitempty"`

	// URL is the endpoint of an http server.
	URL string `json:"url,omitempty"`

	// Headers are sent with every request to an http server. Values for
	// sensitive header names must be "${env:NAME}" references.
	Headers map[string]string `json:"headers,omitempty"`

	// Disabled records the user's intent to keep this server switched off.
	// It matches the field name used by other MCP clients so that a pasted
	// configuration behaves the way its author expected.
	Disabled bool `json:"disabled,omitempty"`

	// extra preserves fields this version of Ollama does not understand.
	extra map[string]json.RawMessage
}

// specFields are the keys ServerSpec serialises itself. Anything else found in
// a server object is preserved in extra.
var specFields = map[string]bool{
	"type": true, "command": true, "args": true, "env": true,
	"url": true, "headers": true, "disabled": true,
}

func (s *ServerSpec) UnmarshalJSON(data []byte) error {
	type plain ServerSpec
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*s = ServerSpec(p)

	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	for key, raw := range all {
		if specFields[key] {
			continue
		}
		if s.extra == nil {
			s.extra = make(map[string]json.RawMessage)
		}
		s.extra[key] = raw
	}
	return nil
}

func (s ServerSpec) MarshalJSON() ([]byte, error) {
	type plain ServerSpec
	data, err := json.Marshal(plain(s))
	if err != nil {
		return nil, err
	}
	if len(s.extra) == 0 {
		return data, nil
	}

	var merged map[string]json.RawMessage
	if err := json.Unmarshal(data, &merged); err != nil {
		return nil, err
	}
	if merged == nil {
		merged = make(map[string]json.RawMessage, len(s.extra))
	}
	for key, raw := range s.extra {
		if _, taken := merged[key]; !taken {
			merged[key] = raw
		}
	}
	return json.Marshal(merged)
}

// Transport reports the transport this spec uses, applying the inference rule
// documented on Type. It returns an empty Transport when the spec names
// neither a command nor a URL.
func (s *ServerSpec) transport() Transport {
	switch {
	case s.Type != "":
		return s.Type
	case s.Command != "":
		return TransportStdio
	case s.URL != "":
		return TransportHTTP
	default:
		return ""
	}
}

// Config is the parsed contents of mcp.json.
type Config struct {
	// Servers is keyed by server name. Each spec's Name field is populated on
	// load and kept in step by Set and Remove.
	Servers map[string]*ServerSpec

	// extra preserves top-level fields this version of Ollama does not
	// understand, so a save does not strip another client's settings.
	extra map[string]json.RawMessage
}

// ConfigPath returns the location of the MCP configuration file. It mirrors the
// resolution order used for skills (agent.SkillsDir): an explicit environment
// override, then XDG_CONFIG_HOME, then ~/.ollama.
func ConfigPath() (string, error) {
	if path := strings.TrimSpace(os.Getenv(ConfigPathEnv)); path != "" {
		return filepath.Abs(path)
	}
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return filepath.Join(xdg, "ollama", configFilename), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ollama", configFilename), nil
}

// Load reads the configuration at path. A missing file is not an error: it
// yields an empty configuration, because "no servers configured yet" is the
// ordinary starting state.
//
// Load fails only when the file cannot be read or is not valid JSON. Problems
// with individual servers are reported by Problems, so that one malformed entry
// does not make every other server unreachable.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{Servers: map[string]*ServerSpec{}}, nil
		}
		return nil, fmt.Errorf("read mcp config %s: %w", path, err)
	}
	return parseConfig(data, path)
}

func parseConfig(data []byte, path string) (*Config, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return &Config{Servers: map[string]*ServerSpec{}}, nil
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return nil, fmt.Errorf("parse mcp config %s: %w", path, err)
	}

	cfg := &Config{Servers: map[string]*ServerSpec{}}
	for key, raw := range top {
		if key == "mcpServers" {
			if err := json.Unmarshal(raw, &cfg.Servers); err != nil {
				return nil, fmt.Errorf("parse mcp config %s: mcpServers: %w", path, err)
			}
			continue
		}
		if cfg.extra == nil {
			cfg.extra = make(map[string]json.RawMessage)
		}
		cfg.extra[key] = raw
	}

	for name, spec := range cfg.Servers {
		if spec == nil {
			return nil, fmt.Errorf("parse mcp config %s: server %q is null", path, name)
		}
		spec.Name = name
	}
	return cfg, nil
}

// Marshal renders the configuration as it would be written to disk.
func (c *Config) Marshal() ([]byte, error) {
	top := make(map[string]json.RawMessage, len(c.extra)+1)
	maps.Copy(top, c.extra)

	servers, err := json.Marshal(c.servers())
	if err != nil {
		return nil, fmt.Errorf("marshal mcp servers: %w", err)
	}
	top["mcpServers"] = servers

	data, err := json.MarshalIndent(top, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal mcp config: %w", err)
	}
	return append(data, '\n'), nil
}

// servers returns a non-nil map so that an empty configuration serialises as
// "mcpServers": {} rather than null.
func (c *Config) servers() map[string]*ServerSpec {
	if c.Servers == nil {
		return map[string]*ServerSpec{}
	}
	return c.Servers
}

// Save writes the configuration to path, creating the directory if needed. The
// write is atomic — a temporary file in the same directory followed by a
// rename — so a crash mid-write cannot leave a truncated configuration behind.
//
// The file is written 0600 and its directory 0700. This file names executables
// that Ollama will run, so anything able to write it can run code as this user.
func (c *Config) Save(path string) error {
	data, err := c.Marshal()
	if err != nil {
		return err
	}
	return writeFilePrivate(path, data)
}

func writeFilePrivate(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create mcp config directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".mcp-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp mcp config: %w", err)
	}
	tmpPath := tmp.Name()

	cleanup := func() {
		tmp.Close()
		os.Remove(tmpPath)
	}

	if err := tmp.Chmod(0o600); err != nil {
		cleanup()
		return fmt.Errorf("chmod temp mcp config: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write temp mcp config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync temp mcp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp mcp config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("replace mcp config: %w", err)
	}
	return nil
}

// Names returns the configured server names in a stable order.
func (c *Config) Names() []string {
	return slices.Sorted(maps.Keys(c.servers()))
}

// Get returns the spec stored under name.
func (c *Config) Get(name string) (*ServerSpec, bool) {
	spec, ok := c.servers()[name]
	return spec, ok
}

// Set stores spec under name, replacing any existing entry.
func (c *Config) Set(name string, spec *ServerSpec) {
	if c.Servers == nil {
		c.Servers = map[string]*ServerSpec{}
	}
	spec.Name = name
	c.Servers[name] = spec
}

// Remove deletes the named server. It reports whether the server was present.
func (c *Config) Remove(name string) bool {
	if _, ok := c.servers()[name]; !ok {
		return false
	}
	delete(c.Servers, name)
	return true
}

// Problems returns a validation error for every server that cannot safely be
// used, keyed by server name. An empty result means every configured server is
// usable. Callers must not connect a server that appears here.
func (c *Config) Problems() map[string]error {
	var problems map[string]error
	for _, name := range c.Names() {
		spec, _ := c.Get(name)
		if err := validateServer(name, spec); err != nil {
			if problems == nil {
				problems = make(map[string]error)
			}
			problems[name] = err
		}
	}
	return problems
}

func validateServer(name string, spec *ServerSpec) error {
	var errs []error

	if !serverName.MatchString(name) {
		errs = append(errs, fmt.Errorf("server name %q must be 1-64 characters of letters, digits, underscore or hyphen and start with a letter or digit", name))
	}
	if strings.Contains(name, "__") {
		errs = append(errs, fmt.Errorf("server name %q must not contain %q, which separates the server from the tool in a namespaced tool name", name, "__"))
	}

	switch spec.transport() {
	case TransportStdio:
		if strings.TrimSpace(spec.Command) == "" {
			errs = append(errs, errors.New(`stdio server needs a "command"`))
		}
		if spec.URL != "" {
			errs = append(errs, errors.New(`stdio server must not set "url"`))
		}
		if len(spec.Headers) > 0 {
			errs = append(errs, errors.New(`stdio server must not set "headers"`))
		}
	case TransportHTTP:
		if err := validateURL(spec.URL); err != nil {
			errs = append(errs, err)
		}
		if spec.Command != "" || len(spec.Args) > 0 {
			errs = append(errs, errors.New(`http server must not set "command" or "args"`))
		}
	default:
		errs = append(errs, errors.New(`server needs either a "command" (stdio) or a "url" (http)`))
	}

	errs = append(errs, validateEnv(spec.Env)...)
	errs = append(errs, validateHeaders(spec.Headers)...)

	return errors.Join(errs...)
}

func validateEnv(env map[string]string) []error {
	var errs []error
	for _, key := range slices.Sorted(maps.Keys(env)) {
		value := env[key]
		if secretishEnvKey.MatchString(key) && !isEnvRef(value) && value != "" {
			errs = append(errs, fmt.Errorf(`env %q looks like a credential and must be written as "${env:NAME}" rather than a literal value`, key))
		}
	}
	return errs
}

func validateHeaders(headers map[string]string) []error {
	var errs []error
	for _, key := range slices.Sorted(maps.Keys(headers)) {
		if sensitiveHeaders[strings.ToLower(strings.TrimSpace(key))] && !isEnvRef(headers[key]) {
			errs = append(errs, fmt.Errorf(`header %q must be written as "${env:NAME}" rather than a literal value`, key))
		}
	}
	return errs
}

func isEnvRef(value string) bool {
	return envRef.MatchString(strings.TrimSpace(value))
}

func validateURL(raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return errors.New(`http server needs a "url"`)
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("url %q is not a valid URL: %w", value, err)
	}
	switch parsed.Scheme {
	case "https":
	case "http":
		if !isLoopbackHost(parsed.Hostname()) {
			return fmt.Errorf("url %q must use https; plain http is allowed only for loopback addresses", value)
		}
	default:
		return fmt.Errorf("url %q must use https (or http on loopback), not %q", value, parsed.Scheme)
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("url %q has no host", value)
	}
	// Userinfo is unambiguous: it is a credential or it is nothing. Accepting
	// it would put a literal secret in mcp.json, which is the one thing this
	// file exists to prevent — and configurations pasted from other MCP clients
	// arrive this way, because several of them put the token in the URL. The
	// refusal names the alternative, because a rule with no way round it is a
	// wall rather than a gate.
	if parsed.User != nil {
		return fmt.Errorf(`url %q carries a credential in the address; move it to a header, for example "Authorization": "${env:MY_TOKEN}"`, redactURL(value))
	}
	return nil
}

// redactURL renders a URL with any embedded password removed, for messages and
// for anything written to disk. The username is kept: it is not the secret and
// it is what lets a user recognise which entry is being talked about.
func redactURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.User == nil {
		return raw
	}
	if _, hasPassword := parsed.User.Password(); hasPassword {
		parsed.User = url.User(parsed.User.Username())
	} else {
		// A bare userinfo with no colon is a token often enough that keeping it
		// is not worth the risk, and it names nothing useful.
		parsed.User = url.User(redactedMarker)
	}
	return parsed.String()
}

// redactedMarker stands in for a value that has been withheld.
const redactedMarker = "[redacted]"

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// ErrMissingEnv reports an environment reference whose variable is not set.
type ErrMissingEnv struct {
	// Field names where the reference appeared, such as `header "Authorization"`.
	Field string
	// Variable is the environment variable that was referenced.
	Variable string
}

func (e *ErrMissingEnv) Error() string {
	return fmt.Sprintf("%s references environment variable %s, which is not set", e.Field, e.Variable)
}

// resolveValue expands a whole-value "${env:NAME}" reference from the process
// environment. Values that are not references are returned unchanged, so
// ordinary configuration values pass through untouched.
func resolveValue(field, value string) (string, error) {
	match := envRef.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return value, nil
	}
	resolved, ok := os.LookupEnv(match[1])
	if !ok {
		return "", &ErrMissingEnv{Field: field, Variable: match[1]}
	}
	return resolved, nil
}

// ResolveEnv returns the spec's environment with references expanded. It fails
// rather than substituting an empty string when a referenced variable is unset,
// so a missing credential surfaces as a clear error instead of an
// authentication failure from the server.
func (s *ServerSpec) ResolveEnv() (map[string]string, error) {
	return resolveMap(s.Env, "env")
}

// ResolveHeaders returns the spec's headers with references expanded.
func (s *ServerSpec) ResolveHeaders() (map[string]string, error) {
	return resolveMap(s.Headers, "header")
}

func resolveMap(in map[string]string, kind string) (map[string]string, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(in))
	var errs []error
	for _, key := range slices.Sorted(maps.Keys(in)) {
		value, err := resolveValue(fmt.Sprintf("%s %q", kind, key), in[key])
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out[key] = value
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return out, nil
}
