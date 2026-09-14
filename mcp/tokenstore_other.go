//go:build (!darwin || !cgo) && !windows

package mcp

// DefaultTokenStore returns the best place this build can keep tokens.
//
// On this platform that is a file protected by its permissions alone, at
// OLLAMA_MCP_TOKENS if it is set. macOS keeps tokens in the keychain and
// Windows encrypts the same file with DPAPI, both of which the operating system
// provides with no daemon and no interaction.
//
// Linux has no equivalent, and the Secret Service is not one. It is a desktop
// service reached over a session bus, and Ollama's own installer creates a
// system account — "useradd -r -s /bin/false" in scripts/install.sh — that has
// no login session, no session bus and no keyring to unlock. On that
// configuration a Secret Service store would fall back to this file every time
// while carrying a new dependency for the privilege.
//
// It would help a desktop user running the CLI as themselves. Against that:
// unlocking a keyring is inherently interactive, and a token save during a
// background reconnect would then be able to raise a prompt with nobody at the
// keyboard — the failure the Windows store deliberately designs out with
// CRYPTPROTECT_UI_FORBIDDEN, and one this code should not introduce on another
// platform.
//
// So the file says what it is rather than a keyring path pretending to protect
// machines it cannot reach, and every surface that offers a sign-in shows
// TokenStore.Description() before a token exists. Building it anyway is a
// decision for the maintainers, and the tradeoff is recorded here rather than
// left to be rediscovered.
func DefaultTokenStore() TokenStore {
	return &FileTokenStore{}
}
