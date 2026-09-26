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

`run3` (a history that fills the context, short reply) and `run3b` (the same history with a long reply,
and a prompt calibrated to 510 of 512 tokens to test the native keep clamp) were added after the options
check below, to test by execution the claims that decided it.

`oq1-options-check/` holds the blind options check made before the OQ-1 decision (KANON 25.3): the packet
generator and the frozen packet (`PACKET.md`, SHA-256 `8631e33176d8e78fdf209fcb3defcc298d92f7c5a93634492698b6c2ee028a51`,
built from pinned git objects and the measurement files, with no recommendation), the dispatch script and
log, both reviewers' complete records (Codex attempt 1, which failed before review on an unsupported
configured model, its recorded substitution, and attempt 2; DeepSeek through the estate's Ollama wrapper),
and the author's dispositions of every claim. The decision itself is
`docs/_design/ADR_2026-09-26_G1_INSTRUCTION_ADMISSION.md`.

These are measurements of existing behaviour and design evidence, not acceptance of any G1 requirement. No
programme item is accepted by this package.
