//go:build windows || darwin

package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ollama/ollama/app/ui/responses"
	"github.com/ollama/ollama/mcp"
)

// mcpState loads both files the MCP surface works with. They are read together
// so a request can never take the configuration from one place and the
// approvals from another.
func mcpState() (*mcp.Config, *mcp.Approvals, string, string, error) {
	configPath, err := mcp.ConfigPath()
	if err != nil {
		return nil, nil, "", "", err
	}
	approvalsPath, err := mcp.ApprovalsPath()
	if err != nil {
		return nil, nil, "", "", err
	}
	cfg, err := mcp.Load(configPath)
	if err != nil {
		return nil, nil, "", "", err
	}
	approvals, err := mcp.LoadApprovals(approvalsPath)
	if err != nil {
		return nil, nil, "", "", err
	}
	return cfg, approvals, configPath, approvalsPath, nil
}

// listMCPServers describes every configured server: what it would run, whether
// Ollama may run it, and what it is currently offering.
//
// It does not start or contact anything. The live half comes from the manager
// that is already running; a page load must never be able to launch a server.
func (s *Server) listMCPServers(w http.ResponseWriter, _ *http.Request) error {
	cfg, approvals, _, _, err := mcpState()
	if err != nil {
		return err
	}
	problems := cfg.Problems()

	live := map[string]mcp.ServerState{}
	if s.MCP != nil {
		for _, state := range s.MCP.States() {
			live[state.Name] = state
		}
	}

	servers := make([]responses.MCPServer, 0, len(cfg.Names()))
	for _, name := range cfg.Names() {
		spec, _ := cfg.Get(name)
		servers = append(servers, describeMCPServer(name, spec, problems[name], approvals, live[name]))
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(responses.MCPServersResponse{Servers: servers})
}

// describeMCPServer merges the three things a user needs to see about a server:
// what the configuration says, whether it has been approved, and what the live
// connection is doing.
func describeMCPServer(name string, spec *mcp.ServerSpec, problem error, approvals *mcp.Approvals, state mcp.ServerState) responses.MCPServer {
	server := responses.MCPServer{
		Name:      name,
		Runs:      spec.Summary(),
		Transport: string(spec.Type),
		Enabled:   !spec.Disabled,
		Approved:  approvals.Allows(spec),
	}
	if server.Transport == "" {
		if spec.Command != "" {
			server.Transport = string(mcp.TransportStdio)
		} else if spec.URL != "" {
			server.Transport = string(mcp.TransportHTTP)
		}
	}

	// Approved once, but not for what it now says it would run. This is the
	// case the ledger exists to reveal, so it is surfaced as its own flag
	// rather than folded into "not approved".
	if !server.Approved && approvals.Entries[name].Fingerprint != "" {
		server.Changed = true
		server.PreviouslyRan = approvals.Entries[name].Summary
	}

	switch {
	case problem != nil:
		server.Status = string(mcp.StatusInvalid)
		server.Error = problem.Error()
	case spec.Disabled:
		server.Status = string(mcp.StatusDisabled)
	case !server.Approved:
		server.Status = string(mcp.StatusNeedsApproval)
	case state.Name != "":
		server.Status = string(state.Status)
		if state.Err != nil {
			server.Error = state.Err.Error()
		}
	default:
		// Configured, enabled and approved, but this process has no connection
		// for it — it was added since the app started.
		server.Status = string(mcp.StatusNeedsApproval)
		server.Error = "restart Ollama to connect this server"
	}

	for _, tool := range state.Tools {
		server.Tools = append(server.Tools, responses.MCPTool{
			Name:        tool.QualifiedName(),
			Description: tool.Description,
		})
	}
	for _, skipped := range state.Skipped {
		server.Skipped = append(server.Skipped, responses.MCPSkippedTool{
			Name:   skipped.Name,
			Reason: skipped.Reason,
		})
	}
	return server
}

// addMCPServer writes a new server to the configuration. It does not approve
// it: the command line came over HTTP rather than from the user's keyboard, so
// it must be shown and agreed to before Ollama will run it.
func (s *Server) addMCPServer(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Name    string            `json:"name"`
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("invalid request body: %w", err)
	}
	if body.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return errors.New("name is required")
	}

	cfg, approvals, configPath, _, err := mcpState()
	if err != nil {
		return err
	}
	if _, exists := cfg.Get(body.Name); exists {
		w.WriteHeader(http.StatusConflict)
		return fmt.Errorf("an MCP server named %q already exists", body.Name)
	}

	spec := &mcp.ServerSpec{
		Command: body.Command,
		Args:    body.Args,
		Env:     body.Env,
		URL:     body.URL,
		Headers: body.Headers,
	}
	switch {
	case body.URL != "":
		spec.Type = mcp.TransportHTTP
	case body.Command != "":
		spec.Type = mcp.TransportStdio
	}

	cfg.Set(body.Name, spec)
	// Validate through the same path the runtime uses, so a server that could
	// never start is refused here rather than at the next launch.
	if problem := cfg.Problems()[body.Name]; problem != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("%s: %w", body.Name, problem)
	}
	if err := cfg.Save(configPath); err != nil {
		return err
	}

	stored, _ := cfg.Get(body.Name)
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(describeMCPServer(body.Name, stored, nil, approvals, mcp.ServerState{}))
}

