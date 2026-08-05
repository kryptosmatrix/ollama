# Handoff — MCP client support, session 1

From: Grommet (Claude Opus 5)
Date: 2026-08-05
Branch: `mcp/server-support`, pushed to `origin` (kryptosmatrix/ollama)
Head at handoff: `0506f8e7`

## Read this first

`docs/_design/MCP_CLIENT_PLAN.md` is the governing document. It carries the codebase analysis, the operator's rulings, the security model, the UI contract, and the phase plan with each phase's proof obligations. This handoff says only what changed in session 1 and what the next session should do.

## State

| Phase | Status |
|---|---|
| 0 — toolchain and baseline | **DONE** (`bc2d60ae`) |
| 1 — `mcp/config.go` | **DONE** (`f984d4ec`) |
| 1b — approval ledger | **DONE** (`b672284c`) |
| 2a — namespacing + schema conversion | **DONE** (`7dbec4fa`) |
| 2 — manager and transports | **DONE** (`594f6104`) |
| 2b — OAuth 2.1 | not started |
| 3a — CLI surface | **DONE** (`12ea7760`, `09a954cb`, `7e533582`, `f12a2d6a`) |
| 3b — app approval path | **BACKEND DONE** (`0506f8e7`) — React side outstanding |
| 3c/3d — app surfaces | not started |
| 4 / 4b — MCP Servers page, registry browse | not started |
| 5 — docs, cross-substrate review, closeout | not started |

The whole tree builds and vets clean; `mcp`, `agent/...` and `cmd` all pass. MCP is live in `ollama` agent mode: configured, approved servers are connected at start-up and their tools are offered to the model behind per-tool approval.

**The feature is now usable end to end from the terminal.** `ollama mcp add|list|remove|enable|disable|approve|revoke` exists and is registered on the root command; `ollama` in agent mode connects approved servers and offers their tools behind per-tool approval.

`/mcp` inside the agent TUI lists servers and toggles them; it deliberately cannot approve one. **Phase 3a is complete.** The next surface is the desktop app.

## Operator rulings in force

Ruled 2026-08-05, recorded in plan §5 and §8.4. Do not re-open without Ash.

1. Surfaces: desktop app **and** CLI agent TUI.
2. Remote: **full OAuth 2.1 with sign-in**, sequenced last (Phase 3d).
3. Protocol: **official `github.com/modelcontextprotocol/go-sdk` v1.7.0**, contained so no SDK type appears outside `mcp/`.
4. Target: **aimed at an upstream PR**, so keep the diff small and touch unrelated files as little as possible.
5. The MCP Servers page is a **manager plus a browse-and-install surface over the official MCP Registry**.

## Things that will cost you a session if you don't know them

**`go build ./...` fails on a clean checkout.** `app/ui/app.go` embeds `app/dist`, so the frontend must be built first: `npm ci && npm run build` in `app/ui/app`. This is not a broken tree.

**Three checks are already red and are not yours to fix.** `go vet ./...` fails on `tokenizer/bytepairencoding_test.go:542`. `npm run lint` fails with 122 problems across 14 files. `npm run prettier:check` fails on 9 files. Full detail in `docs/_design/proof/phase0-baseline.txt`. Completion is "no new problems against that baseline", not "zero problems".

**Do not run `prettier --write` on an existing file.** `ChatSidebar.tsx` must be edited for the sidebar entry and is already format-dirty; reformatting it would bury a three-line change in a hundred-line diff and work against the upstream ruling. Hand-edit in the surrounding style.

**`mcp/` cannot import `cmd/internal/fileutil`.** Go's internal rule. It writes atomically itself.

**The upstream branch `Parth/agents` does not compile.** Commit `5c0caaff` looks like finished MCP support and its manager was never committed. It is a design reference for config shape and CLI naming, nothing more. Plan §3.1.

**`go test ./... | tail` reports the exit code of `tail`.** Redirect to a file and echo `$?` on its own.

