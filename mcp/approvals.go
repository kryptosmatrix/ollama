package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// ApprovalsPathEnv overrides the location of the approval ledger.
const ApprovalsPathEnv = "OLLAMA_MCP_APPROVALS"

const approvalsFilename = "mcp-approvals.json"

// ApprovalPolicy decides whether a configured server may be connected.
//
// It exists because "the user switched this server on" and "the user agreed to
// run this particular command" are different questions. The configuration file
// answers the first; only the ledger answers the second.
type ApprovalPolicy interface {
	// Allows reports whether this exact spec has been approved. An
	// implementation must key on what the spec would execute, not on its name:
	// a name that has been approved once must not launder a command that has
	// been changed since.
	Allows(spec *ServerSpec) bool
}

// allowAll approves everything. It is unexported so it cannot be reached from
// outside this package: a production caller must supply a real ledger, and the
// only legitimate users of a blanket policy are this package's own tests, which
// exercise the manager's mechanics rather than the policy.
type allowAll struct{}

func (allowAll) Allows(*ServerSpec) bool { return true }

// approvalsFile is an ApprovalPolicy that reads the ledger from disk on every
// question rather than holding a snapshot.
//
// This matters because approvals change while Ollama is running: a user
// approves a server from the app or from another terminal and expects it to
// start. A manager built with a snapshot would answer from the ledger as it
// stood at launch and refuse for ever, which looks exactly like the approval
// not having been recorded.
//
// A ledger that cannot be read denies, which is the safe direction.
type approvalsFile struct {
	path   string
	logger *slog.Logger
}

// ApprovalsFile returns a policy backed by the ledger at path, re-read on each
// question. Pass it to Options.Approvals when approvals may change while the
// process is running, which is every production caller.
func ApprovalsFile(path string, logger *slog.Logger) ApprovalPolicy {
	if logger == nil {
		logger = slog.Default()
	}
	return &approvalsFile{path: path, logger: logger}
}

func (a *approvalsFile) Allows(spec *ServerSpec) bool {
	ledger, err := LoadApprovals(a.path)
	if err != nil {
		a.logger.Warn("could not read mcp approvals; refusing to run any server", "path", a.path, "error", err)
		return false
	}
	return ledger.Allows(spec)
}

// Approval is one recorded agreement to run a particular server.
type Approval struct {
	// Fingerprint is the hash of the spec the user approved.
	Fingerprint string `json:"fingerprint"`
	// ApprovedAt is when it was approved, for display and audit.
	ApprovedAt time.Time `json:"approvedAt"`
	// Summary is the command line or URL as it stood at approval time. It is
	// recorded for the user's benefit — when a fingerprint stops matching, this
	// is what they are being asked to compare against.
	Summary string `json:"summary,omitempty"`
}

// Approvals is the ledger of servers the user has agreed to run.
//
// It is deliberately separate from mcp.json. Every other MCP client treats a
// server without "disabled" as enabled, so a configuration block pasted from
// elsewhere is expected to work; inverting that default would break the paste
// and surprise anyone editing the file by hand. The ledger achieves the same
// protection without fighting the convention: configuration says what exists,
// the ledger says what may run.
type Approvals struct {
	Entries map[string]Approval `json:"approvals"`
}

// ApprovalsPath returns the ledger's location, resolved like ConfigPath.
func ApprovalsPath() (string, error) {
	if path := strings.TrimSpace(os.Getenv(ApprovalsPathEnv)); path != "" {
		return filepath.Abs(path)
	}
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return filepath.Join(xdg, "ollama", approvalsFilename), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ollama", approvalsFilename), nil
}

// LoadApprovals reads the ledger. A missing file yields an empty ledger, which
// means nothing is approved yet — the safe direction.
//
// A ledger that cannot be parsed is an error rather than an empty ledger.
// Treating a corrupt file as "nothing approved" would be safe; treating it as
// an error is safe *and* visible, and a silently emptied ledger would ask the
// user to re-approve everything with no explanation.
func LoadApprovals(path string) (*Approvals, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Approvals{Entries: map[string]Approval{}}, nil
		}
		return nil, fmt.Errorf("read mcp approvals %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return &Approvals{Entries: map[string]Approval{}}, nil
	}

	var ledger Approvals
	if err := json.Unmarshal(data, &ledger); err != nil {
		return nil, fmt.Errorf("parse mcp approvals %s: %w", path, err)
	}
	if ledger.Entries == nil {
		ledger.Entries = map[string]Approval{}
	}
	return &ledger, nil
}

// Save writes the ledger with the same private permissions as the config: it
// records which commands are allowed to run, so write access to it is write
// access to the approval decision.
func (a *Approvals) Save(path string) error {
	if a.Entries == nil {
		a.Entries = map[string]Approval{}
	}
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal mcp approvals: %w", err)
	}
	return writeFilePrivate(path, append(data, '\n'))
}