// updateMCPServer switches a server on or off and applies the change, which
// includes stopping a server that was switched off.
func (s *Server) updateMCPServer(w http.ResponseWriter, r *http.Request) error {
	name := r.PathValue("name")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return errors.New("server name is required")
	}

	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("invalid request body: %w", err)
	}
	if body.Enabled == nil {
		w.WriteHeader(http.StatusBadRequest)
		return errors.New("enabled is required")
	}

	cfg, approvals, configPath, _, err := mcpState()
	if err != nil {
		return err
	}
	spec, ok := cfg.Get(name)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return fmt.Errorf("no MCP server named %q", name)
	}

	spec.Disabled = !*body.Enabled
	if err := cfg.Save(configPath); err != nil {
		return err
	}
	s.applyMCPConfig(r, cfg)

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(describeMCPServer(name, spec, cfg.Problems()[name], approvals, s.mcpStateFor(name)))
}

// approveMCPServer records agreement to run a server exactly as it currently
// stands, and connects it.
//
// The page is allowed to approve where the chat slash command is not, because
// a page can show the resolved command line verbatim beside the button. The
// caller must send back the command line it displayed, and it must match what
// is on disk: that is what stops a user approving one thing while looking at
// another, whether from a stale page or a configuration edited underneath them.
func (s *Server) approveMCPServer(w http.ResponseWriter, r *http.Request) error {
	name := r.PathValue("name")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return errors.New("server name is required")
	}

	var body struct {
		Runs string `json:"runs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("invalid request body: %w", err)
	}

	cfg, approvals, _, approvalsPath, err := mcpState()
	if err != nil {
		return err
	}
	spec, ok := cfg.Get(name)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return fmt.Errorf("no MCP server named %q", name)
	}
	if problem := cfg.Problems()[name]; problem != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("%s cannot run as configured: %w", name, problem)
	}
	if body.Runs != spec.Summary() {
		w.WriteHeader(http.StatusConflict)
		return fmt.Errorf("%s now runs %q, not what you were shown; reload and read it again", name, spec.Summary())
	}

	approvals.Approve(spec, time.Now())
	if err := approvals.Save(approvalsPath); err != nil {
		return err
	}
	s.applyMCPConfig(r, cfg)

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(describeMCPServer(name, spec, nil, approvals, s.mcpStateFor(name)))
}

// deleteMCPServer removes a server and its approval. Leaving the approval
// behind would silently re-approve a future server that reused the name and the
// command line.
func (s *Server) deleteMCPServer(w http.ResponseWriter, r *http.Request) error {
	name := r.PathValue("name")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return errors.New("server name is required")
	}

	cfg, approvals, configPath, approvalsPath, err := mcpState()
	if err != nil {
		return err
	}
	if !cfg.Remove(name) {
		w.WriteHeader(http.StatusNotFound)
		return fmt.Errorf("no MCP server named %q", name)
	}
	if err := cfg.Save(configPath); err != nil {
		return err
	}
	if approvals.Revoke(name) {
		if err := approvals.Save(approvalsPath); err != nil {
			return err
		}
	}
	s.applyMCPConfig(r, cfg)

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// applyMCPConfig brings the running manager into line with the configuration on
// disk. The file is the source of truth; the manager follows it.
func (s *Server) applyMCPConfig(r *http.Request, cfg *mcp.Config) {
	if s.MCP == nil {
		return
	}
	s.MCP.Connect(r.Context(), cfg)
}

func (s *Server) mcpStateFor(name string) mcp.ServerState {
	if s.MCP == nil {
		return mcp.ServerState{}
	}
	state, _ := s.MCP.State(name)
	return state
}
