//go:build windows || darwin

package ui

import (
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/ollama/ollama/app/ui/responses"
	"github.com/ollama/ollama/mcp"
)

// browseMCPRegistry searches the official MCP Registry.
//
// Every entry is resolved here rather than in the browser, so the command line
// shown beside an install button is produced by the same code that would write
// it. An entry Ollama cannot run is returned with the reason instead of being
// hidden, because a user searching for a server they know exists deserves to be
// told why it is not offered.
func (s *Server) browseMCPRegistry(w http.ResponseWriter, r *http.Request) error {
	client := mcp.NewRegistryClient(os.Getenv(mcp.RegistryURLEnv))
	page, err := client.Search(r.Context(), r.URL.Query().Get("search"), r.URL.Query().Get("cursor"))
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return err
	}

	entries := make([]responses.MCPRegistryEntry, 0, len(page.Servers))
	for _, server := range page.Servers {
		entries = append(entries, describeRegistryEntry(server))
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(responses.MCPRegistryResponse{
		Entries:    entries,
		NextCursor: page.NextCursor,
		NotVetted:  true,
	})
}

func describeRegistryEntry(entry mcp.RegistryEntry) responses.MCPRegistryEntry {
	described := responses.MCPRegistryEntry{
		Name:          entry.Name,
		Title:         entry.Title,
		Description:   entry.Description,
		Version:       entry.Version,
		Publisher:     entry.Publisher(),
		WebsiteURL:    entry.WebsiteURL,
		SuggestedName: suggestedServerName(entry.Name),
	}
	if entry.Repository != nil {
		described.Repository = entry.Repository.URL
	}

	spec, err := entry.Resolve()
	if err != nil {
		described.Reason = err.Error()
		return described
	}

	described.Installable = true
	described.Runs = spec.Summary()
	described.Transport = string(spec.Type)

	names := slices.Sorted(maps.Keys(spec.Env))
	names = append(names, slices.Sorted(maps.Keys(spec.Headers))...)
	for _, name := range names {
		value := spec.Env[name]
		if value == "" {
			value = spec.Headers[name]
		}
		described.Variables = append(described.Variables, value)
	}
	return described
}

var unsafeNameChars = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

// suggestedServerName turns a reverse-DNS registry name into something that
// passes the configuration layer's own name rules, so the browse surface never
// proposes a name that would be refused on add.
func suggestedServerName(name string) string {
	_, tail, found := strings.Cut(name, "/")
	if !found || strings.TrimSpace(tail) == "" {
		tail = name
	}
	cleaned := strings.Trim(unsafeNameChars.ReplaceAllString(tail, "-"), "-")
	// The separator between a server and its tool may not appear in a name.
	for strings.Contains(cleaned, "__") {
		cleaned = strings.ReplaceAll(cleaned, "__", "_")
	}
	if len(cleaned) > 64 {
		cleaned = strings.Trim(cleaned[:64], "-")
	}
	if cleaned == "" {
		return "server"
	}
	return cleaned
}

// resolveMCPRegistryEntry returns what installing one entry would write,
// without writing it.
//
// It exists so the interface can show the command line at the moment of the
// decision, from the server rather than from a list that may be minutes old.
func (s *Server) resolveMCPRegistryEntry(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}
	if strings.TrimSpace(body.Name) == "" {
		w.WriteHeader(http.StatusBadRequest)
		return errors.New("name is required")
	}

	client := mcp.NewRegistryClient(os.Getenv(mcp.RegistryURLEnv))
	page, err := client.Search(r.Context(), body.Name, "")
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return err
	}

	for _, entry := range page.Servers {
		if entry.Name != body.Name {
			continue
		}
		w.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(w).Encode(describeRegistryEntry(entry))
	}

	w.WriteHeader(http.StatusNotFound)
	return errors.New("that server is no longer in the registry")
}
