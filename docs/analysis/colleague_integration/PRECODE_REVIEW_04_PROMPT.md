# Independent pre-code review: chat-history preservation repair

You are DeepSeek, the operator-selected independent adversarial reviewer. Review this bounded maintenance contract for technical design sufficiency BEFORE implementation. Do not assume it is correct. Return a finite list of substantive defects with source/requirement anchors, then one design-only conclusion: APPROVE or REVISE. This is not a code-acceptance, execution, release, or earned blueprint-promotion verdict. You have only this supplied material; do not claim access to files or tests not included here, or claim you executed anything.

## Authority and boundary

Ash directly granted Eko standing pre-approval for changes Eko reasons necessary and announces, excluding unrecoverable deletion. Recorded verbatim in docs/analysis/colleague_integration/DECISIONS.md, decision ash-eko-standing-preapproval-2026-09-20. Eko has applied that grant to this two-function maintenance repair, dispensing with formal Method 16 promotion and its dependent Method 03 rung for THIS repair only. All six frozen requirements and independent pre-code review, Method 04 T-01 through T-10, permanent tests, deliberate-break controls, regression and independent executable review remain required. Do not reopen the settled authorisation/byte-ratio question. Do challenge deficient technical design or proof. No source edit has yet occurred.

Only app/store/database.go::getChatWithOptions and app/store/store.go::ChatWithOptions change in production, plus their tests and scoped evidence. No schema, signature, permission policy, migration, retries, concurrency redesign, installed app, or live conversation changes. GET-empty-success and rename error wording are separate findings and are not silently claimed repaired. HEAD 953de98d408a9697fa20a48c23973b9dc8eee921.

## Frozen requirements

R-01. Only an absent chat-header row MUST be classified as `not.Found` by `Store.ChatWithOptions`.
R-02. A non-absence chat-load error MUST preserve its underlying error chain through `Store.ChatWithOptions`.
R-03. Continuing an existing chat whose load fails MUST leave its persisted chat, message, attachment and tool-call records unchanged.
R-04. Continuing an existing chat whose load fails MUST return an error before an inference request is issued.
R-05. Valid existing-chat continuation MUST retain the earlier messages.
R-06. Genuinely absent-chat continuation MUST retain the existing create-and-continue behaviour.

## Exact live source excerpts

app/store/database.go raw SHA256 16bde5746f671f7f83c5e4b952186c698f306cd70508d547f7bc91c93ae0a099, lines 675-714:
```go
func (db *database) getChatWithOptions(id string, loadAttachmentData bool) (*Chat, error) {
 query := `SELECT id, title, created_at, browser_state FROM chats WHERE id = ?`
 var chat Chat
 var createdAt time.Time
 var browserState sql.NullString
 err := db.conn.QueryRow(query, id).Scan(&chat.ID, &chat.Title, &createdAt, &browserState)
 if err != nil {
  if err == sql.ErrNoRows { return nil, fmt.Errorf("chat not found") }
  return nil, fmt.Errorf("query chat: %w", err)
 }
 chat.CreatedAt = createdAt
 if browserState.Valid && browserState.String != "" {
  var raw json.RawMessage
  if err := json.Unmarshal([]byte(browserState.String), &raw); err == nil { chat.BrowserState = raw }
 }
 messages, err := db.getMessages(id, loadAttachmentData)
 if err != nil { return nil, fmt.Errorf("get messages: %w", err) }
 chat.Messages = messages
 return &chat, nil
}
```
Whitespace is compacted in this excerpt; tokens and flow are retained. Browser-state decode swallowing is pre-existing and outside the contract's conditional 'whose load fails': no claim that every malformed field produces an error.

app/store/store.go raw SHA256 af324196245292ab5b6ee65abbb8de2f1db5e3d07772db0f98d273c53848349f, lines 428-440:
```go
func (s *Store) ChatWithOptions(id string, loadAttachmentData bool) (*Chat, error) {
 if err := s.ensureDB(); err != nil { return nil, err }
 chat, err := s.db.getChatWithOptions(id, loadAttachmentData)
 if err != nil { return nil, fmt.Errorf("%w: chat %s", not.Found, id) }
 return chat, nil
}
```

app/ui/ui.go raw SHA256 dbba86f3d90ec5edac5b23458fec00f25e39f4d9f3ef43ac124571e0c1a0d48e, lines 727-734:
```go
chat, err := s.Store.ChatWithOptions(cid, true)
if err != nil {
 if !errors.Is(err, not.Found) { return err }
 chat = store.NewChat(cid)
}
```
Then the existing code appends the user prompt and calls Store.SetChat at lines 774-787. Inference begins afterwards, around lines 797-802. database.saveChat at lines 716-774 opens a transaction, upserts the header, deletes old message rows, then inserts the supplied messages and tool calls. Dependent attachments/tool rows cascade on message deletion. No changes to this saver or HTTP handler are proposed.

