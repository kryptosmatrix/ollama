# Handoff — MCP client support

From: Grommet (Claude Opus 5)
Sessions: 1 (2026-08-05) and 2 (2026-08-06 to 2026-08-07)
Branch: `mcp/server-support`, pushed to `origin` (kryptosmatrix/ollama)
Head at handoff: `3f925cfc` — 49 commits ahead of `main`, 108 files, ~24,000 lines added

## Read this first

`docs/_design/MCP_CLIENT_PLAN.md` is the governing document. It carries the codebase analysis, the operator's rulings, the security model, the UI contract, and the phase plan with each phase's proof obligations. This handoff is the current state and what remains.

**The feature is complete and works end to end, in both surfaces, against real hosted servers.** `mcp/` is 13 production files and 147 tests. Two things are outstanding and both are named below: the Windows and Linux token stores, and an independent review by a different substrate.

## State

| Phase | Status |
|---|---|
| 0 — toolchain and baseline | **DONE** (`bc2d60ae`) |
| 1 — `mcp/config.go` | **DONE** (`f984d4ec`) |
| 1b — approval ledger | **DONE** (`b672284c`) |
| 2a — namespacing + schema conversion | **DONE** (`7dbec4fa`) |
| 2 — manager and transports | **DONE** (`594f6104`) |
| 3d — OAuth 2.1 | **DONE** (`da3b9da1`, `0f82fb9b`, `75dc07cd`, `3104fd4e`, `630974c0`, `088215ce`) — macOS keychain landed; Windows and Linux stores owed |
| 3a — CLI surface | **DONE** (`12ea7760`, `09a954cb`, `7e533582`, `f12a2d6a`) |
| 3b — app approval path | **DONE** (`0506f8e7`, `e7a05864`) |
| 3c — app MCP registration | **DONE** (`7b6bfc76`) |
| 3d — OAuth in the surfaces | **DONE** (`3104fd4e`) |
| 4 — MCP Servers page | **DONE** (`273c19bd`) |
| 4b — registry browse | **DONE** (`8317a273`, `07b06d8d`, `0c540dac`) |
| 5 — docs and closeout | **DONE** — docs current, 24 proof artefacts, this handoff |
| 5 — cross-substrate review | **DONE** (glm-5.2:cloud, 2026-08-07) — **found 5 confirmed defects; repairs NOT done** |

### Verified at closeout (2026-08-07)

`go build ./...` clean. `go test ./...` — 65 packages ok, 0 failures. `go vet ./...` — one failure, `tokenizer/bytepairencoding_test.go:542`, which is the pre-existing baseline. Frontend: 102 tests in 11 files pass; eslint reports 43 problems in 13 files and prettier 8 files, both **below** the Phase 0 baseline of 122/14 and 9, and none of them in a file this work owns. The single warning in `ChatSidebar.tsx` is at line 170, in chat date-grouping code this branch never touched.

`app/updater`'s `TestAutoUpdateDisabledSkipsDownload` is flaky and pre-existing; it passed at closeout, which proves nothing either way.

### What works

From the terminal: `ollama mcp add|list|remove|enable|disable|approve|revoke|login|logout`, all registered on the root command. `ollama` in agent mode connects approved servers at start-up and offers their tools to the model behind per-tool approval. `/mcp` in the TUI lists and toggles servers and deliberately cannot approve one.

From the desktop app: an MCP Servers page that lists, approves, switches, removes and adds servers, browses and installs from the official MCP Registry, and signs in to and out of remote servers. Tool calls are gated by an approval prompt.

Against real services: a public hosted server connects and a tool call round-trips; three OAuth-protected services report a needed sign-in rather than failing; and a complete sign-in to Sentry works — discovery, dynamic registration, PKCE, browser, exchange, nine tools, reconnect from the stored token with no browser, and a sign-out Sentry accepted as a revocation.

## Operator rulings in force

Ruled 2026-08-05, recorded in plan §5 and §8.4. Do not re-open without Ash.

1. Surfaces: desktop app **and** CLI agent TUI.
2. Remote: **full OAuth 2.1 with sign-in**, sequenced last (Phase 3d).
3. Protocol: **official `github.com/modelcontextprotocol/go-sdk` v1.7.0**, contained so no SDK type appears outside `mcp/`.
4. Target: **aimed at an upstream PR**, so keep the diff small and touch unrelated files as little as possible.
5. The MCP Servers page is a **manager plus a browse-and-install surface over the official MCP Registry**.

