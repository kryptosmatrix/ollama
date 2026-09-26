The checkout reports HEAD `4025d3b06c66d2c99c91427b4b8cdf5b4e325059`, not `a7c35fd65e732cd378dc6d2e93ccc43f152efcef`. **[Unverified] I cannot attribute the inspected files to the requested commit without establishing their equivalence.** The citations below refer to the available checkout.

I made no edits and ran no builds, tests or baseline instrument. The Git metadata query triggered denied macOS cache-write attempts; subsequent commands only read repository files. The historical measurements in §3.3 remain unverified by this review.

[Inference] My choices are **Q1(a), Q2(a), Q3(a), Q4(a), Q5(c), Q6(b), and Q7(a,b,d,e), with a corrected version of Q7(c)**. Several need qualifications below. The central design change is to separate the delivery score from whether the run was complete and valid: **zero incorrect captured requests is insufficient for acceptance.**

On sections 3 and 4, I found the main source account substantially accurate, with these corrections and qualifications:

- **Desktop entry and composition:** the chat and create-chat routes exist, while unmatched methods fall through to the application handler. `createChat` performs a daemon-readiness check before allocating its ID; “allocates an ID only” correctly describes its conversation-state effect, but not all its activity. MCP text is added when it contains **non-whitespace** text, rather than merely when non-empty. The builder does not incorporate `ShowResponse.System`. See [ui.go:304](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/ui.go:304), [ui.go:352](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/ui.go:352), [ui.go:481](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/ui.go:481) and [ui.go:1909](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/ui.go:1909).
- **Terminal entry and compaction:** the described launcher path and separation from the direct `chatModel` tests are supported. The Bubble Tea constructor is at line **257**, not 256. The summariser supplies its fixed system prompt; `CompactionRequest.SystemPrompt` enters the estimation path. See [cmd.go:2130](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/cmd/cmd.go:2130), [chat.go:257](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/cmd/tui/chat/chat.go:257), [input_test.go:972](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/cmd/tui/chat/input_test.go:972), [compactor.go:236](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/agent/compactor.go:236) and [compactor.go:383](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/agent/compactor.go:383).
- **Absent surfaces:** the CLI registration lacks `instructions`, and the HTTP fallback serves embedded application content when available. This supports the reported failures, but does not independently reproduce their exact exit status or response. See [cmd.go:2533](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/cmd/cmd.go:2533) and [app.go:32](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/app.go:32).
- **Isolation needs more than `HOME`.** The desktop database default is evaluated during package initialisation; Windows uses `LOCALAPPDATA`. Terminal skills can come from `OLLAMA_SKILLS`, `XDG_CONFIG_HOME` and the working project. Production TTS can use the macOS Keychain independently of `HOME`; the existing UI `TestMain` separately redirects TTS storage. Set the child environment before execution, use an isolated working directory and explicitly isolate other configuration sources. See [store.go:192](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/store/store.go:192), [skills.go:81](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/agent/skills.go:81), [skills.go:628](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/agent/skills.go:628), [secrets_darwin.go:184](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/tts/secrets_darwin.go:184) and [tts_test.go:20](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/tts_test.go:20).
- **The fake daemon needs a defined protocol, not just permissive responses.** Existing terminal startup includes `HEAD /`, preload through `/api/generate`, and a running-model lookup. Future successful G1 cases require the advertised admission capability and appropriate final admission records. Without that capability, refusing dispatch is correct. See [client.go:422](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/api/client.go:422), [agent_tui.go:583](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/cmd/agent_tui.go:583) and [blueprint:253](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/docs/_design/G1_PERSISTED_INSTRUCTIONS.md:253).
- **Revision numbers require scenario isolation.** A fresh `HOME` for the entire run does not justify restarting every scenario at revision 1. Use fresh storage per independent scenario, preserving it across that scenario’s processes. Revisions are neither removed nor reused under [blueprint:165](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/docs/_design/G1_PERSISTED_INSTRUCTIONS.md:165).
- **D5 is not a reliable legacy fixture after implementation.** A conversation first used under G1 while unconfigured should already have a revision-zero binding. The legacy branch requires a saved transcript with **no binding**, as distinguished in [blueprint:79](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/docs/_design/G1_PERSISTED_INSTRUCTIONS.md:79).
- **T1’s tool wording needs correction if literal.** The fake daemon emits the tool call; the host executes the tool and supplies its result in a subsequent request. Letting the fake daemon fabricate both sides would bypass tool execution. See [session.go:573](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/agent/session.go:573) and [ui.go:1118](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/ui.go:1118).

