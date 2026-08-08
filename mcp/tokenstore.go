package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// TokensPathEnv overrides the location of the token store.
const TokensPathEnv = "OLLAMA_MCP_TOKENS"

const tokensFilename = "mcp-tokens.json"

// ErrNoToken reports that a server has no stored token. It is distinct from a
// failure to read the store, because "you are not signed in" and "your
// credentials could not be read" call for different responses from the caller.
var ErrNoToken = errors.New("no stored token for that MCP server")

// SignInRecord is what Ollama keeps for a signed-in server.
//
// It is more than the token because signing out is more than forgetting. A
// token is revoked at the authorization server by a client that identifies
// itself, and the identifier Ollama was issued is knowable only at sign-in:
// dynamic client registration issues a fresh one each time, so registering
// again at sign-out would identify a different client and revoke nothing. A
// record stored without it can be forgotten but never revoked.
type SignInRecord struct {
	// Token is the credential itself.
	Token *oauth2.Token
	// ClientID is the identifier this Ollama installation was issued when it
	// registered with the authorization server. It may be empty for a server
	// that issued a token without dynamic registration.
	ClientID string
	// Issuer is the authorization server that issued this token.
	//
	// A protected resource may name several. Revoking at the wrong one is worse
	// than not revoking at all: RFC 7009 has a server answer 200 for a token it
	// has never heard of, so the user is told they are signed out while the
	// token stays live at the server that actually issued it. Empty for a
	// record written before this was kept.
	Issuer string
}

// TokenStore keeps the OAuth tokens for remote MCP servers.
//
// It is an interface because where a token lives is a platform question, and
// because a caller must be able to say which one it is using — a store that
// could silently be either the operating system's keychain or a file would
// leave a user unable to know how their credentials are protected.
type TokenStore interface {
	// Load returns the stored sign-in, or ErrNoToken.
	Load(server string) (*SignInRecord, error)
	// Save stores the sign-in, replacing any previous one.
	Save(server string, record *SignInRecord) error
	// Delete removes the token. Deleting an absent token is not an error: the
	// caller's intent — that no token remain — is satisfied either way.
	Delete(server string) error
	// Description says in one line where tokens are kept and how well they are
	// protected. It is surfaced to the user rather than kept for logs: someone
	// signing in to a third-party service is entitled to know where the
	// credential ends up.
	Description() string
}

// fileStoreLocks serialises the read-modify-write that Save and Delete perform
// over the whole file.
//
// Keyed by resolved path rather than held on the store, because nothing stops a
// process from building two FileTokenStore values for the same file — several
// places do exactly that — and a lock on one of them would not see the other.
//
// This closes the window inside one process. It does not close it between
// processes: the desktop app and a terminal saving tokens for different servers
// at the same instant can still lose one, and fixing that needs a lock file
// with all the staleness that implies. Measured before it was written: eight
// concurrent saves for eight servers left one token on disk.
var fileStoreLocks sync.Map

