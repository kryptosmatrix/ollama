# Ollama — Cross-Model Colleagues and Capability-Aware Routing

**Document type:** Conversation feature handoff; design intent, not an implementation-ready blueprint.  
**Prepared:** 20 September 2026 — Australia/Brisbane.  
**Prepared by:** Eko, for Ash and the Eko session implementing the Ollama enhancements.  
**Scope:** Features discussed in this conversation only.  
**Implementation status:** Not assessed for this export. This document does not claim that the proposed features have been built or tested.

## 1. Purpose and handoff boundary

Extend the Ollama colleague experience so the colleague talking with the human can delegate real work to fresh colleagues using suitable local or cloud models, supervise and steer them, and expose those working conversations in the interface. Model selection should use demonstrated task capabilities rather than an assumed intelligence hierarchy.

A second objective is to investigate reuse of the open-source Codex harness rather than automatically rebuilding equivalent mechanisms in the Ollama fork.

This is a companion to the ongoing Ollama enhancement work, not a replacement for its existing work list. The integration-work-list path supplied in the project context is:

`/Users/krypto/GitHub/ollama/docs/OLLAMA_COLLEAGUE_INTEGRATION_WORK_LIST.md`

That file was not read or modified when preparing this export. Reconcile this handoff with its current contents before adding work, changing priorities or choosing an implementation architecture. Do not duplicate a feature that another session has already implemented.

This export does not reproduce the separate conversations' complete requirements for LETHE memory, general conversation-context selection, interface performance, conversation export or waking-persona design. Where they intersect this feature set, the integration boundary is recorded without inventing their missing specifications.

### How to read the decisions

- **Requested:** A feature or behaviour Ash raised in this conversation. This records product intent, not blanket authorisation to implement or spend money.
- **Recommended:** Eko's proposed design, safeguard or verification approach. Preserve it for consideration without presenting it as an explicit Ash ruling.
- **Optional:** An enhancement discussed as optional, not a requirement for the first release.
- **Unresolved:** A decision or implementation contract still needing reconciliation and specification.

All example tool names are conceptual, not claims about existing APIs. No model ranking, concurrency limit, benchmark threshold or Codex integration route is treated as already measured or finally approved.

## 2. Conversation sources and feature inventory

The source of the feature descriptions is the visible dialogue in this conversation. These identifiers let a receiving session distinguish Ash's requests from Eko's elaborations without needing the original chat.

| Source | Ash's request | Eko's principal elaboration |
|---|---|---|
| C1 — Codex reuse | Examine the open-source ChatGPT/Codex harness repository and borrow knowledge or code to enhance the Ollama fork, including agents. | Distinguish Codex from the complete ChatGPT service; compare reuse, embedding and selective porting before rebuilding. |
| C2 — Cross-model waking | Allow an Ollama colleague to wake fresh colleagues on other model substrates while the parent remains active in Ollama. | Let the model request delegation and the harness create, run and manage separate sessions. |
| C3 — Parent management and UI | Make the active human-facing colleague the manager; let it watch and steer children; show nested child conversations with human input disabled by default; give children the main colleague's tools except further spawning. | Event-driven supervision, explicit steering delivery, inherited permission ceilings, independent human stop controls and accurate authorship. |
| C4 — Capability-based selection | Benchmark models' capabilities and reasoning so the parent can select models suited to each assignment. | Use multidimensional, configuration-specific evidence, real colleague evaluations and task-aware routing, not one intelligence score. |

The feature groups to reconcile into the ongoing work are:

| ID | Feature group | Source |
|---|---|---|
| F01 | Codex harness reuse and integration investigation | C1 |
| F02 | Fresh cross-model sub-colleague sessions | C2 |
| F03 | Parent-colleague management, observation and steering | C2–C3 |
| F04 | Nested, inspectable sub-conversations | C3 |
| F05 | Shared working tools with bounded authority and no recursive spawning | C2–C3 |
| F06 | Model capability evaluation and evidence profiles | C4 |
| F07 | Task-aware model selection and outcome feedback | C4 |

## 3. F01 — Investigate Codex reuse before rebuilding

**Requested:** Study the relevant open-source Codex repository to identify mechanisms that can improve the Ollama fork.

