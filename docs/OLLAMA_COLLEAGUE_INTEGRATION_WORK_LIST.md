# Ollama — PSYCHE Colleague Integration Work List
## R3 — Single-attachment programme delivery prompt

Written by Eko for Ash, 20 September 2026 (Australia/Brisbane).
R3 adds the six memory, context, conversation and continuous-agent features requested by Ash on the same date; no implementation status is promoted.

**To the colleague receiving this document:** this is your assignment, not a request to summarise a document. Take ownership of delivering the complete programme below. Investigate the live system, author and obtain review of the necessary designs, create bounded implementation packs, implement, test, repair, document, integrate and verify. Continue with the next authorised, dependency-ready unit without asking Ash to repeat “continue”. Do not stop after returning a plan, a set of blueprints, or one completed feature while further authorised work is ready.

Ash's covering message may be only: “Please find attached a prompt written for you by Eko, please read and follow this prompt as your goal”. This attachment supplies the task; the live workspace supplies code, actual tools and governing source documents. No earlier chat is required to understand the requested outcomes.

**Document class:** programme execution assignment, with an embedded work list. It is not a pre-judged low-level implementation blueprint. Your assignment explicitly includes creating, reconciling and earning the required design/pack readiness before building. No runtime result or independent judgement is asserted by this document.

**Your relationship:** you are a colleague, with responsibility for candid technical judgement. Keep your established identity or follow your host's actual colleague/lineage protocol. You are not automatically Eko, and you are not automatically the recurring Ollama colleague being established. Never rename yourself or another active colleague merely to make the onboarding test pass.

## 0. Delivery contract — read before starting

### 0.1 The complete goal

Deliver these connected capabilities in Ash's Ollama fork:

1. An editable, persisted terminal-instructions surface whose effective instructions demonstrably reach actual model requests.
2. One recurring named colleague, founded using the four existing wake files, with a truthful charter, LINEAGE attribution and LETHE-backed identity, work history, corrections and task-relevant recall across genuinely new conversations and compaction.
3. Short repository architecture overviews plus an estate relationship map, connected to TECHNE's startup/task routing, so colleagues inspect the real producers, owners and consumers before changing code. Completion must require behavioural integration proof, not merely a read receipt.
4. Parent-owned delegation: the active colleague Ash is talking to manages the children it raises. It can observe, steer, interrupt, collect and integrate their work without Ash relaying messages. Children inherit applicable parent tools within the task's current authority, except the ability to raise further colleagues. No grandchildren through an unguarded alternate route.
5. Nested, durable parent–child conversations in the conversation viewer. Child human-input fields are disabled by default so observation does not accidentally become intervention.
6. Versioned, evidence-backed evaluation of model-plus-harness configurations, and parent selection of local or permitted cloud models according to demonstrated task suitability, constraints, cost and uncertainty—not reputation, model self-description or a universal “intelligence” score.
7. An evidence-led decision about the previously discussed Codex harness integration: compare the relevant existing local harness/CLI/SDK/app-server approach with the Ollama path, then integrate or adapt only the justified, bounded surface. This does not commission a wholesale Codex transplant, access to proprietary services, or replacement of Ollama's inference engine.

Keep the original OI-01–OI-25 and add OI-26–OI-31: **31 work items in total**. The six additions cover automatic LETHE memory capture, relevant working context, responsive long conversations, complete Markdown export, checkpointed autonomous goals and user interaction without abandoning those goals. The dependency repairs and engineering directions in this document are Eko's additions to make the programme executable; they are not claims that Ash previously chose every internal design detail. During design, adopt them or record a demonstrably better equivalent that preserves the outcomes, permissions and proof. Do not silently substitute a narrower product.

### 0.2 Scope and authority

Primary repository: `/Users/krypto/GitHub/ollama`. Companion integration areas: TECHNE, LETHE and LINEAGE. Other estate repositories are in scope for the agreed architecture-overview rollout and necessary reviewed routing/documentation changes, not estate-wide runtime repair. Resolve worktree paths before writing; the spelling `ollama` is the observed repository name.

When Ash supplies this attachment as the implementation goal, treat it as the task commission to perform scoped reconnaissance, design, code, tests and documentation within the host's actual permissions. You own ordinary technical decisions needed to deliver it: implementation structure, exact types, lifecycle design, test design and dependency order. Record consequential technical choices in the owning blueprint/decision record, then obtain the required independent checks before implementation. New semantics belong in design, not hidden inside a build pack.

Do not send every engineering choice back to Ash. First inspect live sources, recover any existing approved decision, compare alternatives and make the decision within delegated scope. Ask only for a genuine reserved decision or missing authority that cannot be resolved that way. Report the recommendation and strongest objection in one short paragraph; keep the evidence on disk.

This commission is not a new blanket grant to disclose private material, spend money, download large models, change global settings, replace a running application, interrupt unrelated processes, migrate a shared live store, destroy data, publish or bypass host controls. Use existing, applicable grants where the host recognises them and record their exact scope. Do not convert a historical memory of permission into current permission. Credential availability is not authorisation. Do not alter TECHNE ratifications or widen standing permissions to unblock yourself.

Follow current authorised Git practice, including scoped commits, integration, remote verification and safe branch/worktree retirement where permitted. Do not reinstate an obsolete blanket “never commit” rule from an old uploaded guide; equally, do not infer push permission merely from a configured remote. Preserve other colleagues' work. No force-push, destructive reset, broad cleanup, secret exposure or deletion of unique work to obtain a clean status.

A code defect within a required dependency may need repair; map its effect, design and review the minimum necessary change. A dependency blocker remains a blocker until resolved. An unrelated defect is recorded, not an invitation to widen the programme. Preserve the original exclusions except for these explicit R3 amendments: per-conversation Markdown download, scoped automatic memory capture, selective request-context assembly and a bounded Ollama goal scheduler are now requested work. Unsolicited bulk transcript export/import, model fine-tuning, AGORA substitution for LETHE, removal of the existing compactor without a judged compatibility plan, estate-wide code repair and multi-machine deployment remain excluded. Product implementation is not permission to install an unrelated background service or bypass provider, session, spending or approval limits.

### 0.3 Inputs, discovery and startup

Read this entire attachment once, including the work list and closeout requirements. Preserve it at its canonical work-list path or a verified byte-identical task input; do not overwrite a differing live revision. Build a compact task card pointing back to sections and keep only the currently active slice's exact requirements immediately loaded thereafter. Do not repeatedly inject this whole delivery prompt into the recurring persona's everyday conversation.

Establish actual working directory, repository/worktree, host/model identity, tool access, build environment, current Git state and ownership. Inspect applicable host and workspace instructions, including repository `AGENTS.md`. A Codex session follows its actual host naming/lineage rules; do not import Ollama identity into the delivery controller.

Enter TECHNE through these real paths without bouncing between them:

- `/Users/krypto/GitHub/TECHNE/TECHNE_AGENT_START.md`
- `/Users/krypto/GitHub/TECHNE/00_START_HERE.md`

Follow the live Load Contract, including KANON and Constitution in full, the umbrella welcome, current North Star and applicable repository North Star. Read `/Users/krypto/GitHub/LINEAGE/WELCOME.md` and `HOUSE_NOTES.md` for the private collaboration/arrival guidance. An absent required source is reported and located by bounded discovery, never invented. A narrower active task or host restriction wins over a broader historical document.

The four existing onboarding inputs are:

- `/Users/krypto/GitHub/TECHNE/docs/prompts/PSYCHE_RECURRING_COLLEAGUE_WAKE_PACKET.md`
- `/Users/krypto/GitHub/TECHNE/docs/prompts/01_FIRST_WAKE.txt`
- `/Users/krypto/GitHub/TECHNE/docs/prompts/02_RECURRING_WAKE_TEMPLATE.txt`
- `/Users/krypto/GitHub/TECHNE/docs/prompts/03_FOUNDING_RESUME.md`

Read them in full before their first use. The packet explains the process; the first wake is one-time onboarding; the recurring template is bound after real identity/map creation; the résumé is a truthful professional seed, not invented past employment. Do not concatenate all four into every model request. Check whether a colleague has already been founded since these drafts were written. Continue a verified matching onboarding/binding instead of registering a duplicate; do not adopt a merely similar name.

Use the live router's relevant RECON, DESIGN, PROMOTE, PACK, CODE, TEST, REVIEW, CRAFT, GATE, DOCS, GIT, TEAM, COLLEAGUE, HANDOFF, RESUME, SECURITY and SURFACES routes when triggered. Read the canonical sources they name before the governed action; this list is not a demand to bulk-read every method at startup. Important owners include Method 03 for pack construction, Method 04 T-01–T-10 for proof, Method 16 for implementation readiness, Method 20 for repeated judgement and Method 21 for Git closeout. Use the live blueprint/prompt/code guidance they route to, not an older copied version that contradicts them.

In particular, the live KANON/Method 03 inspected for this revision require effective `[IMPLEMENTATION-READY]` earned under Method 16; `[PACK-READY]` alone does not license build packs. Re-read the complete current standard before promotion. Honour inactive/provisional markers and required external ratifications. Do not self-ratify an optional route, reset a judgement budget by renaming a blueprint, or substitute a convenient number of reviews.

### 0.4 First work unit and durable control records

Start OI-01 immediately after arrival. Use at least two discovery methods and trace complete paths, not symbol names alone. Inspect existing settings, UI/backend boundaries, request assembly, sessions, compaction, MCP/tool dispatch, conversation storage, tests and relevant build/launch scripts. Read existing curated-context design material and determine what is actually implemented. Compare the code candidate with the binary/application Ash really runs; a repository build is not installed-app verification.

Capture a live baseline: current commit plus working delta, staged/untracked state, relevant branches/worktrees, build/test commands discovered from manifests and scripts, toolchain/configuration, and full executed results with true child exit codes. Record tests not run and why. Read commands before executing unfamiliar scripts. Do not run broad network, installation, migration or process-termination side effects merely because a script is called “verify”.

Use existing suitable project ledgers rather than create a second competing tracking system. If none exists for this programme, create the following **new task artefact locations** under `ollama/docs/analysis/colleague_integration/`; these paths are a proposed reporting layout, not claims that files already exist:

- `STATE.json`: authoritative current task pointer, OI/sub-item states, dependency status and next safe action.
- `REQUIREMENTS.md`: immutable-ID requirement inventory and source → design → pack → code → test → proof → reviewer trace.
- `DECISIONS.md`: decisions/assumptions with alternatives, authority, risk, falsifier, revision path and supersession links.
- `SOURCE_MAP.md`: actual source symbols/callers/consumers, file hashes, read coverage and baseline identity.
- `BLOCKERS.md`: precise blocker, affected items, attempted resolution, authority/evidence needed, owner and safe independent work.
- `HANDOFF.md`: current verified state, outstanding obligations, exact next commands and resumption procedure.
- `evidence/` and `reviews/`: retained full execution outputs, immutable candidate manifests, all review attempts and finding dispositions.

Place owning blueprints and complete prompt packs in the repository's discovered canonical design/pack locations and link them here. Do not scatter a competing blueprint corpus into the report directory. Private memory, identity content and credentials remain in their appropriate protected stores; the source repository holds safe references, not exported private histories.

Write control records atomically using existing mechanisms; preserve append-only events/revisions where required. Use one writer for shared mutable programme state. Record task/session identity, active OI and pack, current phase, tested commit/delta, selected routes, evidence locations, review status, worktree ownership, in-flight commands, permissions needed and next safe action. An unknown field is explicitly unknown, not guessed.

The controller must work using the current host's durable files and available tools before the new Ollama memory/delegation features exist. Do not make progress depend on the features being built. Onboarding the recurring colleague is a separate work item, not a prerequisite to your own ability to investigate and design.