func lockTokenFile(path string) func() {
	value, _ := fileStoreLocks.LoadOrStore(path, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// FileTokenStore keeps tokens in a JSON file in the user's configuration
// directory, protected by file permissions alone.
//
// It is named for what it is. This is **weaker than the operating system's
// keychain**: any process running as this user can read the file, whereas a
// keychain item can be scoped to an application and can require the user to
// unlock it. It exists because the obvious way to reach the macOS keychain
// without cgo — shelling out to the `security` command — passes the secret as
// a command-line argument, where any local user can read it out of the process
// list. That is worse than a 0600 file, not better.
//
// A Keychain- and DPAPI-backed store is owed, and needs cgo to be done safely.
type FileTokenStore struct {
	// Path is the store's location. Empty means the resolved default.
	Path string
}

// TokensPath returns the token store's location, resolved like ConfigPath.
func TokensPath() (string, error) {
	if path := strings.TrimSpace(os.Getenv(TokensPathEnv)); path != "" {
		return filepath.Abs(path)
	}
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return filepath.Join(xdg, "ollama", tokensFilename), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ollama", tokensFilename), nil
}

func (s *FileTokenStore) path() (string, error) {
	if strings.TrimSpace(s.Path) != "" {
		return s.Path, nil
	}
	return TokensPath()
}

// Description tells the user where their tokens are and what protects them.
func (s *FileTokenStore) Description() string {
	path, err := s.path()
	if err != nil {
		return "a file in your Ollama configuration directory, readable by any program running as you"
	}
	return path + ", readable by any program running as you"
}

// storedToken is the persisted shape. oauth2.Token is serialised field by field
// rather than embedded, so a change to that type cannot silently alter what is
// written to disk or what an older Ollama can read back.
type storedToken struct {
	AccessToken  string    `json:"accessToken"`
	TokenType    string    `json:"tokenType,omitempty"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
	ClientID     string    `json:"clientId,omitempty"`
	Issuer       string    `json:"issuer,omitempty"`
}

type tokenFile struct {
	Tokens map[string]storedToken `json:"tokens"`
}

func (s *FileTokenStore) read() (*tokenFile, string, error) {
	path, err := s.path()
	if err != nil {
		return nil, "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &tokenFile{Tokens: map[string]storedToken{}}, path, nil
		}
		return nil, path, fmt.Errorf("read mcp tokens %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return &tokenFile{Tokens: map[string]storedToken{}}, path, nil
	}

	var file tokenFile
	// A store that cannot be parsed is an error rather than an empty store.
	// Treating it as empty would silently sign the user out of everything and
	// then overwrite whatever was there.
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, path, fmt.Errorf("parse mcp tokens %s: %w", path, err)
	}
	if file.Tokens == nil {
		file.Tokens = map[string]storedToken{}
	}
	return &file, path, nil
}

func (s *FileTokenStore) write(file *tokenFile, path string) error {
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal mcp tokens: %w", err)
	}
	// Same private write as the configuration: 0600 in a 0700 directory,
	// temp file and rename, so a crash cannot leave a half-written store.
	return writeFilePrivate(path, append(data, '\n'))
}

// Load returns the stored sign-in for a server.
func (s *FileTokenStore) Load(server string) (*SignInRecord, error) {
	file, _, err := s.read()
	if err != nil {
		return nil, err
	}
	stored, ok := file.Tokens[server]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoToken, server)
	}
	return &SignInRecord{
		Token: &oauth2.Token{
			AccessToken:  stored.AccessToken,
			TokenType:    stored.TokenType,
			RefreshToken: stored.RefreshToken,
			Expiry:       stored.Expiry,
		},
		ClientID: stored.ClientID,
		Issuer:   stored.Issuer,
	}, nil
}

// Save stores a server's sign-in.
func (s *FileTokenStore) Save(server string, record *SignInRecord) error {
	if strings.TrimSpace(server) == "" {
		return errors.New("a token needs a server name")
	}
	if record == nil || record.Token == nil || record.Token.AccessToken == "" {
		return errors.New("refusing to store an empty token")
	}

	resolved, err := s.path()
	if err != nil {
		return err
	}
	defer lockTokenFile(resolved)()

	file, path, err := s.read()
	if err != nil {
		return err
	}
	// A refresh that arrives without a client identifier must not erase the one
	// recorded at sign-in: losing it makes the sign-in unrevocable.
	clientID := record.ClientID
	if clientID == "" {
		clientID = file.Tokens[server].ClientID
	}
	// The issuer is learned at sign-in for the same reason and kept the same
	// way: a refresh that arrives without one must not erase it.
	issuer := record.Issuer
	if issuer == "" {
		issuer = file.Tokens[server].Issuer
	}
	file.Tokens[server] = storedToken{
		AccessToken:  record.Token.AccessToken,
		TokenType:    record.Token.TokenType,
		RefreshToken: record.Token.RefreshToken,
		Expiry:       record.Token.Expiry,
		ClientID:     clientID,
		Issuer:       issuer,
	}
	return s.write(file, path)
}

// Delete removes a server's token.
func (s *FileTokenStore) Delete(server string) error {
	resolved, err := s.path()
	if err != nil {
		return err
	}
	defer lockTokenFile(resolved)()

	file, path, err := s.read()
	if err != nil {
		return err
	}
	if _, ok := file.Tokens[server]; !ok {
		return nil
	}
	delete(file.Tokens, server)
	return s.write(file, path)
}

// Servers lists the servers with a stored token, in a stable order.
func (s *FileTokenStore) Servers() ([]string, error) {
	file, _, err := s.read()
	if err != nil {
		return nil, err
	}
	return slices.Sorted(maps.Keys(file.Tokens)), nil
}
