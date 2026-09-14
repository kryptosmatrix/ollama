package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/ollama/ollama/agent"
	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/mcp"
)

// MCP adapts one tool advertised by an MCP server to the agent harness's Tool
// interface.
//
// The protocol library stays behind mcp.Manager: this type deals only in
// mcp.Tool and api.ToolFunction, so the harness has no idea which library — or
// whether any library — is speaking to the server.
type MCP struct {
	manager *mcp.Manager
	tool    mcp.Tool
	schema  api.ToolFunction
}

// NewMCP adapts a tool for the harness. It converts the schema once, here,
// rather than on every call: the harness's Schema method cannot report a
// failure, and a tool whose schema cannot be represented must be refused at
// registration rather than silently offered with an empty parameter list.
//
// In practice mcp.Manager has already filtered those out at connect time, so an
// error here means the two disagree, which is worth surfacing rather than
// papering over.
func NewMCP(manager *mcp.Manager, tool mcp.Tool) (*MCP, error) {
	if manager == nil {
		return nil, errors.New("mcp tool needs a manager")
	}
	schema, err := tool.Schema()
	if err != nil {
		return nil, err
	}
	return &MCP{manager: manager, tool: tool, schema: schema}, nil
}

// RegisterMCP adds every usable tool from the manager to the registry. It
// returns the names registered, and an error for any tool that could not be
// adapted — registration continues past a failure so one bad tool does not cost
// the user every other one.
func RegisterMCP(registry *agent.Registry, manager *mcp.Manager) ([]string, error) {
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
func (m *MCP) Name() string { return m.schema.Name }

// Description is the server's own text, already sanitised and capped, with any
// schema constraints Ollama's tool type cannot carry appended to it.
func (m *MCP) Description() string { return m.schema.Description }

// Schema is the converted tool definition.
func (m *MCP) Schema() api.ToolFunction { return m.schema }

// RequiresApproval is always true.
//
// Approving a server (mcp.Approvals) is agreement to *run* it. It is not
// agreement for the model to invoke any of its tools with any arguments it
// likes, and the two are genuinely different questions: a filesystem server the
// user was happy to start is still a filesystem server the model can ask to
// delete something. The harness's scope mechanism means the user is asked once
// per tool per session, not once per call.
func (m *MCP) RequiresApproval(map[string]any) bool { return true }

// ApprovalScope makes an approval cover this tool on this server, and nothing
// else. Approving "files__read" must not approve "files__write", and must not
// approve another server's "read".
func (m *MCP) ApprovalScope(map[string]any) string { return m.schema.Name }

// Execute calls the tool on its server.
//
// A tool-level failure reported by the server — as opposed to a transport
// failure — is returned as an error, because that is how the harness renders a
// failed tool call to both the model and the user. The server's own message is
// preserved so the model can react to it rather than to a generic failure.
func (m *MCP) Execute(ctx context.Context, _ agent.ToolContext, args map[string]any) (agent.ToolResult, error) {
	result, err := m.manager.Call(ctx, m.schema.Name, args)
	if err != nil {
		return agent.ToolResult{}, err
	}
	if result.IsError {
		if result.Content == "" {
			return agent.ToolResult{}, fmt.Errorf("%s reported a failure", m.schema.Name)
		}
		return agent.ToolResult{}, errors.New(result.Content)
	}
	return agent.ToolResult{Content: result.Content}, nil
}