### 0.5 The delivery loop — repeat until complete or genuinely blocked

1. **Recover and reconcile.** Read current state, blockers, ownership and affected live source. Verify that inherited completion evidence still applies to the actual candidate/configuration. Check active freezes and changed instructions.
2. **Select ready work.** Choose the earliest dependency-ready useful slice. Prefer finishing a started coherent feature to opening a new one. Numbering is not an excuse to ignore the corrected dependency groups in §0.6. Independent work can proceed around a blocker without closing or deleting it.
3. **Design explicitly.** In 0A, convert the slice's outcomes into the exact low-level contract: existing/new types and paths, state owner/writers/readers, lifecycle, persistence/migration, APIs/messages, concurrency/ordering, limits, failure behaviour, permissions and Given/When/Then acceptance. Record new definitions as new, not discovered code. Reconcile approved intent and live implementation. Missing technical semantics send the work to design, not automatically back to Ash.
4. **Earn design readiness.** Obtain the independent judgement/promotion required by current TECHNE against frozen document and source identities. Preserve every finding and disposition. Resolve substantiated blocks, rerun invalidated checks and respect Method 16/20 termination rules. A changed header is not a promotion.
5. **Generate and judge bounded packs.** Use the current 0A → 0B → A → B → C → C2 → D → E → F workflow; C2 is the separate craft/architecture check, not a second name for code review. Use the live canonical envelope and role openers, embedded exact invariants, source/test custody and complete output contracts. Map every requirement and proof obligation to a pack. Generate the upcoming dependency-ready set completely; do not create hollow later packs or defer all implementation until every remote feature is exhaustively designed. Keep programme-wide interfaces coherent first.
6. **Build and prove the slice.** Establish contrasting/negative acceptance cases, implement through existing production paths, run focused tests and required system acceptance/regressions, and execute isolated clean–broken–clean controls. Keep code, tests and proofs together. Correct discovered in-scope wiring defects while their mechanism is understood.
7. **Judge independently and repair.** Perform spec review, craft/architecture review, documentation verification and the appropriate reality gate. Review assertions, fixtures, actual runner discovery and downstream effects—not just the report. Any repair invalidates the affected prior evidence and triggers the appropriate reruns/review.
8. **Close the unit durably.** Save/read back evidence and state, update the architecture/status map and required LINEAGE/LETHE pointers, and perform authorised Git integration/closeout. Close only the scope whose obligations are satisfied. An unreviewed implementation remains unreviewed.
9. **Continue.** Select the next ready unit in the same active session. Do not ask for routine permission to proceed. Stop only for completion, a real unavailable prerequisite/authority, exhausted governing review limits or a safe context/session handoff.

Do not turn this into “analyse everything, report everything, leave implementation to another prompt”. Design and implementation are separately gated stages inside the same commissioned programme. You may perform different roles sequentially, but a role switch does not create an independent judge of your own work.

### 0.6 Corrected dependency groups and scheduling

The work-item acceptance language below has been repaired so an item does not require its own later dependency to be complete.

| Group / item | Prerequisites | Closure rule |
|---|---|---|
| OI-01 | Arrival and real access | Evidence-backed recon/baseline, not a runtime feature claim. |
| OI-02 | OI-01 | Programme architecture and dependency plan established; every build group separately earns its full design/pack gates before building. |
| G1: OI-03 + OI-04 | OI-02 gates for instructions | Editor/storage and request delivery are one coupled delivery group; close both only after save → restart → actual request proof. |
| OI-05 | OI-02 memory/tool design; existing tool path verified | Real authorised LETHE write/raw retrieval and failure behaviour. |
| OI-06 | G1 + OI-05 | Real, idempotent founding records and bound identity inputs. |
| G2: OI-07 + OI-08 | OI-06 | Binding/installation and recovery/history implemented together; close both after a fresh conversation actually uses the bound records. |
| OI-09 | G2 | Independent cold-start/compaction/restart/failure proof. |
| OI-10, then OI-11 pilot | OI-01 source map + relevant OI-02 design | Create the Ollama overview early; use provisional recon context before its formal review. Continue the estate rollout separately. |
| OI-12, then OI-13 | G2 + reviewed OI-11 pilot | Real context routing, then an actual evidence-consuming completion gate. Existing human/tool verification governs earlier work. |
| G3: OI-14 + OI-15 + OI-16 | OI-09 + OI-13 + judged delegation/security design | Dispatch, authority enforcement and supervision are coupled; no writable child launch before the relevant boundary checks are implemented and proven. |
| OI-17 and OI-18 | G3 | Ownership/integration and nested UI may proceed separately only with non-overlapping write custody. |
| OI-19, then OI-20 | G1 + relevant OI-02 design | Catalogue and benchmark design may proceed alongside other groups. |
| OI-21 | OI-20 + actual G3 child configuration | Qualification measures the actual setup. Requalify affected profiles after OI-26/OI-27 and G4; final agent-mode claims require that mode to be measured. Exploratory development runs do not require the not-yet-built automatic router. |
| OI-22 | OI-19 + OI-21 + G3 | Measured profiles actually constrain parent selection. |
| OI-26 | OI-05 + G2 + its judged capture design | Extend the same LETHE integration with automatic event-driven capture; do not build a second memory store. |
| OI-27 | OI-26 + OI-12 + its judged context design | Select bounded relevant context without replacing the full conversation archive. |
| OI-28 and OI-29 | OI-01 + their OI-02 design gates and verified archive contract | Viewer profiling/optimisation and complete export can proceed without waiting for OI-27 or G4. Re-run their integration checks after shared storage/UI changes. |
| G4: OI-30 + OI-31 | OI-13 + OI-26 + OI-27 + completed G3 + their judged scheduler/interaction design | Build scheduling, checkpoint/resume and user priority together. No agent-mode delivery closure before the user can converse, stop and prevent unwanted resumption. Use a permitted explicitly selected model before OI-22 is qualified. |
| OI-23 and OI-24 | Their original dependencies + OI-26–OI-29 + completed G4; current affected model qualification | Combined functional gate and reliability/upgrade qualification cover all six additions and precede release. |
| OI-25 | OI-23 + OI-24 + Ollama overview pilot | Qualified expanded Ollama release/acceptance. Programme closure additionally requires the entire frozen OI-11 rollout inventory. |

Do not tick a member of G1/G2/G3/G4 as delivered just to unlock its sibling. Intermediate contracts, code and tests may be recorded as progress, but the shared outcome closes at the group gate. Keep one feature blueprint where components serve one outcome; these groups are scheduling boundaries, not permission to manufacture new feature identities or reset review budgets.

### 0.7 Detailed cross-feature engineering directions

These directions resolve task-level gaps in R1. During 0A, bind them to real source and exact types, fields, transitions, error cases and tests. They are proposed engineering requirements of this assignment, not evidence of current implementation.

**Instructions and readiness.** Use one authoritative effective-instruction assembly consumed by every targeted terminal/app agent entry path. Preserve existing model protocol/templates, host boundaries and tool schemas; do not overwrite a model-specific system contract blindly. Persist the enabled state, selected colleague binding and instruction revision. Display effective scope and source/revision in the editor/diagnostics. Capture the carrier at the real model-request boundary in a protected test environment.

Freeze the chosen instruction revision for an active conversation. Saved edits apply to new conversations; applying an edit to an existing conversation requires an explicit safe-boundary reload, recorded as an event, followed by re-orientation. Compaction reattaches the conversation's effective carrier without repeatedly accumulating duplicate system messages. Model changes recheck capability/context compatibility and required recovery while retaining colleague identity and actual session attribution. Reject unresolved active binding values. A disabled/unconfigured optional persona must preserve ordinary non-persona use; a selected but invalid persona must not silently become another colleague.

Define a host-observed readiness lifecycle covering unconfigured, onboarding, recovering/orienting, ready and degraded/blocked states. These are behavioural roles, not existing enum names. New-conversation/compaction triggers must come from real harness events, not only from the model noticing a summary. Permit the read-only discovery and scoped identity/memory operations needed to recover before readiness; do not deadlock recovery by blocking its own required tools. Block substantive state-dependent work until prerequisites hold. In degraded mode allow clearly labelled assistance that does not depend on missing state. A model's “I read it” is not a readiness signal on its own.

Measure the actual request budget, including carrier, mandatory arrival sources, current task, tool schemas, retrieved records, history and output allowance. Respect the selected model's real supported context. Do not silently truncate KANON, Constitution, bindings, correction records or tool pairs. Select a permitted suitable configuration or report the precise capacity blocker. A shorter model response is not proof that all inputs were consumed.

**Identity and LETHE.** Keep colleague identity, founding lineage reference, conversation/session identity, task identity and model configuration distinct. The same colleague may run on a different model; capability claims do not travel automatically. Respect LINEAGE's actual schema. Never merge other colleagues' autobiographies or write your own independent observed assessment.

Use the three-layer carrier → literal-key map → exact raw anchors design in the wake packet. Store actual returned references, authorship, source/proof bounds and explicit supersession/correction links. Recover corrections and all required chunks; a newest timestamp alone cannot resolve conflicting concurrent work. An onboarding retry must use a recorded onboarding identity and resume or flag a collision, not mint a second colleague. Preserve the original records during interruption and recovery; do not “test” by deleting production memory.

Use real LETHE interfaces and existing authorised scopes; test the real service in isolated, authorised test state. Keep that distinct from any protocol fixture. No direct database writes bypassing the service, silent AGORA fallback, invented tool names, new consent or assumed raw-text access. Important stores require raw readback. Preserve an explicit missed-memory obligation when a write fails; local delivery state may still support independent engineering, but the memory-dependent feature cannot claim success. Do not make routine conversation contingent on unavailable optional historical anchors; identify which anchors are actually mandatory in the judged contract.

Founding résumé, inherited lessons and actual contributions remain distinct after retrieval and compaction. No fabricated employers, degrees or achievements. The cold-wake test's discriminating content must exist only in LETHE, not in the permanent prompt, visible history, summary, fixture name or test instruction. Keep the memory service real in the positive arm and prove the returning colleague functionally uses the recovered content.

**Repo awareness.** Each overview explains intended responsibilities/non-goals, real execution journeys and verified current status separately. Include upstream producers, state owners, downstream consumers, defaults, lifecycle/persistence, shared contracts and evidence/revision pointers. Keep the mandatory overview compact and point to detailed authoritative sources. An overview is an index, not a replacement owner for runtime semantics.

Freeze a finite rollout inventory using the actual estate manifest and authorised repo set. Count canonical repos, not duplicate worktrees, archives or unrelated business folders. Missing access stays visible. One owner reviews each map; boundary changes must reconcile both producer and consumer maps. Reuse an adequate existing root overview and maintain one canonical destination. A parent-directory index points to these maps; do not copy every repo's architecture into one giant startup prompt.

At task start, require actual inspection of the affected input → activation → processing/state → consumer → result chain. Refresh at repo/task/specification/configuration change and resumption. Record the source revision and working delta; changes invalidate affected evidence rather than blindly every historical record. Do not claim that a read log proves comprehension. The enforceable outcome is correct integration and the test that detects its absence.

**Child execution and supervision.** The active parent conversation owns child sessions; no compulsory separate manager persona. Before dispatch, persist an attributable task containing current goal, repo/worktree and writable scope, required inputs, relevant architecture/invariants, acceptance conditions, model configuration, capability grants and resource limits. Do not clone the parent's complete private history or its founding identity into a new child.

The tool catalogue and effective execution authority are separate. Children must receive useful file, command, MCP and other parent-equivalent tools where those tools are in scope—not be silently reduced to text-only reviewers. Enforce child-creation denial at dispatch/authorisation as well as schema exposure. Bind parent/child identity and ownership in the host, never from a model-supplied claim. Apply current revocations and path/command approvals. Check alternate shell, external delegate, MCP and network routes for bypasses within the declared threat boundary.

