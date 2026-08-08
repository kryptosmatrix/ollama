package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultRegistryURL is the official MCP Registry.
//
// It is a metadata registry, not a vetting service: anyone may publish to it,
// and a listing is a claim by its publisher rather than anything Ollama has
// checked. Everything in this file is written on that assumption.
const DefaultRegistryURL = "https://registry.modelcontextprotocol.io"

// RegistryURLEnv overrides the registry, for testing and for private mirrors.
const RegistryURLEnv = "OLLAMA_MCP_REGISTRY"

const (
	defaultRegistryTimeout = 15 * time.Second
	// maxRegistryResponse caps a registry reply. The registry is a third party;
	// a page load must not be able to consume unbounded memory.
	maxRegistryResponse = 4 << 20
	// registryPageLimit is the page size requested. The API's maximum is 100.
	registryPageLimit = 30
)

// RegistryEntry is one server as the registry describes it.
type RegistryEntry struct {
	// Name is the publisher-scoped reverse-DNS identifier, such as
	// "io.github.someone/weather". The namespace is shown to the user because
	// it is the only provenance the registry offers.
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	WebsiteURL  string `json:"websiteUrl,omitempty"`

	Repository *RegistryRepository `json:"repository,omitempty"`
	Packages   []RegistryPackage   `json:"packages,omitempty"`
	Remotes    []RegistryRemote    `json:"remotes,omitempty"`
}

// RegistryRepository is where the source lives, if the publisher said.
type RegistryRepository struct {
	URL    string `json:"url,omitempty"`
	Source string `json:"source,omitempty"`
}

// RegistryPackage is a way to run the server locally.
type RegistryPackage struct {
	RegistryType string `json:"registryType"`
	Identifier   string `json:"identifier"`
	Version      string `json:"version,omitempty"`
	// FileSHA256 is an integrity hash, where the publisher provided one.
	FileSHA256       string             `json:"fileSha256,omitempty"`
	RuntimeArguments []RegistryArgument `json:"runtimeArguments,omitempty"`
	PackageArguments []RegistryArgument `json:"packageArguments,omitempty"`
	EnvironmentVars  []RegistryVariable `json:"environmentVariables,omitempty"`
}

// RegistryArgument is one argument the publisher says the package needs.
type RegistryArgument struct {
	Type  string `json:"type,omitempty"`
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// RegistryVariable is a value the user must supply, such as an API key.
type RegistryVariable struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsSecret    bool   `json:"isSecret,omitempty"`
	IsRequired  bool   `json:"isRequired,omitempty"`
}

// RegistryRemote is a hosted endpoint for the server.
type RegistryRemote struct {
	Type    string             `json:"type"`
	URL     string             `json:"url"`
	Headers []RegistryVariable `json:"headers,omitempty"`
}

// RegistryPage is one page of search results.
type RegistryPage struct {
	Servers []RegistryEntry
	// NextCursor is empty when there are no more results.
	NextCursor string
}

// RegistryClient reads the official MCP Registry.
type RegistryClient struct {
	BaseURL string
	HTTP    *http.Client
}

// NewRegistryClient returns a client for the registry named by
// OLLAMA_MCP_REGISTRY, or the official one.
func NewRegistryClient(baseURL string) *RegistryClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultRegistryURL
	}
	return &RegistryClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: defaultRegistryTimeout},
	}
}

// Search returns one page of servers. An empty query lists everything.
func (c *RegistryClient) Search(ctx context.Context, query, cursor string) (RegistryPage, error) {
	params := url.Values{}
	params.Set("limit", fmt.Sprint(registryPageLimit))
	params.Set("version", "latest")
	if strings.TrimSpace(query) != "" {
		params.Set("search", query)
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}

	var body struct {
		Servers  []RegistryEntry `json:"servers"`
		Metadata struct {
			NextCursor string `json:"nextCursor"`
		} `json:"metadata"`
	}
	if err := c.get(ctx, "/v0/servers?"+params.Encode(), &body); err != nil {
		return RegistryPage{}, err
	}
	return RegistryPage{Servers: body.Servers, NextCursor: body.Metadata.NextCursor}, nil
}

func (c *RegistryClient) get(ctx context.Context, path string, into any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")

	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: defaultRegistryTimeout}
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("reach the MCP registry: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("MCP registry returned %s", response.Status)
	}

	decoder := json.NewDecoder(http.MaxBytesReader(nil, response.Body, maxRegistryResponse))
	if err := decoder.Decode(into); err != nil {
		return fmt.Errorf("read the MCP registry response: %w", err)
	}
	return nil
}

// Publisher is the namespace half of a reverse-DNS registry name. It is the
// only provenance the registry carries, so it is surfaced rather than hidden.
func (e RegistryEntry) Publisher() string {
	namespace, _, found := strings.Cut(e.Name, "/")
	if !found {
		return e.Name
	}
	return namespace
}

// ErrUnresolvable reports that an entry cannot be turned into something Ollama
// could run. It is returned rather than guessed at: presenting an install
// button that would produce a command line nobody has verified is worse than
// saying the entry is not usable here.
var ErrUnresolvable = errors.New("this entry cannot be installed by Ollama")

