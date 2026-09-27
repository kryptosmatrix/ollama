# Pre-code technical review, revision 2 — exact repair and assertions

You are DeepSeek, Ash's selected independent reviewer. Judge the concrete contract below, not the existence of a formal blueprint stamp. The operator has explicitly delegated the bounded maintenance exception; the six requirements and all implementation/testing/independent-execution obligations remain. No production edits have happened. Return APPROVE or REVISE for design sufficiency only, with finite source-grounded findings. Do not claim tests ran or issue code/release acceptance. Do not manufacture objections from hypothetical APIs not present in the supplied source.

## Contract and scope

R-01. Only an absent chat-header row MUST be classified as `not.Found` by `Store.ChatWithOptions`.
R-02. A non-absence chat-load error MUST preserve its underlying error chain through `Store.ChatWithOptions`.
R-03. Continuing an existing chat whose load fails MUST leave its persisted chat, message, attachment and tool-call records unchanged.
R-04. Continuing an existing chat whose load fails MUST return an error before an inference request is issued.
R-05. Valid existing-chat continuation MUST retain the earlier messages.
R-06. Genuinely absent-chat continuation MUST retain the existing create-and-continue behaviour.

Only production files database.go and store.go change. No schema, method signature, authentication policy, default config, retry, concurrent-writer repair or installed application changes. The existing HTTP caller remains unchanged. No whole-programme feature is marked delivered from this prerequisite fix. Authorisation is recorded in DECISIONS.md as ash-eko-standing-preapproval-2026-09-20.

## Exact candidate, replacing ambiguity with code

The entire resulting Store method is:
```go
func (s *Store) ChatWithOptions(id string, loadAttachmentData bool) (*Chat, error) {
    if err := s.ensureDB(); err != nil {
        return nil, err
    }
    return s.db.getChatWithOptions(id, loadAttachmentData)
}
```
Store returns the database's error unchanged and never adds not.Found. database.go adds the existing app/types/not import. Only the chat-header QueryRow/Scan error arm becomes:
```go
if err != nil {
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("%w: chat %s", not.Found, id)
    }
    return nil, fmt.Errorf("query chat: %w", err)
}
```
The header-absence branch alone produces the wrapped sentinel on this read path. The message/attachment/tool-call functions and their wrappers remain unchanged. A downstream error, including any error wrapping sql.ErrNoRows, is transported without not.Found. Empty message/attachment/tool result sets are successful empty collections, not absent chats.

The unchanged HTTP consumer is:
```go
chat, err := s.Store.ChatWithOptions(cid, true)
if err != nil {
    if !errors.Is(err, not.Found) { return err }
    chat = store.NewChat(cid)
}
```
Then it appends the user's prompt and calls Store.SetChat; only afterwards does it construct the inference client and issue Show/Chat calls. On a non-absence error the save and inference are unreachable.

## Live source and provenance

HEAD 953de98d408a9697fa20a48c23973b9dc8eee921. Raw file SHA256: app/store/store.go af324196245292ab5b6ee65abbb8de2f1db5e3d07772db0f98d273c53848349f; app/store/database.go 16bde5746f671f7f83c5e4b952186c698f306cd70508d547f7bc91c93ae0a099; app/ui/ui.go dbba86f3d90ec5edac5b23458fec00f25e39f4d9f3ef43ac124571e0c1a0d48e.

Current Store lines 428-440 relabel every database failure with not.Found. Header query at database lines 675-714 currently returns untyped fmt.Errorf("chat not found") for direct sql.ErrNoRows; all other errors retain %w causes. Saver lines 716-774 upserts the header and deletes/reinserts messages transactionally, cascading their attachments/tool rows. UI lines 727-734 are exactly the consumer above, and save/inference follow at 785/797.

Freshly read database.go:910-991 getMessages uses db.conn.Query, loops rows.Next(), scans and wraps actual failures as `scan message: %w`, loads attachments/tools with their %w wrappers, checks rows.Err(), returns messages,nil. At 1052-1091 getAttachments uses the same Query/Next/Scan/Err pattern. At 1093-1140 getToolCalls uses that pattern too. None uses QueryRow/Scan for missing children or emits not.Found. A successful empty result has zero loop iterations, rows.Err()==nil and no error. The actual absence of children is covered by valid empty-chat tests. Inventing a driver-version scenario in which ordinary empty Query becomes sql.ErrNoRows is not evidence about this path. The algorithm explicitly preserves a future downstream ErrNoRows without reclassification, but no new production injection seam is justified solely to manufacture that otherwise-unreachable case.

## Exact assertions and planned changes to the retained diagnostics

