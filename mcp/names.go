package mcp

import (
	"fmt"
	"regexp"
	"strings"
)

// NameSeparator joins a server name to a tool name in the namespaced tool name
// the model sees. Server names may not contain it (see config.go), so the first
// occurrence always divides the two halves.
const NameSeparator = "__"

// toolName constrains the raw tool names Ollama will accept from a server.
// Dots are permitted because Ollama's own tools already use them
// (browser.search, browser.open).
var toolName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)

// reservedToolNames are the tool names Ollama's own tools occupy across both
// tool stacks. No MCP tool may claim one.
//
// Namespacing already makes a collision impossible — every MCP tool carries a
// "<server>__" prefix and no first-party name contains the separator — so this
// is a second, cheap barrier rather than the primary defence. If a first-party
// tool is ever renamed to something containing the separator, this list is what
// stops the rename from silently handing a server the ability to impersonate it.
var reservedToolNames = map[string]bool{
	"bash":           true,
	"read":           true,
	"edit":           true,
	"write":          true,
	"skill":          true,
	"web_search":     true,
	"web_fetch":      true,
	"browser.search": true,
	"browser.open":   true,
	"browser.find":   true,
}

// QualifyName returns the name a model sees for a tool advertised by a server.
func QualifyName(server, tool string) string {
	return server + NameSeparator + tool
}

// SplitQualifiedName divides a namespaced tool name back into its server and
// tool halves. It reports false for a name that carries no separator, which is
// how a first-party tool is distinguished from an MCP one.
func SplitQualifiedName(qualified string) (server, tool string, ok bool) {
	server, tool, ok = strings.Cut(qualified, NameSeparator)
	if !ok || server == "" || tool == "" {
		return "", "", false
	}
	return server, tool, true
}

// validateToolName reports why a tool advertised by a server cannot be offered
// to a model, or nil when it can.
func validateToolName(server, tool string) error {
	if !toolName.MatchString(tool) {
		return fmt.Errorf("tool name %q from server %q must be 1-128 characters of letters, digits, underscore, dot or hyphen and start with a letter or digit", tool, server)
	}
	if reservedToolNames[strings.ToLower(tool)] {
		return fmt.Errorf("tool name %q from server %q is reserved by Ollama", tool, server)
	}
	if qualified := QualifyName(server, tool); reservedToolNames[strings.ToLower(qualified)] {
		return fmt.Errorf("namespaced tool name %q from server %q is reserved by Ollama", qualified, server)
	}
	return nil
}