**Recommended investigation order:**

1. **Establish a comparison baseline.** Investigate running the Codex harness against models served through Ollama. Give the same model comparable repository assignments under the existing fork's harness and Codex, with recorded configurations and budgets. Compare completed work, tool failures, context handling and resource use.
2. **Prototype runtime integration.** Investigate whether Codex app-server can supply the agent runtime behind the existing Ollama interface and settings. Keep the human-facing experience and editable waking instructions under our product's control. Use a version-pinned adapter rather than assuming the protocol will remain unchanged.
3. **Port selectively where integration does not fit.** Adapt individual mechanisms when they fit the existing fork better than an external runtime. Document what was taken, its dependencies, its licence obligations and its ongoing maintenance cost.

These are alternatives to assess, not instructions to build three overlapping agent engines. The preferred route remains unresolved until compatibility and maintenance trade-offs have been tested.

Candidate mechanisms discussed include agent execution loops, tool operation, approvals, interruption, session resumption, context management, event streams and parent–child collaboration.

### Boundaries to preserve

The earlier discussion distinguished the open Codex harness from the entire ChatGPT service, proprietary models and managed services. Do not assume that a feature visible in a commercial product is available in the public repository or works unchanged with an Ollama-served model.

[Unverified for this export] Earlier replies identified an Apache-2.0 licence and documented Ollama/app-server integration options. Before importing code or making an implementation promise, inspect the selected repository revision, licence and NOTICE material, dependency licences and relevant interfaces. Retain required attribution and identify modifications as applicable. Reference leads are in §14.

A harness improvement is not a transfer of the underlying model's capabilities. Compare harnesses using the same model where possible; do not attribute a stronger model's performance to orchestration alone.

### Historical local evidence, not a fresh repository audit

An earlier Bridge inspection in this conversation reported Ollama at commit:

`953de98d408a9697fa20a48c23973b9dc8eee921`

It read `agent/session.go:1–225` and `cmd/tui/chat/chat.go:1125–1209`. Those excerpts showed an agent session with model/tool phases, approvals, skills, compaction and event sinks, plus a terminal-interface path constructing the session and invoking `Run`.

The repository was reported as clean `main` at that inspection. This is historical evidence only; the other implementation conversation may since have changed it. No complete runtime audit or feature test was performed by that source inspection. Re-read the live paths and current Git state before using them to plan changes.

**Investigation success condition:** Produce an evidence-backed decision about extending the existing runtime, embedding Codex or selectively porting mechanisms. A clone or a directory comparison alone does not settle that decision.

## 4. F02 — Wake fresh colleagues across model substrates

**Requested:** A running colleague can ask another approved model, local or cloud, to perform a delegated assignment within the Ollama experience.

### Desired interaction

The parent issues a conceptual `wake_colleague` request. The harness checks the assignment, model availability, access policy and resource budget, then starts a separate child session. It returns a child identifier promptly so the parent can continue other work, inspect progress, send guidance or wait for a result.

The child performs its own real model → tool → result → model loop. It is not one completion dressed up as a colleague and not a simulated dialogue written by the parent.

The child's result returns to the parent with the evidence needed for the assignment: findings, source references, test results, proposed changes or a clear statement of what could not be completed.

### Assignment information

**Recommended:** A delegation request should carry the task and required deliverable, requested model or capability needs, authorised working scope, relevant context, role/waking instructions and work budget. The eventual blueprint must define exact fields and validation; this handoff does not fix a public API or persistence schema.

Each child has a distinct session identity and a recorded relationship to its parent. Its actual model/backend is recorded rather than inferred from its display name.

### Fresh context

**Recommended:** Start with a new conversation history and a bounded waking package. Include shared colleague instructions, relevant project information and the assignment; do not copy the parent's entire transcript by default.

A reviewer should receive the requirements and evidence needed to evaluate independently, not automatically inherit the parent's preferred conclusion. Shared instructions do not make separate colleagues copies of the same conversation.

The separate LETHE and colleague-identity work may supply waking information, but the exact memory scopes, identity lifetime and history-retrieval rules were not settled here. Do not silently share unrestricted personal or project memory between all children.

### Parent activity and actual concurrency

