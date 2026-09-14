## Findings

### HIGH — Migration can leave a credential in cleartext permanently

**File:** `mcp/tokenstore_darwin.go`, lines 107-112

In `KeychainStore.Load`, the migration path saves to the keychain and then deletes from the fallback file store. If `s.Save` succeeds but `s.Fallback.Delete` fails (e.g. the file became read-only, or a transient I/O error on re-read inside `Delete`), the function returns the delete error — but the token now exists in **both** the keychain and the cleartext file. Every subsequent `Load` for that server finds the item in the keychain (line 89-96) and returns immediately, never attempting to remove the file copy again. The cleartext credential persists indefinitely, defeating the stated purpose of the migration ("takes the credential out of cleartext rather than leaving a copy behind," line 87). The user is told about the error once, but the secret remains readable by any process as the user forever, with no further attempt to clean it up.

### MEDIUM — `Servers()` omits servers whose tokens are only in the fallback store

**File:** `mcp/tokenstore_darwin.go`, lines 165-200

`KeychainStore.Servers()` lists only keychain items. A server whose token exists solely in the `Fallback` file store (because `Load` has not yet been called for it) is invisible to `Servers()`. A caller that enumerates signed-in servers via `Servers()` to offer sign-out, or to display status, will miss a server the user is actually signed into. The migration is lazy (only on `Load`), so any server not yet loaded is absent from the list.

### MEDIUM — File store read-modify-write is not concurrency-safe; cross-server token loss

**File:** `mcp/tokenstore.go`, lines 190-216 (Save) and 219-229 (Delete)

`FileTokenStore.Save` and `Delete` both do read-entire-file → modify → write-entire-file. If two operations for *different* servers interleave (process A reads, process B reads, A writes, B writes), B's write overwrites A's change, silently losing the token for A's server. The atomic rename in `writeFilePrivate` prevents a torn file but does not prevent lost updates across servers. This is data loss of a credential.

### LOW — `goString` silently returns empty string on conversion failure

**File:** `mcp/tokenstore_darwin.go`, lines 334-336

If `CFStringGetCString` returns `false` (e.g. the CFString contains characters not representable in the requested encoding, or a buffer allocation issue), `goString` returns `""`. In `Servers()` (line 196) this produces an empty server name that gets appended to the result slice and sorted, silently corrupting the server list without any error.