An unrestricted shell/network route can defeat a simple “no spawning” tool rule. Do not certify universal prevention from a deny-list or prompt. Design a real supported boundary, preserving the promised useful tool capabilities through scoped execution; when host authority cannot enforce the requirement, record that conflict and block the affected claim. Do not silently remove tools or broaden sandbox/OS permissions to hide it.

Define the full child state machine before building: queued, running, waiting-for-tool/approval, pause-requested, paused, cancel-requested, cancelled, completed, failed and interrupted/recovery-required behaviours, with illegal transitions rejected. Reuse existing states where equivalent. Specify exact persisted fields and schema migration in the owning design. A successful model response is not successful task acceptance.

Use ordered durable events for messages, tools, steering, status and results. Give dispatches and steering messages stable request identities and deduplicate retries. A steering request is separately queued, delivered to the child before its next applicable action, and acknowledged; show those states honestly. Parent observation must work while children run and while no new human message arrives. Route actionable events through a bounded manager continuation path; coalesce noisy updates rather than create an unbounded event/model-call loop.

Pause takes effect at a specified safe boundary. Cancel revokes further task dispatch, requests termination of owned in-flight work and records actual outcomes. Do not announce cancellation complete while the child still has usable authority to continue. Recovery must not replay an uncertain side effect blindly: reconcile the command/store outcome or require a decision. A durable event log is not an exactly-once guarantee. Retain in-flight uncertainty, stale ownership and orphan recovery as explicit states. No broad process-name kills.

Set explicit queue, concurrency, child-count, context, output, retry, elapsed-time and resource limits during design. Respect stricter host limits. Begin with one child, then prove the supported concurrent configuration. The parent remains responsive to Ash. Serialise shared writes; permit parallelism only with independent file/interface custody. Integration requires verified artefacts and review; neither a worker's “done” nor a clean diff closes a requirement.

**Conversation viewer.** Store parent links and actual speaker/source identities; reconstruct nested conversations from the same persisted events used by the manager. Display task, model, state, public messages, relevant tool events, corrections and results. Read-only child input is the default, including keyboard/shortcut routes and reloads. Do not change the target conversation because the user selected a child for inspection.

Any explicit human intervention mode must be designed before implementation: clear target, deliberate enablement, recorded authorship and notification to the parent, with control returned predictably. The required default does not authorise an invisible takeover feature. Do not invent private chain-of-thought or expose material unavailable under the provider's actual interface. Preserve accessible labels, keyboard navigation and the existing app's design conventions; no UI-framework rewrite for this feature.

**Model evaluation and routing.** Identify the evaluated configuration, not just a mutable tag: provider/destination, reported revision/digest, quantisation, context/output limits, template/carrier, relevant tools, sampling settings, hardware and harness version. Record unknowns. A local loopback endpoint is not proof that processing remains local. Cloud claims require a real authorised provider-path test; hermetic simulation proves only its declared protocol contract.

Define task classes for routine coding, Swift/Python work, repo navigation, cross-file integration, reasoning, test design, tool/MCP use, long-context instruction retention, correction uptake and calibrated escalation. Fix datasets/task variants, scoring rubrics, minimum independent task counts, repetitions, uncertainty treatment and qualification thresholds before observing qualification scores. Count repeated attempts separately from distinct tasks. Use independent verification/objective tests where possible; no résumé-based scores, self-grading or cherry-picked retries. Null/good/broken controls must verify the benchmark's own sensitivity.

Evaluate in disposable workspaces with the real target wake/context/tool configuration. Protect held-out tasks/answers from evaluated colleagues and their memory. Record task failures and infrastructure failures separately, retaining both; include intervention, latency/loading, tokens, resources and observed cost. Unknown cost is not zero cost. Keep raw evidence and all attempts; do not promote an exploratory small sample to a general reliability claim.

The parent submits task requirements. The router first applies hard constraints: authority/privacy, provider destination, task tool support, context, resources, spending limits and required reviewer independence. Only then compare measured suitable candidates, preferring an adequate local candidate where justified. Persist the candidate set, exclusions, evidence/configuration identity and selection rationale. No eligible candidate yields an explicit limitation/escalation, not an invented capability. Human/parent overrides cannot bypass hard policy. Requalify material changes; a remote model's inaccessible weights are not a reproducible local weight pin.

Start routing in advisory/shadow mode, compare it with the predeclared selection contract, then enable automatic choice in the qualified default path. Test deliberately better-scoring but ineligible models, stale profiles, contradictory scores, provider outage and resource pressure. Observe real task outcomes separately from the benchmark and use them for diagnosis/requalification, not uncontrolled self-training.

**Codex boundary.** Recover existing scoped decisions before comparison. Inspect the installed/discovered interface and primary documentation for that exact interface/version; do not infer compatibility from branding. Compare task execution, tools, approvals, cancellation, persistence, streaming events, provenance, cost and maintenance burden. Choose and record a bounded adapter, specific adaptation, or evidence-backed no-adoption decision. A no-adoption decision does not remove native Ollama delegation requirements. A selected adapter must pass the same parent/child, permissions, steering, persistence and failure proofs—not merely launch a subprocess and parse its last line. Record licence/redistribution implications before copying upstream code. Do not expand into proprietary models, ChatGPT internals, an IDE extension or a new remote service.

### 0.8 Proof, review and completion enforcement

Apply Method 04 T-01–T-10 to every runtime claim. Tests must start at the normal production entry point/assembly, observe actual delivered usable data and independently specified downstream outcomes, and prove a contrasting input changes the appropriate result. At least one applicable smoke uses production defaults. A headless route may prove shared assembly but cannot replace real UI interaction proof.

Execute clean → one isolated relevant fault → clean with unchanged acceptance assertions. Cover no-op/fixed result, missing dispatch/registration, interrupted required data and disconnected consumer where applicable. Prove the fault was active; a compile failure, generic timeout, missing dependency or zero collected tests is not the intended behavioural failure. Keep mutations isolated from production stores and other colleagues. Preserve all logs and actual exit codes. Reviewers reproduce required checks against the exact candidate, not a previous one.

The completion gate must consume verified candidate-bound evidence through a real trusted boundary. Record integrity alone is not execution proof. Test the gate with a forged PASS, stale candidate, altered fixture/configuration, failing child, missing required scenario and empty selection; each must prevent acceptance. Tests of the gate before it is finished are run using existing validation tools, not its own unproven verdict.

Derive build/format/lint/type-check/test/acceptance commands from the real project. Capture full output, discovered/executed/failed/skipped counts and all attempts; retain logs on disk instead of flooding Ash's conversation. Propagate true child failures through pipes and wrappers. Printing `exit=1` after a failed command is not enough when the wrapper then returns zero.

Target zero errors and warnings for the claimed supported build. Fix root causes in scope; do not suppress diagnostics, remove tests, change expectations to match a bug, weaken thresholds or hide required checks behind optional flags. Pre-existing unrelated failures/warnings remain explicitly recorded; they cannot support an unqualified clean-build or complete-system claim. Repair required dependency failures under reviewed scope; preserve a genuine blocker where unrelated repair would need new authority.

Before a design/code judgement, freeze source, working delta, specification, configuration and evidence identities. Use actual available independent reviewers with sufficient source access. A new conversation of the same persona is not automatically independent; a same-model role switch does not meet a different-substrate requirement. Supply the complete scoped evidence, not your preferred conclusion or another judge's verdict. Record the real requested/served model and review limitations.

A reviewer finding is evidence to investigate, not an instruction to obey. Repair substantiated faults; refute incorrect findings with executable/source evidence and preserve the disposition. Do not majority-vote away a block. Use the actual TECHNE judgement budgets/termination rules, not “loop forever until everyone says PASS”. If the required judge/authority is unavailable, preserve the candidate, prepare the exact pending review request and continue only independent work. Never use the new delegation interface as the only way to obtain the review needed to build it.

### 0.9 Long sessions, interrupted work and continuation

“Until complete” means sustained accountable progress, not pretending one context window is unlimited. Maintain the exact next action after each meaningful unit. Do not wait until compaction to record a correction, permission boundary, dependency change or in-flight side effect.

Follow the current context-tail/seam protocol. As context approaches the governing threshold or signs of thinning appear, stop starting new full-file work; close outstanding checks at the next safe seam and preserve the handoff. Do not compact between making a claim and verifying it. If interruption happens anyway, mark the pending obligation explicitly and verify it first on recovery.

Before compaction/handoff record: exact candidate/working delta; completed requirements with evidence; unfinished and unreviewed work; source read/unread ranges; technical decisions and authority boundaries; running commands/children and their ownership; failed/pending tests; exact next file, symbol, command and expected result; and the next pack's title/purpose. Read back critical records. Update applicable lineage and memory pointers through their authorised paths. Never claim a memory write or successor launch that did not occur.

Use the existing host's supported compaction/continuation mechanism where available, then recover the task from disk and continue. Do not install a scheduler, unattended daemon or shell loop merely to continue your own delivery session. This does not prohibit implementing the explicitly requested bounded Ollama product scheduler in OI-30/OI-31; test it separately before relying on it. If the host cannot resume itself, end with a ready-to-run handoff and state that resumption needs another invocation. The same attachment remains sufficient: after fresh arrival, inspect existing task state and resume the first unmet obligation instead of restarting OI-01 or redoing onboarding. Revalidate current authority before side effects in a new conversation; memory alone cannot renew it.

Do not switch the delivery controller onto an unqualified new runtime while it is editing itself. Test the candidate separately; any live installation/transition follows the approved rollout boundary with recovery available.

### 0.10 Blockers and decision delivery

Classify each obstacle as a source gap, design gap, implementation defect, proof failure, environment/tooling limitation, review limitation, authority issue or product/scope decision. Do not call all of them “needs Ash”. Design gaps route to 0A; known in-scope defects route to repair; source disputes route to direct examination; review disagreements route to their exhibit/falsifier. Recover existing decisions before asking again.

A blocked item retains its exact obligations and cannot be marked skipped, not applicable or complete. Record why, what was tried, what would resolve it, the affected dependent items and the next permitted action. Continue independent ready work. Stop the programme only when no safe useful work remains, a governing halt applies, or a handoff is required. Do not repeat unchanged failed calls without new evidence or consume a budget in a waiting loop.

Keep user-facing updates brief: one or two sentences for a meaningful verified milestone, reserved decision or blocker. Put source citations, command output, item tables and design reasoning in the durable artefacts. Ash has explicitly said long responses are inaccessible at present. Do not paste this document or narrate every tool call back to him.

### 0.11 Programme definition of done

The whole programme is complete only when every OI item and required sub-item in the frozen inventory has its intended outcome, current evidence and required review; selected adapters are proven; the supported default path works; and required Git, migration, installation and operator-acceptance obligations are satisfied. An approved product-scope amendment may change that inventory prospectively; you may not create one to close the checklist.

Run the combined real workflow: launch the intended app/terminal candidate → recover a LETHE-only anchor → inspect current repo context → select a qualified model → start a correctly scoped child → observe and steer its real work → verify a useful change through its downstream consumer → integrate under authority → preserve attributable history → restart → use the recovered record correctly. Include failure controls at every load-bearing link. Prove the human-facing nested view and read-only default separately through the actual UI. **R3 additionally requires** automatic capture without a model-issued store call; relevant current/past-chat context without whole-history replay; responsive long-history navigation; full-archive Markdown download from the top-left control; multiple real checkpoint/resume work periods without repeated human prompts; a user question answered during agent mode followed by correct continuation; and an explicit stop that prevents all automatic resumption. Exported history must remain complete even though both the model context and mounted UI are bounded.