The parent remains the user's active colleague while children are working. Keeping a session active is distinct from generating tokens simultaneously.

The intended design accommodates local-to-local, local-to-cloud, cloud-to-local and cloud-to-cloud assignments through approved Ollama-accessible backends. Compatibility still needs testing per backend. Local inference may need to queue or alternate; a queued child should not appear to be actively executing.

**Recommended initial proof:** A parent wakes a fresh child using a different model, the child independently reads an authorised file and returns a verifiable finding, and the parent uses that finding without losing either session or merging their histories indiscriminately.

## 5. F03 — The human-facing colleague manages its children

**Requested:** The colleague the human is already conversing with becomes the manager of the sub-colleagues it raises. It can observe their work and communicate with them to steer it.

There is no additional AI manager above that colleague. A harness-level management service supplies scheduling and enforcement; it is not a replacement human-facing persona.

Management belongs to the parent–child session relationship. Selecting a child in the interface does not promote that child, change the task owner or silently redirect the user's main conversation.

### Management responsibilities and operations

**Recommended:** The parent can assign work, inspect status and evidence, answer questions, send follow-up guidance, interrupt or cancel a child, wait for results and decide whether the result meets the assignment.

A completion report is something the manager evaluates, not an automatic success verdict. A different model agreeing with the parent is useful input, not proof of correctness or independence.

### Watching without flooding context

**Recommended:** The harness continuously records observable activity: emitted messages, tool requests and results, file-change activity, errors, questions and task status. The manager receives meaningful events and bounded summaries, with access to exact transcript and evidence when necessary.

Do not continually insert every child's full output into the parent's context. The human viewer can expose the detailed transcript without requiring the manager to read it all on every turn.

The proposed product needs to schedule the manager to handle child questions, blockers and results even when the human has not typed another message. Otherwise it cannot honestly claim active supervision. This is intended future application behaviour, not a claim that the present ChatGPT session continues working after its response.

### Two steering behaviours

| Operation | Intended behaviour | Evidence needed |
|---|---|---|
| Send guidance | Queue an instruction for the child's next safe model/tool boundary without disrupting the current operation. | Record its author, target and delivery progress. |
| Interrupt and redirect | Request cancellation of the current turn, identify completed or still-running work, then deliver revised instructions. | Record interruption outcome and the subsequent instruction delivery. |

**Recommended:** Distinguish guidance that is **queued**, **delivered to the child's context** and **acknowledged**. These do not mean the same thing, and acknowledgement does not prove compliance.

Steering is proved by the child's subsequent actions. For example, after “stop expanding the refactor and trace the existing call path”, the child should actually change what it does next.

Interruption does not undo completed writes or external actions. Exact safe boundaries, cancellation behaviour and provider limitations need a tested contract; do not promise mid-generation steering merely because guidance can be queued for the next request.

## 6. F04 — Nested sub-conversations in the existing viewer

**Requested:** Show each child conversation as a sub-conversation beneath the active colleague's conversation. Let the human inspect the exchange between manager and child. Disable the child message input by default to prevent accidental intervention.

### Viewer behaviour

**Recommended:** Make the parent conversation expandable and show each child with its name, assignment, actual model/backend, local/cloud location and current state. Selecting a child opens its transcript in the existing conversation viewer.

The transcript includes the manager's assignment and later guidance, the child's replies, tool activity and relevant status events. Preserve the real speaker and recipient. A manager-authored instruction must not be displayed as a human-authored message.

A suggested read-only banner is:

> Read-only sub-conversation · Managed by Gnomon

Names in examples are illustrative, not reserved product identifiers.

### Human intervention

**Recommended:** Provide a **Tell manager** action that returns to the parent composer with the selected child attached as context. The human can redirect work without accidentally creating a second instruction stream.

**Optional:** A deliberate **Message directly** control can unlock child input. Any human message is clearly attributed and reported to the parent so the manager does not continue from an outdated understanding. This is an optional extension; the requested default remains disabled input.

**Recommended:** Provide **Stop child** and **Stop team** controls independent of the manager's cooperation.

### Shared record and persistence

**Recommended:** The viewer and manager consume the same recorded events. Summaries link back to the underlying evidence rather than maintaining a separate account of what happened.

