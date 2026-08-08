//go:build (!darwin || !cgo) && !windows

package mcp

// DefaultTokenStore returns the best place this build can keep tokens.
//
// On this platform that is a file protected by its permissions alone, at
// OLLAMA_MCP_TOKENS if it is set. macOS uses the keychain and Windows encrypts
// the same file with DPAPI; Linux has no equivalent that can be relied on,
// because the Secret Service is a desktop service and a great many Ollama
// installs are headless. Rather than a path that silently does nothing on the
// machines most likely to run this, the store says what it is — and every
// surface that offers a sign-in shows TokenStore.Description() before a token
// exists.
func DefaultTokenStore() TokenStore {
	return &FileTokenStore{}
}
