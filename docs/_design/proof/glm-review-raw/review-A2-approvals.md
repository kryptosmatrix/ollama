## Findings

### MEDIUM — Credentials in URL or args are persisted in plaintext to the approvals ledger

**File:** `mcp/approvals.go`, lines 161–170 (`Approve`), via `Summary()` at lines 235–245.

`Approve` stores `spec.Summary()` in the ledger's `Summary` field, and `Save` (line 157) writes that to disk. For `TransportHTTP`, `Summary()` returns `s.URL` verbatim (line 241); for `TransportStdio`, it joins `Command` and all `Args` (line 239). If a user configures an HTTP server with credentials embedded in the URL (e.g. `https://token:x-oauth@host`), or a stdio server that passes a secret as a command-line argument (e.g. `--api-key=sk-...`), those secrets are written in cleartext to `mcp-approvals.json`. The fingerprint already captures the spec securely as a SHA-256 hash; the `Summary` field duplicates the secret in plaintext in a second on-disk file. If the user later edits `mcp.json` to remove the credential, the secret persists in the approvals ledger, which the user may not think to scrub.

### LOW — `Fingerprint()` returns a constant string on marshal failure, creating a collision class

**File:** `mcp/approvals.go`, lines 223–228.

If `json.Marshal(clone)` fails, `Fingerprint()` returns the constant string `"unmarshalable"`. The comment claims this "matches nothing" for safety, but it actually matches *every other spec that also fails to marshal*. If a user were to approve one such spec, the ledger stores `"unmarshalable"`, and every subsequent spec that fails to marshal would produce the same fingerprint and be allowed. This is a fingerprint collision: approving one unmarshalable spec approves all of them. In practice, standard Go structs with string/slice/map fields do not fail to marshal, so this requires a custom `MarshalJSON` or an unusual field type on `ServerSpec` to trigger—but the safety reasoning in the comment is incorrect, and the failure mode is a collision rather than a denial.

---

**No defect found** in the following categories:
- **CRITICAL**: No path was found by which a server can be started without a current approval. The name-keyed lookup plus fingerprint comparison is sound against config tampering: changing the command, args, env, URL, or headers changes the fingerprint and causes denial.
- **HIGH**: No high-severity wrong behaviour was found in the approval or fingerprint logic under normal operation.