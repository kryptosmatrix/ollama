# Continuation 13 proof package (Thole, 26 September 2026)

Curated, byte-identical copies of evidence whose working originals live untracked in the primary
checkout at `docs/analysis/colleague_integration/evidence/2026-09-26_cont13/`. `MANIFEST.sha256` lists
every file here; verify with `shasum -a 256 -c MANIFEST.sha256` from this directory.

`control-records-pre-edit/` holds the five programme control records exactly as they stood before
Thole's first edit (custody transfer and pointer reconciliation), with their own `MANIFEST.json`.

`oq1-measurements/` holds the G1 OQ-1 context-admission measurements: the two harness scripts
(`oq1_probe.py`, `oq1_probe_r2.py`), their stdout, and per-probe results, rendered prompts, responses and
daemon/runner logs for the native and rendered chat routes (`run1`: default shift, shift off, oversized
input; `run2`: keep-whole-prompt and the reload cost of changing shift). They were produced against the
unchanged candidate daemon built from `04009c432e313eef65172293e75feb038ebe476b` and the pinned native
llama-server b10091 (`b4d6c7d8ff69c2e05e4e8ee7e6e710a08abd7b45`, binary SHA-256
`795c8de26497272d3c56869042b52944eeeaee6fe90dc2874c1bcf7bb0b7bda8`), model qwen3:8b, in an isolated
sandbox (separate HOME, model directory of APFS clones, port 11500, `OLLAMA_NOPRUNE=1`); every probe
used a fresh daemon and every teardown was verified. The one key mentioned in a log is the sandbox's
freshly generated public key; no private key or credential is included.

These are measurements of existing behaviour, not acceptance of any G1 requirement. No programme item
is accepted by this package.