Q1: I choose **(a)** as the closest interpretation of the stated unit, with Q5’s omission rule made explicit.

- **(a), strongest objection:** request-heavy conversations contribute more failures, so the value depends on the scripted workload and can exceed one. It also leaves absent requests undefined.
- **(b), strongest objection:** one bad delivery and twenty bad deliveries become indistinguishable. This measures affected conversations, not incorrect deliveries.
- **(c), strongest objection:** it changes the denominator, and additional correct requests dilute the rate.

A materially better option is missing: publish the requested measure together with its components and a separate validity result.

Use `bad observed deliveries`, `missing required deliveries`, `observed deliveries`, and `controlled conversation trials`. Count an observed request once in the numerator even when it has several defects. Category counts may overlap, but must say so.

For Q5(c), explicitly define a failed required delivery to include an omission. Otherwise Q1(a) and Q5(c) quietly use different meanings of “delivery”. The resulting headline is:

`(bad observed deliveries + missing required deliveries) / controlled conversation trials`

The proposed matrix contains **16 intended conversation trials**, counting T0. Freeze those trial identities before execution; do not deduplicate them using defective application IDs or remove trials that fail to start. Report realised application identities separately.

Q2: I choose **(a)**, with classification based on the operation that caused the request.

- **(a), strongest objection:** exclusions can hide failures. A defective turn request must not become “background” merely because its carrier or expected content is missing. Separate summariser checks must also block acceptance when they fail.
- **(b), strongest objection:** it changes the compactor’s job by injecting standing conversational instructions into its summarisation request. The source deliberately supplies its own prompt. It would also mishandle empty `/api/chat` load requests, which the daemon explicitly supports at [routes.go:2622](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/server/routes.go:2622).
- **(c), strongest objection:** instructions could disappear immediately after the first tool call while every counted request passes.

The better missing option is **an explicit request-purpose inventory**. Count user-turn requests, their model retries and tool continuations. Give summarisation, warm-up and other auxiliary operations their own expectations. An unrecognised request prevents a complete verdict until classified.

For these controlled fixtures, the summariser should receive its existing prompt and conversation material without G1 injecting a carrier into that material. The post-compaction conversation request must independently reconstruct the selected carrier. Do not turn this into a universal prohibition on users quoting instruction text in ordinary messages.

Q3: I choose **(a)** for the authoritative terminal result.

- **(a), strongest objection:** PTY driving introduces terminal negotiation, menu, focus, completion and timing failures that are unrelated to G1. The driver needs observable readiness and completion conditions, not sleeps and guessed key sequences.
- **(b), strongest objection:** it skips command dispatch, launcher selection and some startup behaviour. Those could be broken while every lifecycle test passes.
- **(c), strongest objection:** it bypasses exactly the new wiring most at risk: normal creation, selection loading, ID allocation, process persistence and HTTP transport. The capture helper records a request pointer directly at [test_helpers_test.go:39](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/cmd/tui/chat/test_helpers_test.go:39).

A useful missing option is a layered suite: **(a) for entry acceptance, (b) for detailed process lifecycle coverage, and (c) for fast component diagnosis**, sharing scenario definitions and scoring rules where practical.

If only (b) is delivered, describe it as testing the production launch function, not the built CLI’s complete entry path. Preserve or explicitly amend the blueprint’s designated test ownership; an overlay-only test must not disappear from final discovery.

Q4: I choose **(a)**.

- **(a), strongest objection:** a re-executed test server can still assemble dependencies differently from the desktop application. It proves a real Handler/process boundary, not automatically the native application’s startup wiring.
- **(b), strongest objection:** process globals, cached defaults, live handles and process-owned state survive. It does not demonstrate persistence across process exit or release of OS-held leases.

The missing improvement is **(a) with production-equivalent assembly and verified process termination**, plus final application/browser coverage. Keep the driver and capture service alive outside the restarted host, preserve the storage root, and verify that P1 has exited before claiming P2 read persisted state.

Q5: I choose **(c)**, restricted to required delivery opportunities.