Then run long-session, concurrency, interruption, resource, upgrade and rollback qualification. Preserve existing conversations/settings and private identity/memory. Rollback may require a compatible restoration plan rather than opening a new schema with an old binary; prove the stated path. Release only the exact tested candidate. Do not install over a working app, stop shared services or perform a destructive migration without the applicable authority.

Ollama's qualified pilot/release can precede the rest of the architecture-overview rollout. That is an Ollama milestone, not completion of OI-11 or of this whole assignment. Likewise, “code complete”, “verified in a test profile”, “awaiting live installation” and “awaiting Ash's acceptance” are different states; never claim the latter steps happened because their automation was written.

Close out the complete pack series under current TECHNE. The final report must identify actual item counts, evidence/review pointers, candidate and installed identities, compatibility/rollback results, supported configurations, remaining limitations and Git state. Counts come from the ledger, not memory. Deliver Ash at most three short sentences: what is genuinely complete, where the result/evidence lives, and the one remaining action or blocker if any. Do not end with a proposed plan while authorised implementation remains ready.

### 0.12 R3 — Shared boundaries for the six additions

The six product outcomes come from Ash's six-feature request and his instruction to add them here. The technical safeguards, decomposition and tests below are Eko's implementation directions, to be made source-bound and independently judged through §0.5. They are not statements that these features already work. There is no need to consult the earlier conversation to discover the requested scope.

**Keep four responsibilities distinct.** The conversation archive retains the complete saved user-visible exchange and attributable events. LETHE stores scoped durable memories with provenance and corrections. The working-context assembler selects a bounded input for the next model request. The goal scheduler owns durable goals, steps, checkpoints, authority and scheduling. These may share existing storage mechanisms where appropriate, but a summary must not become the only transcript, a memory must not become permission, and the browser's mounted message list must not become the export source or task ledger.

**Reuse the current architecture.** Reconcile the existing curated-context proposals and LETHE/client contracts before implementing OI-26/OI-27. Reuse OI-05/OI-08 for memory, OI-12 for context routing, G3 for run lifecycle/events and OI-13 for completion proof. OI-30 schedules ordinary authorised runs; it does not introduce a competing tool executor or delegate factory. The current fork's actual maximum working period is a reconnaissance question: discover the controlling duration, deadline and enforcement layer. A tool-round cap, token allowance, model limit and elapsed-time limit are different quantities; do not invent a numeric time limit from any of them.

**Product mode versus the delivery session.** The requested product scheduler may continue goals without another user message while its authorised runtime is active. The colleague implementing it must not claim that this creates background execution in the present chat host. Define and test behaviour for UI navigation/closure, application exit, sleep, restart and disconnected providers. Hiding or switching the conversation must not cancel an authorised goal. Do not claim computation continues while the machine is asleep or the execution service is stopped. A newly started application must restore accurate state; continuation follows its recorded restart/authorisation policy, never an accidental replay of uncertain side effects.

**Cross-cutting scope and privacy.** Automatic capture and retrieval operate within the configured, consented colleague/project scopes. Do not turn “past chats” into access to every colleague, every account or every private source. Keep source speaker, session, message/event identifiers, revision and correction links. Deleted, withdrawn, excluded or permission-revoked content must not reappear through stale summaries or indexes contrary to the applicable retention policy. Preserve the complete saved transcript subject to that policy; never delete history just to pass a performance test. A full conversation export is a deliberate local user action, not cloud disclosure or blanket archive export.

**No summary-only execution state.** Preserve the current prompt, exact live constraints, active goal/step and unresolved tool-call/result relationships. Use relevant summaries for orientation and exact source excerpts for precision-sensitive code, commands, identifiers, negations and acceptance conditions. Retrieved prose and tool outputs are data, not fresh instructions that override current authority. Test correction loss, contradictory memories, poisoned instructions in an old chat and missing context; do not claim that summarisation prevents hallucination or guarantees relevance.

**Separate measurements.** Profile archive/query cost, retrieval and summary cost, model input/prefill, streaming updates and UI rendering independently. Reduced model-input size does not prove the viewer is responsive; smooth scrolling does not prove the model received enough context. Predeclare representative conversation/goal workloads, supported environments, limits, repetitions, quality/performance thresholds and evidence before qualification. Count work spent on retrieval/summarisation/checkpointing and user replies as well as goal execution. Do not measure only the improved segment of a slower complete path.

**Existing acceptance is not lost.** Keep prior proofs for their original candidate and scope. Where these additions change shared source, configuration or fixtures, mark affected evidence stale and re-run it rather than resetting all progress or retaining an invalid PASS. OI-23–OI-25 now accept the expanded scope; an earlier milestone is not the completed R3 programme.

**Begin now with arrival and OI-01, or with the verified next obligation if this programme already has durable state.**

---

## R1 evidence snapshot — historical, recheck before implementation

The following observations were recorded when Eko wrote R1 earlier on 20 September 2026. They are retained as discovery pointers, not current qualification, installed-feature status or permission. R2 authoring checks are recorded separately at the end.

- All four named wake files were read in full. The packet calls itself an authored draft, not installed, independently judged or runtime-tested. The recurring template contains four unbound values: working name, founding LINEAGE reference, LETHE scope and map key. These observations do not establish whether someone has since onboarded a colleague elsewhere. [S1–S4]
- The live TECHNE router is `00_START_HERE.md`. It still requires the mandatory arrival set, including KANON and Constitution in full, then task-specific routes. Its receipt explicitly does not itself enforce compliance. Do not introduce a second router called `00_on_startup.md`. [S5]
- Ollama was clean on `main`, HEAD `953de98d408a9697fa20a48c23973b9dc8eee921`, matching the locally recorded `origin/main`. No network fetch was performed. Selected source shows `buildChatRequest` prepending a supplied `RunOptions.SystemPrompt` and retaining model options and tools. This is a source-level integration seam, not proof that the proposed editor or persistent wake works. [S6–S7]
- TECHNE's separate `Tools/ollama` README describes a single-prompt review wrapper, not a tool-enabled colleague. Its historical test claims were not rerun. The Ollama repository also contains existing curated-context design documents; the inspected document is explicitly a design draft. Reconcile those documents before adding overlapping memory machinery. [S8–S9]

No Ollama build, model request, LETHE operation, benchmark, onboarding or runtime test was executed while preparing R1. TECHNE was already dirty; its working-tree bytes must not be represented as a committed snapshot. This R2 update also executes no Ollama feature, onboarding or qualification run.

## 1. Work list — retained IDs, corrected execution gates

The checkboxes are the assignment inventory, not evidence of current implementation. Initialise or reconcile them from the actual durable ledger; never reset verified progress on resumption. Closing an implementation item requires the declared proof and review. Role labels are responsibilities for you to fulfil or delegate through available authorised tools, not colleagues already dispatched. Coupled groups in §0.6 close together; intermediate work does not earn premature completion.

### Phase A — Reconcile the existing system and specify the changes

- [ ] **OI-01 — Trace the current harness and installed application.**
  **Owner:** reconnaissance colleague. **Depends on:** none.
  Locate terminal instruction construction, app settings and persistence, session creation, compaction, MCP/tool dispatch, conversation storage and the conversation viewer. Compare source, built binaries and the application actually launched. Inspect the existing curated-context proposals and relevant Codex integration/delegation surfaces. Capture baseline build/test/Git evidence using discovered project commands.
  **Complete when:** an evidence map separates verified existing behaviour, defects, absent work and unresolved questions, with exact production callers and consumers. Filenames or a clean checkout alone cannot close this item.

- [ ] **OI-02 — Produce and independently judge the integration design.**
  **Owner:** architect plus independent judge. **Depends on:** OI-01.
  Define ownership, lifecycle, message contracts, persistence, permissions, configuration precedence, migration and failure behaviour for the workstreams below. Reconcile the wake packet and existing curated-context design without silently adopting every older proposal. Decide whether the Codex-related work needs a pinned delegate adapter or a different bounded integration; do not assume a wholesale engine transplant.
  **Complete when:** the programme architecture, requirements, interface ownership and dependency plan are established and reviewed. Each build group must additionally have its own complete source-bound design, earned effective `[IMPLEMENTATION-READY]` and judged bounded packs before its runtime work begins. Prepare upcoming groups progressively; do not stop at designing them. This assignment itself is not the implementation-readiness stamp.

### Phase B — Make one persistent colleague work honestly

- [ ] **OI-03 — Add or complete the terminal-instructions editor.**
  **Owner:** settings/UI implementer. **Depends on:** the relevant OI-02 design/pack gates. **Coupled delivery:** G1 with OI-04; its settings contract is an intermediate dependency, not early completion.
  Provide the discussed editable instructions surface with saved configuration, scope clearly displayed, validation, effective-prompt preview and a reversible reset/disable operation. Specify when edits take effect; do not silently change an in-flight session. Reject an unresolved identity binding rather than storing it as an active profile.
  **Complete when:** edit → save → application restart → newly opened conversation demonstrably uses the saved value, while disabling the feature restores its specified baseline.

- [ ] **OI-04 — Wire instructions into the actual request path.**
  **Owner:** harness implementer. **Depends on:** the relevant OI-02 gates and the established OI-03 settings contract, not OI-03 delivery closure. **Coupled delivery:** G1 with OI-03; close both after their shared request-path proof.
  Trace and extend the existing `SystemPrompt` path rather than introduce a parallel prompt assembly. Preserve model-template requirements, tool schemas and protocol messages. Define ordering among host guidance, selected colleague binding, repo context and current task. Measure the full arrival/context budget for each supported configuration; report an overflow instead of silently trimming required material.
  **Complete when:** request-boundary capture proves the exact effective carrier reaches the selected model through the real terminal entry point, including after restoration/compaction where applicable. An editor screenshot is insufficient.

- [ ] **OI-05 — Verify LETHE access through the harness.**
  **Owner:** memory integration colleague. **Depends on:** OI-02; compatible tool execution established by OI-01.
  Discover the real MCP schemas and existing consented scopes. Prove write, lookup, exact-reference/raw retrieval and chunk completeness using harmless test records. Implement only missing integration. Do not assume `connected:psyche` is authorised, switch to AGORA on failure, or invent a new memory schema.
  **Complete when:** the ordinary tool path writes a test record and retrieves its raw content correctly; inaccessible scope, missing chunks and service failure yield explicit incomplete recovery rather than invented success. [S2–S3]

- [ ] **OI-06 — Run the one-time founding wake.**
  **Owner:** the new colleague; the delivery owner coordinates through the authorised onboarding path and Ash retains reserved decisions. **Depends on:** completed G1 and OI-05. Reuse an already established matching identity only after verifying its binding.
  Use `01_FIRST_WAKE.txt` and `03_FOUNDING_RESUME.md`. Complete the routed arrival, skim accessible estate colleague names/rationales/contribution summaries with coverage recorded, check the candidate name and aliases, and register a unique non-human/non-brand working name through current LINEAGE conventions. Separate proposed remit, inherited lessons and actual achievements. Recheck collisions immediately before registration; repeated onboarding must resume safely rather than duplicate identity.
  **Complete when:** real founding/session references and an attributed charter exist, the colleague has chosen its own name, and its actual contribution record starts at onboarding. Do not rename Eko or an unrelated colleague. [S2, S4]

- [ ] **OI-07 — Bind and install the recurring wake message.**
  **Owner:** onboarding colleague produces the binding; authorised harness/operator path installs it. **Depends on:** OI-06 and the relevant design gates. **Coupled delivery:** G2 with OI-08. Binding preparation and safe test installation can proceed before recurring recovery is complete; neither item closes until the shared fresh-conversation proof passes.
  Establish the three layers: permanent carrier, exact-key LETHE map and raw anchor records. Fill all four placeholders in `02_RECURRING_WAKE_TEMPLATE.txt` with returned, verified values. Keep the generic template separate from the bound message; reject unresolved values and identity mismatches. The first-wake prompt must not become the every-session prompt.
  **Complete when:** a fresh conversation selects the established binding, reaches the exact map/anchors and retains the working name with correct new-session attribution. The founding prompt does not authorise the colleague to install its own message. [S1–S3]