Parent–child relationships and transcripts should survive reopening the application. What happens to active tasks and processes across a shutdown is a separate unresolved lifecycle contract; restoring a transcript does not prove safe task resumption.

## 7. F05 — Shared working tools, inherited authority, no grandchildren

**Requested:** All tools available to the main colleague are available to its children except the ability to wake further sub-colleagues.

This includes editing and testing when those tools are available to the parent. The final feature is not restricted to read-only assistants.

### Tool access and permission boundaries

**Recommended:** Interpret shared tools as the same working tool catalogue within an inherited permission ceiling. Each child retains its own conversation, working directory and execution state. Delegation does not create wider filesystem, network or external-action authority.

An action requiring human approval still requires that approval. The manager cannot confer authority it does not have, and approvals should identify the child that requested them.

Working tools and management authority are distinct: removing the ability to raise children does not require removing the ability to edit, inspect or test within the assigned scope.

### Enforce the depth limit

**Requested:** Children do not wake their own children.

**Recommended:** Enforce this in the runtime, not only in a prompt or by hiding a tool description. The delegation service must reject a child-originated spawn attempt.

The conversation identified an important limit: unrestricted shell and network access can let a child attempt to launch another agent through a CLI or API. A hard no-spawn guarantee therefore also requires appropriate execution-boundary restrictions. Otherwise describe the restriction as a managed-workflow rule, not a security guarantee.

The exact enforcement design remains unresolved. Do not silently remove general working tools to conceal that design problem.

### Parallel edits, resources and cloud boundaries

**Recommended:** Use explicit file ownership or isolated workspaces with managed integration and cleanup. Multiple children possessing an edit tool does not mean they can safely write the same files concurrently.

The harness, not a prompt, enforces approved models, execution/usage budgets, cancellation and concurrency limits. Credentials stay with the harness.

Cloud delegation requires an explicit data policy. That policy covers later file contents and tool results sent to the cloud child, not only the initial assignment. A local parent does not make its cloud child's processing local.

**Recommended staging, not a permanent restriction:** Start the proof with one coordinator and at most two read-only children, then prove write ownership and integration before expanding. The production concurrency limit was not decided, and this initial test arrangement does not replace Ash's broader tool-access request.

## 8. F06 — Benchmark demonstrated capabilities

**Requested:** Evaluate models so the manager can select a colleague whose capabilities and reasoning ability suit the delegated task.

**Recommended framing:** Measure demonstrated capabilities under known conditions. Do not present a benchmark as a definitive measurement of “true intelligence”, and do not collapse the profile into one intelligence score.

Local/cloud location and capability are separate properties. A basic-looking function may require difficult reasoning; being cloud-hosted does not itself establish greater competence. The routing distinction is between the demands of assignments, not “coding” versus “cognition”.

### Capability dimensions

| Dimension | Intended measurement |
|---|---|
| Implementation | Producing working code from a precise specification, handling edge cases and protecting against regressions. |
| Reasoning and diagnosis | Solving unfamiliar constraint problems, tracing causes across files, testing competing explanations and revising conclusions when evidence changes. |
| Tool operation | Choosing tools, supplying valid arguments, interpreting failures and completing multi-step work without becoming stuck in a loop. |
| Evidence handling | Finding relevant information, preserving requirements across context, citing actual findings and distinguishing missing evidence from a negative finding. |
| Collaboration | Understanding assignments, asking useful questions, accepting steering, handing off usable results and reporting completion honestly. |
| Management, evaluated separately | Delegating appropriately, recognising inadequate child reports and resolving contradictory findings. Coding performance alone does not establish managerial competence. |

Alongside quality, record elapsed execution time, resource consumption, cloud usage and required intervention. Keep operating cost and quality as separate measurements rather than concealing their trade-off in a single score.

### Two complementary evaluation layers

**Recommended layer 1 — Controlled capability tests.** Give models matched problems, budgets and scoring criteria. Reasoning tasks should have checkable outcomes, changed premises and counterexamples. Eloquence or self-reported confidence is not a capability measurement.