app/types/not/found.go defines `var Found = errors.New("not found")`; its documented usage is wrapping with context via fmt.Errorf or errors.Join, not direct sentinel returns. The database file currently imports database/sql, encoding/json, fmt, strings, time and the sqlite3 driver. It does not yet import app/types/not.

## Proposed exact algorithm

1. Add the existing github.com/ollama/ollama/app/types/not import to database.go.
2. ONLY the header QueryRow/Scan `err == sql.ErrNoRows` arm changes to `return nil, fmt.Errorf("%w: chat %s", not.Found, id)`.
3. Preserve ensureDB unchanged; replace the Store's whole relabelling block with `return s.db.getChatWithOptions(id, loadAttachmentData)`.
4. Preserve every other database error wrapper and every successful code path. Do not classify nested ErrNoRows globally, inspect error strings, return an empty Chat on failure, repair data implicitly, or add new database reads.

Expected effect: genuine absence keeps the existing Store-level sentinel/text; other failures preserve their underlying cause and reach the HTTP handler's existing abort branch before any save/inference. There is no new state, lifecycle, asymptotic cost, schema, or flag.

## Test/acceptance contract

Permanent default Go tests will be in app/store/chat_read_errors_test.go and app/ui/chat_read_errors_test.go; neither a skip nor an optional build tag hides the macOS/Windows application obligation. Existing windows||darwin tags match the production package boundary.

Store scenarios: both attachment-loading modes; genuine absent header gives nil chat plus errors.Is(not.Found); valid empty/non-empty chat succeeds; a real closed SQLite database retains the original error identity as obtained by native Ping; a deliberately malformed message column causes a real loader error and is not classified as absence; ensureDB failure remains non-absence. R-01/R-02/R-05/R-06 annotations and exact names retained.

HTTP scenario: four fresh test processes, each with isolated HOME/USERPROFILE/LOCALAPPDATA set BEFORE package initialisation, use Store's default platform path. Seed via Store.SetChat with two distinguishable messages, title, browser state, non-empty attachment and tool-call record. Fault changes one message boolean to invalid text in only this test database, verifies exactly one affected row and actual loader failure, then POSTs a new prompt through the real Server.Handler with Dev=false and a valid synthetic test cookie. Assert explicit HTTP 500 JSON error, zero inference requests and independent raw SQL equality across EVERY column/row in chats/messages/attachments/tool_calls; empty fixture tables fail rather than allowing vacuous equality. Include a no-cookie request that is refused before data/inference effects. Recovery in a fresh process corrects only the injected boolean, proves original records remain, continues through the same handler, asserts both originals plus the new user/assistant messages persist, and observes the original history arriving in the inference request. Another fresh process creates a genuinely absent chat and continues it again with two distinct prompts; exact persisted reply values reject an always-refusing or constant-output path.

Internal Store, SQLite, routing and error-to-HTTP handling are real. Inference is explicitly an external simulated HTTP service returning input-dependent replies; this proves data transport, not live-model compatibility. The existing ttsServer/countingSynth fixture supplies an authenticated Server without modifying credential initialisation, and Dev=false is asserted. This tests the actual production authentication branch, not packaged-app session issuance, browser sign-in, or installed-binary startup. Those wider boundaries are not claimed proved.

The historical authenticated diagnostic at unchanged HEAD compiled and returned exit 1: fault/recovery failed with both original messages, the attachment, tool record and title lost; seed and independent healthy/absent controls passed. A second compiled diagnostic found that the native closed-database cause becomes not.Found. These are retained red baselines, NOT post-repair acceptance.

Fault controls: run unchanged acceptance assertions on clean candidate, then in isolated compilation overlays or disposable copies on (a) restored relabelling bug, (b) HTTP consumer that recreates a chat after any read failure, (c) always-error getter, and (d) getter returning a fabricated empty chat. Require specific behavioural assertion failures, not compilation errors, no test discovery, timeout or unrelated faults; retain each patch and logs, and clean rerun. Broken controls must never affect the user's installed app or real database.

Regression obligations remain the original declared set: core Go build; complete ./agent/... ./cmd/tui/chat/... ./app/... suites where safely isolated; complete frontend tests/typecheck/lint; targeted tests/controls; git diff --check and source containment. Historical unrelated lint failure (137 errors, 11 warnings) remains visible and cannot be labelled green. No root Darwin test reaches the actual login Keychain just because HOME is changed. Missing necessary environment or independent re-execution prevents acceptance, not diagnostics/implementation work under this approved contract. T-10 independent execution remains owed, not supplied by this source review. Default-run inclusion and expected test discovery are explicitly checked before acceptance.

## Requested review

Challenge the production design and adequacy of the planned proof for the six requirements. Identify the smallest fix to any actual gap without unrelated redesign. Distinguish pre-code design sufficiency from unperformed mandatory execution and wider product proof. Return APPROVE only if implementation can follow this contract without inventing semantics; list remaining execution obligations separately. No passing code or release verdict.
