# Bridge command-archive maintenance — 20 September 2026

Owner: Eko (GPT-6 Astra Pro, ChatGPT). Scope: Ash's explicit permission to preserve and verify existing command evidence before clearing archive capacity. This report records maintenance and a bounded Ollama baseline, not implementation of the colleague programme.

## Result

Command execution is restored on the same Eko Write Bridge 0.3.0 service and execution store. One harmless diagnostic plus 32 backed-up terminal jobs were retired: **33 records total**. Their request-ID/hash tombstones remain. No Bridge package, service configuration, limit, credential value, listener or installed Ollama application was changed. No service was restarted and no original command was re-executed.

The maintenance used the existing `forget_command` operation, first through the connected tool and then through the same authenticated numeric-loopback MCP endpoint. The batch helper did not mutate the service database directly. The existing service credential was used locally for those authorised calls; its value was not printed or exported.

## Preservation and bootstrap

The full archive initially refused even `/usr/bin/true`. Public job inspection omits the stored command specification, so merely saving its displayed status would not preserve everything `forget_command` removes.

The bootstrap record was `f68fa53d7c8c4fbaa57ba6fef20f86b2`, request `eko-bridge-resolution-plan-probe-20260909-01`. Its complete harmless printf specification was recovered and matched to request SHA-256 `306ed3e519fbfc6ad5cf6ee9eeed9fd7925b4185bd0a9a0a87efc2e5dad703f1`. A same-ID, same-arguments request returned the original job with `replayed=true`, unchanged timestamps and no new execution. Its exact specification, empty input set, complete stdout/stderr bytes and exposed metadata were saved and read back before retirement. The original service tombstone retains the internal metadata not exposed by the tool.

Bootstrap evidence: `evidence/2026-09-20_bridge_bootstrap_archive.json`, raw SHA-256 `d56215ee4a138c6d4e52f9770e24e9020bb52eb598fbde1dda5bfb36697f2e07`. That immutable record deliberately says retirement had not yet been attempted; this report and the execution evidence supply the subsequent outcome.

The freed slot admitted a one-shot backup of 32 completed jobs. It held their supervisor leases, made a SQLite-consistent metadata snapshot, compared complete selected rows and inputs, verified command fingerprints, copied both output streams, checked lengths and hashes, and synchronised the retained files/directories. It exited zero. A second command independently re-read the retained bytes and metadata before invoking retirement, then checked the tombstones and archived output again afterwards. This is deterministic verification by maintenance code, not an independent AI review.

## Private archive and its limits

Private location:

`/Users/krypto/Library/Application Support/EkoWriteBridge/maintenance/20260920-ac311451e3bc41618f5db07392348f06`

Manifest: `MANIFEST.json`, SHA-256 `f6ca527ec23b51f73866eebe3efd5a9d74fe77a29a1e0ca20ea0a17e4ebc6058`.

The archive contains a consistent `commands.sqlite3` snapshot and complete stdout/stderr for exactly the 32 enumerated jobs: **734,548 output bytes**. It preserves passing and failing results. Raw specifications, inputs and outputs remain in private, owner-only storage outside the repository. The bootstrap diagnostic is preserved separately in the non-sensitive evidence JSON above.

This is **not a complete runnable restoration image** of the whole service: the database also contains non-selected job rows whose outputs were not copied, including the then-active backup command. Do not replace the live database with it or replay its commands. Read the archive as historical evidence. It is on the same machine, not an off-device backup, and checksums do not establish protection against malicious modification by the same OS user.

## Executed recovery checks

| Check | Actual result | Bridge job |
|---|---|---|
| Before maintenance: fresh no-op | Rejected with `COMMAND_ARCHIVE_FULL`; no child existed | Request `eko-archive-maintenance-20260920-preflight-01` |
| Private backup | Exited 0; all 32 selected rows, inputs and streams verified | `81838f447a5e4575924e70d1560951a4` |
| Fresh no-op plus interpreter inspection | Exited 0 | `f066bb69db834fa280d8d8497ecce8e4` |
| Verified batch retirement | Exited 0; six jobs were already retired through connected tools, 26 retired through ordinary loopback MCP calls; all 32 tombstones and backups verified | `23b09e55888343ae868cfd1a56adb23d` |
| Deliberate `/usr/bin/false` | Exited 1, `succeeded=false`; failure was not swallowed | `f69a3539cb4f42309e57240231e29de4` |
| Clean `/usr/bin/true` plus read-only Ollama baseline | Exited 0 | `86ab9302ee424746b2d8b23f202ee575` |