## Things that will cost you a session if you don't know them

**`go build ./...` fails on a clean checkout.** `app/ui/app.go` embeds `app/dist`, so the frontend must be built first: `npm ci && npm run build` in `app/ui/app`. This is not a broken tree.

**Three checks were already red at Phase 0 and are not yours to fix.** `go vet ./...` on `tokenizer/bytepairencoding_test.go:542`; `npm run lint` with 122 problems across 14 files; `npm run prettier:check` on 9 files. Full detail in `docs/_design/proof/phase0-baseline.txt`. Completion is "no new problems against that baseline", not "zero problems". At closeout the frontend numbers had *fallen* to 43 and 8 — something else in the tree improved — so measure before you assume a number is yours.

**Do not run `prettier --write` on a file you did not create.** Reformatting buries a small change in a large diff and works against the upstream ruling. Hand-edit in the surrounding style. (`ChatSidebar.tsx` was the case that mattered; it is prettier-clean now, but the rule stands for the eight files that are not.)

**`mcp/` cannot import `cmd/internal/fileutil`.** Go's internal rule. It writes atomically itself.

**The upstream branch `Parth/agents` does not compile.** Commit `5c0caaff` looks like finished MCP support and its manager was never committed. It is a design reference for config shape and CLI naming, nothing more. Plan §3.1.

**`go test ./... | tail` reports the exit code of `tail`.** Redirect to a file and echo `$?` on its own.

**The missing `app/dist` also takes out `mcp`'s `TestSDKIsContainedToThisPackage`**, because that test is a `go list` sweep of the whole module and `go list` cannot load `app/ui` without the embed. The failure names the embed, so it reads as nothing to do with SDK containment. `go test ./app/ui/` goes the same way. If you do not need a real frontend, a placeholder `index.html` in `app/ui/app/dist` is enough to unblock both — it is gitignored at `app/ui/app/.gitignore:11`.

**`app/updater`'s `TestAutoUpdateDisabledSkipsDownload` is flaky.** It fails roughly one full app-tree run in three and always passes when that package is run alone — proven against a clean tree with this work stashed. It is not MCP's. Do not chase it, and do not read a single green app-tree run as proof of anything.

## What remains

**Do the repairs first. The branch is not ready for an upstream PR.**

### 1. Repair what the cross-substrate review found — 2 of 7 DONE

A blind review by `glm-5.2:cloud` over commit `3f925cfc` found seven confirmed defects, four of them by execution. Full record with per-finding verification status in `docs/_design/proof/phase5-cross-substrate-review.txt`; the reviewer's raw output is beside it under `glm-review-raw/`.

**The two structural defects are repaired** (`4d0e9120`, proof in `phase5-repairs-falsification.txt`): a server removed from the configuration now stops and is forgotten, and two properties may share one `$ref`. Read that proof before touching the resolver — the obvious fix, making the reference chain local to one resolution, silently traded sibling sharing for the accurate cycle diagnosis, and only the pre-existing cyclic-schema test caught it. The chain is threaded down the nesting path for that reason.

**Five confirmed defects remain, and the credential pair is the one to take next.**

**A password embedded in a server URL is accepted and written to `mcp.json` as a literal** (`mcp/config.go`, `validateURL` never looks at the userinfo). The one rule that file exists to enforce is that credentials appear only as `${env:NAME}`.

And then: a refused sign-in loses the server's reason (`mcp/oauth.go` error branch calls `deliver` rather than `deliverFailure`); a failed migration leaves the credential in cleartext for ever (`mcp/tokenstore_darwin.go`, `Load`); and a registry identifier is not checked for being a flag (`mcp/registry.go`, `resolvePackage`).

**The same credential is copied into the approval ledger** (`mcp/approvals.go`, `Approve`, via `Summary()`), so a secret in a URL or in stdio args lands in `mcp-approvals.json` too — and a user who scrubs `mcp.json` has no reason to look there. Confirmed by probe. Fix these two together; they are one defect with two outlets.

The approval ledger *was* reviewed in the end, after its first shard returned an empty response, and it produced the one negative result worth having: **no path was found by which a server can be started without a current approval**, and the fingerprint holds against config tampering. That is the claim this branch rests on and it is no longer only mine.

