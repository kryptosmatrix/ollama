# ADR — one settings switch that approves every tool call

| Field | Value |
|---|---|
| Status | Decided and implemented on branch `isochron/auto-approve-tools` (2026-09-14). Not merged. Not a Standard 16 blueprint; a bounded feature under an operator ask |
| Decider | Isochron (Claude Fable 5.1), a technical decision under KANON 25, tier 2 (a persisted schema field and a behaviour another session would have to re-derive) |
| Check before deciding | Codex CLI 0.153.4, `codex exec --sandbox read-only` in the fork, model the account default (not exposed in the transcript). Packet: a prompt naming the requirement, the relevant files, and three candidate routes, with six questions; the repository readable to the reviewer. Prompt and verdict kept in the session scratchpad; the verdict is summarised below. The working tree already held the store half when the reviewer read it, and it said so |
| Operator ask | Ash, 2026-09-14: "When a colleague calls a tool the UI prompts the human user for permission on every individual tool call. Can we add a single toggle switch setting to the settings page that automatically approves all tool use by default when toggled on. No pop-up alerts should appear in this user mode." |
| Commits | `a0b83743e` settings field, column, v16->v17 migration, tests, regenerated TypeScript type; `ffa369f12` settings-page switch and test; `c4d4dfec1` the gate and its tests |

## Context

The desktop app gates tool calls in `app/ui/ui.go` `awaitToolApproval`: a tool that implements `ApprovalRequired` (every MCP tool does, `app/tools/mcp.go`) makes the chat's streaming response emit a `tool_approval` event and block on `app/tools/approval.go` `Approvals.Await` until `POST /api/v1/chat/{id}/approval` answers it. Per-chat `ApprovalState` already remembers a scope (`Grant`) or everything (`GrantAll`, the "always allow" button) for the rest of that chat. A separate ledger in `mcp/approvals.go` decides which servers may run at all; it is per server, not per call.

Settings persist as one row in SQLite (`app/store`), one column per field, with numbered migrations; the React side reads a `Settings` class generated from the Go struct by `tscriptify`.

## Decision

Route A. A persisted global setting `Settings.AutoApproveTools` (column `auto_approve_tools BOOLEAN NOT NULL DEFAULT 0`, schema version 17), consulted inside `awaitToolApproval` after the per-chat `Allows` check and before a request is created. When on, the call proceeds, no event is emitted, nothing is left pending, and one `Info` log line records chat, tool and scope, never the arguments. The setting is read from the store on every call, as the MCP ledger is re-read on every question, so switching off takes effect on the next call and no blanket grant is left in any chat. `Server.Store == nil` means no switch. A store that cannot be read logs a warning and asks the user.

The settings page gains one card with the switch and a description that states whichever mode is current. Reset to defaults sets it off explicitly.

## Alternatives considered

B. The same setting applied through the existing per-chat `GrantAll`. Rejected: `allowAll` has no revoke, so switching off would not restore prompting in chats that had run while it was on.

C. The same setting consulted in the tool layer, with the MCP adapter's `RequiresApproval` returning false. Rejected: it covers only MCP tools, couples the adapter to application settings, and blurs "this tool requires approval" with "the user has waived it".

## What the check said, and what was done with it

Adopted: route A; a nil-Store guard (both existing test helpers build a Server without a store); the off-state wording, which had overclaimed that every call would prompt when explicit per-chat grants still apply; an explicit false on reset; keeping the per-server ledger separate; logging without raw arguments; the column default in both the fresh `CREATE TABLE` and the `ALTER TABLE`, with `duplicateColumnError` tolerated so a half-applied migration recovers; and, most usefully, that the existing approval tests never reach `Execute`, so a test through the real chat handler was required. Two such tests now exist.

Refuted or deferred, with reasons:

1. Release calls already waiting for an answer when the switch is turned on (the reviewer marked this an inference from "no pop-up alerts"). Decided not to: a question already put to the user is theirs to answer; releasing it on a settings save would run a call the user is looking at without their click, and would need a registration barrier to avoid stranding a call that registers just after the release. The switch governs calls from the next one on, and the description says so. Falsifier: if Ash rules that a visible prompt must vanish when the switch flips, add the release behind a barrier and clear the React state.

2. On a settings read error, refuse the call through the inline tool-error path rather than prompt. Decided to prompt: both directions are fail-closed (nothing runs without consent); prompting keeps the user able to proceed and is the behaviour that existed before the switch; the "no alerts" requirement applies to a mode that cannot be read in that branch. Falsifier: a real read error in the field that Ash considers an unacceptable prompt.

3. A DOM click-through test for the React switch. Not added: the frontend test environment is `node` with no DOM or testing-library dependencies, so this would be a project tooling change; the wiring is two lines, and the behaviour is carried by the Go tests through the real handler and by the app run. The static tests cover the label, that the switch reflects the stored value, and the wording of each mode.

Recorded, not fixed: settings are written as whole snapshots and the server replaces every field, so a delayed unrelated save from another settings writer can carry a stale `AutoApproveTools` within the refetch window. This is pre-existing and affects every setting; merge or patch semantics for the settings endpoint are out of this change's scope.

## Security, stated once

With the switch on, every connected MCP server becomes unattended execution driven by model output, which prompt injection through tool results or web content can steer. The default is off; the switch's own text says what it allows; the per-server approval is unchanged; the right place to bound the blast radius is each server's own permissions. The workspace server this switch will most likely front, `agora-workspace-mcp`, has a write ceiling of `/Users/krypto/GitHub`, which contains every repository and the LINEAGE archive.

## Proof

Go, `app/ui`: `TestAutoApproveSwitchSkipsThePrompt`, `TestAutoApproveSwitchOffAsksAgain`, `TestAutoApproveWithoutAStoreStillAsks`, `TestAutoApproveUnreadableSettingsStillAsks`, `TestChatLoopRunsMCPToolWithoutAskingWhenSwitchedOn`, `TestChatLoopAsksBeforeMCPToolWhenSwitchedOff`. Go, `app/store`: the settings round-trip with the field, a fresh database off-on-off, and the epoch migration ending readable with the switch off. React: `ToolApprovalSetting.test.tsx` (three). Suites: `go test ./app/ui/ ./app/tools/ ./app/store/` green; `vitest run` 136 of 136; `tsc -b` clean; the three changed frontend files lint clean (the project's whole-tree lint carries 137 pre-existing errors, none in changed files).

Not proven here: the packaged app on Darwin with the switch flipped by hand. Building lands an app bundle on the operator's machine, so it is offered rather than done. Windows and the TUI are out of scope; the TUI keeps its own approval flow.

## Rollback

Revert the three commits. The column is harmless if left (default off); the repository's own migrations already use `ALTER TABLE ... DROP COLUMN` if it must go.

— Isochron, 2026-09-14
