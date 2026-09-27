# OQ-4 census working notes (continuation 16, Abuttal; a working record, not the census)

Source revision: 58999716 (production and test sources identical to 04009c43; `git diff --stat 04009c43..HEAD -- . ':(exclude)docs/**'` empty).

## Method 16 Tier 2, rule by rule (author-side obligations before the first judged round)

- R-0: header line 14, PACKS_BLOCKED derived from [SPEC-DRAFT]. Clear.
- R-1: concept cited by path + SHA-256 (line 11; 9396cd68 re-derived). N = 18 (lines 39-56); the only lines with upper-case RFC 2119 keywords are 39-56. Clear. (Normative prose in §3-§9 is incorporated by the requirements' "under §n" references.)
- R-2: reach closure NOT published (no file:symbol closure set). OPEN - main census output.
- R-3: itemised estimate §10 (2,181 LOC); bytes 81,731 <= 87,240 (headroom 5,509); every current line count re-derived and equal (19 files). Ceiling UNDECLARED (line 374). Clear now; the census will not fit without restructuring.
- R-4: citations mostly carry file:line + token at 04009c43 / 17197224; a full resolve check is owed at the panel.
- R-5: dependencies: G2 (not built; bindings refused), OI-27 curated context (separate draft, not adopted). Label is the floor.
- R-6: worked tables at producer-attainable extremes: MISSING for A3 reserve (C/8 clamp 16..4096, <= C/2), A5 fit (C-R), A13 bound B vs V, 80% notice, capacity caps, revision overflow, body bound (prose only). OPEN.
- R-7: carriers per requirement: §3 types + §11 mapping (partial); Cannot-Represent: §8 has author examples (judge supplies five). Partial.
- R-8: Write-path Guards table across the whole closure: MISSING (§8 table is a design guard list; line 329 says so). OPEN - main census output.
- R-9: REACHABLE chains per field/flag: partial (§3 producer -> consumer column, no hop chains). OPEN.
- R-10: default-closed gates -> production emitters (Profile.enabled default false -> PUT / CLI set; capability absent -> refuse); ORGANIC/INJECTED labels on I-01..I-14: MISSING. OPEN.
- R-11: configuration x trigger table (desktop app, terminal agent chat, CLI set, headless daemon, cloud): MISSING as such (§6.4 is route-based). OPEN.
- R-12: eight twins named (line 354); killing assertions partial; metamorphic relations/mutations partial. Partial.
- R-13: instruments + capability demonstrations: OQ-3 instrument (not fit); others designated. Partial.
- R-14: requirement ids <-> acceptance ids: §11 maps G1-R -> I-xx; I-xx rows do not cite G1-R ids; no published grep command. OPEN.
- R-15: boundaries: terminal/desktop -> daemon (HTTP), daemon -> cloud provider, frontend -> desktop server, CLI -> store; boundary enums (error codes, admission fields, capability strings) totality + unknown-value failure: partial. OPEN.

## Closure findings so far

1. Request call sites in the adapters' families (IP-005: enumerated from callers of the client, not a pattern):
   agent/session.go:396 (agent chat turns, G1), agent/compactor.go:256 (summariser; fixed system prompt :31, :236-249),
   cmd/agent_tui.go:598 (preload, no carrier), cmd/cmd.go:595 (loadOrUnloadModel preload for `ollama run`),
   cmd/cmd.go:1852 (classic `ollama run` chat, NOT in any G1 record), cmd/cmd.go:1984 (one-shot generate, NOT in any G1 record),
   app/ui/ui.go:978 (desktop chat, G1). cmd/bench (tool, out of scope).
2. Consumers of the terminal's effective system prompt: agent/session.go buildChatRequest (:441-467); compactor estimate (agent/compactor.go:383-389);
   chat token estimate; /prompt and /save (cmd/tui/chat/debug.go:28-64, :160-222) - /save writes JSON 0644 into the working directory; no admission field.
3. Terminal base producer: chatModel.systemPrompt (cmd/tui/chat/input.go:1719-1728) trims and joins opts.SystemPrompt and extra with "\n\n".
4. Desktop base producers: details from c.Show (app/ui/ui.go:870), mcpInstructions (ui.go:904, :930, only when tools capability and no attachments).
5. Daemon: ChatHandler (server/routes.go:2417-2911): explicit cloud proxied first (:2440-2448); remote-host stubs (:2495-2585); m.Messages (Modelfile MESSAGE) prepended BEFORE request messages (:2633), m.System only when first request message is not system (:2634-2636); chatPrompt (server/prompt.go:23-92) drops messages from the front one at a time keeping system messages, keeps the last message even when it does not fit (:66-70); native: truncateNativeChatMessages (:3010-3065) same shape, early return when !truncate or NumCtx<=0 or <=1 message (:3011).
6. Diagnostics sink undefined (§5 line 208).
7. R-3.2 headroom 5,509 bytes.
8. TECHNE-PA: no prior-art entry maps the Ollama G1 blueprint (the register's prior-art sections map NEXUS BP01-BP44); relevant internal pitfalls: IP-005, IP-007, IP-008, IP-082, IP-084.
9. Live-workload values for R-8 (bindings vs Ash's desktop chat count) need a read of Ash's live desktop store: reserved to Ash.

## Added after the options checks (27 September 2026)

10. Ash ruled the classic `ollama run` paths into G1 (DECISIONS `ash-classic-run-scope-2026-09-27`; ADR D-11 and amendment 1), so they join the closure: `cmd/cmd.go` RunHandler (:736, interactive decision :792-816), `generate` (:1884, request :1972-1984), `chat` (:1758, request :1840-1852); `cmd/interactive.go` `generateInteractive` (:34), `/set system` (:369-397, multi-line :182-185), `/clear` (:279-286), `/load` (:209-253), `/save` (:254-277, `NewCreateRequest` :548, :568-580); daemon `GenerateHandler` (`server/routes.go:250`: cloud :272, remote :313, raw guard :427, unload :409, load :483, system choice :520-525, chat-like path :556-615, direct template :617-624, `Completion` :659, final response :706); `api/types.go` `GenerateRequest` (`Suffix`, `System`, `Template` serialised without omitempty, :70-77; `Context` omitempty :81).
11. One-shot runs do not preload; only the interactive branch does (`cmd/cmd.go:899-900`, one-shot `generate` at :927).
12. The Go runner's cut (`llm/llama_server.go:276-316`): skipped when truncate is off, media is present, NumCtx <= 1 or the prompt's byte length is below NumCtx (:278); refuses with 400 when the runner cannot shift (:291-296); otherwise keeps nKeep tokens plus the tail and logs a warning only (:298-316).
13. Cloud qualification harnesses called only `/api/chat` (`proof_cont13/cloud-qualification/cloud_check.py:28`, `cloud_check2.py:110`); generate on cloud is unqualified.
14. Live citations of the blueprint by line that the (non-line-neutral) OQ-4 revision will make stale: the instrument's `DESIGN.md` and `oq3_oracle_test.go` (owed to instrument revision 5), and `ADR_2026-09-27_G1_OQ4_OPERATOR_CONTRACT.md` (re-anchored by clause in the revision). Historical packets, answers and dispositions keep the line numbers of their commit. No `*.sha256` or manifest pins the current blueprint hash (`5ce93604`).