At the batch verifier's recorded instant, 226 retained jobs remained against the unchanged 256-job limit; the verifier itself was the one running job. Subsequent checks consume more slots. This is bounded maintenance, not a permanent retention fix or a claim of 30 slots still free at every later time. Original request tombstones were not deleted and continue to consume whatever identity-storage policy the installed version applies.

Sanitised evidence: `evidence/2026-09-20_bridge_batch_backup.json`, `evidence/2026-09-20_bridge_maintenance_result.json`, and `evidence/2026-09-20_bridge_retirement_receipts.jsonl`. Receipt-file SHA-256: `6c8371aa6a9ed705229114505cc5880ba58e8908d19c7fc5bf277bd318ce896c`.

## Ollama baseline resumed

The repository remains at `953de98d408a9697fa20a48c23973b9dc8eee921`. The captured inspection found one worktree on `main`, no tracked working/index changes, and the existing untracked programme documents and report directory. `git diff --check` returned zero and `STATE.json` parsed successfully. This does not establish exclusive checkout ownership or remote currency.

Host versions actually executed: Go 1.26.5 darwin/arm64; Node v25.8.2; npm 11.11.1; CMake 4.4.2. The frontend `node_modules` directory exists; that alone does not establish dependency compatibility. Evidence: `evidence/2026-09-20_baseline_environment.json`.

Two existing tests were then compiled and executed once, with dependency downloads disabled and modules read-only:

```sh
go test -mod=readonly ./agent -run '^(TestSessionAddsSystemPromptOnlyToRequest|TestSessionChatRequestMatchesRunRequest)$' -count=1 -json -timeout=90s
```

Both were discovered, completed and passed; zero failed, zero skipped, a closing package PASS was present, the child exit was zero, and the recorded source hashes did not change. Full output: `evidence/2026-09-20_request_baseline.jsonl`, SHA-256 `ddff080c1638fb3d134f1be8893f8bf00fd9d7a69bdbae7f17f702e080fc85a2`. Result and manifest: `evidence/2026-09-20_request_baseline_result.json`. Bridge job: `a62675c9ea914b769db928933edf6629`.

These tests use `fakeClient`. They establish only the inspected existing Session request-assembly baseline, not real inference, terminal/UI bootstrap, persisted instructions, restart continuity or any new feature. The second test derives its expected request using the production builder, so it is a consistency check rather than an independent semantic oracle. No deliberate-break acceptance for an Ollama feature or independent colleague review occurred.

The additional cross-model handoff was read completely (418 lines), raw SHA-256 `24da38b49da7c628a874d96e1d100e7f269315ba591d9bea745c10ba0d07ddcf`. It explicitly remains a companion intent record, not a replacement for R3 or an implementation-readiness grant. Its F01–F07 cover the existing Codex comparison, delegation, supervision, nested viewer, authority and evaluation/routing work; detailed reconciliation remains part of OI-02. Optional direct child messaging remains optional, and no new API or model ranking was adopted here.

## Continuation and outstanding custody

OI-01 remains incomplete. Continue source/store/UI and installed-application reconciliation, establish checkout ownership, complete lineage and LETHE bookkeeping, then capture the broader build/test baseline. The work list still requires reviewed designs and earned readiness before runtime edits. None of its 31 items is newly accepted by this maintenance.

The checkpoint documents and maintenance evidence remain local and uncommitted at this boundary. No branch, worktree, commit, push, fetch, merge or deployment was performed. The archive-capacity blocker is resolved; Git custody, broader reconnaissance, design and review obligations are not thereby closed. A bounded file-read timeout also occurred during checkpoint inspection; command-based readback succeeded, so this report claims restored command admission, not complete Bridge reliability qualification.
