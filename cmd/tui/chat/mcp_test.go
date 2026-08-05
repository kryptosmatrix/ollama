package chat

import (
	"context"
	"errors"
	"strings"
	"testing"

	coreagent "github.com/ollama/ollama/agent"
	"github.com/ollama/ollama/mcp"
)

func submitSlash(t *testing.T, m chatModel) chatModel {
	t.Helper()
	updated, cmd := m.handleSubmit()
	if cmd != nil {
		t.Fatal("a slash command should not start a model run")
	}
	model, ok := updated.(chatModel)
	if !ok {
		t.Fatalf("handleSubmit returned %T", updated)
	}
	return model
}

func lastEntry(t *testing.T, m chatModel) string {
	t.Helper()
	if len(m.entries) == 0 {
		t.Fatal("no entries were produced")
	}
	return m.entries[len(m.entries)-1].content
}

func TestMCPSlashCommandIsOffered(t *testing.T) {
	var found bool
	for _, command := range chatSlashCommands {
		if command.name == "/mcp" {
			found = true
			if !strings.Contains(command.usage, "enable") || !strings.Contains(command.usage, "disable") {
				t.Errorf("usage = %q, want it to mention enable and disable", command.usage)
			}
		}
	}
	if !found {
		t.Fatal("/mcp is not in the slash command list, so nothing surfaces it to the user")
	}
}

func TestMCPListsServersWithStatusAndSkippedTools(t *testing.T) {
	states := []mcp.ServerState{
		{
			Name:    "files",
			Spec:    &mcp.ServerSpec{Command: "uvx", Args: []string{"mcp-server-files"}},
			Status:  mcp.StatusConnected,
			Tools:   []mcp.Tool{{Server: "files", Name: "read"}, {Server: "files", Name: "write"}},
			Skipped: []mcp.SkippedTool{{Name: "bash", Reason: "is reserved by Ollama"}},
		},
		{
			Name:   "hosted",
			Spec:   &mcp.ServerSpec{URL: "https://mcp.example.com/v1"},
			Status: mcp.StatusNeedsApproval,
		},
		{
			Name:   "broken",
			Spec:   &mcp.ServerSpec{Command: "missing"},
			Status: mcp.StatusFailed,
			Err:    errors.New("command not found"),
		},
	}

	m := chatModel{
		opts:  Options{MCPServers: func() []mcp.ServerState { return states }},
		input: []rune("/mcp"),
	}
	content := lastEntry(t, submitSlash(t, m))

	for _, want := range []string{
		"files", "connected, 2 tools", "uvx mcp-server-files",
		"skipped tool", "is reserved by Ollama",
		"hosted", "https://mcp.example.com/v1",
		"broken", "command not found",
		"ollama mcp approve",
		"/mcp enable",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("/mcp output should contain %q, got:\n%s", want, content)
		}
	}
}

func TestMCPWithNothingConfigured(t *testing.T) {
	m := chatModel{
		opts:  Options{MCPServers: func() []mcp.ServerState { return nil }},
		input: []rune("/mcp"),
	}
	content := lastEntry(t, submitSlash(t, m))
	if !strings.Contains(content, "ollama mcp add") {
		t.Errorf("an empty list should say how to add a server, got:\n%s", content)
	}
}

func TestMCPWhenUnavailable(t *testing.T) {
	m := chatModel{opts: Options{}, input: []rune("/mcp")}
	content := lastEntry(t, submitSlash(t, m))
	if !strings.Contains(content, "not available") {
		t.Errorf("with no MCP wiring the command should say so, got:\n%s", content)
	}
}