// Resolve turns a registry entry into a server specification.
//
// The result is deliberately not enabled or approved by anything here. It is
// what the user must read — Summary() renders the exact command line — before
// agreeing to it.
//
// A remote endpoint is preferred when the publisher offers one, because it runs
// no code on the user's machine. Otherwise a package is resolved into a
// concrete command line for its ecosystem.
func (e RegistryEntry) Resolve() (*ServerSpec, error) {
	for _, remote := range e.Remotes {
		switch remote.Type {
		case "streamable-http", "sse", "http":
			if strings.TrimSpace(remote.URL) == "" {
				continue
			}
			spec := &ServerSpec{Type: TransportHTTP, URL: remote.URL}
			if headers := declaredValues(remote.Headers); len(headers) > 0 {
				spec.Headers = headers
			}
			return spec, nil
		}
	}

	for _, pkg := range e.Packages {
		command, args, err := resolvePackage(pkg)
		if err != nil {
			continue
		}
		spec := &ServerSpec{Type: TransportStdio, Command: command, Args: args}
		if env := declaredValues(pkg.EnvironmentVars); len(env) > 0 {
			spec.Env = env
		}
		return spec, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrUnresolvable, e.Name)
}

// resolvePackage maps a package to the command line that runs it.
//
// Only ecosystems whose runner is unambiguous are supported. An unknown
// registryType is refused rather than guessed: inventing a command line for an
// ecosystem nobody has checked is how a user ends up approving something that
// does not do what its name suggests.
func resolvePackage(pkg RegistryPackage) (string, []string, error) {
	identifier := strings.TrimSpace(pkg.Identifier)
	if identifier == "" {
		return "", nil, fmt.Errorf("%w: package has no identifier", ErrUnresolvable)
	}
	// A package name never begins with a dash. One that does is read by the
	// runner as an instruction rather than a package — "npx -y --call=..." runs
	// the command it names. The user would be shown that command line before
	// agreeing to it, but a resolver whose contract is that a command line is
	// derived rather than guessed must not derive one from a value it never
	// checked, and refusing costs a publisher nothing they can legitimately
	// want.
	if strings.HasPrefix(identifier, "-") {
		return "", nil, fmt.Errorf("%w: package identifier %q begins with a dash, which a runner reads as an option rather than a package", ErrUnresolvable, identifier)
	}
	if version := strings.TrimSpace(pkg.Version); strings.HasPrefix(version, "-") {
		return "", nil, fmt.Errorf("%w: package version %q begins with a dash", ErrUnresolvable, version)
	}
	versioned := identifier
	if version := strings.TrimSpace(pkg.Version); version != "" {
		versioned = identifier + "@" + version
	}

	switch strings.ToLower(pkg.RegistryType) {
	case "npm":
		return "npx", append([]string{"-y", versioned}, argumentValues(pkg.PackageArguments)...), nil
	case "pypi":
		args := []string{}
		if version := strings.TrimSpace(pkg.Version); version != "" {
			args = append(args, identifier+"=="+version)
		} else {
			args = append(args, identifier)
		}
		return "uvx", append(args, argumentValues(pkg.PackageArguments)...), nil
	case "oci":
		args := append([]string{"run", "--rm", "-i"}, argumentValues(pkg.RuntimeArguments)...)
		args = append(args, identifier)
		return "docker", append(args, argumentValues(pkg.PackageArguments)...), nil
	default:
		return "", nil, fmt.Errorf("%w: Ollama does not know how to run a %q package", ErrUnresolvable, pkg.RegistryType)
	}
}

// argumentValues renders the publisher's declared arguments. Only positional
// values and named flags with a value are carried; an argument with nothing to
// say is dropped rather than emitted as an empty string.
func argumentValues(arguments []RegistryArgument) []string {
	var values []string
	for _, argument := range arguments {
		name := strings.TrimSpace(argument.Name)
		value := strings.TrimSpace(argument.Value)
		switch {
		case name != "" && value != "":
			values = append(values, name, value)
		case name != "":
			values = append(values, name)
		case value != "":
			values = append(values, value)
		}
	}
	return values
}

// declaredValues turns the variables a publisher says are needed into
// environment references rather than literals.
//
// The registry describes what a server wants; it never carries the user's
// secret. Emitting "${env:NAME}" means the value is read from the user's own
// environment at connect time and never written into the configuration file,
// which is the same rule the rest of this package enforces.
func declaredValues(variables []RegistryVariable) map[string]string {
	if len(variables) == 0 {
		return nil
	}
	values := make(map[string]string, len(variables))
	// Two names that differ only in punctuation reduce to the same environment
	// variable — "X-Custom-Auth" and "X_Custom_Auth" both become
	// X_CUSTOM_AUTH. Sharing one variable between two headers means the user
	// sets one value and both receive it, and if they were meant to carry
	// different credentials one of them is silently wrong. Each name gets its
	// own reference instead.
	taken := make(map[string]bool, len(variables))
	for _, variable := range variables {
		name := strings.TrimSpace(variable.Name)
		if name == "" {
			continue
		}
		reference := envReferenceName(name)
		if taken[reference] {
			for suffix := 2; ; suffix++ {
				candidate := reference + "_" + strconv.Itoa(suffix)
				if !taken[candidate] {
					reference = candidate
					break
				}
			}
		}
		taken[reference] = true
		values[name] = "${env:" + reference + "}"
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

// envReferenceName makes a header or variable name usable as an environment
// variable name, since a header may contain characters an env name may not.
func envReferenceName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	resolved := b.String()
	if resolved == "" || (resolved[0] >= '0' && resolved[0] <= '9') {
		resolved = "MCP_" + resolved
	}
	return resolved
}
