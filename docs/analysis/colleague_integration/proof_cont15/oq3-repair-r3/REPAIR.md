# Repair of the round-3 findings: OQ-3 instrument revision 4

Continuation 15, Treadle (Claude Opus 5.5), 27 September 2026. The round-3 review (a judge and a repair reader on Codex) found eight distinct blocking defects in revision 3, all accepted (`../oq3-instrument-review-r3/DISPOSITIONS_CODEX.md`). This record states what revision 4 built for each, the control that shows it working, and the deliberate break that makes that control fail. Design evidence only: nothing outside `proof_cont14/oq3-instrument/` and `proof_cont15/` changed.

Commits on `treadle/g1-oq3-repair-20260927`: `a88ee4f2` (revision 4: code, controls, `DESIGN.md`), `ffe0423d` (in-place corrections of two revision-3 records: the false preload claim in `../oq3-repair-r2/REPAIR.md`, and the header statement in `../oq3-baseline-r3/BASELINE.md`, re-pinned in its manifest), `37b966bd` (the revision-3 goldens moved aside as `goldens-superseded-01b86567/`), `071fc4e9` and `f57b6b18` (the prediction for the revision-4 baseline, and a marked correction of when the goldens moved; the expected figures were not changed).

## The eight findings

**1. The idle proof accepted an active preload (J1, R1).** Revision 3 claimed the probe's status is drawn only when no turn, compaction, preload or approval prompt is active; for a preload that is false (`render.go:1449` draws no activity line during a preload, and `notificationLine`, `:1344`, does not check for one). Built: `oq3Daemon.preloadDone` and `oq3Run.mustPreload`. The preload function (`cmd/agent_tui.go:583-608`) sends `POST /api/show`, `POST /api/generate`, then `GET /api/ps`, and returns once that is answered; every step that starts a preload (the terminal start, the launcher's arrival in the chat, a model switch, and a capability-less start that preloads) waits for that sequence under its own label before the idle proof. The false claim is corrected in `DESIGN.md` and in place in the revision-3 record. Control: a model switch whose preload is held back two seconds at the daemon (`oq3Daemon.delay`, which records the request under the label current at its arrival and then holds only the response) keeps its `/api/generate` under the switch and none under the next turn. Break: M13 (no barrier on the switch) fails it. M23 (no barrier at the terminal start) survives, as declared before the run: the start already waited for the preload's `/api/generate` under its own label, so the barrier there adds only the `GET /api/ps`, a metadata request that is never compared with a golden.

**2. T8's busy reload might never be handled while the turn was busy (J2).** Built: after typing the refused command, the driver types a marker (`OQACKZ`, no digit, so an approval prompt could not act on it) and waits until the chat's input box shows it, then clears it, before the await step releases the held response. Keys are handled in order, so the marker proves the Enter reached the chat while the response was held, when no approval prompt can exist. Control: a held tool turn with the command sent during the hold; the step records that it was handled while held, and in the session's transcript the marker comes before the approval prompt, which can open only after the release. Break: M16 (revision 3's behaviour: no marker) fails it. Limit: the race itself cannot be forced in a control.

**3. An unrelated start-up failure could pass as the refusal (J3, R2).** Built: `oq3RefusalReported` requires the daemon's version and a name for the refusal (`context_unavailable`, "context unavailable" or `chat.admission.v1`), compared case-insensitively with white space removed. The blueprint gives no exact text for the terminal's report: gap 7 in `DESIGN.md`. Controls: a stand-in that prints a banner naming the version and then fails for an unrelated reason makes the run invalid; the stand-in that reports the refusal is held. Break: M14 (version alone) fails the first.

**4. A permitted refusal that left the session open outside an idle chat was rejected (J4).** Built: in `terminalStartWithoutCapability`, a session that has reported the refusal and cannot prove an idle chat is recorded as refused and open (`termOpenNotIdle`); the no-dispatch turn is not typed, the driver waits five seconds (`oq3RefusedTurnWait`) during which any request under the turn's step is still captured and scored, and the stop step ends the session. Control: a stand-in that reports the refusal and then stays open ignoring its input is held, its turn scored as refused as required, the run valid. Break: M15 (a refusal must reach an idle chat) fails it.

**5. Duplicate JSON members hid leaks from the scan and the normaliser (J5, R3).** Built: `oq3JSONScan` walks a body token by token: every key and value decoded, in order, duplicates reported by object path and key, and a string that is itself a JSON document walked in turn. A duplicate member makes a scored delivery incorrect (`duplicate-member`), fails a checked summariser or preload, is an auxiliary failure on any other request, and is never frozen as a golden. Controls: instruction text in the first of two duplicate members, plain and escaped; a clean request with a duplicate member; a desktop turn's own request with its model repeated, and with a carrier-bearing `messages` member ahead of the correct one (each reads as the golden to a decoder that keeps the last member, and each scores incorrect). Breaks: M17 (duplicates ignored) and M18 (bodies scanned through a map, as revision 3 did) fail them.