- **(a), strongest objection:** “never dispatch” can score zero.
- **(b), strongest objection:** it replaces delivery measurement with a scenario-failure flag and loses the later requests that would reveal additional behaviour.
- **(c), strongest objection:** it can manufacture failures for legitimate refusals, infrastructure faults or hypothetical downstream requests whose triggering event never happened.

A materially better option is missing: a **predeclared event ledger** distinguishing required dispatch, required refusal, failed production operation, and invalid harness execution.

Attempt absent surfaces and record their real responses. Continue independently executable steps. Do not revise “expect A” to “expect nothing” merely because saving A failed—that would excuse the missing feature.

Conversely, an intentional invalid-profile or unavailable-capability case expects refusal. It must not acquire a “request never sent” penalty. A broken PTY or failed fake daemon makes the run incomplete, rather than proving an instruction-delivery defect.

Required save/reload/disable outcomes also need assertions independent of the numerator. D4 could deliver the correct no-carrier request even though both attempted saves failed.

Q6: I choose **(b), strengthened to enforce the complete composition contract**.

- **(a), strongest objection:** matching one expected string does not establish its role, leading position, uniqueness or absence elsewhere. A no-MCP fixture also leaves the combined-base path untested.
- **(b), strongest objection:** as written, it does not explicitly require the carrier message to be first, and its desktop base expectation excludes MCP. A terminal baseline obtained from the implementation under test could also make the oracle circular.
- **(c), strongest objection:** sentinels survive many corruptions: removed prose between them, altered whitespace, wrong role, wrong revision, duplicate carriers or a lost base.

The better missing option combines exact content with structural and compatibility checks:

- The leading message has role `system`, with the exact expected base, separator, header and profile bytes.
- Revision and generation use their specified representations.
- No additional carrier or fixture sentinel appears in other messages or fields.
- Conversation messages, tool schemas, model, options, thinking and format obey their independent expectations.
- Enabled requests include the admission field; disabled requests preserve its absence.

The exact carrier separator is specified at [blueprint:200](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/docs/_design/G1_PERSISTED_INSTRUCTIONS.md:200). The separator between desktop model-system and MCP blocks is not specified at [blueprint:204](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/docs/_design/G1_PERSISTED_INSTRUCTIONS.md:204). Resolve that before freezing the combined-base golden.

Freeze pre-feature compatibility expectations independently. Parameterise only identified variable inputs, such as date and fixture paths. Compare decoded JSON meaning while preserving exact string bytes and meaningful field presence; retain raw bodies as evidence.

Q7: I require **(a), (b), (d), (e)** and a corrected **(c)**.

- **(a), strongest objection:** scorer and synthetic fixtures can share the same mistaken assumptions. Include independently authored cases, multi-defect requests, omissions, unexpected requests and arithmetic reconciliation.
- **(b), strongest objection:** a correct reference sender proves capture/scoring calibration, not production wiring. Keep its captures outside the candidate’s numerator and denominator.
- **(c), strongest objection:** dropping a system message *after the daemon receives it* does not make the host’s delivery incorrect. Likewise, submitting a valid step twice can produce two correct carriers. These faults should change the appropriate **integrity or cardinality verdict**, not necessarily the delivery numerator.
- **(d), strongest objection:** child exit codes and scenario names do not prove assertions ran. Require discovered, started and completed cases with reconciled evidence. Account separately for deliberately expected command failures.
- **(e), strongest objection:** repeatable wrong answers are still wrong. Repetition detects instability, not validity. Require stable case/category verdicts, not identical random identifiers or raw bytes.

Missing required controls include truncated or malformed capture records, wrong request attribution, an omitted continuation, an extra request, failed discovery, timeout, and an unauthenticated request paired with an authenticated existing-route control.

The final instrument also needs **production mutants**, including the blueprint’s disconnected composer, stale-binding loader and failed CAS, alongside its degenerate twins. Baseline redness gives no mutation-detection credit. Credit requires a passing clean candidate, a compiling mutant that fails its intended assertion, and a passing restored candidate, as specified at [blueprint:354](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/docs/_design/G1_PERSISTED_INSTRUCTIONS.md:354).

Q8: These are the material ways I can identify for a false zero; this cannot be an exhaustive enumeration of arbitrary faulty implementations.