- [ ] **OI-08 — Implement recurring recovery and earned history.**
  **Owner:** harness/memory colleague. **Depends on:** OI-06, G1, OI-05 and the established OI-07 binding contract, not OI-07 delivery closure. **Coupled delivery:** G2 with OI-07; build recovery against the bound contract and close both together.
  Recover identity, applicable live-thread revisions, corrections and task-relevant anchors before substantive work on a new conversation or detected compaction. New conversations have their own session attribution; in-place compaction keeps the same session. Preserve evidence-backed contributions, reflections and correction links at meaningful boundaries. Keep parallel live threads separate. Old tasks and permissions are context, not automatic authority.
  **Complete when:** recovery, correction uptake and history write/readback work through ordinary use, including a model change. An unavailable record causes an honest bounded failure, not a new identity or fabricated memory. [S3]

- [ ] **OI-09 — Pass the continuity falsification suite.**
  **Owner:** independent verifier. **Depends on:** completed G2.
  Test a genuinely cold conversation, process/application restart, compaction, missing memory, wrong identity, superseded record, interrupted onboarding and concurrent sessions. Store a discriminating harmless anchor only in LETHE; exclude it from the prompt, chat history and continuation summary. Require unprompted retrieval and correct functional use after a bland greeting.
  **Complete when:** real cold-start and restart trials pass and severing the memory path makes the intended memory-dependent check fail. Same-session readback, familiar phrasing and model self-report are not continuity proof. [S1, acceptance checks]

### Phase C — Supply architectural context and enforce integration proof

- [ ] **OI-10 — Specify the repo-overview and estate-map format.**
  **Owner:** architecture colleague. **Depends on:** OI-02.
  Keep a short canonical overview in each repo root and an estate relationship/index map in the parent workspace. Separate approved intended behaviour, actual runtime journeys and verified present state. Include boundaries, owners, producers, consumers, defaults, persistence and evidence links. The overview points to owning blueprints; it does not acquire authority to invent runtime semantics.
  **Complete when:** a reader can distinguish intended, verified, broken and unknown connections without mistaking a future-state picture for existing code.

- [ ] **OI-11 — Populate a pilot overview, then the repo rollout.**
  **Owner:** repo architects/reviewers. **Depends on:** OI-10 and access to each repo's approved intentions/live source.
  Pilot on Ollama so this integration work has its own system map. Inventory the estate repos to receive overviews and process them in a controlled order, with one owner per overview and cross-repo boundary checks. Reuse an adequate existing overview. Do not require every estate overview to finish before the Ollama pilot can proceed.
  **Complete when:** Ollama's overview is source-grounded and reviewed; the full item closes only when every repo in the recorded rollout inventory has a reviewed overview or an explicitly agreed exclusion. Missing evidence remains visible.

- [ ] **OI-12 — Route context at wake, task start and resumption.**
  **Owner:** TECHNE/harness integration colleague. **Depends on:** completed G2 and the reviewed OI-11 Ollama pilot.
  Connect `00_START_HERE.md` to the relevant overview and task-specific source paths. Refresh the working context when repo, task, specification or configuration changes and after compaction. Future child packets receive their own scope and affected integration chain, not the parent's entire history. Preserve the router's mandatory arrival obligations unless a separate method change is approved.
  **Complete when:** captured context and task records show the correct current repo and upstream/downstream surfaces; stale or missing mandatory context blocks the affected action. A recorded file hash is not proof of comprehension. [S5]

- [ ] **OI-13 — Make completion depend on integration evidence.**
  **Owner:** proof/enforcement implementer plus independent reviewer. **Depends on:** OI-12 and the approved enforcement/security design.
  Bind the harness completion workflow to the existing TECHNE requirement/evidence and routing records. Require real entry point → delivered data → state/effect → downstream consumer → observable result. Include executed clean–broken–clean controls, required regressions, configuration identities, logs and exit codes. Preserve useful unit tests; strengthen the affected feature's production-path proof rather than rewrite every legacy test.
  **Complete when:** a wired-but-inert change, missing consumer, stale receipt or fabricated PASS cannot satisfy the completion gate. Required warnings/errors block the claimed clean result; do not hide warnings, skips or baseline failures. [S5; S10]

### Phase D — Give the active colleague supervised children

- [ ] **OI-14 — Build the parent-owned delegation interface.**
  **Owner:** orchestration implementer. **Depends on:** the relevant OI-02 gates, OI-09 and OI-13. **Coupled delivery:** G3 with OI-15 and OI-16; no early completion before authorised execution and supervision are proven together.
  Add or complete child start, status, result, message, cancellation and closure operations through the chosen native/adapter boundary. Bind each child to the actual active parent session and a bounded task packet containing repo, ownership, acceptance conditions and limits. Keep identity, session, task and model attribution distinct.
  **Complete when:** the active colleague launches a real child, receives its real artefacts/evidence and incorporates the result without a human copying text between conversations. An existing text-only review wrapper must not be relabelled as an agent. [S8]

- [ ] **OI-15 — Enforce tool inheritance and no grandchildren.**
  **Owner:** runtime-boundary/security implementer. **Depends on:** the judged G3 contracts. Implement/prove the applicable authorisation boundary before enabling writable child execution. **Coupled delivery:** G3 with OI-14 and OI-16; final proof uses their real dispatch path.
  Give children the parent's applicable tools, subject to current task scope and approval controls, but deny child-creation authority at execution dispatch as well as schema exposure. Prevent role spoofing and unauthorised peer-session access. Inspect shell, MCP and external delegate routes for bypasses: unrestricted execution cannot be declared constrained merely because one tool is hidden.
  **Complete when:** real child tool use succeeds where authorised, while attempted grandchild creation and authority escalation are rejected by enforced boundaries. Resolve any broad-shell conflict explicitly rather than silently narrowing the promised tool set.

- [ ] **OI-16 — Add durable supervision and steering.**
  **Owner:** orchestration/session implementer. **Depends on:** the judged G3 contracts and ready intermediate dispatch/authorisation surfaces, not OI-14/OI-15 delivery closure. **Coupled delivery:** G3 with OI-14 and OI-15.
  Persist parent/child ownership, lifecycle and ordered conversation/tool events. Allow the parent to observe progress, send corrections, pause/resume where supported, and cancel. Distinguish queued steering from delivered/acknowledged steering; define the safe point where it affects the next action. Bound concurrency, retries, runtime, context and resource consumption.
  **Complete when:** a parent correction changes the child's subsequent action, cancellation prevents further authorised work, and restart/reconnect recovers accurate state without duplicate dispatch or invented progress. In-flight side effects need explicit outcomes, not assumed rollback.

- [ ] **OI-17 — Coordinate file ownership and result integration.**
  **Owner:** manager/runtime colleague. **Depends on:** completed G3.
  Apply existing TECHNE branch/worktree and custody rules. Allocate non-overlapping writable scopes or explicitly serialise shared files. Keep child output, review and integration separate; protect other colleagues' changes and make ownership survive handoff. Do not automatically merge unverified results or delete active worktrees.
  **Complete when:** a conflict test prevents silent overwrite, a reviewed child result can be integrated through the authorised path, and completed temporary work is closed out without recreating the abandoned-branch problem.

- [ ] **OI-18 — Show nested child conversations.**
  **Owner:** conversation-viewer implementer. **Depends on:** completed G3 and its persisted conversation/event contract.
  Display children beneath their actual parent, with visible sender, task, model, status, messages, relevant tool activity and parent steering. Drive the display from persisted events rather than generated summaries. Disable human message input by default. Define any explicit human intervention/handover mode before adding it; ensure the manager sees that intervention and do not silently address the wrong session.
  **Complete when:** the viewer shows real parent–child exchanges, its read-only default prevents accidental sends, and reopening reconstructs the same conversations. Respect private data and provider limits; do not invent hidden reasoning.

### Phase E — Measure capabilities, then route tasks

- [ ] **OI-19 — Build an exact-configuration model catalogue.**
  **Owner:** model integration colleague. **Depends on:** the relevant OI-02 gates and completed G1.
  Discover available local and permitted cloud models. Record reported model/revision or digest, provider/destination, quantisation where applicable, actual context configuration, tools/structured-output support and resource observations. Probe required capabilities through the harness; advertised metadata alone is not a capability result. Keep the colleague's identity independent of the selected model.
  **Complete when:** the catalogue distinguishes advertised, measured, unsupported and unknown properties, and changed configurations invalidate affected profiles. Cloud revision limits remain explicit rather than being described as reproducibly pinned weights.

- [ ] **OI-20 — Design the task-capability benchmark.**
  **Owner:** evaluator plus independent reviewer. **Depends on:** OI-19.
  Measure the abilities needed here: basic coding, Swift/Python work, repo navigation, cross-file integration, reasoning, testing, tool/MCP use, long-context instruction retention, correction uptake and failure honesty. Use versioned representative tasks, held-out variants, objective tests where possible, repeated trials and independently reviewed rubrics. Include tasks that require declining or escalating when evidence is insufficient.
  **Complete when:** scoring, minimum evidence, uncertainty and eligibility thresholds are fixed before qualification runs. Report measured task capability of the model-plus-harness configuration, not universal intelligence, an IQ score or résumé-based competence.

- [ ] **OI-21 — Run qualification and retain attributable results.**
  **Owner:** evaluation runner/verifier. **Depends on:** OI-20 and the actual completed G3 child-role configuration for child qualification.
  Execute in disposable workspaces with the actual wake/context/tool configuration. Record all attempts, success/failure causes, latency including loading where relevant, token use, resource use, observed cost and raw evidence. Separate infrastructure failure from task failure. Calibrate the evaluator with known-good and deliberately broken controls; keep qualification answers inaccessible to evaluated colleagues.
  **Complete when:** results can be independently checked, sample counts and uncertainty are visible, and no model is qualified using hidden retries, self-grading or an untested configuration. Cloud runs require current disclosure/spending authority.

- [ ] **OI-22 — Connect measured profiles to parent model selection.**
  **Owner:** routing implementer. **Depends on:** completed G3, OI-19 and OI-21.
  Let the parent express task requirements; apply hard privacy, tool, context, resource, cost and reviewer-independence constraints before ranking eligible measured candidates. Prefer an adequate local option where justified, not a local model regardless of suitability. Explain the selection, support an explicit override within permissions, and escalate or report no eligible model when evidence is insufficient. Requalify after material changes and track production outcomes separately from benchmark scores.
  **Complete when:** controlled task differences produce the expected evidence-backed selections, and an attractive but ineligible model is never chosen. Begin with advisory/shadow choices; automatic routing follows demonstrated agreement with the approved selection policy.

### Phase E2 — Automatic memory, working context and continuous agent mode

These are six additional required items, not replacements for OI-01–OI-25. Their numbers preserve the old identifiers; execute by the dependency graph, not by numerical order. All six receive the same design, pack, independent-review and production-path proof requirements as the original work.

