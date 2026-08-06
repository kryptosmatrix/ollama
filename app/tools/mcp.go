//go:build windows || darwin

package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/mcp"
)

// OllamaTool is implemented by tools that already hold a faithful
// api.ToolFunction.
//
// The alternative path describes a tool as a plain map and rebuilds it with
// toolFunctionFromSchema, which keeps only each property's type and description
// — enums, nested objects, array items and alternatives are all lost. That is
// not survivable for an MCP tool, whose schema was converted carefully from the
// server's own and would arrive at the model stripped of exactly the parts that
// make a call valid. Nor is it survivable for browser.open, whose "id" is a URL
// or a link index and so needs anyOf to say so.
//
// Every tool the desktop app registers now implements this, so no tool the user
// can reach goes through the derived path. That path stays because implementing
// this interface is optional: Tool is the published contract and Register
// accepts anything satisfying it, so a tool that only has a map must still get
// a usable definition rather than none.
type OllamaTool interface {
	ToolFunction() api.ToolFunction
}

// MCP adapts one tool advertised by an MCP server to the desktop app's Tool
// interface. The protocol library stays behind mcp.Manager; this type deals
// only in mcp.Tool and api.ToolFunction.
type MCP struct {
	manager *mcp.Manager
	tool    mcp.Tool
	fn      api.ToolFunction
	schema  map[string]any
}

// NewMCP adapts a tool for the app. The schema is converted once here rather
// than per call, because the Tool interface's Schema method cannot report a
// failure and a tool whose schema cannot be represented must be refused at
// registration rather than offered with no parameters.
func NewMCP(manager *mcp.Manager, tool mcp.Tool) (*MCP, error) {
	if manager == nil {
		return nil, errors.New("mcp tool needs a manager")
	}
	fn, err := tool.Schema()
	if err != nil {
		return nil, err
	}

	// A map form is kept as well, because the registry's own listing describes
	// tools that way and other callers read it.
	schema, err := toolSchemaMap(fn.Parameters)
	if err != nil {
		return nil, fmt.Errorf("tool %q: %w", fn.Name, err)
	}

	return &MCP{manager: manager, tool: tool, fn: fn, schema: schema}, nil
}

// RegisterMCP adds every usable tool from the manager to the registry. It
// returns the names registered and an error for any tool that could not be
// adapted; registration continues past a failure so one bad tool does not cost
// the user the rest.
func RegisterMCP(registry *Registry, manager *mcp.Manager) ([]string, error) {
	if registry == nil || manager == nil {
		return nil, nil
	}

	var registered []string
	var errs []error
	for _, tool := range manager.Tools() {
		adapted, err := NewMCP(manager, tool)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		registry.Register(adapted)
		registered = append(registered, adapted.Name())
	}
	return registered, errors.Join(errs...)
}

// Name is the namespaced name the model sees: "<server>__<tool>".
func (m *MCP) Name() string { return m.fn.Name }

// Description is the server's own text, already sanitised and capped, with any
// schema constraints Ollama's tool type cannot carry appended to it.
func (m *MCP) Description() string { return m.fn.Description }

// Schema is the parameter schema as a plain map, for the registry's listing.
func (m *MCP) Schema() map[string]any { return m.schema }

// ToolFunction is the faithful definition, used in preference to the map when
// the request to the model is built.
func (m *MCP) ToolFunction() api.ToolFunction { return m.fn }

// Prompt adds nothing to the system prompt. What the tool does is carried by
// its description, which came from the server and is treated as untrusted data
// rather than as instructions to the model.
func (m *MCP) Prompt() string { return "" }

// RequiresApproval is always true.
//
// Approving a server is agreement to run it. It is not agreement for the model
// to invoke any of its tools with any arguments it likes, and the two are
// different questions: a filesystem server the user was happy to start is still
// one the model can ask to delete something.
func (m *MCP) RequiresApproval(map[string]any) bool { return true }

// ApprovalScope makes an approval cover this tool on this server and nothing
// else, so agreeing to one tool never agrees to its siblings.
func (m *MCP) ApprovalScope(map[string]any) string { return m.fn.Name }

// Execute calls the tool on its server. It returns the structured result for
// storage and display, and the text the model reads.
//
// A failure the server itself reports is returned as an error, because that is
// how the app renders a failed tool call; the server's own message is preserved
// so the model reacts to the real reason.
func (m *MCP) Execute(ctx context.Context, args map[string]any) (any, string, error) {
	result, err := m.manager.Call(ctx, m.fn.Name, args)
	if err != nil {
		return nil, "", err
	}
	if result.IsError {
		if result.Content == "" {
			return nil, "", fmt.Errorf("%s reported a failure", m.fn.Name)
		}
		return nil, "", errors.New(result.Content)
	}
	return result.Structured, result.Content, nil
}
