package mcp

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// baseEnvVars are the only variables inherited from Ollama's own environment by
// a stdio MCP server. A server process is arbitrary code running with the
// user's privileges, so it is given what it needs to find its interpreter and
// its home and nothing else; anything further is declared in the server's own
// "env" block, where the user can see it.
var baseEnvVars = []string{
	"PATH",
	"HOME",
	"LANG",
	"LC_ALL",
	"TMPDIR",
}

// windowsEnvVars are additionally inherited on Windows, where a process that
// cannot see them frequently fails in ways that are hard to diagnose.
var windowsEnvVars = []string{
	"SystemRoot",
	"SystemDrive",
	"USERPROFILE",
	"APPDATA",
	"LOCALAPPDATA",
	"TEMP",
	"TMP",
	"PATHEXT",
	"COMSPEC",
	"ProgramData",
	"ProgramFiles",
	"ProgramFiles(x86)",
}

// childEnv builds the environment for a stdio server: the small inherited base
// above, overlaid with the server's own declared variables.
func childEnv(declared map[string]string) []string {
	names := slices.Clone(baseEnvVars)
	if runtime.GOOS == "windows" {
		names = append(names, windowsEnvVars...)
	}

	env := make([]string, 0, len(names)+len(declared))
	for _, name := range names {
		if _, shadowed := declared[name]; shadowed {
			continue
		}
		if value, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+value)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(declared)) {
		env = append(env, name+"="+declared[name])
	}
	return env
}

// headerTransport adds fixed headers to every request. It wraps rather than
// replaces the base transport so proxy and TLS settings are preserved.
type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// RoundTrippers must not modify the request they are given.
	clone := req.Clone(req.Context())
	for name, value := range t.headers {
		clone.Header.Set(name, value)
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(clone)
}

// newTransport builds the SDK transport for a spec. It resolves any ${env:NAME}
// references at this moment rather than at load time, so a credential lives in
// memory only for as long as a connection needs it.
//
// A stdio server is executed directly. It is never passed through a shell:
// there is no point in the pipeline where a command string could be
// re-interpreted, so a server name or argument cannot smuggle shell syntax.
func newTransport(ctx context.Context, spec *ServerSpec) (sdk.Transport, error) {
	switch spec.transport() {
	case TransportStdio:
		env, err := spec.ResolveEnv()
		if err != nil {
			return nil, err
		}
		if _, err := exec.LookPath(spec.Command); err != nil {
			return nil, fmt.Errorf("command %q not found: %w", spec.Command, err)
		}

		cmd := exec.CommandContext(ctx, spec.Command, spec.Args...)
		cmd.Env = childEnv(env)
		cmd.Stderr = os.Stderr
		return &sdk.CommandTransport{Command: cmd}, nil

	case TransportHTTP:
		headers, err := spec.ResolveHeaders()
		if err != nil {
			return nil, err
		}
		client := &http.Client{}
		if len(headers) > 0 {
			client.Transport = &headerTransport{headers: headers}
		}
		return &sdk.StreamableClientTransport{
			Endpoint:   spec.URL,
			HTTPClient: client,
		}, nil

	default:
		return nil, fmt.Errorf("server %q declares no usable transport", spec.Name)
	}
}

// flattenContent renders a tool result for the model. MCP content is a list of
// typed blocks; the model consumes text, so text blocks are joined and other
// kinds are named rather than silently dropped — a caller that sees "[image
// content omitted]" can act on it, whereas an empty result looks like success.
func flattenContent(content []sdk.Content, limit int) string {
	var parts []string
	for _, block := range content {
		switch typed := block.(type) {
		case *sdk.TextContent:
			parts = append(parts, typed.Text)
		case *sdk.ImageContent:
			parts = append(parts, "[image content omitted]")
		case *sdk.AudioContent:
			parts = append(parts, "[audio content omitted]")
		case *sdk.ResourceLink:
			parts = append(parts, fmt.Sprintf("[resource link: %s]", typed.URI))
		case *sdk.EmbeddedResource:
			parts = append(parts, "[embedded resource omitted]")
		default:
			parts = append(parts, "[unsupported content omitted]")
		}
	}
	return sanitiseText(strings.Join(parts, "\n"), limit)
}