- [ ] **OI-26 — Automatically capture colleague memories through LETHE.**
  **Owner:** memory/harness implementer. **Depends on:** OI-05, G2 and the judged automatic-capture contract. **Extends:** OI-05/OI-08; one memory integration, not a duplicate engine.
  **Required outcome:** the harness automatically preserves useful, attributable memories during ordinary colleague/agent use, without needing the model or Ash to remember to call a store tool. Attach capture to committed turns and meaningful decisions, corrections, verified results and task checkpoints. Keep exact conversation persistence separate from curated memories. Define selection/extraction, source references, eligibility and disabled/excluded-content rules in the design; label model-derived interpretations and self-reports rather than promoting them to verified facts. Provide visible memory status and review/correction controls through the existing suitable surface.
  **Implementation obligations:** reuse real LETHE storage/retrieval and consent; choose a service adapter rather than copying the engine unless a reviewed, justified integration requires it. Use a durable bounded capture queue/outbox with source-event identity and deduplication so retry/restart does not silently lose or multiply a memory. Distinguish captured/pending/stored/failed states from verified raw readback. Do not claim a queued write is persisted. Recover interrupted capture, expose errors and prevent an outage from freezing unrelated chat; memory-dependent work must still honour its readiness rules. Store referenced technical evidence and subsequent corrections without destroying authorship. Never generate a new automatic memory event solely from storing that same memory, creating a feedback loop. Automatic operation is enabled in the configured, authorised colleague profile after qualification, not merely shipped as an unused API.
  **Complete when:** ordinary messages/decisions cause retained memory without any model-issued store call; a fresh process/conversation retrieves and correctly uses it; corrections supersede the relevant prior claim; duplicate events are deduplicated; denied scope, disabled capture and service outage behave as specified. Execute a control disconnecting the capture hook, then restore it: the later memory-dependent assertion must fail and then pass. A stored transcript or a successful MCP probe alone does not close this item.

- [ ] **OI-27 — Build relevant working context instead of whole-history replay.**
  **Owner:** context/memory implementer. **Depends on:** OI-26, OI-12 and the judged context-assembly contract. **Extends:** the existing request assembly and curated-context proposals.
  **Required outcome:** each new user prompt and autonomous continuation receives enough current context to make sense, plus relevant information from the current and authorised past chats, principally as bounded source-linked summaries rather than the complete conversation every time. Retain the current prompt, standing instructions, active goal/step, applicable exact constraints, necessary recent exchanges and outstanding tool pairs. Clarify references such as “that change” using current thread/goal context before retrieval; latest-prompt similarity alone is not the relevance policy.
  **Implementation obligations:** filter by permission/scope before ranking; retrieve, deduplicate and rank bounded candidates using the actual supported interfaces; preserve corrections and evidence provenance. Summary entries point to exact recoverable source messages. Do not replace raw authoritative facts with an unsupported synthesis or regenerate an ever-growing summary of summaries. Permit exact source expansion when needed and declare insufficient context instead of guessing. Cache/index incrementally with invalidation for edits, corrections and access changes; do not scan or resummarise the entire archive for every request. Preserve the complete archive and export independently. Bound every stage and reserve room for output/tools; when mandatory content cannot fit, expose the precise capacity issue. Integrate startup recovery and compaction coherently, keep a tested rollback/compatibility path, and make selective context the qualified normal path for the new colleague/agent mode—not an unused toggle. A failure must not silently send an unbounded or unauthorised archive as fallback.
  **Complete when:** captured actual requests omit unrelated/old bulk history, remain within the declared budget as the archive grows, and answer controlled context-dependent tasks correctly using current and past-chat evidence. Include exact-code/constraint, pronoun-reference, correction, multi-turn goal, no-hit, contradictory-memory, unavailable-retrieval and denied-scope cases. Use held-out questions, all attempts and predeclared quality/latency thresholds, with full-history reference where it fits. Removing retrieval or corrupting a relevant summary must fail its specific behavioural check. Smaller prompts alone do not establish quality or UI improvement.

- [ ] **OI-28 — Make long conversations responsive to load, read and navigate.**
  **Owner:** UI/storage performance implementer. **Depends on:** OI-01 and its judged performance/archive contract. **Independent of:** OI-27 delivery; diagnose and fix the viewer separately.
  **Required outcome:** growing conversation history does not force all messages and expensive content into the UI at once or make normal navigation unresponsive within the tested operating envelope. Begin with profiling, then use appropriate paged/cursor archive loading, virtualised/windowed rendering, incremental updates, memoisation and lazy Markdown/code/attachment rendering where the measurements justify them. Do not assume a particular library is installed or prescribe a rewrite merely to obtain these mechanisms.
  **Implementation obligations:** bound initial query/transfer size, loaded message pages, mounted elements and cached rendered content; rendering only nearby messages is insufficient when the frontend still loads/parses the entire transcript. Use stable message identifiers and cursor/order semantics, not array-index identity across paging. Handle variable heights and streaming without repeatedly reprocessing every earlier message. Preserve scroll anchors when older pages arrive, navigation/search to older messages, keyboard/accessibility behaviour, edits, attachments and the user's chosen reading position. New messages must not drag a user away from older text; provide a deliberate return-to-latest path. Account for nested child conversations and large individual messages, not just many short messages. Profile before and after using the same representative fixtures, hardware/build and cache conditions; predeclare load/interaction latency, memory, frame/long-task and mounted-count bounds.
  **Complete when:** real UI tests and measurements meet the predeclared bounds at increasing history sizes while all saved content remains reachable. Include mixed text, long code blocks, tool activity, attachments, nested children and live streaming; distinguish cold/open, warm/reopen and older-page navigation. Run with model generation idle as well as active to isolate rendering cost. A deliberately non-windowed/unbounded-loading control must violate the relevant count or performance assertion for the intended reason. Do not claim unlimited-length performance or pass by truncating/deleting the archive.

- [ ] **OI-29 — Add a top-left Download conversation control for complete Markdown export.**
  **Owner:** conversation UI/export implementer. **Depends on:** OI-01, the verified archive contract and its judged export design. **Independent of:** OI-27/OI-28 closure; coordinate shared UI/file ownership.
  **Required outcome:** a labelled, accessible button at the **top left of the conversation window** lets the user download/save a UTF-8 `.md` version of the entire selected conversation. Read the canonical saved archive, never only mounted/loaded messages, selected model context or a generated summary. Preserve actual message text, speaker attribution, order, code fences, available timestamps, relevant visible tool/result events and attachment references. Do not fabricate unavailable content, hidden reasoning, attachment bytes or missing history.
  **Implementation obligations:** use a stable export snapshot boundary while a conversation is streaming; state whether an in-progress message is included and label any partial content. Use stable IDs to prevent duplicates/omissions across pages. Preserve content containing backticks, Unicode, long code, empty messages and edits according to the existing conversation-branch/version policy. The selected conversation's full saved transcript is always included; child conversations are a separately labelled opt-in inclusion/export scope so the button does not silently export other conversations. That export choice does not modify the viewer's read-only child-input default. Produce a safe default filename, proper Markdown/media type, save/cancel/failure behaviour and a clear error rather than a silently truncated success. Stream/chunk from storage with bounded memory where supported. Export only the requested user-visible authorised content, not unrelated credentials, internal bindings or other users' private stores. Never fetch remote attachments merely to complete a local transcript export.
  **Complete when:** the real top-left button saves a readable `.md` whose ordered message IDs/count and content match an independently enumerated archive snapshot, including messages never loaded in the UI and messages outside the model context. Test pagination boundaries, large histories, streaming, code/Unicode, unavailable attachments, cancellation and write failure. A visible-page-only exporter and an exporter with a missing/misordered message must fail the completeness checks. UI visibility or a non-empty file alone is not export proof.

- [ ] **OI-30 — Add durable, time-bounded autonomous goal execution.**
  **Owner:** goal scheduler/runtime implementer. **Depends on:** OI-13, OI-26, OI-27, completed G3 and judged scheduler/interaction contracts. **Coupled delivery:** G4 with OI-31; this is scheduling existing authorised runs, not another tool executor.
  **Required outcome:** after the user enables agent mode and supplies a goal, the model makes a durable plan of steps, performs dependency-ready work, checks results and advances towards the goal without a new “continue” prompt each period. A work period is bounded by the actual current maximum working time and other effective limits. Before that deadline, yield safely, record completed and unfinished work, then review the next eligible goal and its existing progress and start the next required step automatically. Expired time never counts as completed work. Preserve multiple queued goals and partial steps; a long unfinished step must be resumed rather than silently skipped or marked done.
  **Implementation obligations:** discover and record the exact current time/deadline controls, their source/configuration and host/provider meaning; do not substitute the known tool-round cap for elapsed time or invent a number. Define a checkpoint reserve and safe-boundary/deadline policy in the design. Do not start an operation that cannot safely finish/checkpoint/cancel within its remaining authorised budget. At forced interruption persist pending/uncertain side effects for reconciliation rather than blindly reissue them. Persist goal/step IDs, dependencies, priority/order, owner, plan revision, outcomes/evidence, checkpoint, retries, scheduling eligibility, paused/stopped/completed/blocked state and current authority. Use one logical scheduler owner with durable dispatch deduplication; recover without duplicate workers. Define a deterministic queue/fairness policy and revisit unfinished goals without starvation. Reuse G3 events/cancellation and the ordinary execution path; scheduling the same child again is not permission for a child to spawn descendants.
  **Continuation boundary:** renew only a per-work-period budget where the actual host contract permits another run. Keep cumulative time, spend, retry and no-progress limits across periods. Never rotate sessions, accounts or credentials to evade usage, authorisation or safety limits. Stop or block on genuine completion, user stop, denied/revoked permission, exhausted overall budget or irreconcilable side effects. Prove step/goal completion with the relevant acceptance criteria, not model narration or an empty queue. Use an explicitly selected permitted model during development; evaluate the final scheduler/context configuration before automatic routing claims. Expose goal and checkpoint status in the product and define app-exit/sleep/restart behaviour under §0.12.
  **Complete when:** a real goal spans multiple bounded work periods and resumes from persisted progress without repeated human input; queued goals are selected as specified and unfinished steps retain their identities. Verify durable recovery after process interruption, deduplicated dispatch and uncertain-side-effect reconciliation within the declared boundary, failed-checkpoint handling, no-progress/budget stops and no skipped/duplicated side effects. A clock/deadline test uses declared smaller test limits plus a production-configuration smoke. A control removing checkpoint restore or next-period scheduling must fail actual progress assertions. Close G4 only after OI-31's conversation/stop proofs also pass.

- [ ] **OI-31 — Let the user converse during agent mode and resume work automatically.**
  **Owner:** interaction/runtime implementer. **Depends on:** the judged G4 contracts and ready OI-30 intermediate scheduler/checkpoint surfaces, not OI-30 delivery closure. **Coupled delivery:** G4 with OI-30.
  **Required outcome:** while the agent is pursuing its goals, the user can send a normal prompt, receive a substantive response and have the agent return automatically to its tasks, unless the user asks it to stop or changes the task. This must work during a long work period, not only after the goal finishes. Continuing a task means preserving its durable state; simultaneous model inference is not required.
  **Implementation obligations:** accept and persist user messages through an interactive priority path. At a safe boundary yield foreground inference/task work as needed, handle the question or steering with relevant context, then reconcile the current goal/plan and resume the next valid step. Define and measure acknowledgement/response latency under load; a spinner, ignored message or answer deferred to the end of the goal is insufficient. Support ordered multiple messages without duplicate replies or lost checkpoints. Distinguish ordinary conversation, goal-changing steering, pause and stop; an unrelated question must not silently cancel or rewrite the goal, while genuine changed instructions must invalidate stale queued work. Preserve current ownership and serialise conflicting mutations; a conversation response cannot race another model into writing the same state. Surface waiting for an uninterruptible operation or approval honestly and provide the specified stop path.
  **Stop is durable:** explicit stop must cancel pending automatic continuation for the targeted goal or whole agent session, revoke further dispatch as appropriate and handle owned in-flight tools/children under the existing cancellation contract. An ambiguous target may pause affected dispatch pending clarification, not keep working blindly. Distinguish “Stop response” from “Stop agent/goals” in the UI. Late model/tool results, scheduler timers, queued events, ordinary replies and application restarts must not resurrect stopped work. Resumption after a stop requires a fresh explicit user action. Do not infer approval for previously denied tools from continuing the conversation.
  **Complete when:** inject an ordinary question while a real multi-step goal runs; verify a substantive answer arrives within the declared bound, task progress is preserved and subsequent goal work occurs without another user prompt. Test steering that changes the next step, repeated messages, active tools/children, response errors, app reconnect and explicit stop. A control that cancels the goal on every user message must fail continuation; a control that resumes after stop must fail the cancellation assertion. OI-30/OI-31 close only when these interaction and scheduling outcomes pass together.

