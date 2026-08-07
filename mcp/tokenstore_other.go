//go:build !darwin || !cgo

package mcp

// DefaultTokenStore returns the best place this build can keep tokens.
//
// On this platform that is a file protected by its permissions alone, at
// OLLAMA_MCP_TOKENS if it is set. A Windows build should keep them behind DPAPI
// and a Linux one behind the desktop secret service; neither is written yet,
// and until one is, the store says what it is rather than implying otherwise —
// every surface that offers a sign-in shows TokenStore.Description() before a
// token exists.
func DefaultTokenStore() TokenStore {
	return &FileTokenStore{}
}
