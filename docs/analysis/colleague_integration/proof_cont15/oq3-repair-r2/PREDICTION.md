# Prediction for the revision-3 baseline (pre-registered)

Continuation 15, Treadle (Claude Opus 5.5). Written at 2026-09-27 08:18:39 AEST, before the revision-3 baseline runs, at commit bceda9a4 (instrument revision 3 at 33f72b8f, whitespace-only gofmt at a12e28d9, revision-2 goldens moved aside at bceda9a4).

**Expected:** a valid run reading **67/39 = 1.7179**: 84 observed deliveries, 67 incorrect, 17 correct, 0 missing. The one extra delivery is T8's new turn 3, which requires a carrier and which the unchanged candidate sends with neither carrier nor admission field (categories admission-missing and missing, each one higher than in the revision-2 baseline). Every other scenario reads as in the revision-2 baseline (66/39): the repairs to the method check, the leak scan, the idle proof and the capability-less start change what the instrument can see, not what the unchanged candidate sends. Auxiliary: 16 preloads and 4 summariser requests equal to their goldens; no leak, no method failure. T10 starts normally (it preloads), so its no-dispatch turn is still sent and still scores bad. Reference arm: 82 of 82 correct (the two capability-less deliveries have no expectation to re-send). Repeat run: identical.

**Falsifier:** any other figure. A difference is explained from the evidence before anything is relied on, and this prediction is kept, not edited.
