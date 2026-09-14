//go:build windows || darwin

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/ollama/ollama/api"
)

// Tool defines the interface that all tools must implement
type Tool interface {
	// Name returns the unique identifier for the tool
	Name() string

	// Description returns a human-readable description of what the tool does
	Description() string

	// Schema returns the JSON schema for the tool's parameters
	Schema() map[string]any

	// Execute runs the tool with the given arguments and returns result to store in db, and a string result for the model
	Execute(ctx context.Context, args map[string]any) (any, string, error)

	// Prompt returns a prompt for the tool
	Prompt() string
}

// Registry manages the available tools and their execution
type Registry struct {
	tools      map[string]Tool
	workingDir string // Working directory for all tool operations
}

// NewRegistry creates a new tool registry with no tools
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all available tools
func (r *Registry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// SetWorkingDir sets the working directory for all tool operations
func (r *Registry) SetWorkingDir(dir string) {
	r.workingDir = dir
}

// Execute runs a tool with the given name and arguments
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (any, string, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, "", fmt.Errorf("unknown tool: %s", name)
	}

	result, text, err := tool.Execute(ctx, args)
	if err != nil {
		return nil, "", err
	}
	return result, text, nil
}

// ToolCall represents a request to execute a tool
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction represents the function call details
type ToolFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    any    `json:"content"`
	Error      string `json:"error,omitempty"`
}

// ToolSchemas returns all tools as schema maps suitable for API calls
func (r *Registry) AvailableTools() []map[string]any {
	schemas := make([]map[string]any, 0, len(r.tools))
	for _, tool := range r.tools {
		schema := map[string]any{
			"name":        tool.Name(),
			"description": tool.Description(),
			"schema":      tool.Schema(),
		}
		schemas = append(schemas, schema)
	}
	return schemas
}

// OllamaTools returns every registered tool as an Ollama tool definition, in a
// stable order.
//
// A tool that already holds a faithful definition supplies it directly; the
// rest are derived from their schema map. Deriving is lossy — it keeps only a
// property's type and description — so anything that has the real thing must
// not be round-tripped through it.
func (r *Registry) OllamaTools() api.Tools {
	names := r.ToolNames()
	sort.Strings(names)

	definitions := make(api.Tools, 0, len(names))
	for _, name := range names {
		tool := r.tools[name]
		if faithful, ok := tool.(OllamaTool); ok {
			definitions = append(definitions, api.Tool{Type: "function", Function: faithful.ToolFunction()})
			continue
		}
		definitions = append(definitions, api.Tool{Type: "function", Function: toolFunctionFromSchema(tool)})
	}
	return definitions
}

// toolFunctionFromSchema derives a definition from a tool's schema map. It
// carries each property's type and description and the required list, which is
// all the map form has ever expressed.
func toolFunctionFromSchema(tool Tool) api.ToolFunction {
	fn := api.ToolFunction{Name: tool.Name(), Description: tool.Description()}
	fn.Parameters.Type = "object"
	fn.Parameters.Required = []string{}
	fn.Parameters.Properties = api.NewToolPropertiesMap()

	schema := tool.Schema()
	if schema == nil {
		return fn
	}
	if declared, ok := schema["type"].(string); ok && declared != "" {
		fn.Parameters.Type = declared
	}
	if properties, ok := schema["properties"].(map[string]any); ok {
		for _, name := range sortedKeys(properties) {
			definition, ok := properties[name].(map[string]any)
			if !ok {
				continue
			}
			property := api.ToolProperty{}
			if declared, ok := definition["type"].(string); ok && declared != "" {
				property.Type = api.PropertyType{declared}
			}
			if description, ok := definition["description"].(string); ok {
				property.Description = description
			}
			fn.Parameters.Properties.Set(name, property)
		}
	}
	switch required := schema["required"].(type) {
	case []string:
		fn.Parameters.Required = required
	case []any:
		list := make([]string, 0, len(required))
		for _, entry := range required {
			if name, ok := entry.(string); ok {
				list = append(list, name)
			}
		}
		fn.Parameters.Required = list
	}
	return fn
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// ToolNames returns a list of all tool names
func (r *Registry) ToolNames() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}
