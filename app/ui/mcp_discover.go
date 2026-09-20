//go:build windows || darwin

package ui

import (
	"encoding/json"
	"net/http"

	"github.com/ollama/ollama/app/ui/responses"
	"github.com/ollama/ollama/mcp"
)

// discoverMCPServers lists the MCP servers other applications on this machine
// have been configured with.
//
// It reads named configuration files and nothing else. Loading this page does
// not contact anything, which is why it is a GET while the loopback probe —
// which does — is not.
func (s *Server) discoverMCPServers(w http.ResponseWriter, r *http.Request) error {
	found, searched := mcp.DiscoverConfigured(s.discoveryHome)

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(responses.MCPDiscoveryResponse{
		Servers:  describeDiscovered(found),
		Searched: searched,
	})
}

// probeMCPServers looks for MCP servers answering on this machine right now.
//
// It is a POST because it is an act rather than a reading: it contacts every
// listening loopback port with the MCP handshake. Nothing calls it except a
// user asking for it.
func (s *Server) probeMCPServers(w http.ResponseWriter, r *http.Request) error {
	body := responses.MCPDiscoveryResponse{Servers: []responses.MCPDiscoveredServer{}}

	found, err := mcp.ProbeLoopback(r.Context())
	if err != nil {
		// The probe failing is not the request failing. Saying why in the body
		// lets the page show the reason beside an otherwise empty list, rather
		// than an error that replaces everything.
		body.Error = err.Error()
	} else {
		body.Servers = describeDiscovered(found)
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(body)
}

func describeDiscovered(found []mcp.DiscoveredServer) []responses.MCPDiscoveredServer {
	described := make([]responses.MCPDiscoveredServer, 0, len(found))
	for _, server := range found {
		one := responses.MCPDiscoveredServer{
			Name:    server.Name,
			Sources: server.Sources,
			Paths:   server.Paths,
			Runs:    server.Runs,
			Origin:  server.Origin,
			Notes:   server.Notes,
			Problem: server.Problem,
		}
		if server.Spec != nil {
			one.Command = server.Spec.Command
			one.Args = server.Spec.Args
			one.Env = server.Spec.Env
			one.URL = server.Spec.URL
			one.Headers = server.Spec.Headers
			one.Disabled = server.Spec.Disabled
		}
		described = append(described, one)
	}
	return described
}