Store diagnostic already compiled at the above HEAD. It uses the real Store fixture (s, cleanup := setupTestStore(t), t.Cleanup(cleanup)), saves an existing chat, closes the database and obtains cause := s.db.conn.Ping(). For both loading modes, it asserts:
```go
chat, err := s.ChatWithOptions("existing-chat", attachments)
if chat != nil || err == nil { t.Fatal("closed database returned success") }
if errors.Is(err, not.Found) { t.Error("database failure became absence") }
if !errors.Is(err, cause) { t.Error("database error identity lost") }
```
Its genuine-absence control asserts nil chat and errors.Is(err,not.Found). Additional permanent Store tests cover empty/non-empty valid chats with exact content; ensureDB initialisation failure; malformed message data with a live header, direct SQL observation of the invalid value, an actual scan error, !errors.Is(not.Found), !errors.Is(sql.ErrNoRows), and non-empty cause wrapping. The dynamically generated Scan conversion error has no exported stable sentinel shared between separate query attempts: string-equality alone is not used as error-identity proof; the native closed-database test above is the independent chain-identity check.

HTTP diagnostic uses four fresh OS processes (seed, fault, recover, positive). HOME/USERPROFILE/LOCALAPPDATA are set before each child's package initialisation; the Store uses its real platform-default location within that isolated home. Internal SQLite and Server.Handler are real; inference alone is an explicit loopback HTTP simulation. Authenticated fixture is existing ttsServer(t,&countingSynth{}); Dev=false asserted; a valid synthetic cookie exercises the production auth branch. This is not packaged-app/browser session issuance proof. Add an unauthenticated request using the same endpoint and no cookie; assert 403, no inference and byte-equal raw-table snapshot.

Seed: Store.SetChat with title, browser state, two known different message bodies, non-empty attachment and tool record. Fault: mutate precisely one persisted message boolean, verify one row affected, verify Store actually fails, then POST a continuation. The inference service's actual HTTP handler increments an atomic counter on EVERY request, including Show; this is the observation point, not a caller-side configured counter. Assert HTTP 500 with explicit JSON error, counter==0, and raw SQL equality across every column/row of all four relevant tables. All are non-empty before the fault; emptiness after is failure. The same counter implementation must be positive in each successful/recovery request, explicitly hits.Load()>0, and the recovery arm additionally captures and inspects the actual api.ChatRequest containing the original history. Each phase starts a fresh service/counter and no state survives in memory across phases.

Recovery: fresh process corrects only the injected boolean and requires exactly one affected row. Before continuing, assert title, BOTH exact original bodies, attachment bytes and tool identity. After continuation, assert message length 4 AND exact four bodies [original user, original assistant, new prompt, reply:new prompt] in order; require captured inference input includes original history and the positive service counter. The raw fault snapshot already protects original row identifiers and all column values. Successful save legitimately reinserts messages with new row IDs, so requiring stable message IDs across successful continuation would invent a compatibility requirement and break existing supported behaviour.

Positive: a new ID demonstrably absent before POST, then two distinct prompts; assert created chat.ID equals requested ID and all four stored body values exactly. The real unchanged UI NewChat/SetChat branch is used; no alternative writer exists in the test. Source containment verifies that branch has not changed. Assertions on internal function-call names would prove instrumentation rather than R-06's behavioural contract.

## Deliberate breaks and regression obligations

Run the unchanged permanent acceptance tests on candidate, then compile isolated variants, never altering installed/shared data:
A. Restore the OLD STORE METHOD's blanket not.Found wrapping. This IS the Store-rewrapping control (not a different database control). It must fail both cause preservation and HTTP record preservation/no-inference.
B. Change only UI error handling to recreate a chat on any read error. It must fail the same authenticated HTTP preservation/no-inference assertions.
C. Store always returns a non-absence error: valid/missing positive controls fail.
D. Store returns fabricated empty Chat success: original-body/data preservation or malformed-error assertions fail.
Keep patches, source identities, discovered tests, full output and actual child exit codes. Compilation errors, zero tests, unrelated exceptions and timeouts do not count. Clean rerun must pass. No returned verdict substitutes for actual execution.

The original full affected set remains required before acceptance: core Go build; ./agent/... ./cmd/tui/chat/... ./app/... suites with verified safe isolation; complete frontend Vitest/typecheck/lint; direct test/controls; diff check and containment. Previous unrelated lint failures stay visible. Root Darwin login-Keychain tests are not run against real credentials. Independent executable review is still owed; this source-only review does not discharge T-10. No release/install or overall programme completion claim.

## Reconciliation of prior review claims

Prior D-01/D-02 requested explicit pass-through/!not.Found/errors.Is(cause): code and existing assertions above supply them. D-03 requested identity of a dynamically generated scan error: malformed-input evidence and native closed-database identity prove different properties without pretending two separately constructed conversion errors share identity. D-04 clarified the server-side counter and positive-control liveness: made explicit above and added positive assertions. D-05 wanted exact original content after successful continuation: strengthen to all four exact bodies. D-06 wanted existing route: real unchanged handler, ID assertion and source containment rather than a mock call counter. D-07 claimed restored relabelling did not cover Store rewrapping: that claim was false; control A explicitly restores the whole old Store method. D-08 requested normal missing-child ErrNoRows: source proves missing children yield success; preserve the valid-empty case and never add broad ErrNoRows classification.

Review independently. Reject any disposition contradicted by supplied code. Judge whether there is a real remaining implementation-design blocker, not whether unperformed post-code obligations have already passed. Return one design-only APPROVE or REVISE.
