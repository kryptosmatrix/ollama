# Decision request — bounded chat-history preservation repair

Status: proposed by Eko; awaiting Ash's explicit decision. This document grants no authority.
Date: 20 September 2026, Australia/Brisbane.
Repository: `/Users/krypto/GitHub/ollama`, HEAD `953de98d408a9697fa20a48c23973b9dc8eee921`.

## Decision being requested

Authorise this one existing-code repair under an independently reviewed repair contract rather than requiring ordinary Method 16 blueprint promotion and its dependent Method 03 implementation-rung gate. Retain all six frozen requirements, independent design review before production edits, Method 04 T-01 through T-10, and independent code/test/execution review before accepting the repair. Keep `deepseek-v4.1-flash:cloud` as Ash's selected primary adversarial reviewer.

This is a proposed scope-specific operator exception, not a claim that TECHNE already exempts maintenance. It is not an earned `[IMPLEMENTATION-READY]` status, a change to TECHNE's general rules, a reset of any review budget, or permission to weaken proof. Method 16 section 7 requires an operator override to identify what was not earned and prohibits recording it as `[IMPLEMENTATION-READY]`.

## Exact production boundary

Only these production changes are proposed:

- `app/store/database.go::getChatWithOptions`: classify the chat as absent only when the chat-header query returns `sql.ErrNoRows`, using the existing wrapped `app/types/not.Found` sentinel.
- `app/store/store.go::ChatWithOptions`: preserve every other database error instead of relabelling it as `not.Found`.

Permanent default-run regression tests, isolated fault controls and scoped evidence/documentation are part of this boundary. No database schema, public signature, valid-continuation behaviour, authentication policy, new product feature or unrelated cleanup is included. No live database, installed application, service or global setting is to be changed. Other programme items remain governed by the original commission and current TECHNE rules.

## Frozen requirements — unchanged

R-01. Only an absent chat-header row MUST be classified as `not.Found` by `Store.ChatWithOptions`.

R-02. A non-absence chat-load error MUST preserve its underlying error chain through `Store.ChatWithOptions`.

R-03. Continuing an existing chat whose load fails MUST leave its persisted chat, message, attachment and tool-call records unchanged.

R-04. Continuing an existing chat whose load fails MUST return an error before an inference request is issued.

R-05. Valid existing-chat continuation MUST retain the earlier messages.

R-06. Genuinely absent-chat continuation MUST retain the existing create-and-continue behaviour.

Source: `docs/_design/CHAT_READ_ERROR_INTEGRITY.md`, raw SHA-256 `b691c8004ed82674c82ebcde3a2f876cd85f65cb5f8167ffadc5e3fd6e6e0570`. That draft remains SPEC-DRAFT; this request does not modify it.

## Why the decision is necessary

The draft's provisional 12-added/changed-production-line estimate yields a 480-byte whole-blueprint ceiling under Method 16 R-3.2. Its six frozen requirement lines alone occupy 625 UTF-8 bytes, and the current draft occupies 10,541 bytes. This establishes noncompliance at the stated estimate, not a theorem that every conceivable specification is impossible. R-3.1's itemised estimate is also still owed. Inflating the estimate with all existing target-file lines, writing unnecessary production code, or labelling future repaired behaviour AS-BUILT would not resolve the issue honestly.

The task commission requires earned readiness before implementation. The general router's existing-code repair entries do not themselves waive that commission. The last DeepSeek routing report contradicted itself and is not permission. Repeating advisory reviews cannot create operator authority.

Relevant live source: `TECHNE/Method/16_Implementation_Ready_Standard.md` R-3.1/R-3.2, sections 2.1 and 7; raw SHA-256 `df589510432ba5e60848feca4eed9d93b5fb031f7bd07dee03e1d46b2daed564`. The quoted operator-override provision is at line 485 in that revision. The commission is `docs/OLLAMA_COLLEAGUE_INTEGRATION_WORK_LIST.md` sections 0.3 and 0.5.

## Evidence already available; work still owed

The initial Store diagnostic compilation error was corrected before the interrupted response ended. Its corrected raw SHA-256 is `cb497b9f4d2751dff6acc3de003e6453a1f7d13d33059e789752a3b1d62df5c4`. The executed rerun, Bridge job `46773b72aaa54706a22b8d4a8199a70e`, compiled successfully: genuine absence passed and preservation of the real closed-database error failed. This is evidence of the current production defect, not a repaired candidate.

The separate authenticated, process-separated HTTP diagnostic reproduced deletion of original messages, attachment and tool-call records, and title loss. The original red results and every failed instrument attempt remain under `evidence/2026-09-20_cont03/`. No new execution is claimed merely because those records were re-read in this retry.

Still owed after a decision: independent review of the final bounded repair/test contract; permanent tests; the production correction; candidate/broken/candidate runs with compiled behavioural controls; the declared regression/proof set; and independent re-execution and adjudication. Text-only review cannot be reported as an independent test run. The tool-enabled reviewer route remains unresolved; the earlier Codex attempt produced no review and an Ollama harness-discovery attempt was blocked before execution. Those limitations must not be bypassed or relabelled as passes.

## Strongest objection

A maintenance exception can become a habitual escape from design review. Prevent that by binding it to the two named production functions, the six unchanged requirements, and this single defect; preserving both pre-code design review and post-code executable review; and recording that ordinary formal blueprint promotion was not earned. No exception carries forward to the wider 31-item programme.

## Suggested operator wording

“I authorise the bounded chat-history preservation repair described in DECISION_REQUEST_CHAT_READ_REPAIR.md under a reviewed repair contract, without formal Method 16 promotion or its dependent Method 03 readiness gate for this slice only. Keep all six requirements, independent pre-code design review, Method 04 testing and independent executable adversarial review. Do not claim an earned IMPLEMENTATION-READY stamp or expand the scope.”

No operator approval has been received or inferred from a generic request to continue or retry. No production code was changed in preparing this request.