1. **Vacuous execution:** empty selection, skipped scenarios, early child exit, missing captures, integer division or rounded display produces zero.
2. **Wrong subject:** a stale binary, substituted handler, direct model fixture, modified overlay or reference sender supplies the scored traffic.
3. **Incomplete capture:** requests escape to another host or route, arrive after collection ends, or are lost through truncation, races or deduplication.
4. **Biased classification:** missing-carrier requests are excluded because classification itself looks for the carrier.
5. **Wrong attribution:** concurrent sessions are mixed, `/new` reuses an identity, or expected state is inferred from the very revision header being tested.
6. **Circular expectations:** the scorer calls production composition code, trusts the returned save value as its input oracle, or continually regenerates compatibility goldens from the changed candidate.
7. **Weak comparison:** sentinels remain while text, role, ordering, base, generation, fields or duplicates are wrong.
8. **Unasserted operation failure:** an HTML 200 is accepted as Saved; reload or disable fails without failing the scenario; later expectations are weakened to match.
9. **Unexercised transitions:** compaction is skipped, a tool never executes, the legacy fixture already has a binding, or a “restart” retains process state.
10. **Untested combinations:** only one data shape, tool round, model, context size, toggle state or successful serial schedule is exercised.
11. **Carrier-only success:** the text is correct but the selected model, tools, history, approvals or admission protection is wrong.
12. **Binding defects hidden by matching text:** identity collisions, incorrect generations or shared state happen to produce the same A/B output in the sampled order.
13. **Surface gaps:** HTTP works while the editor, actual CLI registration, launcher or desktop assembly is disconnected.
14. **Failures below capture:** the daemon changes, drops or misrenders correct incoming text, or loses it during generation. The existing daemon performs additional model-system insertion and later rendering; capture at its input cannot certify those stages. See [routes.go:2633](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/server/routes.go:2633) and [routes.go:2882](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/server/routes.go:2882).
15. **Failures outside the sampled lifetime:** another process race, interrupted initialisation, abandoned lease, corruption, capacity boundary or later restart behaves differently.

A zero here proves only the classified, observed boundary obligations in a valid run. It does not prove model compliance or replace daemon/real-model qualification.

Q9: The corresponding false-alarm risks are:

1. **Correct refusal scored as missing delivery:** absent capability, unsupported route, exceeded capacity, invalid state, stale CAS or a busy conversation.
2. **Harness failure blamed on G1:** missing assets, failed child startup, PTY input loss, wrong authentication, network timeout or malformed fake-daemon responses.
3. **Incorrect fake metadata:** missing tools capability, inconsistent model identity, inadequate context information or missing admission records prevents an otherwise correct path.
4. **Uncontrolled expected base:** date, timezone, working directory, platform shell wording, model-system sentinel, skills or tools differ from the frozen fixture. Terminal construction explicitly uses date, model and directory at [agent_tui.go:218](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/cmd/agent_tui.go:218).
5. **Revision/generation mistakes in the oracle:** scenarios share storage; identical saves are assumed to allocate revisions; idempotent reloads are assumed to increment generation. The latter is explicitly a no-op under [blueprint:169](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/docs/_design/G1_PERSISTED_INSTRUCTIONS.md:169).
6. **Wrong binding-time assumption:** the oracle binds desktop at ID allocation or terminal at first submission, reversing the specified timing.
7. **Incorrect request attribution:** a summariser, warm-up, allowed retry or another conversation is scored as the next user turn.
8. **Overstrict byte comparison:** harmless JSON encoding differences are treated as changed instruction text, or an unspecified MCP separator is treated as settled.
9. **Sentinel reuse in legitimate content:** a user quotation, controlled tool result or assistant reply contains profile text and is mislabelled as injected history. Design fixtures to avoid accidental ambiguity.
10. **Premature collection:** the driver advances before a save commits, reload completes, response finishes or child exits.
11. **Double counting:** one defective request is added once per category; a single omission is counted repeatedly at successive dependent steps.
12. **Compatibility expectations contradict the old path:** disabled desktop requests are required to contain model-system text they previously left for the daemon to supply.

Some of these should still block acceptance as infrastructure or operation failures. They should not be reported as observed incorrect instruction deliveries.

Q10: **The six named cases are represented as broad scenario shapes. The lifecycle table and both adapters’ tool continuations are not completely covered.**