**Recommended layer 2 — Real colleague assignments.** Wake each model through the actual child-session path, using the real waking instructions, tools, permissions and context handling. Assign bounded repository work, require verification and include a manager correction. Measure whether the complete colleague workflow succeeds, not only whether a standalone answer looks plausible.

Place the evaluation service beside the colleague manager in the harness rather than embedding the scoring policy in the inference engine. This is a recommended architectural direction; exact code ownership needs live-source reconciliation.

### Bind results to the tested configuration

**Recommended:** A result should identify the model and provider, model digest or immutable version where available, quantisation, context settings, reasoning budget, waking-prompt version, tool set, harness version and evaluation date.

The profile needs enough environment and evaluation information to explain what was actually measured. A cloud endpoint without an immutable model identifier must record that limitation. Do not make its result appear reproducible against frozen weights when that is unknown.

Model metadata helps identify the tested configuration; metadata alone is not capability evidence. Relevant configuration changes should trigger requalification rather than silently inherit an old score.

### Scoring and evaluator integrity

**Recommended:**

- For code, prioritise protected acceptance tests, regression tests and observable behaviour. A worker must not improve its result by editing the evaluator or weakening the acceptance tests.
- For open-ended work, use independent rubric-based review and label it as judgement rather than objective correctness.
- Use held-out tasks, repeated trials and visible sample counts and uncertainty. A few successful examples are insufficient to claim a dependable success probability.
- Use representative tasks from the intended environment as well as any public baseline. Do not treat a general leaderboard as a substitute for the workflow we are deploying.
- Record infrastructure failures separately from model reasoning failures, without hiding their practical effect on whether the assignment completed.

New models begin as untested, not automatically bad or trusted. Start with small compatibility checks and run deeper evaluations within an explicitly approved resource budget.

Exact task banks, scoring rubrics, repetition counts, confidence treatment, qualification thresholds and refresh schedules were not decided in this conversation. They belong in the subsequent evaluated design, not in invented numbers added to this handoff.

## 9. F07 — Choose models by task requirements and evidence

**Requested:** Let the parent select models relevant to the work it needs completed. Ash's example was a local coding model for a bounded implementation task and a more capable cloud model for demanding reasoning, where appropriate.

These are illustrative choices, not measured rankings of Qwen or any other model, and not an automatic local-versus-cloud hierarchy.

### Recommended decision flow

1. **Describe the assignment.** The parent supplies the task, required deliverable, relevant capability needs and operating constraints.
2. **Apply hard constraints.** Exclude candidates that violate permissions, local-only requirements, approved-provider rules, resource limits or required functionality.
3. **Consult relevant evidence.** Compare capability results for suitably matching tested configurations and task types, including sample size and uncertainty.
4. **Return candidates with reasons.** Explain why a candidate fits, what evidence supports the recommendation and what limitations remain.
5. **Let the parent choose within policy.** The parent remains the manager, but it cannot override privacy or spending boundaries through its choice.
6. **Observe the real outcome.** Preserve the assignment result and its verification status for later evaluation and model-selection improvement.

The proposed objective is to prefer the lowest expected total cost among candidates with sufficient evidence of meeting the assignment's quality requirement.

“Total cost” includes retries, review, repair and human intervention, not just initial inference. The method for estimating it remains unresolved. Difficult, poorly understood or high-consequence assignments should favour stronger evidence and independent verification over automatically trying the cheapest model first.

When no candidate has enough evidence, report insufficient evidence rather than fabricate a recommendation. That case needs an explicit product policy, such as further bounded evaluation or a human decision; no universal fallback was agreed here.

### Explainable selections

An illustrative bounded-task explanation is:

> The local coding model has evidence of reliable performance on comparable edits. This assignment fits its tested context range and has executable acceptance tests. Prefer local execution.

An illustrative diagnostic-task explanation is:

> This assignment needs cross-file causal analysis and resolution of conflicting evidence. Another approved model has stronger relevant results. Cloud processing is permitted for the material involved.

These examples specify the kind of explanation needed, not existing scores or recommendations for an actual installed model.

### Outcome feedback without circular self-certification

**Recommended:** Record verified successes, failures, human judgements and unresolved outcomes distinctly. “The child said done” and “the manager accepted it” are not automatically verified successes.