Twenty further findings are recorded as UNVERIFIED with reasons — concurrency in the manager, the file store's read-modify-write across processes, revocation targeting, several schema-conversion gaps. Work them down; do not treat the list as conclusions.

One finding was REFUTED, and why is worth reading: my shard boundaries cut a call site away from its callee, so the reviewer correctly reported dead code that is not dead. Give the next reviewer whole call paths.

### 2. The Windows and Linux token stores

macOS is done (`mcp/tokenstore_darwin.go`, cgo + Security framework, one generic password item per server). Windows should use DPAPI, Linux the desktop secret service. They were **not written** rather than written unproven: neither can be executed on this machine, and code that compiles but has never run is what full-or-stop forbids. `DefaultTokenStore` on those platforms returns the file store, whose `Description()` says what protects it, and every surface shows that string before a token exists.

Follow the darwin file's shape. The interface takes a `SignInRecord`, not a bare token. `Save` must preserve a `ClientID` that a refresh does not carry. `Load` must return `ErrNoToken` for a miss and is also where migration from the file store belongs. `Delete` of an absent item is not an error. And whatever you write, make its `Description()` say only what you have measured it to deliver.

### Also worth knowing

**The real-service harness.** `mcp/hosted_test.go` is skipped unless `OLLAMA_MCP_TEST_URL` names an endpoint; `OLLAMA_MCP_TEST_SIGNIN=1` additionally allows a browser sign-in, and `OLLAMA_MCP_TEST_TOOL`/`OLLAMA_MCP_TEST_ARGS` call one tool. It uses a token store in a temporary directory, never the platform default, and the sign-in test signs out at the end so a run leaves no credential at either end. Use it whenever you touch the transport or the OAuth path.

**An upstream issue is open and may change this code.** https://github.com/modelcontextprotocol/go-sdk/issues/1152 — `auth/authorization_code.go`, `validateIssuerResponse` rejects a correct-but-unadvertised RFC 9207 `iss`. A real sign-in to Sentry is what found it, and it is why `LoopbackRedirect` compares the issuer itself. If the SDK is fixed, `IgnoreIssuer` can stop withholding the value — but **the comparison itself must stay**, because RFC 9207 §2.4 puts that obligation on the client whenever the parameter is present, and the library only ever sees the values it was told to expect.

## The shape of what landed, so it is not re-derived

, so it is not re-derived: the transport seam is `newTransport(ctx, spec, transportOptions) (sdk.Transport, func(), error)` — the mode says whether a browser may open, and the returned function releases the redirect listener and must run on every path. `Manager.SignIn` is the only caller that passes `signInAllowed`. `TokenStore` keeps a `SignInRecord`, not a bare token, because RFC 7009 revocation needs the client identifier and dynamic registration issues a fresh one each time, so it is knowable only at sign-in. `SignOut` revokes then deletes, and returns `ErrSignedOutLocallyOnly` when the revocation did not happen. `StatusNeedsSignIn` is its own status, not a failure, and is never retried.

## What must not soften