**`app/updater`'s `TestAutoUpdateDisabledSkipsDownload` is flaky.** It fails roughly one full app-tree run in three and always passes when that package is run alone — proven against a clean tree with this work stashed. It is not MCP's. Do not chase it, and do not read a single green app-tree run as proof of anything.

## What the next session should do

**Finish 3b, then 3c.** Two pieces, in this order:

1. **The React approval prompt.** The backend is complete and proven; a user still cannot answer. Regenerate `app/ui/app/codegen/gotypes.gen.ts` (the `tscriptify` generate directive at `app/ui/ui.go:39`) so the new `tool_approval` event and its `approvalId` / `approvalScope` / `approvalArgs` fields reach TypeScript, then render the prompt in the message stream and POST the answer to `/api/v1/chat/{id}/approval` with `{approvalId, allow, remember, rememberAll}`. Do **not** hand-edit the generated file.
2. **3c: register MCP tools in the app.** `app/tools/mcp.go` adapting `mcp.Tool` to `app/tools.Tool` plus `ApprovalRequired` and `ScopedTool` (mirror `agent/tools/mcp.go`), a process-lifetime manager in `app/ui/ui.go` — **not** per chat request, or every message restarts the subprocesses — the routes in plan §8.2, `responses/types.go` additions, and schema v17 for `mcp_server_state`.

**Until the React prompt exists, do not register MCP tools in the app.** The gate would block every call for ten minutes and then refuse it, which is safe but looks exactly like a hang.

Then Phase 4 (the MCP Servers page) and 4b (registry browse).

## What must not soften

- Tool descriptions from a server are untrusted input that lands in the model's prompt. Sanitised and capped in `mcp/tools.go`; keep it that way when the manager starts feeding real ones through.
- Schema conversion is lossy and says so in the description. Do not "simplify" that into dropping constraints.
- `mcp.json` is a code-execution config: `0600` in a `0700` directory, no shell, credentials only as `${env:NAME}`.
- The desktop app's approval path now exists but **has no user interface**. Registering MCP tools in the app before the React prompt lands would gate every call on a question nobody can answer.
- Everything about the approval wait fails safe: a timeout, a cancelled context and a failed notify all refuse the call. Do not "simplify" any of those into allowing it.
- `/mcp` must never grow an approve verb. Approval needs the command line shown verbatim and a deliberate answer; a chat input line cannot give either. There is a test guarding the decision, not the implementation.
- If Phase 4b runs long, the browse polish gives way; the install gate — resolved command line shown verbatim, landed disabled, hash verified — never does.

## Proof artefacts

- `docs/_design/proof/phase0-baseline.txt` — toolchain, build, vet, frontend, lint, format, and what was already red.
- `docs/_design/proof/phase0-gotest.txt` — full Go suite at baseline: 64 packages ok, 34 without tests, 0 failures.
- `docs/_design/proof/phase1-falsification.txt` — three protections broken and observed failing, then restored.
- `docs/_design/proof/phase2a-falsification.txt` — four more, same method.
- `docs/_design/proof/phase1b-falsification.txt` — the approval ledger, four protections.
- `docs/_design/proof/phase3a-falsification.txt` — the harness adapter and the CLI entry point, including two falsifiers that initially passed and the test repairs they forced.
- `docs/_design/proof/phase3a-commands-falsification.txt` — the `ollama mcp` commands, five protections including root-command registration.
- `docs/_design/proof/phase3a-slash-falsification.txt` — `/mcp`, five protections including a disabled server left running.
- `docs/_design/proof/phase3b-falsification.txt` — the app approval rendezvous, six protections, plus the attribution of a pre-existing flaky test.
- `docs/_design/proof/phase2-falsification.txt` — the manager, including three attempts that **failed to falsify** and the removal that followed. Read that section before adding any safety mechanism to this package: the protocol library already reaps processes, and code written to do it again cannot be tested.