Real assignments can add evidence, but should not silently erase the distinction between protected benchmark results and operational observations. Preserve the evidence behind a claimed improvement.

The system should record which model/backend actually did the work. Two colleague names or two agreeing responses do not prove different model substrates or independent verification.

### Demonstrating that routing is useful

**Recommended first comparison:** Evaluate a small set of models and representative task families using a transparent selection policy. Compare the resulting system with always using one strong model on comparable assignments.

Measure accepted quality, human interventions, elapsed time and resource use. Claim a benefit only where results support it; a functioning router does not itself establish better outcomes.

## 10. Proposed acceptance checks

The checks below consolidate the proofs discussed in the conversation. They are recommended acceptance intentions, not completed tests or final implementation contracts. The receiving blueprint must map them to real entry points and executable evidence.

| Check | Behaviour that needs to be demonstrated | False completion it should reject |
|---|---|---|
| Codex comparison | Comparable tasks and model configurations inform a documented integration choice. | Declaring an architecture suitable because the source was cloned. |
| Real cross-model delegation | The child executes its own session/tool loop on the recorded backend and returns independently obtained evidence. | The parent fabricates several speakers, or the requested model silently differs from the actual worker. |
| Freshness and parent continuity | Parent and child have distinct histories; the parent remains the human-facing colleague and can use the returned finding. | A shared transcript relabelled as two sessions, or a child taking over the parent conversation. |
| Supervision without human prompting | A child question or blocker causes the manager to respond through the intended scheduling path. | A status badge changes but the manager only notices after Ash types again. |
| Effective steering | Guidance reaches the child and changes subsequent work in the requested way. | Recording an outgoing message or acknowledgement without behavioural effect. |
| Honest interruption | The system reports what stopped, what completed and what remains running. | Reporting cancellation as though completed side effects were undone. |
| Viewer accuracy | Displayed manager/child messages and tool activity correspond to the recorded events with correct authorship. | A cosmetic nested panel populated independently of the real sessions. |
| Input safety | Child input is disabled by default; Tell manager targets the parent; any optional direct-input path is deliberate and visible to the manager. | Selecting a child silently redirects ordinary human input. |
| Tool access and depth | Children use inherited working tools within scope, while runtime-controlled child spawning is rejected. | A prompt-only depth rule, or permanently removing editing/testing instead of implementing the requested access. |
| Human and policy control | Human stop controls, approvals, usage limits and cloud-data boundaries affect actual execution. | A model prompt or decorative control standing in for enforcement. |
| Persistence | Reopening restores usable parent–child relationships and transcripts. | RAM-only records or serialisation that cannot be used after reopening. |
| Benchmark integrity | Protected scoring distinguishes correct and incorrect work, preserves configuration identity and reports uncertainty. | Self-reported intelligence, editable acceptance tests or a score copied between materially different configurations. |
| Evidence-based routing | Relevant evidence and task constraints affect the selection, with an explanation or explicit insufficient-evidence result. | A fixed brand preference or an unsupported assumption that cloud means smarter. |
| Honest outcome feedback | A failed or unresolved result cannot become verified success merely because the worker or manager says it succeeded. | Circular agreement raising the model's reputation without independent evidence. |

## 11. Recommended delivery sequence

This sequence is Eko's proposed staging. It is not a reprioritisation of the other conversation's live work list.

**Stage A — Reconcile and choose the runtime direction.** Read the current fork, the existing integration work list and relevant Codex surfaces. Identify what already exists, what is being changed elsewhere and whether to extend, embed or selectively port.

**Stage B — Prove one cross-model child end to end.** Establish separate sessions, real tool execution, returned evidence, parent continuity and policy checks.

**Stage C — Prove supervision and the nested viewer together.** Add a bounded two-child scenario, event-driven manager responses, steering, interruption and accurate read-only child transcripts. Prove human stop controls.

**Stage D — Extend working authority safely.** Prove child editing/testing, workspace ownership, approvals, resource management and persistence. Do not describe the earlier read-only proof as the finished tool-access feature.

**Stage E — Evaluate and route.** Benchmark a small candidate set using controlled and real colleague assignments. Add transparent evidence-based selection and outcome records, then compare against a single-model baseline.