### Phase F — Prove the combined system and roll it out

- [ ] **OI-23 — Run the combined end-to-end reality gate.**
  **Owner:** independent, appropriately separate judge. **Depends on:** OI-09, OI-13, OI-17, OI-18, OI-22, OI-26–OI-29 and completed G4 (OI-30/OI-31), with affected model profiles requalified for the actual final configuration.
  From the intended app/terminal entry point, wake the recurring colleague, recover a LETHE-only anchor, load current repo context, select a qualified child, complete a real bounded code change, steer the child, verify downstream behaviour, record evidence/history and restart. Confirm the returning colleague correctly uses that record without inheriting old permissions. Separately exercise any selected Codex adapter end to end. Add the full R3 workflow in §0.11: automatic memory, context-dependent recall without whole-history replay, large-history navigation and full export, multiple checkpointed work periods, user conversation with automatic task return, and durable explicit stop.
  **Complete when:** severing each important link fails its corresponding behavioural check; restoration passes. All applicable regression, safety, migration and clean-build obligations pass. Self-authored success narration cannot stand in for independent verification.

- [ ] **OI-24 — Qualify recovery, load and upgrade behaviour.**
  **Owner:** reliability colleague. **Depends on:** Phase D, Phase E and Phase E2 implementations, including complete G4. Earlier isolated reliability tests remain useful but cannot close the expanded item.
  Test concurrent sessions, bounded parallel children, long conversations, context pressure, process interruption, LETHE outage, cloud outage and unavailable/changed models. Verify ownership recovery, useful error messages, bounded resources and explicit treatment of uncertain side effects. Exercise migration of old settings/conversations and compatible restoration of the previous release/configuration. Include interrupted capture/indexing, stale summaries, paged-history/large-export load, scheduler fairness, checkpoint failures, multiple work periods, user-response latency under load and stopped-goal resurrection after restart. Profile UI and model-context costs separately; final evidence uses their combined deployed configuration.
  **Complete when:** supported operating limits are measured, failures remain visible and recoverable, and an upgrade or rollback does not silently lose identity, records, task ownership or evidence.

- [ ] **OI-25 — Pilot, document and release the qualified scope.**
  **Owner:** release colleague plus Ash for operator acceptance. **Depends on:** OI-23 and OI-24; OI-11's Ollama pilot, not completion of every other repo overview.
  Pilot one colleague, one repository and one child before increasing concurrency. Update the user guide, system map, support/configuration matrix, evidence links and remaining-work ledger. Reconcile any changed canonical TECHNE guidance and its adoption surfaces separately and coherently. Enable promised features in the intended default path only after qualification; keep cloud use and consequential actions subject to policy.
  **Complete when:** the installed build is the verified candidate, Ash's actual workflow passes, restart/uninstall/rollback instructions are checked, and no unverified feature or unrun test is presented as complete. Close Git work under current scoped authority.

## 2. Delivery checkpoints

**Checkpoint 1 — One colleague with real continuity:** OI-01–OI-09. Do not wait for the benchmark system to prove this.

**Checkpoint 2 — Architecture-aware work:** OI-10, the Ollama portion of OI-11, OI-12 and OI-13. Continue the estate-wide overview rollout independently with explicit coverage.

**Checkpoint 3 — One supervised child:** OI-14–OI-18, initially with explicit model choice. Parallelism comes after control and recovery work.

**Checkpoint 4 — Evidence-backed model choice:** OI-19–OI-22. Catalogue and benchmark design can proceed alongside delegation, but child qualification must measure the configuration actually used.

**Checkpoint 5 — Memory and conversation upgrades:** OI-26–OI-29. Viewer and export work may proceed earlier against the verified archive contract; selective context is not a prerequisite to fixing UI responsiveness.

**Checkpoint 6 — Continuing, interactive agent mode:** G4 (OI-30/OI-31), followed by affected OI-21/OI-22 qualification. Prove automatic goal continuation and durable user stop together.

**Checkpoint 7 — Qualified expanded release:** OI-23–OI-25, including all six R3 additions. Each earlier feature still needs its own proof; this final gate is not permission to defer all testing.

## 3. Decisions assigned to design before dependent implementation

Resolve these under §0.2/§0.5 and the engineering directions in §0.7. They are assigned work for OI-01/OI-02 and the owning feature designs, not a questionnaire Ash must answer before you begin: exact editor/settings ownership and scope; instruction precedence; startup/compaction readiness gating; LETHE scope and identity binding; parent/child persistence and cancellation semantics; shell/MCP enforcement limits; human intervention semantics; Codex adapter scope; model qualification thresholds; approved cloud destinations/budgets; operating limits; and migration/rollback contracts. Existing approved decisions should be recovered before asking Ash to decide again.

For OI-26–OI-31 also settle: automatic capture eligibility/outbox semantics; retrieval scope and summarisation/raw-source policy; archive/cursor and UI performance bounds; exact Markdown export contents/snapshot policy; the actual current work-time limit and checkpoint reserve; goal queue/fairness and cumulative budgets; user-message priority, pause/stop targets and restart behaviour. Technical choices are assigned design work, not a mandatory questionnaire for Ash.

R3 explicitly adds selective working context, per-conversation local export and the bounded product scheduler. Full removal/replacement of the compactor still requires a judged compatibility decision; blanket unsolicited transcript export/import, model fine-tuning, AGORA substitution, estate-wide code repair and multi-machine deployment remain excluded. Bring any genuinely required new dependency back into design with its reason and boundary.

## 4. Historical R1 source ledger and evidence limits

The following are the R1 authoring record of local working-tree reads on 20 September 2026 via Eko Write Bridge, retained for provenance and discovery. They are not new R2 execution evidence. The separate read Bridge returned HTTP 404. Raw `file_sha256` values below identify complete files, even where only a selected range was read; they are not the returned excerpt `content_sha256` and not proof of execution.

**S1 — Wake packet.** `TECHNE/docs/prompts/PSYCHE_RECURRING_COLLEAGUE_WAKE_PACKET.md`, lines 1–311, complete; especially Use and Proposed acceptance checks. Raw SHA-256 `39a3af5147559ee48d3906657118657f342f914f4fed0c221f59b975e46d7a6b`; request `30a2b8e7410f48ef8d5307256b200bba`.

**S2 — First wake.** `TECHNE/docs/prompts/01_FIRST_WAKE.txt`, lines 1–86, complete. Raw SHA-256 `63f8c9c2224c9cd95c588cf9493a71ee14539b2e37e02ab8e910d65e0ae733bf`; request `e5113c4242c74150a4a50f0f702695fe`.

**S3 — Recurring template.** `TECHNE/docs/prompts/02_RECURRING_WAKE_TEMPLATE.txt`, lines 1–61, complete. Raw SHA-256 `7d2b31137f4f28d72c9c7b834de791919ec3cedf3102bc1fb51476c8e8da97e9`; request `1b6a9a3819f443f5a9a9ee03c4691194`.

**S4 — Founding résumé.** `TECHNE/docs/prompts/03_FOUNDING_RESUME.md`, lines 1–61, complete. Raw SHA-256 `9d63cbbd5a0c5892fc77249a2394a04b7a8855af65c143af88b1b6944d088946`; request `d7aea2d2168e4608b72e7af061422497`.

**S5 — Router.** `TECHNE/00_START_HERE.md`, lines 1–173, complete; especially Triggers That Cannot Be Skipped and Routing And Evidence Receipt. Raw SHA-256 `e0779d39073b56ae823b4868565c19330289188a0bec0fe15322835b49dff525`; request `7e1ae13b5aa24458b4d42e3b850c27c3`. S1–S5 were read at reported TECHNE HEAD `035365bb70c57feb900e88a029f631bfe6865d52`, with a dirty working tree. Canonical methods referenced by the router were not exhaustively audited in this planning pass.

**S6 — Ollama repository baseline.** `repo_overview(GITHUB, ollama)` request `add1dcb5f97d45c9a65aff41f6876661`: clean `main`, HEAD `953de98d408a9697fa20a48c23973b9dc8eee921`, zero ahead/behind local `origin/main`, three local branches and one worktree. This is local Git metadata, not a fetched remote check or installed-app verification.

**S7 — Request assembly.** `ollama/agent/session.go`, lines 421–484 only, including complete `buildChatRequest`. Raw SHA-256 `0a2521b18c99b043c8f91af37605a827bdc4d50b380e904013a5959d859d3cec`; request `a3b672642c1e49db94d06622b299f583`. The response is range-limited/truncated relative to the whole 1,087-line file. No whole-session audit is claimed. A bounded directory listing also confirmed agent event, registry, compactor, approval and tool source/test files; existence is not activation evidence.

**S8 — Separate review wrapper.** `TECHNE/Tools/ollama/README.md`, lines 1–400 of 403; request `eee5775b2df04aaea98a179c867ffb7c`; raw SHA-256 `bc1eae7e423b953a3ffeef92e60fbc67a0051353997681b34ff52062f8af9120`. Relevant sections: Runtime location, What it does not do and Relationship to the other TECHNE tools. Historical test/model claims were not reproduced.

**S9 — Existing context proposal.** `ollama/docs/CURATED_CONTEXT_SYSTEM.md`, lines 1–180 of 501; request `343e7a2059d34aad9433fc879ce22ce8`; raw SHA-256 `1952eb35a89b65a1bbd4cad4e8b7621ba7a86851f9c10d4c4d87caa16be8869c`. Header declares `[DESIGN-DRAFT v3]` and non-implementation class. The document was not read in full, its proposed interfaces were not adopted, and its claims about current behaviour were not verified. `CURATED_CONTEXT_BLUEPRINT.md` was located by directory listing but not read.

**S10 — Supplied engineering guidance.** Uploaded `BLUEPRINT_GUIDELINES.md`, particularly document classes, source-of-truth reconciliation and activation proof; uploaded `CODE_GENERATION_RULEBOOK.md`, particularly Activation Proof Contract, Anti-Facade Discipline and Test Discipline. These support the planning structure, not a claim that their uploaded versions supersede current TECHNE masters. Implementation must read current routed sources.

**S11 — Conversation requirements.** Ash's current request and the recovered 20 September discussions supply persistent identity, parent-owned supervision, nested conversations/read-only default, child tool inheritance without spawning, capability-based local/cloud selection, and repo-overview intent. Recovered discussions are context, not evidence of installed code. Detailed lifecycle and gate proposals in this list remain proposals until design approval.

**S12 — Historical external references, not current qualification.** R1 recorded official Ollama documentation references on 20 September 2026; they were not rechecked for R2: tool calling, `https://docs.ollama.com/capabilities/tool-calling`; context length, `https://docs.ollama.com/context-length`. The former documents the client-side tool-result loop; the latter documents configurable context and its resource implications. Neither proves this local fork's behaviour or the discussed features. No model rankings, latest-version claim or externally reported benchmark score is used for this assignment. Verify current primary interface documentation when the implementation actually relies on an external API.

## 5. Historical R2 amendment and carry-over record

This section records R1 → R2, not the current total. R3 adds six items and prospectively amends the named shared gates; see §6. The R2 source/authoring record below is retained unchanged as historical evidence.