- Tool descriptions from a server are untrusted input that lands in the model's prompt. Sanitised and capped in `mcp/tools.go`; keep it that way when the manager starts feeding real ones through.
- Schema conversion is lossy and says so in the description. Do not "simplify" that into dropping constraints.
- The desktop app's own tools declare an `api.ToolFunction` and derive their schema map from it, never the reverse. `browser.open`'s `id` is either a URL or the index of a link on the page, which needs `anyOf` to express; rebuild that tool from a map and the property arrives with no type at all. A new first-party tool that ships only a map is back on the lossy path, and `TestFirstPartyToolsSupplyTheirOwnDefinition` is what catches it.
- `defaultWebSearchResults` and `defaultBrowserSearchResults` are each read by both `Execute` and the tool definition. Do not inline either back into one of them: the hand-written schema they replaced advertised 3 while `Execute` asked for 5, for as long as both existed.
- `maxBrowserSearchResults` equals the default on purpose. `browser.search` discarded `topn` for its whole life, so every search it ever ran asked for five; honouring the argument without a ceiling would have widened what the model can request from the search service as a side effect of describing the tool honestly. Raising it is a deliberate decision that needs its own evidence, not a cleanup.
- `mcp.json` is a code-execution config: `0600` in a `0700` directory, no shell, credentials only as `${env:NAME}`.
- The desktop app's approval path is complete, backend and interface. Any tool registered there that implements `ApprovalRequired` will be gated; anything that does not will run the moment the model names it, so an MCP adapter must implement it.
- Everything about the approval wait fails safe: a timeout, a cancelled context and a failed notify all refuse the call. Do not "simplify" any of those into allowing it.
- `/mcp` must never grow an approve verb. Approval needs the command line shown verbatim and a deliberate answer; a chat input line cannot give either. There is a test guarding the decision, not the implementation.
- If Phase 4b runs long, the browse polish gives way; the install gate — resolved command line shown verbatim, landed disabled, hash verified — never does.
- The approval policy must keep reading the ledger from disk on every question (`mcp.ApprovalsFile`). A snapshot means approving a server can never start it, and the symptom is indistinguishable from the approval not having been recorded.
- Approving from the page must keep sending back the command line it displayed. Without that check, a stale page or a configuration edited underneath approves something the user never read.
- A registry entry Ollama cannot run must never be offered with a command line. Refusing is correct; guessing a runner for an unverified ecosystem is how a user approves something that is not what its name suggests.
- The OAuth redirect must keep refusing a wrong-state callback *without ending the wait*. Refusing it is not enough on its own: a stray request that aborts a sign-in in progress is a denial of service on the loopback port that anything on the machine can reach.
- The browser must keep being opened only by `LoopbackRedirect.Fetch`, and only for http and https.
- A token must never reach `mcp.json`, and `FileTokenStore.Description()` must never start implying the operating system's keychain. Both have tests.
- Never reach the macOS keychain through the `security` command. Its secret is a command-line argument and shows up in the process list; this was measured, not assumed.
- Registry tests must keep using the recorded fixtures. Pointing them at the live service makes the suite fail for reasons unrelated to this code.
- An ordinary connection must never be able to open a browser. Servers connect at start-up, on a configuration change and on every reconnect; a 401 answered the ordinary way would open a sign-in page on a machine nobody is sitting at. `signInDisallowed` is the default and `Manager.SignIn` is the only exception.
- Signing out must keep revoking at the server before it deletes. Forgetting a token locally leaves it valid at the service while the user believes they withdrew it. When revocation cannot happen the token is still deleted **and the user is told which of the two occurred** — that distinction is the whole point of `ErrSignedOutLocallyOnly`.
- The client identifier must keep being written down at sign-in, and a refresh without one must not erase it. It is visible exactly once, in the `oauth2.Config` handed to `NewTokenSource`; registering again later issues a different client and revokes nothing.
- `StatusNeedsSignIn` must not collapse back into `StatusFailed`. They ask the user for different things, and a sign-in is never fixed by retrying.
- A sign-in must stay one-at-a-time per server. Two would open two browser windows and two redirect listeners, and the callback that arrived second would answer a request that no longer exists.
- Both surfaces must keep naming the token store before a token exists. This is what makes the file store honest rather than merely convenient.
- A present `iss` must keep being **compared** in `LoopbackRedirect.handleCallback`, not merely dropped. This was got wrong once: the first version withheld an unadvertised issuer from the library and did no comparison of its own, so a callback naming a different authorization server was accepted silently — on exactly the services that need it, since none of Sentry, Linear or Notion advertises the parameter. RFC 9207 §2.4 conditions the obligation on the parameter being *present*, not on it having been advertised. `IgnoreIssuer` decides only what the library is shown, after the check has run.
- `issuerExpectation` answering `("", true)` when it cannot read the metadata is deliberate: nothing to compare against, and the library's stricter handling left in place. Not knowing must never relax anything.
- `signInRequiredTransport` must stay in front of every connection that is not an explicit sign-in. The protocol library performs discovery and dynamic client registration **before** it asks whether a browser may be opened, so a refusal at the fetcher comes too late: it has already announced this installation to the service and left a client registration behind, on every launch and every reconnect. Attaching an OAuth handler to a server with no stored token brings that back. `TestAServerThatNeedsASignInSaysSoRatherThanFailing` asserts zero registrations and zero authorization requests on the ordinary path.
- A 401 without a Bearer challenge must keep reading as an ordinary failure. Reporting it as a needed sign-in sends the user to a browser for nothing.
- There is exactly one place a token is written: `persistingTokenSource`. A second writer was deleted after two attempts to falsify it both passed. Do not add one back beside it.
- `KeychainStore.Description()` must never start implying that other programs cannot read the item. An item added by an unsigned build **is** readable by another process running as the same user — measured, not assumed, by writing from one binary and reading from another. `TestKeychainDescriptionDoesNotOverclaim` fails on the wording, not the mechanism.
- Every test package that can reach `mcp.DefaultTokenStore()` must set `OLLAMA_MCP_TOKENS`. On macOS the default is the real login keychain, and five `cmd` tests build the production manager; without it they read and delete the developer's own credentials. `mcpEnv` and `mcpFiles` set it unconditionally.
- The keychain tests use the real login keychain on purpose, scoped to a per-test service name. Their cleanup must keep sweeping by service through `security delete-generic-password`, which shares no code with the store under test — a sabotage that broke the item query also broke a cleanup that went through it, and left two items behind. `security` is refused for *storing* a token because the secret lands in the process list; deleting by service name passes no secret.
- When falsifying, do not back files up by basename. This repository has two `mcp.go`, two `tools.go` and more; a harness that did cost an hour and nearly lost a surface. `docs/_design/proof/phase3d-wiring-falsification.txt` records it.

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
- `docs/_design/proof/phase5-falsification.txt` — the first-party tool definitions, five protections. Only one of them discriminates the faithful path from the derived one — `browser.open`'s `anyOf` — and the file says which, because the rest assert things both paths preserve.
- `docs/_design/proof/phase3b-react-falsification.txt` — the React prompt, four protections, and a statement of what the node-environment suite cannot cover.
- `docs/_design/proof/phase3c-falsification.txt` — MCP tools in the app, four protections, plus a falsifier that initially passed and the test repair it forced.
- `docs/_design/proof/phase4-falsification.txt` — the MCP Servers page, five protections, plus two falsifiers that initially passed: one a wrong sabotage, one a test that did not discriminate.
- `docs/_design/proof/phase4b-falsification.txt` — the registry client and browse routes, nine protections, all against recorded fixtures.
- `docs/_design/proof/phase2-falsification.txt` — the manager, including three attempts that **failed to falsify** and the removal that followed. Read that section before adding any safety mechanism to this package: the protocol library already reaps processes, and code written to do it again cannot be tested.
- `docs/_design/proof/phase3d-oauthhandler-falsification.txt` — the OAuth handler, ten protections, plus two of my own tests rewritten first because they did not discriminate.
- `docs/_design/proof/phase3d-signin-falsification.txt` — sign-in and sign-out, ten protections against a fake authorization server.
- `docs/_design/proof/phase3d-wiring-falsification.txt` — the transport wiring and the connect surfaces, ten protections, and a harness fault that overwrote a surface file and how it was recovered.
- `docs/_design/proof/phase3d-flow-falsification.txt` — the complete authorization-code flow, ten protections, two defects found before any sabotage, and one protection recorded as **not falsifiable against our code** (PKCE lives in the library) rather than counted.
- `docs/_design/proof/phase3d-keychain-falsification.txt` — the macOS keychain store, eleven protections against the real login keychain, and the measurement that decided the wording of its description.
- `docs/_design/proof/phase3d-hosted-servers.txt` — runs against DeepWiki, Linear, Sentry and Notion, with what they do and do not prove stated separately.
- `docs/_design/proof/phase3d-real-signin.txt` — the complete sign-in to Sentry: the run that failed, the defect it found, and the run that succeeded.
- `docs/_design/proof/phase3d-issuer-validation.txt` — the correction to that fix, where reading RFC 9207 rather than paraphrasing it showed my own first attempt accepted a mismatched issuer.

## What this session got wrong

Kept deliberately, because a handoff that lists only successes teaches the next reader nothing about where the traps are.

- I wrote a confident paragraph about what RFC 9207 required before reading it, and the paragraph was wrong in the direction that weakened a security check. The tests I had written all passed over the hole. **Read the specification text, not your memory of it, whenever a check is being relaxed.**
- A falsification harness that backed files up by basename overwrote `app/ui/mcp.go` with `cmd/mcp.go`. Recovered from HEAD plus re-applied edits.
- A keychain sabotage broke the item query and with it the test's own cleanup, leaving two items in the real login keychain. Cleanups must not go through the code under test.
- `oauthHandlerFor` went dead when the transport was restructured and survived only because its own test kept passing. A green test is not evidence that production uses the thing it tests.
- Two protections could not be falsified and the code behind them was deleted rather than kept: the per-server reaping in Phase 2, and a second token writer in Phase 3d.