// Approve records agreement to run the spec exactly as it currently stands.
func (a *Approvals) Approve(spec *ServerSpec, at time.Time) {
	fingerprint := spec.Fingerprint()
	if fingerprint == unmarshalableFingerprint {
		// Storing it would put a value in the ledger that every other
		// unmarshalable spec also produces. Nothing is recorded, so nothing is
		// approved, and the server simply stays unapproved.
		return
	}
	if a.Entries == nil {
		a.Entries = map[string]Approval{}
	}
	a.Entries[spec.Name] = Approval{
		Fingerprint: fingerprint,
		ApprovedAt:  at.UTC(),
		Summary:     spec.redactedSummary(),
	}
}

// Revoke removes a server's approval. It reports whether one was present.
func (a *Approvals) Revoke(name string) bool {
	if _, ok := a.Entries[name]; !ok {
		return false
	}
	delete(a.Entries, name)
	return true
}

// Allows reports whether this exact spec has been approved.
func (a *Approvals) Allows(spec *ServerSpec) bool {
	if a == nil || spec == nil {
		return false
	}
	entry, ok := a.Entries[spec.Name]
	if !ok {
		return false
	}
	fingerprint := spec.Fingerprint()
	if fingerprint == unmarshalableFingerprint {
		// Not a fingerprint: a marker that one could not be computed. Two specs
		// bearing it are not the same spec, and treating them as equal would
		// approve one server on the strength of another.
		return false
	}
	return entry.Fingerprint == fingerprint
}

// Names returns the approved server names in a stable order.
func (a *Approvals) Names() []string {
	if a == nil {
		return nil
	}
	return slices.Sorted(maps.Keys(a.Entries))
}

// Fingerprint hashes everything about a spec that determines what Ollama would
// run and where it would send data: the transport, the command and its
// arguments, the environment handed to it, the URL, the headers, and any field
// this version of Ollama does not understand.
//
// Only "disabled" is excluded, because it is the user's own switch rather than
// a description of the server. Everything else is included deliberately, and
// the direction of the trade is chosen on purpose: including a field means a
// benign edit asks for re-approval, while excluding one means an edit to it
// runs unreviewed. An unknown future field is hashed for the same reason — we
// cannot know whether it is executable, so we must not assume it is not.
func (s *ServerSpec) Fingerprint() string {
	if s == nil {
		return ""
	}
	// Hash the canonical serialised form with the user's own switch cleared,
	// so the fingerprint follows the spec rather than a hand-maintained list of
	// fields that would drift as the type grows.
	clone := *s
	clone.Disabled = false
	clone.Name = ""

	data, err := json.Marshal(clone)
	if err != nil {
		// A spec that cannot be marshalled cannot be approved. This value was
		// once described as matching nothing; it matches every other spec that
		// also fails to marshal, so approving one such spec would have approved
		// them all. Approve refuses to store it and Allows refuses to accept
		// it, which is what actually makes it match nothing.
		return unmarshalableFingerprint
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// redactedSummary is the form written to the approval ledger.
//
// The ledger is a record, not a decision. What the user is asked to approve
// must be shown in full — that is Summary, and hollowing it out would hollow
// out the gate this package is built around — but the copy kept on disk
// afterwards has no need of the secret, and keeping one puts a second cleartext
// copy in a second file that a user scrubbing mcp.json has no reason to look
// in. Matching is done by the fingerprint, which is a hash; this string only
// has to let a person recognise what they agreed to.
//
// Redaction here is deliberately eager. A value withheld that was not a secret
// costs a little detail in a record; a value kept that was one costs the user
// their credential.
func (s *ServerSpec) redactedSummary() string {
	switch s.transport() {
	case TransportStdio:
		parts := append([]string{s.Command}, redactArgs(s.Args)...)
		return strings.Join(parts, " ")
	case TransportHTTP:
		return redactURL(s.URL)
	default:
		return ""
	}
}

// redactArgs withholds the values of arguments whose names suggest a
// credential, in both the "--api-key=value" and "--api-key value" forms.
func redactArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, 0, len(args))
	redactNext := false
	for _, arg := range args {
		if redactNext {
			redactNext = false
			// A following flag is not the withheld value; the flag simply had
			// none, and swallowing it would misrepresent the command line.
			if !strings.HasPrefix(arg, "-") {
				out = append(out, redactedMarker)
				continue
			}
		}
		if name, _, found := strings.Cut(arg, "="); found && secretishEnvKey.MatchString(name) {
			out = append(out, name+"="+redactedMarker)
			continue
		}
		if strings.HasPrefix(arg, "-") && secretishEnvKey.MatchString(arg) {
			redactNext = true
		}
		out = append(out, arg)
	}
	return out
}

// unmarshalableFingerprint marks a spec whose fingerprint could not be
// computed. It is never stored and never matched.
const unmarshalableFingerprint = "unmarshalable"

// Summary is the one-line description of what a server would do, for the user
// to read when approving it or when its fingerprint stops matching.
func (s *ServerSpec) Summary() string {
	switch s.transport() {
	case TransportStdio:
		parts := append([]string{s.Command}, s.Args...)
		return strings.Join(parts, " ")
	case TransportHTTP:
		return s.URL
	default:
		return ""
	}
}
