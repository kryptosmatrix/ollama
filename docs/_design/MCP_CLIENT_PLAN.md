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

### Phase 1 — `mcp/config.go`: the config file

`ServerSpec` (name, transport `stdio|http`, command, args, env, url, headers, enabled, trusted), `Config{MCPServers map[string]ServerSpec}`, load/save with atomic write via `cmd/internal/fileutil`, `0600` perms, validation (name charset, mutually exclusive transport fields, `${env:VAR}` credential indirection, rejection of literal secrets), and merge semantics with `$OLLAMA_MCP_CONFIG` override.

Proof: table tests over malformed configs; a test asserting a literal `Authorization: Bearer sk-...` value is **rejected**; a test asserting file mode is `0600` after save; a round-trip test proving an unknown future field survives save (forward-compat).

### Phase 2 — `mcp/` core: manager, transports, schema conversion

`Manager` connects N servers concurrently with per-server timeout, records capabilities, caches `tools/list`, exposes `Tools()` and `Call(ctx, server, tool, args)`, retries with backoff, and `Close()` reaps every child process. Schema conversion (`tools.go`) implements the §2.4 policy: representable keywords map through; unrepresentable constraints are **appended to the tool description in a machine-parseable constraints block** rather than dropped silently, so the model still sees them; a tool whose schema cannot be represented at all is skipped with a visible warning rather than offered broken. Namespacing and collision rules in `names.go`.

Proof (this is the phase where a facade is easiest and least visible):
- An **in-process fake MCP server** (`fake_test.go`) that speaks real JSON-RPC over an in-memory pipe: initialise, list tools, call a tool, return an error, hang past timeout, close its pipe mid-call, and advertise a changed tool list on reconnect. Every manager behaviour is proven against it.
- A **real subprocess test** launching an actual stdio MCP server binary and calling a real tool — the test a mock cannot pass. Gated behind a build tag if it needs network/npx, but it exists and it is run.
- A conversion test asserting a schema with `oneOf`, `minimum` and `$ref` produces either a faithful `api.ToolFunction` or a documented, asserted degradation — never a silently wrong one.
- A process-reaping test: kill the parent context, assert no orphaned child remains.

A containment test also lands here: a test that walks the module and **fails if any file outside `mcp/` imports `github.com/modelcontextprotocol/go-sdk`** (§5.1).

### Phase 2b — OAuth 2.1 for remote servers *(ruling B3)*

`mcp/oauth.go` and the platform keystores. Authorization-server metadata discovery, dynamic client registration where advertised, PKCE S256, loopback redirect with `state` verification and immediate listener teardown, token persistence in Keychain/DPAPI, single-flight refresh inside the transport, and revoke-on-disconnect. The SDK's OAuth package is used where it is sound at the pinned version and replaced with our own where its experimental status shows; whichever way each piece goes is recorded in the commit, not left implicit.

Proof:
- A **fake authorization server** in-process: issues codes, rejects a mismatched `state`, rejects a missing or wrong PKCE verifier, expires an access token so refresh must fire, and rejects a revoked refresh token. Each is a separate test with an asserted failure, not a happy path with error branches unvisited.
- A test asserting **no token ever appears in `mcp.json`** — write a config after a full flow, read the file bytes, assert the token substring is absent.
- A test asserting the loopback listener is **closed** after the flow, and that a second unsolicited request to the callback path after completion is refused.
- A keystore round-trip test per platform, with the non-Keychain/DPAPI fallback asserting `0600` and emitting its warning.

### Phase 3a — CLI surface *(ruling A2)*

`agent/tools/mcp.go` adapter implementing `agent.Tool` + `ApprovalRequired` + `ScopedTool`; `cmd/agent_tui.go` builds the Manager and registers tools; `cmd/mcp.go` gives `ollama mcp add|remove|list|enable|disable`; `/mcp` in `cmd/tui/chat/input.go`.

Proof: a session-level test driving `agent.Session` with a fake MCP server registered, asserting the model's tool call reaches the fake and the result reaches the messages — signal chain from emitter to consumer to observable effect. Plus an approval test asserting an MCP tool call is **refused** when the prompter denies it.

### Phase 3b — Desktop app approval path *(prerequisite for 3c)*

`app/tools/approval.go`, the `tool_approval` SSE event, `POST /api/v1/chat/{id}/approval`, persistence of per-scope grants, and the frontend approval affordance. Existing first-party tools keep their current behaviour (no approval required) so nothing regresses.

Proof: an `app/ui` handler test asserting a tool marked as requiring approval does not execute until an approval arrives, and does execute after; and that a denial produces a tool-error message rather than a hang.

### Phase 3c — Desktop app MCP registration and settings API

`app/tools/mcp.go` adapter; Manager construction in `app/ui/ui.go` (process-lifetime, not per-request — connecting servers per chat request would relaunch subprocesses on every message); routes per §8; `responses/types.go` additions + `tscriptify` regeneration; schema v17 `mcp_server_state` table (UI-local state: trusted flag, last-seen tool hash, last error — the *config* stays in `mcp.json`).

Proof: handler tests for every new route; a migration test proving v16→v17 preserves existing rows (the store already has `migration_test.go` conventions to follow).

### Phase 3d — OAuth in the surfaces

The desktop app gains a connect/disconnect affordance that opens the system browser and reports back (`POST /api/v1/mcp/{name}/connect`, `POST /api/v1/mcp/{name}/disconnect`); the CLI gains `ollama mcp login <name>` / `logout <name>`. Sequenced last deliberately: everything before this point is a shippable feature, so OAuth slipping delays OAuth and nothing else.

Proof: a handler test driving the full flow against the fake authorization server of Phase 2b, asserting the connected state is reached and that disconnect revokes and clears the keystore entry.

### Phase 4 — The MCP Servers page *(mock-up received; §8)*

A new `/mcp` route and page component, plus the sidebar entry after Launch with the storefront icon, plus route-derived active state so Chat de-highlights when MCP Servers is open. Not a Settings tab — this was the mock-up's main correction to the plan.

Proof: Vitest coverage of the list / add / edit / remove / enable flows against a mocked API; a test asserting the sidebar's active row follows the route rather than `currentChatId`; Storybook entries for connected, connecting, failed, needs-approval, needs-sign-in and empty states; and a manual run of the packaged app with a real local MCP server, screenshotted.

### Phase 4b — Registry browse and install *(ruling §8.4)*

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
