# Ollama upstream reconciliation — 2026-10-01

**Mode:** no-production-change reconnaissance plus disposable synthetic merge
**Decision:** defer upstream integration until after the G1 Developer Preview
**Working branch:** `eko/g1-finish-20261001`
**Fork candidate:** `465b963e35f6afa46f15e6fdec043189a905d1d8`
**Fresh upstream/HEAD:** `1abe35e6e6e777e858bbfbba283667ee8d516801`
**Merge base:** `8d8c701d6a8a7f4129488d7698e3ae02d24e223c`

## What was actually checked

On 1 October 2026 Eko fetched `upstream` and `origin` with prune, then compared the current G1 candidate with the freshly fetched `upstream/HEAD`.

- Candidate-only commits from the merge base: **193**.
- Upstream-only commits from the merge base: **208**.
- Files modified on both sides: **48**.
- A disposable `git merge --no-commit --no-ff upstream/HEAD` in an isolated worktree produced **37 unmerged paths/conflicts**.
- The merge was then **aborted successfully**. No upstream commit was retained in the G1 candidate, main, or the historical Abuttal worktree.

The upstream delta is not ordinary drift. Upstream commit `c16bf9892` (11 September 2026, “cmd: remove built-in agent (#18393)”) removes the built-in agent/TUI surfaces including `agent/session.go`, `cmd/agent_tui.go` and `cmd/tui/chat/*`. Those are load-bearing production paths for this fork’s commissioned colleague programme and for G1’s agent-TUI acceptance surface. Other upstream changes also remove/move fork-touched conversion and MLX/model surfaces.

## Decision

Do **not** merge upstream before the first G1 Developer Preview.

Reason: the current objective is to finish and test the upgraded fork’s persisted-instructions/request-delivery slice through the fork’s actual desktop, agent-TUI, classic-chat and one-shot paths. Merging upstream now would simultaneously migrate away one of those paths and force resolution of 37 conflicts. That is an architectural migration, not a prerequisite update.

The safer dependency order is:

1. Finish OQ-4 / OQ-3 revision 5 / Method-16 promotion on the existing fork source.
2. Implement and prove G1.
3. Build an isolated Developer Preview `Ollama.app` from the exact proved commit and test it without replacing the installed app.
4. Treat upstream reconciliation as a separate follow-on migration: decide which upstream removals are adopted, which fork capabilities are re-homed, then run a dedicated regression/reality gate.

## Falsifier / reconsideration

Reconsider this order only if a G1 implementation dependency or build blocker is demonstrated to be fixed **only** by an upstream change and cannot be safely backported, or if Ash explicitly chooses to migrate the fork architecture before the G1 preview.

## Containment

The disposable upstream merge was performed only in the isolated worktree later named:

`/Users/krypto/GitHub/ollama-eko-upstream-20261001`

The merge was aborted before G1 design edits began. Historical branches/worktrees, `main`, installed Ollama.app, conversation data and user settings were not changed by the reconciliation probe.
