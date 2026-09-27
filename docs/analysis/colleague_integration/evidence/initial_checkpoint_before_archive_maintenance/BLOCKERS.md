# Active blockers

Owner: Eko. Recorded: 20 September 2026, Australia/Brisbane.
Current item: OI-01. Programme disposition: PARKED at an initial reconnaissance checkpoint.

## BLOCKING — bridge-command-archive-full

| Field | Value |
|---|---|
| id | bridge-command-archive-full |
| category | BLOCKING |
| status | open |
| created_by | Eko |
| summary | The host command tool refuses commands because its persistent archive is full. |
| affected_arc | OI-01 execution baseline; subsequent builds, tests, command-backed reviewers and Git write operations |
| source | `evidence/2026-09-20_eko_execution_blocker.json`; `evidence/2026-09-20_eko_execution_recheck.json` |
| owner | Eko for task recovery; Ash controls permission for archive/service maintenance |
| stop token | PACK_PREREQUISITE_MISSING — working host execution/proof environment |

The first rejected request was `eko-ollama-20260920-arrival-01`, a read-only arrival-document probe. A separate late recheck, `eko-ollama-20260920-execution-recheck-01`, attempted only `/usr/bin/true`. Both returned the same complete error:

```json
{"ok": false, "error": {"code": "COMMAND_ARCHIVE_FULL", "message": "Command archive is full; explicitly forget completed jobs or archive the service state"}, "is_error": true}
```

Neither call returned a job identifier or child exit code. No command execution was observed. These are admission failures, not failing application builds or test outcomes. File reads, local Git inspection and create-only report writes still work.

### Recovery boundary

Permission to implement Ollama is not blanket permission to erase earlier Bridge job histories or reconfigure the Bridge service. No completed job has been forgotten, no logs deleted, no processes stopped and no service settings changed in this task.

Only five recent command records were inspected. Several were already forgotten tombstones. The declared limit is 256 saved jobs; the actual archive count, size and exact retention condition were not measured. Do not assert that forgetting one particular job or deleting tombstones will solve the problem.

The next action is to obtain Ash's approval for bounded, preservation-first Bridge archive maintenance, or have the Bridge owner restore command capacity. Before any deletion, identify the exact records and preserve and verify the required command/input/output evidence in approved custody. Use supported maintenance operations; do not directly edit the service database, bypass its cap or destroy idempotency records as a shortcut. No maintenance method is approved by this report.

After capacity is restored, submit a new explicitly identified no-op probe, poll the returned job and verify its real terminal exit status. Then recheck repository state and ownership before running the actual discovered baseline. A previously rejected request has no successful result to inherit.

## Obligations still pending, not additional discoveries of defects

- OI-01 is incomplete: installed/running app identity, toolchain/dependency discovery, canonical test commands, source/test coverage and baseline execution are still owed.
- No blueprint or prompt pack has been authored or promoted. The work list is the commission, not an earned implementation-readiness verdict.
- No independent colleague has reviewed this checkpoint, a design or code. The command-backed review routes have not been launched. Eko's own role changes are not independent review.
- Lineage registration, predecessor observation and LETHE document-pointer recording have not been performed. Record their completion through the actual normal paths on continuation; do not fabricate prior observations or memory receipts.
- Git write operations and remote verification have not run. The new checkpoint is local, uncommitted and unpushed. No recovery branch, accepted candidate or verified remote tip exists for this new work.

## Concurrent material

`docs/OLLAMA_CROSS_MODEL_COLLEAGUES_AND_CAPABILITY_ROUTING_HANDOFF.md` appeared as a second untracked file during the read. Eko did not create, inspect, edit, stage or delete it. Its owner and contents remain unverified. Preserve it, inspect it on continuation and establish occupancy before a branch switch or commit. The original work-list file is likewise preserved unchanged.

## Safe work completed before stopping

Read the complete commissioning work list and routed arrival/method sources; used directory traversal and exact-symbol searches; read the limited terminal and desktop request paths recorded in `SOURCE_MAP.md`; discovered actual frontend script declarations; saved a complete 31-ID navigation index and exact command-admission errors.

No runtime feature is claimed complete. No failure has been hidden by substituting a sandbox build, fabricated reviewer, mock positive acceptance path or stale test report.
