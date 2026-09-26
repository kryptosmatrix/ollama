# Amendment 2 result (run 3, 26 September 2026, 21:16:14 to 21:16:30 AEST)

Twelve synthetic requests through Ash's installed daemon, exactly as QUALIFICATION_PLAN.md amendment 2 fixed them;
harness `cloud_check2.py` (SHA-256 6b2b4595…, committed in 70c210b6 before the run), exit 0.

All twelve PASS. Every expected value was exact in the reply's content, and every provider count was at or below its prompt bound.

| Model | T0a / T0b / T0c tokens | T1 tokens | T1 prompt bound | Bound ÷ tokens |
|---|---|---|---|---|
| deepseek-v4.1-flash | 52 / 289 / 314 | 11,001 (1,000 lines) | 30,269 | 2.75 |
| glm-5.2 | 51 / 174 / 180 | 13,266 (1,000 lines) | 30,269 | 2.28 |
| glm-5.3 | 59 / 182 / 182 | 25,722 (2,000 lines) | 57,269 | 2.23 |

T0 compared think false, false with one minimal tool, and omitted with that tool. T1 was the clients' shape: streaming,
think true, five tool definitions, a tool call with id `call_1` and its result, and CARRIER, FIRST, MID and LAST asked
for. The tool allowance rule gave A = 304. The six excesses over the minimal tool's 110 bytes were 127, 152, 13, 19, 13
and 13 tokens; deepseek-v4.1-flash's tool preamble is the largest. Prompt tokens this run: 51,472. The cumulative total
under the plan's cap is 453,965 of 500,000.

Observed and not otherwise used: deepseek-v4.1-flash returned no thinking text in T1 with think true (it did in T0c
with think omitted). glm-5.3 still reasons in its reply content with think false, within the 400-token budget.

glm-5.3's T1 passed at 25,722 tokens, above its run-2 size of 24,999. Under the plan's rule — V is the largest passing
probe's count — its verified size becomes 25,722. The other two models' V are unchanged (99,680 and 103,671, plain
text). What this establishes and what it does not is stated in the plan's amendment 2. The feature profile was checked
at these T1 sizes; admitting it between T1's size and V rests on inference.