func TestMCPEnableAppliesTheChangeAndRebuildsTheToolList(t *testing.T) {
	registry := &coreagent.Registry{}

	var gotName string
	var gotEnabled bool
	var rebuilt, reprompted bool

	m := chatModel{
		opts: Options{
			Model: "test-model",
			SetMCPEnabled: func(_ context.Context, name string, enabled bool) error {
				gotName, gotEnabled = name, enabled
				return nil
			},
			ToolRegistryForModel: func(context.Context, string) *coreagent.Registry {
				rebuilt = true
				return registry
			},
			SystemPromptForModel: func(_ context.Context, _ string, got *coreagent.Registry, _ bool) string {
				reprompted = got == registry
				return "system"
			},
		},
		input: []rune("/mcp enable files"),
	}

	updated := submitSlash(t, m)

	if gotName != "files" || !gotEnabled {
		t.Errorf("SetMCPEnabled called with (%q, %v), want (files, true)", gotName, gotEnabled)
	}
	if !rebuilt {
		t.Error("the tool registry must be rebuilt, or the model keeps offering the old tool list")
	}
	if !reprompted {
		t.Error("the system prompt must be rebuilt from the new registry")
	}
	if updated.opts.Tools != registry {
		t.Error("the rebuilt registry was not adopted")
	}
	if !strings.Contains(lastEntry(t, updated), "Enabled files") {
		t.Errorf("the user should be told, got:\n%s", lastEntry(t, updated))
	}
}

func TestMCPDisablePassesFalse(t *testing.T) {
	var gotEnabled = true
	m := chatModel{
		opts: Options{
			Model: "test-model",
			SetMCPEnabled: func(_ context.Context, _ string, enabled bool) error {
				gotEnabled = enabled
				return nil
			},
		},
		input: []rune("/mcp disable files"),
	}
	updated := submitSlash(t, m)
	if gotEnabled {
		t.Error("/mcp disable must switch the server off")
	}
	if !strings.Contains(lastEntry(t, updated), "Disabled files") {
		t.Errorf("got:\n%s", lastEntry(t, updated))
	}
}

func TestMCPEnableReportsFailureAndDoesNotRebuild(t *testing.T) {
	var rebuilt bool
	m := chatModel{
		opts: Options{
			Model: "test-model",
			SetMCPEnabled: func(context.Context, string, bool) error {
				return errors.New("is not approved to run; approve it with: ollama mcp approve files")
			},
			ToolRegistryForModel: func(context.Context, string) *coreagent.Registry {
				rebuilt = true
				return &coreagent.Registry{}
			},
		},
		input: []rune("/mcp enable files"),
	}

	updated := submitSlash(t, m)
	if rebuilt {
		t.Error("a failed change must not rebuild the tool list; the model would be told about tools that are not connected")
	}
	content := lastEntry(t, updated)
	if !strings.Contains(content, "ollama mcp approve files") {
		t.Errorf("the reason should reach the user, got:\n%s", content)
	}
	if updated.status != "error" {
		t.Errorf("status = %q, want error", updated.status)
	}
}

func TestMCPRejectsMalformedArguments(t *testing.T) {
	for _, input := range []string{"/mcp enable", "/mcp toggle files", "/mcp enable a b", "/mcp disable"} {
		t.Run(input, func(t *testing.T) {
			called := false
			m := chatModel{
				opts: Options{
					MCPServers:    func() []mcp.ServerState { return nil },
					SetMCPEnabled: func(context.Context, string, bool) error { called = true; return nil },
				},
				input: []rune(input),
			}
			updated := submitSlash(t, m)
			if called {
				t.Fatalf("%q should not have changed anything", input)
			}
			if !strings.Contains(lastEntry(t, updated), "usage: /mcp") {
				t.Errorf("got:\n%s", lastEntry(t, updated))
			}
		})
	}
}

// TestMCPCannotApprove is a guard on a deliberate design decision, not on an
// implementation detail. Approving a server means agreeing to a particular
// command line, which has to be shown verbatim and answered deliberately. A
// chat input line is the wrong place for that, so /mcp must never grow an
// approve verb — it points at `ollama mcp approve` instead.
func TestMCPCannotApprove(t *testing.T) {
	called := false
	m := chatModel{
		opts: Options{
			MCPServers:    func() []mcp.ServerState { return nil },
			SetMCPEnabled: func(context.Context, string, bool) error { called = true; return nil },
		},
		input: []rune("/mcp approve files"),
	}
	updated := submitSlash(t, m)
	if called {
		t.Fatal("/mcp must not be able to approve a server")
	}
	if !strings.Contains(lastEntry(t, updated), "usage: /mcp") {
		t.Errorf("got:\n%s", lastEntry(t, updated))
	}
}