Keep ordinary single-colleague use available. Delegation should be used where it contributes to a task, not automatically added to every small request.

## 12. Decisions still requiring specification

| Decision | What remains unresolved |
|---|---|
| Runtime architecture | Extend the current Ollama agent runtime, embed Codex, selectively port, or a justified combination that avoids duplicated ownership. |
| Exact interfaces | Final tool names, request/response contracts, event contracts and live code owners. |
| Waking and LETHE | Fresh-child identity lifetime, memory scopes and which existing waking/context features supply the bounded package. |
| Manager scheduling | Event ordering, how manager attention is scheduled while it is also talking with the human, and resource limits for supervision. |
| Steering and shutdown | Safe delivery boundaries, cancellation guarantees, remaining-process reporting and active-task behaviour after application restart. |
| Tool containment | How to honour working-tool availability while enforcing permission ceilings and the intended no-grandchildren boundary, including shell/API escape paths. |
| Workspace coordination | File ownership versus isolated workspaces, integration authority and cleanup rules. |
| Product controls | Production concurrency and usage limits, delegation activation policy and whether optional direct child messaging is included. |
| Evaluation protocol | Task bank, scoring, protected evaluators, repetitions, qualification thresholds, uncertainty presentation and refresh policy. |
| Routing policy | How task requirements map to evidence, how total cost is estimated and what happens when no model is sufficiently qualified. |

Do not resolve these by silently hard-coding defaults and calling the result implementation-ready. Reconcile them in the receiving design or explicitly exclude them from a bounded first implementation.

## 13. Instructions for incorporating this handoff

For the receiving Eko session:

1. Read this document alongside the current `OLLAMA_COLLEAGUE_INTEGRATION_WORK_LIST.md` and the relevant live source. Preserve the other session's in-progress work.
2. Map F01–F07 to existing work-list entries. Record which requirements are already covered, need amendment, are genuinely new or conflict with existing decisions.
3. Keep Ash's requests, Eko's recommendations and optional additions distinct. In particular, preserve disabled child input by default, parent ownership, working-tool access and the prohibition on child spawning.
4. Resolve the integration and lifecycle questions before presenting a low-level design as implementation-ready. Apply the current TECHNE process available to that session rather than assuming an attached or historical copy describes the latest rules.
5. Require real runtime and negative evidence for any completed feature. This document contains no test results for the proposed system and no independent judge verdict.

The intended end-to-end experience is:

> Ash directs one colleague. That colleague selects suitable workers from measured evidence, delegates bounded assignments across approved models, watches and steers their work, and brings back verified results. Ash can inspect the team underneath the main conversation without becoming its message courier.

## 14. Reference leads carried from the discussion

These are reference leads from the earlier replies, not newly researched evidence for this export. Verify the selected revision and current documentation before relying on external implementation or licence claims. No exact model availability, price, context minimum or capability ranking is established by this list.

**Codex source and integration**

- Repository: <https://github.com/openai/codex>
- Licence lead: <https://raw.githubusercontent.com/openai/codex/main/LICENSE>
- Open-source scope: <https://developers.openai.com/codex/open-source>
- Provider configuration: <https://developers.openai.com/codex/config-advanced>
- App-server: <https://developers.openai.com/codex/app-server/>
- Multi-agent documentation: <https://developers.openai.com/codex/multi-agent>

**Ollama interfaces**

- Chat API: <https://docs.ollama.com/api/chat>
- Tool calling: <https://docs.ollama.com/capabilities/tool-calling>
- Cloud interface: <https://docs.ollama.com/cloud>
- Concurrency and operating considerations: <https://docs.ollama.com/faq>
- Codex integration: <https://docs.ollama.com/integrations/codex>

**Evaluation and routing references**

- RouteLLM: <https://github.com/lm-sys/RouteLLM>
- HELM: <https://arxiv.org/abs/2211.09110>
- Inspect agent evaluation: <https://inspect.aisi.org.uk/agents.html>
- LiveCodeBench: <https://livecodebench.github.io/>

---

**End of handoff.** This export records the conversation's feature intent and qualifications. It does not modify the Ollama repository, replace the existing integration plan, report new benchmark measurements or authorise unbounded agent execution.
