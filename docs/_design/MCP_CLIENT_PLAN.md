# MCP Client Support for Ollama — Codebase Analysis and Implementation Plan

Status: `[CONCEPT]` + decision-lock proposal. Not `[IMPLEMENTATION-READY]`; nothing here is authorised to build until §5 is ruled and §7 Phase 0 completes.
Repo: `kryptosmatrix/ollama` fork, branch `mcp/server-support` (currently 0 commits ahead of `upstream/main` at `8d8c701d`).
Author: Grommet (Claude Opus 5), 2026-08-05.
Method: TECHNE. This document is fork-local; it lives under `docs/_design/` so it never collides with upstream's `docs/*.mdx` corpus and is trivially excluded from any future upstream PR.

---

## 1. What is being asked for

Give Ollama the ability to connect to MCP (Model Context Protocol) servers — both **local** (launched by Ollama as a child process, stdio transport) and **internet-based** (remote HTTP endpoints) — so that the tools those servers expose become callable by models running inside Ollama's own surfaces.

A UI mock-up for the desktop app arrives in a following prompt. §8 of this plan defines the *contract* the UI must express — the entities, states, and endpoints. The mock-up will dress that contract; it should not need to change it. Where the mock-up contradicts §8, the mock-up wins and §8 is corrected (it is Ash's product decision), but the contradiction is recorded rather than absorbed silently.

**Ollama becomes an MCP _client_ / host.** This is the opposite direction from `docs/capabilities/web-search.mdx`, which documents exposing *Ollama's* web search to other MCP clients via a Python server. Both can be true at once; they are unrelated code paths.

---

## 2. What the codebase actually looks like

Read from source at `8d8c701d`. Every claim below is a code fact, not an inference.

### 2.1 There are two separate tool stacks, with two different `Tool` interfaces

**Stack A — the core agent harness** (`agent/`, landed 2026-07-02 in #16963). Consumed by the CLI agent TUI.

- `agent/registry.go:20-25` — `Tool` interface: `Name() string`, `Description() string`, `Schema() api.ToolFunction`, `Execute(context.Context, ToolContext, map[string]any) (ToolResult, error)`.
- `agent/registry.go:27-36` — two optional capability interfaces: `ApprovalRequired` (`RequiresApproval(map[string]any) bool`) and `ScopedTool` (`ApprovalScope(args map[string]any) string`).
- `agent/registry.go:38-50` — `Registry` is a bare `map[string]Tool` with a `Register` method. No lifecycle, no async, no close.
- `agent/session.go:22-31` — `Session` holds `Tools *Registry`, `ApprovalPrompter`, `ApprovalState`, `Compactor`.
- `agent/session.go:505` — approval is checked per call (`s.needsApproval(tool, toolName, args)`); `agent/session.go:575` executes.
- `agent/approval.go` — full approval model: per-scope grants, allow-all, `ApprovalPrompter` interface. The TUI implements the prompter in `cmd/tui/chat/approval.go`.
- Registry is populated at `cmd/agent_tui.go:249-277` (`agentToolsRegistry`): `Bash`, `Read`, `Edit`, `Skill`, `WebSearch`, `WebFetch`, each gated by env var or cloud availability.

**Stack B — the desktop app** (`app/`). A Go binary hosting a WebView over a React/Vite frontend.

- `app/tools/tools.go:12-27` — a *different* `Tool` interface: `Name()`, `Description()`, `Schema() map[string]any`, `Execute(ctx, args) (any, string, error)`, `Prompt() string`.
- `app/tools/tools.go:29-40` — its own `Registry`, constructed per chat request at `app/ui/ui.go:852`.
- Registrations at `app/ui/ui.go:867-872`: browser tools or `WebSearch`/`WebFetch`.
- Execution at `app/ui/ui.go:1030` — `registry.Execute(...)` called directly inside the streaming loop, results persisted as `store.Message` rows and streamed to the frontend as `tool` / `tool_result` SSE events.
- **There is no approval mechanism in Stack B.** No `ApprovalRequired`, no prompter, no scope state. Tools run the moment the model asks. This is safe today only because every tool in Stack B is a first-party, read-only-ish web tool.

**Consequence.** Any MCP protocol code written inside either stack would have to be duplicated into the other, and the two copies would drift. The protocol client belongs in a third, shared package with thin adapters into each stack. See §4.

### 2.2 Config precedent already exists, and it is file-based under `~/.ollama/`

- `app/store/cloud_config.go` — the desktop app reads/writes `~/.ollama/server.json` **directly as a file**, deliberately outside its SQLite database, because the setting is shared with the daemon. This is the exact precedent for MCP config: shared between CLI and app, so it must not live only in the app's DB.
- `cmd/config/config.go` — `~/.ollama/config.json`, written atomically via `cmd/internal/fileutil`, holds integration config for external coding tools.
- `agent/skills.go:79-91` — `SkillsDir()` resolves `$OLLAMA_SKILLS`, else `$XDG_CONFIG_HOME/ollama/skills`, else `~/.ollama/skills`.
- `agent/skills.go:311-315` — Ollama already *imports* cross-client config from `~/.codex/skills`, `~/.claude/skills`, `~/.pi/agent/skills`. The project's own convention is to speak the ecosystem's file conventions rather than invent private ones.

### 2.3 The desktop app's persistence and settings surface

- `app/store/database.go:17` — `currentSchemaVersion = 16`, with a linear `migrateVNtoVN+1` chain. Adding a table means bumping to 17 and adding `migrateV16ToV17`. `app/store/schema.sql` is frozen at v2 as a migration-consistency fixture and **must not be edited**.
- Settings live in a single-row `settings` table; served by `GET/POST /api/v1/settings` (`app/ui/ui.go:291-292`, handlers at `app/ui/ui.go:1439` and `:1461`).
- Frontend types are **generated**: `app/ui/ui.go:39` runs `tscriptify` over `app/ui/responses/types.go` into `app/ui/app/codegen/gotypes.gen.ts`. New API types must be declared in `responses/types.go` and regenerated, not hand-written in TypeScript.
- UI: `app/ui/app/src/components/Settings.tsx` + route `app/ui/app/src/routes/settings.tsx`; React 19, TanStack Router/Query, Tailwind, Headless UI, Heroicons.

### 2.4 The tool-schema type is a lossy subset of JSON Schema — this is a real constraint

`api/types.go:473-490`:

```go
type ToolFunctionParameters struct {
    Type       string             `json:"type"`
    Defs       any                `json:"$defs,omitempty"`
    Items      any                `json:"items,omitempty"`
    Required   []string           `json:"required,omitempty"`
    Properties *ToolPropertiesMap `json:"properties"`
}
```

and `ToolProperty` (`api/types.go:418-426`) carries only `anyOf`, `type`, `items`, `description`, `enum`, `properties`, `required`.

MCP servers publish an arbitrary JSON Schema as `inputSchema`. Keywords Ollama's type **cannot represent** include `additionalProperties`, `oneOf`, `allOf`, `not`, `const`, `format`, `pattern`, `minimum`/`maximum`, `minLength`/`maxLength`, `minItems`/`maxItems`, `default`, and `$ref` chains (`$defs` survives only as opaque `any`, with no way to reference it from a property).

Silently dropping constraints makes the model emit arguments the server will reject. The conversion is therefore a specified, tested component with an explicit policy — see §7 Phase 2 and §9.

### 2.5 Build and toolchain

- `go.mod`: `go 1.26.0`, ~20 direct dependencies, no MCP dependency present, no MCP module in the module cache.
- Build (`AGENTS.md`): `cmake -B build . && cmake --build build --parallel 8` then `./ollama serve`; Go-only iteration via `go build .`.
- **Go is not installed on this machine.** `go`, `/usr/local/go/bin/go`, `/opt/homebrew/bin/go` and `~/go` are all absent. `node` v25.8.2, `npm` 11.11.1 and Xcode are present. Nothing in this plan can be compiled, tested, or proven until Go 1.26+ is installed. This is Phase 0.

---

## 3. Prior art

### 3.1 Upstream already tried this, and the attempt is incomplete

`upstream/Parth/agents`, commit `5c0caaff` (ParthSareen, 2025-12-30): *"agents: add MCP server support and ENTRYPOINT command"*. It claims: `MCPRef` type, `MCP` command in Agentfiles, `mcpManager`, `/mcp` REPL commands, `ollama mcp` CLI, model-bundled and global `~/.ollama/mcp.json` servers, MCPs shown in `ollama show`.

What actually landed:

- `types/model/config.go` — `MCPRef{Name, Digest, Command, Args, Env, Type}`, with `Type` documented as *"currently only stdio is supported"*.
- `api/types.go` — `MCPRef` alias, `CreateRequest.MCPs`, `ShowResponse.MCPs`.
- `server/mcp.go` (315 lines) — the **packaging** half: `MediaTypeMCP`, tar.gz layer creation, blob extraction with path-escape checks, `mcp/`-namespaced registry references.
- `cmd/cmd.go` and `cmd/interactive.go` — call sites: `newMCPManager()`, `mcpMgr.loadMCPsFromRefs(...)`, `mcpMgr.ToolCount()`, `mcpMgr.Shutdown()`, `loadMCPConfig()`, `saveMCPConfig()`, `config.MCPServers[...]`, `MCPServerConfig{...}`.

What did **not** land: any definition of `mcpManager`, `newMCPManager`, `loadMCPConfig`, `saveMCPConfig`, or `MCPServerConfig`. `git grep` across the whole branch finds no file defining them. **The branch does not build.** The MCP client itself was never committed.

The branch is also *older than the architecture it would target*: it dates from 2025-12/2026-01, while the entire `agent/` harness (`Registry`, `Session`, approval, compaction) landed 2026-07-02. Its call sites hang off a pre-harness chat loop that no longer exists in this shape.

**Verdict: design reference, not a base.** We take from it (a) the config location and shape — `~/.ollama/mcp.json` with an `mcpServers` map keyed by name, fields `command`/`args`/`env`/`type`/`disabled`; (b) the CLI surface names — `ollama mcp add|remove|list|enable|disable` and a `/mcp` slash command; (c) the idea, deferred, that MCP servers can be *packaged and distributed* through Ollama's own registry. We reject (d) building on the branch, and (e) `MCPRef.Digest`/model-bundled MCPs for v1 — that is the packaging feature, and it multiplies scope.

This is also a live instance of the pitfall register's IP-007 (*a complete consumer with no producer is a facade*): the call sites read as finished work and would have been an inviting foundation.

### 3.2 The protocol and the SDK

- MCP spec revisions in the wild: `2024-11-05` (HTTP+SSE transport), `2025-03-26` (Streamable HTTP replaces HTTP+SSE), `2025-06-18`, `2025-11-25`, and `2026-07-28` (changes Streamable HTTP behaviour again). Version negotiation happens at `initialize`; a client that pins one revision will fail against half the ecosystem.
- `github.com/modelcontextprotocol/go-sdk` — the official Go SDK, jointly maintained with Google. Latest **v1.7.0, published 2026-07-27**. Its compatibility table states v1.7.0+ supports spec revisions `2026-07-28`, `2025-11-25`, `2025-06-18`, `2025-03-26`, `2024-11-05`. Packages: `mcp` (client/server), plus JSON-RPC, OAuth and OAuth-extension packages. Transports include `StdioTransport` and `CommandTransport` (subprocess) and the HTTP client transports; **client-side OAuth is marked experimental** at v1.7.0.

Hand-rolling stdio JSON-RPC is genuinely a day's work. Hand-rolling *five revisions of version negotiation, Streamable HTTP with session IDs and resumable event streams, and OAuth 2.1 with dynamic client registration* is not, and every subtle error in it is a security bug. See §5 Decision C.

---

## 4. Where the code goes

```
mcp/                                  NEW — shared core, no dependency on agent/ or app/
  config.go        ~/.ollama/mcp.json load/save (atomic), schema, validation, env expansion
  config_test.go
  server.go        ServerSpec, transport construction, connect/close, capability record
  manager.go       Manager: lifecycle for N servers, connect/retry/health, tool cache, close
  manager_test.go
  tools.go         MCP tool -> api.ToolFunction conversion (the lossy-schema policy, §2.4)
  tools_test.go
  names.go         namespacing + collision resolution ("<server>__<tool>")
  secrets.go       header/token resolution: env indirection, no plaintext secrets in mcp.json
  fake_test.go     in-process fake MCP server used by every layer's tests
  oauth.go         OAuth 2.1: metadata discovery, DCR, PKCE, loopback callback, refresh, revoke
  oauth_test.go
  keystore_darwin.go   Keychain-backed token storage
  keystore_windows.go  DPAPI-backed token storage
  keystore_other.go    build-tagged fallback: file-backed 0600, with a stated warning
  keystore_test.go
  registry.go      official MCP Registry client: search, versions, detail, package->command resolution
  registry_test.go

agent/tools/mcp.go            NEW — adapter: mcp.Tool -> agent.Tool (+ ApprovalRequired, ScopedTool)
agent/tools/mcp_test.go
cmd/agent_tui.go              EDIT — construct Manager, register adapted tools, close on exit
cmd/mcp.go                    NEW — `ollama mcp add|remove|list|enable|disable` cobra commands
cmd/cmd.go                    EDIT — wire the `mcp` command into the root command
cmd/tui/chat/input.go         EDIT — `/mcp` slash command (list/enable/disable) + help text

app/tools/mcp.go              NEW — adapter: mcp.Tool -> app/tools.Tool
app/tools/approval.go         NEW — the approval gap in Stack B (§6.3)
app/ui/ui.go                  EDIT — build Manager, register tools, approval round-trip, new routes
app/ui/responses/types.go     EDIT — MCP API response types (then regenerate gotypes.gen.ts)
app/store/database.go         EDIT — schema v17: mcp_server_state (UI-local state only)
app/ui/app/src/...            NEW/EDIT — settings UI (shape pending mock-up, contract in §8)

docs/mcp.mdx                  NEW — user documentation
```

**Why a new top-level `mcp/` package, not `agent/mcp/`:** the desktop app (`app/`) must import it, and `app/` importing `agent/` would drag the whole CLI harness — session, compaction, skills — into the app binary. `mcp/` depends only on `api/` and the SDK. Both stacks depend on `mcp/`; neither depends on the other. This mirrors how `api/` and `envconfig/` already sit beneath both.

**Why adapters rather than one shared `Tool` interface:** unifying `agent.Tool` and `app/tools.Tool` is a worthwhile refactor and a *different* piece of work. Attempting it inside this feature would put a large, upstream-divergent refactor on the critical path of a feature the user is waiting for. Two ~80-line adapters is the honest cost of not doing it. Recorded here so it is a decision, not an accident.

---

## 5. Decisions that are Ash's — **RULED 2026-08-05**

| Decision | Ruling | Effect |
|---|---|---|
| A — surfaces | **A2: desktop app + CLI agent TUI** | The shared `mcp/` package of §4 is mandatory, not merely preferable. |
| B — remote depth | **B3: full OAuth 2.1 with sign-in** | Largest single scope increase. Adds Phases 2b and 3d; see §6.4 and §7. |
| C — protocol | **C1: official `go-sdk` v1.7.0** | Pinned exactly; contained per §5.1. |
| D — upstream | **D2: aimed at an upstream PR** | Naming follows `5c0caaff`; commits stay small and separable; dependency surface is minimised. |

### 5.1 The two consequences that need writing down

**The dissent beat on B3.** The strongest case against full OAuth in v1: it is the only decision here that can double the calendar cost of the feature, and it does so in the part of the work with the least visible payoff. It brings a loopback redirect listener, dynamic client registration, token refresh, revocation, and OS-keystore storage on two platforms — and the SDK's client-side OAuth is marked *experimental* at v1.7.0, so we should expect to write and own a meaningful share of it rather than call into it. The cheaper path (B1, static credentials from env vars) reaches most hosted servers today and would let v1 ship, with OAuth added afterwards without redesign. That case is stated and the ruling stands: B3 is what makes "connect and sign in" work for consumer-facing servers, which is the experience a settings page implies. It is sequenced last (Phase 3d) so that a working, useful v1 exists before OAuth lands, and so OAuth slipping cannot hold the rest hostage.

**C1 against D2.** These pull in opposite directions: adding a substantial new dependency is the single most likely reason an upstream PR is refused, and upstream's own attempt at `5c0caaff` added no MCP dependency at all — its `go.mod` diff only promoted `gopkg.in/yaml.v3` from indirect. Rather than re-litigate the ruling, the plan contains the risk mechanically: **no SDK type may appear in any package outside `mcp/`.** `agent/`, `app/`, `cmd/` and `api/` see only our own types (`mcp.ServerSpec`, `mcp.Tool`, `mcp.Manager`) and `api.ToolFunction`. Enforced by a test that fails if any file outside `mcp/` imports `github.com/modelcontextprotocol/go-sdk`. The effect is that swapping to a hand-rolled or vendored implementation later — if upstream insists — is a single-package change with no call-site churn, so C1 stops being irreversible.

### 5.2 The decisions as they were put

### Decision A — Which surfaces get MCP in v1?

1. **Desktop app only.** Matches the mock-up, smallest surface. The CLI agent stays without MCP.
2. **Desktop app + CLI agent TUI** *(recommended)*. The shared `mcp/` package makes the second surface roughly one adapter plus wiring, and the CLI is where MCP is most useful. Same config file, so a server added in the UI is immediately available in the terminal.
3. **App + CLI + `ollama serve` daemon.** MCP tools injected server-side into `/api/chat`, so every API client (including third-party ones) inherits them. This is the most ambitious and the most dangerous: it makes a *shared background daemon* execute arbitrary local subprocesses on behalf of any HTTP caller, including ones with no user present to approve anything. It also changes `/api/chat` semantics for existing clients. I would not do it in v1, and if it is wanted it needs its own threat model.

### Decision B — How far does "internet-based" go in v1?

1. **Streamable HTTP with static credentials** — bearer token or custom headers, values resolved from environment variables, never stored in plaintext in `mcp.json` *(recommended)*. Covers the large majority of hosted MCP servers today.
2. **Add legacy HTTP+SSE (`2024-11-05`) fallback.** Small extra cost with the SDK; needed for older hosted servers.
3. **Full OAuth 2.1 with dynamic client registration and token refresh** — the hardest and most thorough option. This is what makes "click connect and sign in with your account" work for consumer-facing hosted servers. It brings a browser round-trip, a redirect listener, token storage (Keychain on macOS, DPAPI on Windows), refresh handling, and revocation. The SDK's client-side OAuth is *experimental* at v1.7.0, so this is the option most likely to need our own code and most likely to churn.

### Decision C — Official Go SDK, or hand-rolled protocol?

1. **`github.com/modelcontextprotocol/go-sdk` v1.7.0** *(recommended)*. Five spec revisions negotiated for us, both transports, maintained. Cost: one substantial new dependency in a project with ~20, and a version-churn surface.
2. **Hand-roll stdio-only JSON-RPC now, revisit for HTTP.** Zero new dependencies, matches upstream's minimal-deps instinct, and keeps a future upstream PR easier to land. Cost: we own version negotiation and every transport bug, and Decision B options 2–3 become expensive.
3. **Vendor a pinned copy of the SDK** into `mcp/internal/`. Dependency-free `go.mod`, full control of the version, at the price of manual updates and a large diff.

### Decision D — Is this fork's MCP work aimed at an upstream PR?

1. **Ash's fork only.** We optimise for the best design and Ash's needs; rebases on upstream are our problem.
2. **Aimed at upstream** *(no recommendation — this is a product/relationship call)*. Changes the plan materially: minimise dependencies (pushes Decision C toward 2), match upstream's existing naming from `5c0caaff`, keep commits small and separable, and expect the design to be negotiated by people who already have opinions about it.

*(Ruled A2, B3, C1, D2 on 2026-08-05 — see §5 header.)*

---

## 6. The security model — the part that matters

An MCP server is *arbitrary code Ollama launches* (stdio) or *an arbitrary network peer whose text lands in the model's context* (HTTP). Three distinct exposures, all of which must be answered before Phase 3.

### 6.1 Execution exposure

A stdio server is a subprocess with the user's full privileges. `mcp.json` is therefore a code-execution config file: anything that can write it can run code as the user, next time Ollama starts.

- Config file written `0600`, its directory `0700`.
- No shell interpretation. `exec.Command(cmd, args...)` only — never `sh -c`.
- Servers are **disabled on first sight**: a server newly added to `mcp.json` by anything other than Ollama's own UI/CLI is listed but not connected until the user enables it. (Cheap, and it defeats the "someone dropped a server into your config" path.)
- Env passed to the child is explicit and minimal, not the parent's full environment.

### 6.2 Context exposure (tool poisoning)

Tool *names and descriptions* from a remote server are injected into the model's prompt. A hostile or compromised server can put instructions there. This is the documented MCP attack class, and it is not hypothetical.

- Descriptions are treated as untrusted data: length-capped, control characters stripped, and rendered in the UI so the user can read what a server is actually telling the model.
- Tool names are namespaced `<server>__<tool>` (`names.go`). No MCP server can shadow `bash`, `read`, `edit`, `web_search`, `web_fetch`, or the browser tools; first-party names are reserved and a collision is resolved in favour of the first-party tool with a visible warning.
- **Description change detection**: the tool list and description hashes for each server are recorded. If a server's advertised tools change between sessions, the user is told before those tools are offered again — this is the "rug pull" defence.

### 6.3 The Stack B approval gap

Stack A has an approval system; Stack B (`app/tools`) has none, and `app/ui/ui.go:1030` executes whatever the model names. Registering MCP tools in Stack B without first closing this gap would give a remote server unreviewed execution inside the desktop app.

So Phase 3 in the app is *two* pieces of work: an approval path in Stack B, then MCP registration behind it. The approval path should mirror `agent/approval.go`'s model — per-scope grants, an "always allow this tool from this server" state persisted in `mcp_server_state`, and blanket allow only as an explicit user act — and surface over the existing SSE channel as a new `tool_approval` event with a `POST /api/v1/chat/{id}/approval` response. Concretely: an MCP tool is `ApprovalRequired` by default; its `ApprovalScope` is `<server>__<tool>`; and a server may be marked trusted-by-user to auto-approve its whole scope set.

I want to be plain about the trade: this is the single largest piece of work in the plan and it is not visible in a mock-up. It is also the difference between a feature and a vulnerability.

### 6.4 Network exposure, and the OAuth surface (ruling B3)

- Remote servers: HTTPS only unless the host is loopback. No plaintext credentials in `mcp.json` — a header value of the form `${env:VAR_NAME}` is resolved at connect time and the literal form is rejected with a clear error.
- Per-server timeouts and a response size cap, because a remote tool result flows straight into the context window.

Full OAuth 2.1 brings its own obligations, and each is a place where a wrong default is a credential leak:

- **PKCE is mandatory** (S256), on every flow, no exceptions. An authorization-code flow without PKCE in a native app is broken by construction.
- **Redirect URI is loopback only** — `http://127.0.0.1:<ephemeral>/callback`, bound before the browser is opened, torn down immediately after, with a `state` parameter checked on return. Never a custom URL scheme; never a wildcard port.
- **Tokens never touch `mcp.json`.** Access and refresh tokens go to the OS keystore — Keychain on macOS, DPAPI on Windows — keyed by server name. `mcp.json` holds only the issuer URL, client ID and scopes.
- **Authorization-server metadata is discovered, not configured** (RFC 8414 / protected-resource metadata), and the resource indicator is sent so a token minted for one server cannot be replayed at another.
- **Refresh and revocation are first-class**: expiry is handled inside the transport with a single-flight refresh, and "disconnect" in the UI revokes at the server and deletes the keystore entry, not just the local record.
- **The browser round-trip is an explicit user act.** Ollama never opens a browser because a model asked it to — only because the user pressed connect.
- **Dynamic client registration (RFC 7591)** is attempted where the server advertises it, with the registered client credentials stored in the keystore alongside the tokens; a server without DCR falls back to a user-supplied client ID.

---

## 7. Implementation phases

Each phase states its files, its obligations, and the falsifiable proof that closes it. No phase is "done" on compilation. Every phase ends in a commit signed `Co-Authored-By: Grommet (Claude Opus 5)`; the branch is pushed at every phase boundary.

### Phase 0 — Toolchain and baseline — **DONE 2026-08-05**

Go 1.26.5 and CMake 4.4.2 installed via Homebrew. Baseline captured at commit `a734e3b1` into `docs/_design/proof/phase0-baseline.txt` (build, vet, frontend install/build/test/lint/format) and `phase0-gotest.txt` (full Go suite), with an explicit exit code echoed after every command.

**What the baseline established, and why each item matters later:**

- `go build ./...` fails on a clean checkout with `app/ui/app.go:15:12: pattern app/dist: no matching files found`. The desktop app embeds the built frontend, so **the frontend must be built before the Go tree will compile**. `npm ci && npm run build` in `app/ui/app` fixes it; after that the build is clean (cgo warnings from `app/webview` only). Any future "the build is broken" report has to check this first.
- `go vet ./...` **already fails**: `tokenizer/bytepairencoding_test.go:542:5: result of slices.Collect call not used`. Pre-existing, not ours, not to be fixed here — fixing unrelated files works against the upstream-PR ruling (D2).
- Frontend tests pass: 4 files, 27 tests.
- `npm run lint` **already fails**: 122 problems (112 errors, 10 warnings) across 14 files including the generated `codegen/gotypes.gen.ts` and `src/components/ChatSidebar.tsx`. `npm run prettier:check` **already fails** on 9 files, also including `ChatSidebar.tsx`, `Settings.tsx` and `api.ts`.

**The constraint that follows, and it is load-bearing.** `ChatSidebar.tsx` is a file this feature must edit (§8.0) and it is both lint-dirty and format-dirty today. Running `prettier --write` on it would reformat the entire file and bury a three-line change in a hundred-line diff. So: **new files must be lint-clean and prettier-clean; existing files are edited by hand in their surrounding style and never reformatted wholesale.** The completion test for lint is "no new problems against the recorded baseline count", not "zero problems".

### Phase 1 — `mcp/config.go`: the config file — **DONE 2026-08-05** (`f984d4ec`)

Delivered as specified, with three corrections to the plan discovered in the writing:

- **It cannot use `cmd/internal/fileutil`.** Go's `internal` rule puts that package out of reach of a top-level `mcp/`. Moving it to `internal/fileutil` would touch ten files across `cmd/` and enlarge an upstream-aimed diff, so `mcp/` writes atomically itself (temp file, `Chmod(0600)`, sync, rename) into a `0700` directory.
- **`Load` does not fail on a bad server.** It fails only when the file cannot be read or parsed; per-server faults come back from `Problems()` keyed by name, so one malformed paste cannot make every other server unreachable. Callers must not connect a server that appears in `Problems()`.
- **Unknown fields are preserved** at both the top level and inside a server object, so a config written by a newer Ollama or another client is not stripped on save.

Config path resolves `OLLAMA_MCP_CONFIG`, then `XDG_CONFIG_HOME`, then `~/.ollama/mcp.json`, mirroring `agent.SkillsDir()`.

Proof in `docs/_design/proof/phase1-falsification.txt`: 15 tests, then file mode weakened to `0644`, the literal-credential check removed, and unknown-field preservation disabled — each observed to fail the test that catches it, then restored byte-identical.

### Phase 1b — the approval ledger — **DONE 2026-08-05** (`b672284c`)

Writing Phase 1 exposed a flaw in §6.1 as originally drafted. "Servers are disabled on first sight" cannot be expressed through the `disabled` field: every other MCP client treats absent-`disabled` as enabled, so inverting that default would break the paste-a-config-block flow that makes the feature usable, and would surprise anyone hand-editing the file.

The mechanism that achieves the same protection without fighting the convention is a **separate approval ledger** at `~/.ollama/mcp-approvals.json`, mode `0600`:

- Keyed by server name, storing a SHA-256 over the spec's executable surface — command, args, resolved transport, and URL. Not over the whole spec: changing a description or a non-executable field should not demand re-approval.
- A server whose current spec hash is absent from the ledger is **listed but never connected**, whatever `disabled` says.
- Adding a server through Ollama's own UI or CLI records the hash in the same action, so the ordinary path has no extra step.
- A spec edited underneath Ollama — by hand, by another tool, or by malware — produces a hash mismatch and returns to needing approval. This is the same mechanism as the tool-description change detection in §6.2, applied to the command line instead of the tool list.

Built as designed, with two refinements. The fingerprint covers the **whole serialised spec except `disabled`** rather than a hand-listed set of fields, so it cannot drift as the type grows, and an unknown future field is hashed on the principle that we cannot know whether it is executable and must not assume it is not. And a **nil policy denies everything**: a caller that has not thought about approval gets a manager that connects to nothing, with the blanket policy used by this package's own tests kept unexported so no production caller can reach it.

Proof in `docs/_design/proof/phase1b-falsification.txt`: a nil policy made permissive, the fingerprint reduced to the server name, the approval gate removed, and the on/off switch folded into the fingerprint — each observed to fail.

### Phase 2a — namespacing and schema conversion — **DONE 2026-08-05** (`7dbec4fa`)

`mcp/names.go` and `mcp/tools.go`, both written without the SDK: `mcp.Tool` is our own type, so the conversion layer is testable before any protocol code exists and the §5.1 containment rule is satisfied by construction rather than by discipline.

Two additions to what the plan specified. Local `$ref` resolution against `$defs` and `definitions` is implemented, because pydantic and zod emit that shape and most real MCP servers are built on one of them — without it their tools arrive untyped and unusable. Remote `$ref`s are refused and reported, since a tool definition must never cause a network fetch, and cyclic references degrade one property rather than hanging or losing the tool.

Proof in `docs/_design/proof/phase2a-falsification.txt`: package now 29 tests (74 with subtests); control-character stripping, the lost-constraint note, sorted iteration and the reserved-name check each broken and observed to fail, then restored byte-identical.

### Phase 2 — `mcp/` core: manager and transports — **DONE 2026-08-05** (`594f6104`)

`mcp/server.go` and `mcp/manager.go`. Delivered as specified, with four things worth recording.

- **The dependency's real weight.** `go-sdk` v1.7.0 brings six indirect dependencies (`google/jsonschema-go`, `segmentio/asm`, `segmentio/encoding`, `yosida95/uritemplate/v3`, `golang.org/x/oauth2`, `golang.org/x/time`) and lifts `golang.org/x/sync` to v0.20.0 and `golang.org/x/sys` to v0.41.0. That is the predicted cost of ruling C1 against ruling D2, and the §5.1 containment test is what keeps it reversible.
- **Retry diverges from the plan, deliberately.** HTTP connections retry three times with doubling backoff; stdio does not retry at all. A failed `exec` is deterministic — the command is missing, or it is not executable — and retrying it only spawns processes. Proven by a test asserting a missing command fails in under two seconds.
- **A real defect, found and fixed.** The server process was tied to the connect-timeout context, which is cancelled on the way out of the dial. Every server therefore died the instant its handshake completed — indistinguishable from a server that connected. It is now tied to the manager's lifetime, and the defect is reintroduced as F8 in the proof file and observed to fail.
- **Three protections could not be falsified, so they were removed.** Attempts to prove Ollama-side process reaping all passed with the mechanism deleted: the library's `CommandTransport` closes stdin, sends `SIGTERM`, then kills, and tears the transport down when a handshake fails. The per-server cancellation was removed rather than kept. Unfalsifiable code that reads as a safety measure is worse than none, because the next reader believes the safety is ours. The two behavioural no-orphan tests remain, since the promise must keep holding if a transport is ever swapped.

Proof in `docs/_design/proof/phase2-falsification.txt`: 47 tests, 104 with subtests, against **two** peers — an in-process server built on the library over its in-memory transport, and a hand-written server in `mcp/testdata/rawserver` with no protocol library at all, compiled and launched as a real subprocess. The second is what proves the client works against an implementation that is not itself, and it is the only way to exercise what a well-behaved SDK server refuses to emit: a non-object input schema, a tool named `bash`, and a description carrying escape and NUL bytes.

Proof (this is the phase where a facade is easiest and least visible):
- An **in-process fake MCP server** (`fake_test.go`) that speaks real JSON-RPC over an in-memory pipe: initialise, list tools, call a tool, return an error, hang past timeout, close its pipe mid-call, and advertise a changed tool list on reconnect. Every manager behaviour is proven against it.
- A **real subprocess test** launching an actual stdio MCP server binary and calling a real tool — the test a mock cannot pass. Gated behind a build tag if it needs network/npx, but it exists and it is run.
- A conversion test asserting a schema with `oneOf`, `minimum` and `$ref` produces either a faithful `api.ToolFunction` or a documented, asserted degradation — never a silently wrong one.
- A process-reaping test: kill the parent context, assert no orphaned child remains.

A containment test also lands here: a test that walks the module and **fails if any file outside `mcp/` imports `github.com/modelcontextprotocol/go-sdk`** (§5.1).

### Phase 2b/3d — OAuth 2.1 — **REDIRECT DONE 2026-08-06** (`da3b9da1`)

**Scope correction, discovered by reading the SDK rather than assuming.** The plan assumed we would write metadata discovery, dynamic client registration, PKCE, the token exchange and refresh. The protocol library's `auth` package already does all of that through `AuthorizationCodeHandler`, which satisfies the `OAuthHandler` the streamable transport accepts. What it explicitly leaves to the caller is the redirect: *"The caller is responsible for handling the redirect out of band."*

So the honest remaining work is three things, not one: the loopback redirect (**done**), token storage (**done, with a caveat below**), and the connect/disconnect surfaces.

**Token storage** (`0f82fb9b`): a `TokenStore` interface and a file-backed implementation, 0600 in a 0700 directory, keyed per server, written atomically, with the persisted shape declared field by field so a change to `oauth2.Token` cannot silently alter the file. Not signed in (`ErrNoToken`) and could-not-read are deliberately distinct: collapsing them would sign the user out of everything and then overwrite what was there. An empty token is refused, because a stored empty credential looks exactly like being signed in.

**There is no Keychain-backed store, and that is a finding rather than a shortcut.** The obvious cgo-free route to the macOS keychain is the `security` command, whose `add-generic-password` takes the secret as a command-line argument with no stdin form. A probe on this machine ran it with a known value and grepped `ps -ax -o command` while it ran: **the secret was in the process list, readable by any local user.** That is worse than a `0600` file, so the CLI route is refused. A Keychain and DPAPI store needs cgo and is owed; the interface is shaped so adding one changes nothing else, and `FileTokenStore.Description()` says plainly that file permissions are the only protection — a test fails if that description ever starts implying the keychain.

`mcp/oauth.go` binds 127.0.0.1 on an ephemeral port and satisfies `sdkauth.AuthorizationCodeFetcher`. The state is checked at the listener **as well as** by the library, and the difference is the point: a callback with the wrong state is refused *and does not end the wait*, so a stray or hostile request on the loopback port cannot abort a sign-in in progress. After a flow completes the expected state is cleared, so a replay is refused. The browser is opened only here, only by an explicit sign-in, and only for http and https. The wait ends on a redirect, the context, or five minutes, and the last two refuse. The listener is torn down on Close.

Proof in `docs/_design/proof/phase3d-falsification.txt`: six protections falsified, driven with an HTTP client rather than a browser.

### Phase 2b (original spec) — *(ruling B3)*

`mcp/oauth.go` and the platform keystores. Authorization-server metadata discovery, dynamic client registration where advertised, PKCE S256, loopback redirect with `state` verification and immediate listener teardown, token persistence in Keychain/DPAPI, single-flight refresh inside the transport, and revoke-on-disconnect. The SDK's OAuth package is used where it is sound at the pinned version and replaced with our own where its experimental status shows; whichever way each piece goes is recorded in the commit, not left implicit.

Proof:
- A **fake authorization server** in-process: issues codes, rejects a mismatched `state`, rejects a missing or wrong PKCE verifier, expires an access token so refresh must fire, and rejects a revoked refresh token. Each is a separate test with an asserted failure, not a happy path with error branches unvisited.
- A test asserting **no token ever appears in `mcp.json`** — write a config after a full flow, read the file bytes, assert the token substring is absent.
- A test asserting the loopback listener is **closed** after the flow, and that a second unsolicited request to the callback path after completion is refused.
- A keystore round-trip test per platform, with the non-Keychain/DPAPI fallback asserting `0600` and emitting its warning.

### Phase 3a — CLI surface *(ruling A2)* — **PARTLY DONE 2026-08-05** (`12ea7760`, `09a954cb`)

Done: `agent/tools/mcp.go`, the adapter implementing `agent.Tool`, `ApprovalRequired` and `ScopedTool`; and `cmd/agent_tui.go`, which builds one manager per session, consults the ledger, connects approved servers, reports every failure and every skipped tool, and registers the rest. `OLLAMA_AGENT_DISABLE_MCP` switches it off.

Proof in `docs/_design/proof/phase3a-falsification.txt`: the full signal chain — scripted model turn → `agent.Session` → `agent.Registry` → adapter → manager → **real `rawserver` subprocess** → reply asserted in the run's messages, with nothing mocked but the model; a second test proving a declined approval stops the call; and activation evidence at the entry point itself, driving `agentToolsRegistry` and asserting the MCP tool is present and the refused ones absent. Six protections falsified, and two falsifiers that initially passed led to test repairs rather than softened claims.

Also done (`7e533582`): `ollama mcp list|add|remove|enable|disable|approve|revoke`, registered on the root command and proven by tests that drive the real cobra tree against isolated config and approval files. `add` approves what the user typed, since the command line came from their own keyboard and requiring a second command there is friction without security gain; `--no-approve` opts out. `approve` is what the ledger actually defends: it prints the resolved command line, environment and headers verbatim, shows the previously-approved command line when one has changed, and refuses to assume yes when there is no terminal to ask on. `remove` drops the approval with the server, so a future server reusing the name and command cannot inherit it. Five protections falsified, including the group being unregistered from the root command — a command tree that exists and is not hung off `ollama` is unreachable.

Also done (`f12a2d6a`): `/mcp` in the agent TUI, listing every server with its status, command line, tool count and skipped tools, and `/mcp enable|disable` to toggle one. It cannot approve, and a test guards that decision rather than the implementation. The TUI receives two narrow closures rather than the manager, so it cannot close a manager it does not own; after a successful toggle the registry and system prompt are rebuilt through the model-switch path, and after a failed one nothing is rebuilt and the reason reaches the user.

Wiring it exposed a defect in the manager, now fixed: `Connect` marked a newly-disabled server as disabled and dropped its tools but left its session open, so its process ran on for the rest of the session — a switch that switched nothing off, and nothing else would have noticed. `Connect` now closes the session of any server it is not connecting.

**Phase 3a is complete.**

### Phase 3b — Desktop app approval path — **BACKEND DONE 2026-08-06** (`0506f8e7`)

Delivered: `app/tools/approval.go` (the rendezvous registry, per-chat state, the two capability interfaces), the `tool_approval` SSE event in `responses.ChatEvent`, `POST /api/v1/chat/{id}/approval`, the gate in the streaming tool loop, cancellation on chat delete, and construction in `app/cmd/app/app.go`.

The design point worth keeping: the two halves live in **different HTTP requests** — the call blocks inside an open streaming response, the answer arrives on a separate POST — so everything about the wait is written to fail safe. It ends on an answer, the caller's context, or a ten-minute timeout, and the last two refuse. Every exit removes the pending entry so a stale answer cannot resolve a request nobody is waiting on; the answer channel is buffered so resolving never blocks; `Resolve` removes the entry itself so a second answer is refused rather than racing the first; and an approval identifier carries its chat, so one conversation cannot answer another's question.

Approvals are per chat and in memory. "Remember" grants one scope, "remember all" grants everything and is set by nothing but an explicit user act, and a refusal grants neither even when the flags are present. Tools that do not implement `ApprovalRequired` are never gated, so the first-party tools are untouched.

Proof in `docs/_design/proof/phase3b-falsification.txt`: 29 tests including a full round trip, plus refusal, cross-chat refusal, timeout, cancellation, out-of-order concurrent answers and no-leak assertions. Six protections falsified.

Also done (`e7a05864`): the React prompt. A gated call raises a panel above the composer naming the tool, listing its arguments, and offering allow once, always allow this tool, or decline. Arguments are shown because the user is agreeing to *this call*; they are stripped of control characters and capped with a visible marker, and a test asserts that markup in a tool name or argument renders as text. A refusal never carries a remember flag on the wire. An event with no identifier is ignored rather than shown, since an unanswerable prompt would strand the chat until the server's timeout, and a 409 is surfaced rather than swallowed. `gotypes.gen.ts` was regenerated with `tscriptify`, a seven-line diff.

Bounding what the proof covers: the frontend suite runs in a node environment and renders with `renderToStaticMarkup` — the pattern already used here — so there is **no DOM event simulation and a button click is not exercised**. The handlers are one call each and everything they call is tested directly.

**Phase 3b is complete.** The app can now gate a tool call end to end, so Phase 3c may register MCP tools.

### Phase 3c — Desktop app MCP registration — **DONE 2026-08-06** (`7b6bfc76`)

`app/tools/mcp.go` adapts `mcp.Tool` to the app's `Tool` interface with `ApprovalRequired` and `ScopedTool`; the manager is built once where the server is constructed in `app/cmd/app/app.go` and closed on shutdown; `Server.registerMCPTools` adds its tools to each request's registry. `OLLAMA_DISABLE_MCP` switches it off.

**A defect in the app's own tool plumbing had to be fixed first.** `convertToOllamaTool` rebuilt every tool definition from a map carrying only each property's type and description, discarding enums, array item types, nested objects and alternatives — and emitting properties in Go map order, so the schema sent to the model differed between requests. Tools holding a faithful `api.ToolFunction` now supply it through a new `OllamaTool` interface; the rest derive theirs through a sorted, deterministic replacement. The dead helper was removed.

Proof in `docs/_design/proof/phase3c-falsification.txt`: tests run against the real `rawserver` subprocess, with activation evidence that the registry the chat handler builds contains the server's tools and excludes the refused ones, and that an MCP tool in the app goes through the approval gate. Four protections falsified; a fifth initially passed and the test was repaired — it had asserted only what the lossy path also preserves.

### Phase 3c (original) — settings API

`app/tools/mcp.go` adapter; Manager construction in `app/ui/ui.go` (process-lifetime, not per-request — connecting servers per chat request would relaunch subprocesses on every message); routes per §8; `responses/types.go` additions + `tscriptify` regeneration; schema v17 `mcp_server_state` table (UI-local state: trusted flag, last-seen tool hash, last error — the *config* stays in `mcp.json`).

Proof: handler tests for every new route; a migration test proving v16→v17 preserves existing rows (the store already has `migration_test.go` conventions to follow).

### Phase 3d — OAuth in the surfaces — **DONE 2026-08-06** (`75dc07cd`, `3104fd4e`)

The routes are `POST /api/v1/mcp/{name}/signin` and `/signout`, not `/connect` and `/disconnect`. The act is unchanged — an interactive sign-in, and a sign-out that revokes then deletes — but "connect" would name it after the ordinary connection it is deliberately not. The CLI is `ollama mcp login <name>` / `logout <name>` as specified.

**The distinction the phase turns on: an ordinary connection must never be able to open a browser.** Servers connect at start-up, on a configuration change and on every reconnect. A 401 answered the ordinary way would open a sign-in page on a machine nobody is sitting at, or three of them, since http connections are retried. So `signInDisallowed` starts no redirect listener and fails with `ErrSignInRequired`; `signInAllowed` is reachable only from `Manager.SignIn`, which is reachable only from the connect button and `ollama mcp login`. A server that needs a sign-in gets `StatusNeedsSignIn` rather than `StatusFailed` — they ask the user for different things — and is not retried, because a server that asks will ask again on every attempt.

**Signing out revokes.** Forgetting a token locally leaves it valid at the service while the user believes they withdrew it. `SignOut` discovers the revocation endpoint through RFC 9728 resource metadata and RFC 8414 authorization-server metadata, revokes the refresh token in preference to the access token (RFC 7009 §2.1: a server SHOULD invalidate the access tokens issued from it), and deletes locally whether or not that succeeded — with `ErrSignedOutLocallyOnly` so the surfaces can say which of the two happened.

**This forced a change to the token store, and it had to happen before any token was written.** RFC 7009 identifies the client, Ollama is a public client, and dynamic client registration issues a fresh identifier every time — so registering again at sign-out would identify a different client and revoke nothing. The identifier is visible exactly once, in the `oauth2.Config` handed to `NewTokenSource`. `TokenStore` therefore keeps a `SignInRecord`, not a bare `oauth2.Token`, and a refresh that arrives without an identifier does not erase the recorded one. Had this been deferred, every token written in the meantime would have been unrevocable for its whole life.

**The app route returns as soon as the sign-in has started.** Holding an HTTP request open for the minutes a person spends in a browser puts the page at the mercy of every idle timeout between it and the process; the page follows the server's status instead, which is where the outcome appears either way. One sign-in at a time per server, enforced in the manager so both surfaces get it.

Proof in `docs/_design/proof/phase3d-oauthhandler-falsification.txt`, `phase3d-signin-falsification.txt` and `phase3d-wiring-falsification.txt`: thirty protections falsified across the handler, sign-in, sign-out and the wiring. Two tests were rewritten before the first pass because they did not discriminate, and the record says which and why.

**The whole flow now runs end to end** (`630974c0`), against a fake authorization server in `mcp/oauthflow_test.go` implementing RFC 9728, 8414, 7591, 7636 and 7009: a 401 with a challenge, discovery, dynamic registration, an authorize endpoint that redirects into Ollama's own loopback listener, a token endpoint that verifies the PKCE verifier against the committed challenge, refresh, and revocation. The fake user is already signed in and is redirected straight back, which is the timing that catches a listener bound after the browser was opened.

**Writing it found the defect this phase's whole design rested on.** The protocol library performs discovery and dynamic client registration *before* it asks whether a browser may be opened. Refusing at the fetcher therefore came too late: every launch and every reconnect of a remote server the user had never signed in to announced this installation to the authorization server and left a client registration behind. `signInRequiredTransport` now answers the challenge at the RoundTripper, so the library never reaches its authorization branch, and a handler is attached only when there is a stored token to send. The ordinary path touches the MCP endpoint and nothing else.

It also found a redundant second writer for the stored token, deleted after two attempts to falsify it both passed — the same rule that removed the reaping code in Phase 2.

**Owed: a real service.** A fake authorization server agrees with whatever you built. Nothing has been signed in to for real, and that is where the remaining surprises are. Also owed: the Keychain and DPAPI store, which needs cgo.

### Phase 4 — The MCP Servers page — **DONE 2026-08-06** (`273c19bd`)

Delivered as §8.0–8.3 specify: a top-level sidebar destination after Launch with the storefront icon, a `/mcp` route, and a page that lists every server with what it runs, its state, its tools and its refused tools, and can approve, switch, remove and add from a pasted configuration block. The sidebar's active highlight is now route-derived.

**The page may approve where `/mcp` in the chat may not**, and the difference is made real rather than nominal: approving sends back the command line the page displayed, and the server refuses with a conflict when it no longer matches disk. A stale page cannot approve something the user never read. Adding from the app does not approve, because the command line arrived over HTTP rather than from the keyboard.

**A real defect was found and fixed here.** The manager snapshotted the approval ledger when built, so approving from the app could never connect anything — the policy answered from the ledger as it stood at launch. `mcp.ApprovalsFile` now re-reads on every question and both surfaces use it.

**Not built, deliberately:** schema v17 and `mcp_server_state`. Neither the trusted flag nor the last-seen digest has a consumer — approvals live in the ledger, the digest is computed live — and an unused table is a facade. Build it when something reads it.

### Phase 4 (original spec) — *(mock-up received; §8)*

A new `/mcp` route and page component, plus the sidebar entry after Launch with the storefront icon, plus route-derived active state so Chat de-highlights when MCP Servers is open. Not a Settings tab — this was the mock-up's main correction to the plan.

Proof: Vitest coverage of the list / add / edit / remove / enable flows against a mocked API; a test asserting the sidebar's active row follows the route rather than `currentChatId`; Storybook entries for connected, connecting, failed, needs-approval, needs-sign-in and empty states; and a manual run of the packaged app with a real local MCP server, screenshotted.

### Phase 4b — Registry browse and install — **BACKEND DONE 2026-08-06** (`8317a273`, `07b06d8d`)

`mcp/registry.go` searches and paginates the official registry and resolves an entry into an ordinary `ServerSpec`; `app/ui/mcp_registry.go` exposes browse and resolve.

The install gate lives in `Resolve`. npm becomes `npx -y` with the pinned version, pypi becomes `uvx` with a version pin, oci becomes `docker run` with the publisher's runtime arguments, and a **hosted endpoint is preferred over any of them** because running nothing on the user's machine is safer than running something. An ecosystem Ollama does not know how to run is **refused rather than guessed at**, and an entry with neither a package nor a remote likewise — inventing a command line for an unverified runner is how a user approves something that does not do what its name suggests. A declared secret is never written down: it becomes an `${env:NAME}` reference, and a test asserts every resolved spec passes exactly the same validation as one a human typed, so a registry entry cannot smuggle in what the configuration layer would refuse.

Every response carries `notVetted: true` as a **field rather than a rendered sentence**, so the honesty of the surface cannot lapse by someone forgetting to display it. Entries that cannot be installed are returned with the reason instead of being hidden, and carry no command line. Suggested names are derived to pass the configuration layer's own rules. An unreachable registry returns a gateway error, because nothing-found and could-not-ask must not look the same.

Installing is deliberately **not** a new route: an entry becomes an ordinary `POST /api/v1/mcp`, so it lands unapproved and goes through the same reading and agreement as a hand-typed server, and the approve endpoint's shown-value check applies unchanged.

Proof in `docs/_design/proof/phase4b-falsification.txt`: recorded fixtures in `mcp/testdata/registry`, never the live registry. Nine protections falsified across the client and the routes.

Also done (`0c540dac`): the browse surface. Results carry the publisher namespace, the repository, and the exact command line; clicking add re-resolves on the server rather than trusting a row that may be minutes old, shows that command line in a confirmation with the environment values the user must set, and then installs through the **ordinary add endpoint** so the entry lands unapproved and faces the same agreement as a hand-typed server.

**One correction made in the writing.** The first version sent only the rendered command line to the browser and split it back apart to build the install request — two implementations of one resolution, free to drift, with quoting silently mangling any argument containing a space. The resolved specification is now carried structurally alongside the string: `runs` is what the user reads, the fields are what gets stored, from one resolution.

**Phase 4b is complete.** Phase 3d is complete too; what remains in the whole plan is the full-flow OAuth proof, the keychain store, and Phase 5 closeout.

### Phase 4b (original spec) — *(ruling §8.4)*

`mcp/registry.go` — a typed client for `/v0/servers` with search, cursor pagination, version listing and detail; package-to-command resolution per `registryType`; the three routes of §8.2; and the browse surface in the page with its search field, entry cards showing publisher namespace and repository, the not-vetted notice, and the install confirmation that shows the resolved command line verbatim.

Proof:
- Client tests against **recorded registry fixtures** (committed JSON, not live network) covering pagination via `nextCursor`, empty results, a malformed entry, and an entry with both `packages` and `remotes`.
- A resolution test per `registryType` asserting the exact argv produced, including a case where resolution is impossible and the entry must be shown as un-installable rather than guessed at.
- A test asserting an installed entry lands in `mcp.json` **disabled**, and a test asserting no code path enables a server as part of install.
- A `fileSha256` mismatch test asserting a hard failure.
- An offline test asserting the manager half of the page still lists and connects configured servers when the registry is unreachable.

### Phase 5 — Documentation, cross-substrate review, closeout

`docs/mcp.mdx`; a Codex review of the whole diff (different substrate, per KANON 19 and the union-not-count lesson — a second Claude is not a second instrument); series closeout reconciling every claim in this document against the live tree; retro entries.

---

## 8. The UI contract — **mock-up received 2026-08-05**

### 8.0 What the mock-up settles

MCP is a **top-level destination in the sidebar**, not a panel inside Settings. The mock-up shows the sidebar header carrying three entries — `Chat`, `Launch`, and `MCP Servers` — above the `Today` / `This week` / `Older` chat groups, with `MCP Servers` drawn using the Heroicons 24/outline building-storefront glyph.

That is a correction to this document's earlier assumption, and it changes the work:

- **Insertion point**: `app/ui/app/src/components/ChatSidebar.tsx`, in the `<header>` block that today holds the New Chat link (line ~268) and the Launch link (line ~286, `RocketLaunchIcon`), immediately after Launch and before the Windows-only Settings link.
- **Route**: a new file `app/ui/app/src/routes/mcp.tsx` registering `/mcp`, with the page component in `app/ui/app/src/components/MCPServers.tsx`. TanStack Router generates `routeTree.gen.ts`, so the route file is the whole registration. Note that Launch is *not* a route — it is `/c/launch`, a chat id with special handling — so MCP Servers is the first genuine second destination in this app besides Settings, and its page owns its own scroll and empty states rather than borrowing the chat layout's.
- **Icon**: `BuildingStorefrontIcon` from `@heroicons/react/24/outline`, `className="h-5 w-5 stroke-current"`, matching Launch exactly.
- **Active state**: the sidebar's active highlight is currently driven by `currentChatId`, which cannot express a non-chat route. The MCP entry needs route-derived active state (`useMatchRoute` or equivalent), and the Chat entry must *lose* its highlight when `/mcp` is open — the mock-up shows exactly one active row.

**One delegated default.** The mock-up labels the first entry `Chat` with a speech-bubble icon; the live app labels it `New Chat` with a compose glyph. Renaming an existing navigation item is not part of MCP support, so the plan leaves it untouched. Say the word and it becomes a one-line change.

**One question left open** — whether the storefront icon means a *manager* or a *directory*: see §8.4. It is raised as a chip rather than assumed, because it is the difference between a settings page and a distribution feature.

### 8.1 Entities the UI must express

- **Server**: name, transport (local command vs remote URL), the command line or URL shown verbatim, enabled toggle, trusted-tools toggle, per-server error text.
- **Connection state**: `disabled` · `connecting` · `connected` · `failed(reason)` · `needs-approval` (config present but never enabled) · `needs-sign-in` (remote server requires OAuth and has no valid token) · `signed-in-as(account)` where the server reports an identity.
- **Tool list per server**: name, description as advertised (marked as coming from the server, not from Ollama), and a warning badge when a description has changed since last session.
- **Add flows**: local (command + args + env vars) and remote (URL + header credentials by env-var reference). Paste-a-JSON-block should be accepted, because that is how every other client distributes server configs.

### 8.2 Endpoints (all under the app's authenticated `/api/v1` surface)

```
GET    /api/v1/mcp                 list servers with state + tools
POST   /api/v1/mcp                 add a server
PUT    /api/v1/mcp/{name}          update (including enable/disable, trust)
DELETE /api/v1/mcp/{name}          remove
POST   /api/v1/mcp/{name}/test        connect once and report result without persisting enablement
POST   /api/v1/mcp/{name}/connect     begin OAuth sign-in (opens the system browser)
POST   /api/v1/mcp/{name}/disconnect  revoke tokens and clear the keystore entry

GET    /api/v1/mcp/registry           browse/search the official registry (search, cursor, limit)
GET    /api/v1/mcp/registry/{name}    entry detail incl. versions, packages, remotes
POST   /api/v1/mcp/registry/{name}/resolve  return the exact command line an install would write
```

The registry is reached through the Go side rather than from the WebView directly, so timeouts, caching, offline behaviour and the user-agent stay under our control and the page keeps working with no network. `resolve` exists so the UI can show the verbatim command line before the user commits (§8.4) without duplicating resolution logic in TypeScript.

Sign-in is always initiated by a user gesture in this UI. There is no path by which a model, a tool result, or a config file can cause a browser to open.

### 8.3 Page shape

The page is a list of configured servers, each row carrying name, transport summary (the command line or the host), connection state, tool count, and a disclosure that reveals the server's advertised tools with their descriptions. Primary action is *Add server*; per-row actions are enable/disable, sign in/out for OAuth servers, edit, and remove. Empty state explains what an MCP server is and offers both add paths (local command, remote URL) plus paste-a-JSON-block.

### 8.4 Storefront — **RULED 2026-08-05: manager + the official MCP Registry**

The page is both a manager (§8.3) and a browse-and-install surface over `registry.modelcontextprotocol.io`. The icon is honest.

Registry API as read from its OpenAPI document on 2026-08-05 (`/v0` and `/v0.1` are both served):

- `GET /v0/servers` — query params `search` (substring on name), `cursor`, `limit` (default 30, max 100), `version` (`latest` or exact), `updated_since` (RFC3339), `include_deleted`. Envelope: `{ servers: [...], metadata: { count, nextCursor? } }`.
- `GET /v0/servers/{serverName}/versions` and `.../versions/{version}` — version list and pinned detail.
- Entry shape: reverse-DNS `name` (e.g. `io.github.user/weather`), `description`, `title`, semver `version`, `repository`, `websiteUrl`, `icons`, and two mutually-informative arrays — `packages` (each with `registryType` of npm/pypi/cargo/oci/nuget/mcpb, `identifier`, `runtimeArguments`, `packageArguments`, optional `fileSha256`) and `remotes` (each with `type` of stdio/streamable-http/sse, URL, headers, variables).

Install semantics, which is where the security lives:

- A registry entry becomes an ordinary `ServerSpec` in `~/.ollama/mcp.json`, **disabled**, exactly as if the user had typed it (§6.1). The registry never enables anything.
- A `packages` entry is resolved into a concrete command line — `npx -y <identifier>`, `uvx <identifier>`, `docker run ...` — and that **fully resolved command line is shown verbatim for the user to read before the add is committed**. Installing from a directory must never be the first time a user sees what will run on their machine.
- Ollama runs the command; it does not separately download, unpack or execute anything on the registry's say-so. Where `fileSha256` is present it is verified and a mismatch is a hard failure.
- `variables` declared by an entry (API keys, account ids) are collected in the add form and stored as `${env:VAR}` indirection, never as literals (§6.4).
- Version is pinned at install time, recorded in the spec, and an available upgrade is surfaced rather than applied.

### 8.5 What the registry is not

It is an open-publish metadata registry, not a vetting service. A listing is a claim by its publisher, and Ollama presenting it is not endorsement. The UI says so in the browse surface, shows the publisher namespace and repository URL on every entry, and does not rank by anything that could be read as a recommendation. This is not decoration: an install flow that reads as curation while surfacing arbitrary published code is the failure mode worth designing against.

---

## 9. Non-goals for v1

Named so they are decisions rather than omissions: MCP *resources* and *prompts* (only `tools` in v1); MCP servers packaged and distributed through the *Ollama* registry (upstream's `server/mcp.go` direction — deferred, not rejected, and now partly displaced by the official registry ruling in §8.4); publishing *to* the official MCP Registry from Ollama; sampling / elicitation callbacks (server-initiated model calls); Ollama acting as an MCP *server*; and per-chat server selection (v1 scope is global enable/disable).

---

## 10. Risks

- **Upstream rebase risk.** `app/ui/ui.go` and `cmd/agent_tui.go` are active files upstream. Keeping edits in new files and touching those two at as few points as possible is deliberate.
- **SDK churn.** v1.7.0 landed 2026-07-27 and the spec revised 2026-07-28. Pin exactly; treat an SDK bump as its own reviewed change. The §5.1 containment rule limits the blast radius to one package.
- **OAuth is the schedule risk.** Ruled in (B3) and sequenced last (Phase 3d) precisely so it cannot hold the rest hostage. The signal to watch is the SDK's experimental OAuth surface: if we find ourselves rewriting more than roughly half of it, that is worth reporting rather than absorbing.
- **Dependency versus upstream (D2).** The §5.1 containment test is the mitigation; if an upstream maintainer refuses the dependency, the swap is one package, not a rewrite.
- **The invisible half.** Most of the work here — approval, poisoning defences, process lifecycle — is not visible in the mock-up. The completion signal for this feature must not be "the page looks like the picture".
- **The registry turns a config page into a distribution channel.** Browse-and-install (§8.4) means a network response can produce a command line that runs on the user's machine. The mitigations are stated — disabled on install, verbatim command shown before commit, hash verified where offered, no ranking that reads as endorsement — and they are the part of Phase 4b that must not be trimmed if the phase runs long. If something has to give there, it is the browse polish, never the install gate.
- **Registry API churn.** It is a `/v0` API with a `/v0.1` alongside it. Fixtures are committed so our tests do not depend on the live service, and the client tolerates unknown fields rather than failing closed on them.

---

## 11. What I could not verify

- Nothing in this plan has been compiled or executed; Go is absent from this machine (§2.5). Every code claim is read from source at `8d8c701d` and every external claim carries its source in §3.
- The Go SDK's HTTP client transport surface was read from its README's compatibility table, not from its source; Phase 2 must confirm the exact transport constructors and OAuth status against the pinned version before the design in §7 hardens.