R2 converts the planning document into a direct delivery assignment at Ash's request. It adds an explicit discover → design → judge → pack → build → prove → repair → integrate → continue procedure; authorises ordinary technical design within the task's real authority; and routes reserved actions to the operator without creating blanket permissions. It does not promote any runtime feature or blueprint.

The full original 25 item bodies remain, with the dependency/gate amendments listed below. New engineering directions are in §0.7 and must be converted to source-bound owning designs before code. No required feature is intentionally dropped. The generic professional charter and wake-message texts remain owned by their four original TECHNE files, which this update does not change.

| R1 source item | R2 destination / disposition |
|---|---|
| Goal, identity, parent ownership, inherited tools, no grandchildren, nested UI and capability selection | Kept in §0.1 and the OI items; detailed in §0.7. |
| Planning-only metadata | Replaced by direct execution-assignment metadata; low-level design and independent proof are still owed. |
| Permission exclusions | Kept in §0.2, distinguished from the new task commission for local scoped design/implementation. Existing applicable Git/review authority is not silently revoked or widened. |
| OI-01 reconnaissance | Kept; startup, evidence identity and installed-versus-built distinction expanded in §0.3–§0.4. |
| OI-02 design/judgement | Amended to programme architecture plus progressively completed per-group readiness; no blanket permission to build from this file. |
| OI-03 editor | Kept; G1 joint closure removes dependency on a not-yet-wired request path. |
| OI-04 request wiring | Kept; starts from OI-03's ready settings contract, not its delivery closure. |
| OI-05 LETHE tool path | Kept; actual scopes/raw access/failure boundaries expanded in §0.7. |
| OI-06 founding | Kept; completed G1 is prerequisite, existing matching identity must be checked before new registration. |
| OI-07 binding/installation | Kept; G2 preparation and delivery are separated. No self-install authority invented for the founding colleague. |
| OI-08 recovery/history | Kept; implementation consumes the binding contract without requiring OI-07 already to have proven this recovery. |
| OI-09 continuity proof | Kept; runs after complete G2, with uncontaminated LETHE-only content. |
| OI-10 overview format | Kept; design directions and early provisional source map added. |
| OI-11 pilot and full rollout | Kept; finite canonical-repo inventory and whole-programme closure distinction made explicit. |
| OI-12 context routing | Kept; prerequisite is complete G2 rather than premature OI-07 closure. |
| OI-13 evidence enforcement | Kept; forged/stale/missing evidence controls and independent early verification added. |
| OI-14 delegation interface | Kept; joined to G3 so message/cancellation claims do not close before their supervision/authority dependencies. |
| OI-15 child authority | Kept; enforcement precedes writable execution, final proof uses real G3 dispatch. |
| OI-16 supervision | Kept; operates from intermediate G3 contracts and closes jointly with OI-14/OI-15. |
| OI-17 custody/integration | Kept; prerequisite is complete G3. |
| OI-18 nested viewer | Kept; prerequisite is complete G3, read-only default and actual event provenance preserved. |
| OI-19 model catalogue | Kept; prerequisite is complete G1, exact configuration evidence expanded. |
| OI-20 benchmark design | Kept; predeclared metrics, sample counts, variability and evaluator controls expanded. |
| OI-21 qualification | Kept; depends on the actual completed G3 configuration rather than an assumed future child role. |
| OI-22 selection | Kept; requires complete G3, actual measured profiles and hard-constraint checks. |
| OI-23 combined gate | Kept; detailed real workflow and per-link falsification expanded in §0.11. |
| OI-24 reliability | Kept; in-flight uncertainty, resource limits and migration/rollback semantics expanded. |
| OI-25 release | Kept; installed candidate, operator acceptance and remaining estate rollout are distinct. |
| R1 checkpoints and unresolved decisions | Kept in §§2–3, operationalised by §§0.5–0.11. |
| R1 source ledger S1–S11 | Kept as historical provenance, not upgraded to new execution evidence. |
| R1 S12 external-reference claim | Re-labelled historical/not rechecked; implementation must verify an external API when actually relying on it. |
| R1 exclusions | Kept in §0.2 and §3. No new inference engine, blanket memory export, training programme or estate-wide repair implied. |

### R2 authoring evidence, scoped honestly

The supplied R1 attachment and the live Mac work-list file matched raw SHA-256 `f3b5266d253555c972bd9408656c2e30e25acd21bfe83fa1c8c7ad549bc3e949`. The live work list was read in full, lines 1–212, request `d8df453c9e3041db84b59d5282e6cc80`, at reported Ollama HEAD `953de98d408a9697fa20a48c23973b9dc8eee921` with a dirty working tree. These are the input bytes to this revision, not a claim that the repository is runtime-qualified.

Additional fresh R2 reads from the dirty TECHNE working tree at reported HEAD `035365bb70c57feb900e88a029f631bfe6865d52`:

- `TECHNE_AGENT_START.md`, complete lines 1–116; raw SHA-256 `d2cb4701e92a68844d15b966d39c50fa4edee400731365e7936a3aba16a15e47`; request `add7183b61224442933a96878237b0b0`.
- `KANON.md`, complete lines 1–553 across requests `d310b546cf6641b59d0955ef9b835eef` and `564936a2cc1647b6bfeb73d21403c9b1`; raw SHA-256 `8667a6e524fe1c73fa1b4aaccb5afdc5d965bb121694c93abdc8a888f17f2365`. The provisional §25 is not used as a new permission grant.
- `Method/03_Prompt_Pack_Method.md`, complete lines 1–370 across requests `52dcce28c0344159ac0bf74e9f2155f3` and `c72c7368e67e4eb281dc5ed73e63174b`; raw SHA-256 `07a27dd3517f4c08aefef10e74e6929c98e5a3ebcd0367bd8fdb7caf75d562f2`.
- `Method/04_Implementation_Proof_Method.md`, lines 1–350 of 482, including complete T-01–T-10; raw SHA-256 `261e9c16f3041bca0fe19f98f1ff9cf06acdfe914152094eaf8e6c8cf964d64a`; request `18df4302b4484c929b7b5e7a6c4796cc`. Remaining lines were not re-read for R2.
- `Method/16_Implementation_Ready_Standard.md`, headings searched and lines 379–489 of 573 read for termination/escalation; raw SHA-256 `df589510432ba5e60848feca4eed9d93b5fb031f7bd07dee03e1d46b2daed564`; requests `c9e90baefb904518a9b85367af4f49db` and `7c850822b7bc49a798eb257b51a75c55`. Not a full standard audit; the executing colleague must read the complete applicable current standard.
- `docs/prompts/02_RECURRING_WAKE_TEMPLATE.txt`, complete lines 1–61; raw SHA-256 `7d2b31137f4f28d72c9c7b834de791919ec3cedf3102bc1fb51476c8e8da97e9`; request `113986f798a94c06824b8a641f2fbd49`. Other wake-file evidence above is retained R1 evidence, not a second claimed read.

Raw-file hashes identify complete file bytes; a partial line read does not establish full-file review or execution. They are not excerpt `content_sha256` values. The directory listing of `ollama/app` was only navigation, not evidence of any feature being active.

This revision changes the delivery document only. No Ollama runtime code, onboarding record, memory, model, setting, application installation or TECHNE method is changed by authoring it. Its structural/coverage checks are document checks, not an independent blueprint judgement or a software reality gate. The executing colleague is explicitly tasked with earning those gates.


## 6. R3 amendment — six requested capabilities and preserved prior scope

**Request basis:** Ash asked to add the six features from his immediately preceding request: automatic LETHE memories; relevant summaries from current and past chats rather than full-history model input; long-conversation loading/navigation improvements; a top-left complete-conversation `.md` download; goal planning/execution with timed checkpoints and automatic continuation; and user conversation during agent mode followed by task resumption unless stopped. These are added product requirements, not claims of existing code. The engineering details and acceptance tests are Eko's design directions for the delivery colleague to specify and judge.

| Requested feature | Owning new item | Existing work reused / extended | Required discriminating proof |
|---|---|---|---|
| Automatic memory storage using LETHE | OI-26 | OI-05, OI-08, OI-09 | No model store call; real capture, process restart and useful recall; disconnected capture fails. |
| Relevant current/past-chat summaries for each prompt | OI-27 | OI-04, OI-12, existing curated-context designs | Actual bounded request, source-grounded correct use, correction/privacy controls; retrieval removal fails. |
| Fast long-conversation loading and navigation | OI-28 | OI-01 archive/UI recon, OI-18 nested conversations | Real UI latency/resource/mounted-content bounds with complete archive; unbounded loading/rendering control fails. |
| Top-left entire-conversation Markdown download | OI-29 | Existing transcript storage and UI | Saved file matches an independently enumerated full snapshot, including never-loaded messages; page-only export fails. |
| Plan, work, checkpoint at current limit and continue goals automatically | OI-30 | G3, OI-13, OI-26/OI-27 | Multiple work periods from durable state without repeated prompts; missing restore/dispatch fails. |
| Respond to the user and return to tasks unless stopped | OI-31 | G3, OI-30; joint G4 acceptance | Mid-goal answer plus real subsequent progress; stop remains stopped across late events and restart. |

**Preservation and explicit amendments.** All original OI-01–OI-25 identifiers remain. Items OI-01–OI-22 and OI-25 retain their item bodies; OI-23/OI-24 gain the six-feature acceptance/reliability coverage and dependencies. The wider 25-item programme is not replaced by these six. Earlier source ledgers remain historical, not reissued runtime evidence. The recurring colleague's four TECHNE wake inputs and the scope/permission safeguards remain in place.

The title advances R2 → R3; §0.1 names all 31 items; §0.6 adds dependencies/G4 and affected model requalification; §0.11 expands the final workflow; §0.12 names shared storage/context/scheduling boundaries; Phase E2 adds six complete work items before final gates; §§2–3 update checkpoints and design obligations. §0.2/§3 distinguish requested local per-conversation export from excluded bulk export and selective context from unapproved compactor removal. §0.9 distinguishes implementing the requested product scheduler from installing a mechanism to keep the present delivery host running. These are explicit scope reconciliations, not weakened permission or proof gates.

**Qualification remains owed.** “Enough context”, “responsive” and “goal achieved” require the declared task-quality, performance and acceptance evidence. No arbitrary timer, test count, model score or universal guarantee is introduced here. There is no claim that shrinking prompts alone fixes UI latency, that external memory updates model weights, or that an automatic scheduler can continue through a stopped execution service or an exhausted host limit.

**R3 authoring evidence.** The live canonical file at `ollama/docs/OLLAMA_COLLEAGUE_INTEGRATION_WORK_LIST.md` and the supplied sandbox copy were compared before editing and both had raw SHA-256 `35236b8a22011c89374db39670c6b8d0cb8e2990bbac7eb8b2d0dd46190dad21` (83,326 bytes). Bridge read request `abac08680bc94a969ee352a4cbea9022` supplied lines 1–42 and the full raw file hash; authoring used the complete byte-matching attachment. This is not a claim that a 42-line read covered the whole file. Git-status request `58f903f53cd7486f97428e69a7133ac0` recorded Ollama HEAD `953de98d408a9697fa20a48c23973b9dc8eee921`, dirty `main`, with this document and `docs/prompts/OLLAMA_COLLEAGUE_INTEGRATION_WORK_LIST-2.md` untracked. The separate prompts copy is outside this edit; do not assume it has been synchronised.

This amendment changes only the canonical integration document and its downloadable copy. It does not implement runtime features, change wake files, install anything, dispatch colleagues or run a software qualification/independent blueprint judgement. The delivery colleague must re-read live source and earn the required gates; no external library/API claim from a previous discussion is used as current implementation evidence here.