The following belong in the **baseline scenario definitions and the reused final transport instrument**. Attempt the designated surfaces now and preserve their actual pre-feature outcomes.

1. **Creation timing and failed first turns.** Allocate a desktop draft at A, save B, then submit its first accepted turn: expect B. Start an idle terminal at A, save B before its first prompt: expect A. Exercise a failed first desktop turn, retry under the same ID, and a genuinely new ID after the global revision changes. These distinguish the events in [blueprint:79](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/docs/_design/G1_PERSISTED_INSTRUCTIONS.md:79) and [blueprint:194](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/docs/_design/G1_PERSISTED_INSTRUCTIONS.md:194).

2. **A genuine legacy transcript and its eventual reload.** Produce the transcript through the pre-feature HTTP path, then reuse it without an instruction binding. Test both revision-zero continuation and explicit adoption. D5 currently stops before that adoption.

3. **Concurrent save and rejected reload.** Hold a real turn, compaction and tool approval wait. Save B from another process; require save to complete while the active selection remains A. Attempt reload while held, then successfully reload at idle. Verify unchanged transcript and selection after refusal. Current D2/T2 edits occur between turns and do not establish the lease contract.

4. **Model and tool changes.** Change model-system text and available tools while A is pinned and B is current. Verify a rebuilt base with A retained. Exercise terminal `/tools` using its actual toggle syntax, MCP enable/disable, and relevant desktop tool changes. Both model selection and tools changes rebuild the terminal base today: [modals.go:205](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/cmd/tui/chat/modals.go:205), [input.go:402](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/cmd/tui/chat/input.go:402).

5. **Cross-adapter persistence.** Save through desktop and consume through terminal; save through CLI and consume through desktop. Test terminal `/instructions off` followed by a new desktop conversation, while another already-bound conversation retains its previous revision. Separate same-adapter tests do not establish a shared owner.

6. **Disable, restore and reset more fully.** Demonstrate A reaching a request before disabling it, retain A in an already-active conversation after global disable, adopt disabled state through reload, restart while disabled, and re-enable or restore retained text. D4 alone can pass a never-enabled implementation.

7. **Terminal identity and failure preservation.** Observe distinct identities for normal invocations and `/new`, not just changed text. Make `/new` selection creation fail and prove the old transcript and selection remain usable. `api.ChatRequest` has no conversation-ID field, so captured bodies alone cannot establish this identity requirement: [types.go:133](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/api/types.go:133).

8. **Compaction that demonstrably happened.** Assert the summariser request, successful resulting summary/history transition and subsequent reconstructed carrier. Save B before compacting A to expose accidental current-profile reads. Add repeated compaction, compaction after reload, automatic threshold compaction and tool-output-triggered compaction. The latter two have distinct production paths at [session.go:714](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/agent/session.go:714) and [session.go:751](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/agent/session.go:751).

9. **Desktop tool continuations and more than one terminal continuation.** Exercise a deterministic approved MCP tool through its real registration/execution path, and the web-search family with isolated external responses. Cover tool-only and text-plus-tool responses, multiple calls and multiple rounds. Assert the carrier and admission field on **every** resulting request, including after reload and compaction. Desktop request construction occurs inside its pass loop at [ui.go:943](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/ui.go:943).

10. **Compatibility controls with useful variation.** Add desktop unconfigured controls and disabled/unconfigured requests containing tools, thinking, options and supported attachments. Include model-system plus MCP text and `/system off` followed by `/system on`. A plain-text control cannot establish the full disabled compatibility claim.

Tool denial needs adapter-specific expectations: desktop creates a tool-error result and continues, whereas the agent can finish the run as denied. Do not declare a missing continuation merely because the other adapter would have sent one. See [ui.go:1093](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/ui.go:1093) and [session.go:354](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/agent/session.go:354).

The **final acceptance suite additionally requires** the real browser journey; storage corruption, interrupted initialisation, CAS and capacity guards; daemon capability/refusal/diagnostic tests; native and rendered admission tests; bounded real-model qualification; and the specified production mutants. Those are explicit obligations at [blueprint:337](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/docs/_design/G1_PERSISTED_INSTRUCTIONS.md:337), including I-09 through I-14. The simulated transport instrument can remain reusable within that suite, but cannot stand in for all of it.