**6. A number such as `1e400` disabled the decoded scan (J6, R4).** Built: the walk reads numbers as `json.Number`, so no value can fail it; a JSON body that still cannot be walked is an auxiliary failure (`undecodable-body`), never scanned by a weaker rule instead. Control: an escaped sentinel beside `1e400` is reported. Break: M18 fails it.

**7. One malformed query component hid the others (J7, R6).** Built: `oq3EnvelopeLeaks` scans the query raw, through the partial map `url.ParseQuery` returns even with an error, and component by component, each key and value decoded on its own. Control: `note=%53ENTINEL-A-MID-9e41&broken=%` is reported. Break: M19 (revision 3's whole-string handling) fails it.

**8. The envelope scan missed Host, trailers and JSON inside header values (J8, R5, R7).** Built: captures record `Host`, the raw request target and trailers (trailers are complete when the fake records a request, because it reads the whole body first); every envelope string goes through `oq3StringLeaks`, which decodes JSON inside a value. Controls: a sentinel as JSON inside a header value, in a trailer, and in `Host`. Breaks: M20 (header values not decoded), M21 (trailers not scanned) and M22 (Host not scanned) each fail one.

## How the choices were checked

Each repair answers a finding a different substrate made, and the round-4 review puts the repairs to a fresh reviewer and a repair reader. The one new design choice, the preload barrier, rests on the production preload function's order of requests, which a change to that function would break as an invalid run, never as a wrong number. The refusal vocabulary is a reading of §8 that OQ-4 should replace with a stated text (gap 7).

## Deliberate-break matrix

`mutations/mutate.py` reverts one repair at a time (a list of replacements per mutation) in a scratch copy of the committed instrument, compiles it into package `cmd` through its own overlay, runs the controls and reads the fatal line. Run at the committed revision 4 (`a88ee4f2`; tree `ffe0423d`, clean), `go.mod` unchanged across all 23 runs: **20 of 20 breaks caught by the intended control with the intended message; 3 of 3 declared survivors survived** (M11 and M12, layered input clearing, as in revision 3; M23, explained under finding 1). `mutations/matrix.json` and one log per mutation.

| Mutation | Result |
|---|---|
| M01-leak-scan-raw-bytes | caught: `auxiliary fault leak-escaped-sentinel was not reported` |
| M02-no-envelope-header-scan | caught: `auxiliary fault leak-in-header was not reported` |
| M03-no-query-scan | caught: `auxiliary fault leak-in-query was not reported` |
| M04-fake-serves-any-method | caught: `GET /api/show answered 404, want 405 as the real daemon answers` |
| M05-scorer-ignores-method | caught: `PUT replay: status 405, delivery {Scenario:RM Step:1 Label:RM/01/desktop-turn Trial:RM-X Index:…` |
| M06-probe-always-idle | caught: `idle control: the probe read an open approval prompt as idle` |
| M07-runner-trusts-quiet | caught: `a turn left waiting after the daemon finished was treated as ended: turn verdict "done", invali…` |
| M08-no-startup-refusal-path | caught: `a permitted refusal at start-up was not held: invalid [RCR step 0 terminal-start: terminal sess…` |
| M09-exit-without-report-accepted | caught: `terminal-exit-silently passed as a refusal: [{Scenario:RCR Step:1 Label:RCR/01/terminal-turn Tr…` |
| M10-t8-inserted-turn-expects-reload | caught: `T8 control: busy-committing adapter 0/5 bad now, 0/4 under the round-2 ledger (want >0 and 0); …` |
| M11-probe-keeps-input | survived, as declared |
| M12-typeline-keeps-input | survived, as declared |
| M13-no-preload-barrier-on-switch | caught: `preload control: invalid []; generate under the switch 0, under the turn 0` |
| M14-refusal-by-version-only | caught: `terminal-banner-then-crash passed as a refusal: [{Scenario:RCR Step:1 Label:RCR/01/terminal-tur…` |
| M15-refusal-must-reach-idle-chat | caught: `a permitted refusal at start-up was not held: invalid [RCR step 0 terminal-start: terminal neve…` |
| M16-no-held-command-marker | caught: `held-command control: invalid []; handled while held false; marker at -1, approval prompt at 62…` |
| M17-duplicates-ignored | caught: `auxiliary fault duplicate-member-clean was not reported` |
| M18-body-scan-through-a-map | caught: `auxiliary fault leak-nested-json was not reported` |
| M19-query-whole-string-only | caught: `auxiliary fault leak-query-partial was not reported` |
| M20-header-values-not-decoded | caught: `auxiliary fault leak-header-json was not reported` |
| M21-trailers-not-scanned | caught: `auxiliary fault leak-trailer was not reported` |
| M22-host-not-scanned | caught: `auxiliary fault leak-host was not reported` |
| M23-no-preload-barrier-at-start | survived, as declared |
