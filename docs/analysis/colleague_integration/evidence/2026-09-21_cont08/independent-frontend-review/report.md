The stderr contains only the known preserved warnings (stale Browserslist, missing sourcemap) — no setup/compile errors. The failures are genuine assertion failures. All required arms are complete.

## Source and requirement findings

- **Contract** (`VALIDATION_CONTRACT.md`) requires: eleven committed parser/real-renderer tests discovered and finished; full suite plus typecheck; three isolated behavioural fault controls (no-output, disconnected citation plugin, wrong-page); a clean rerun after controls; no mocks replacing Streamdown/Shiki/remark; no pass on zero assertions or failed collection.
- **Runner** (`validate_frontend.py`) enforces: `tsc -b --force` compile gate before any test arm; exact expected counts (11 focused / 154 full); `EXPECTED ⊆ titles`; every file has assertions; all statuses in {passed, failed} (no skips/pending/todo); source fingerprints unchanged; mutants run in an isolated copy with `node_modules` symlinked, never touching the live tree; `detected` requires exit 1 **and** the specific required title carrying a failure message. This is a sound instrument.
- **Config**: `vitest.config.ts` sets `test.server.deps.inline: ["streamdown"]` — the exact repair the contract specifies, so Vite (not Node) processes Streamdown's `katex/dist/katex.min.css` import. No mocks introduced.
- **Renderer/parser/highlighter**: real production source. `remarkCitationParser` does two passes (range + generic citations, then adjacent-duplicate coalescing). `StreamingMarkdownContent` selects `pageStack[cursor]` and only emits an `href` for `http`-prefixed pages. `highlighter.ts` uses real Shiki. The corrected assertion (line 24–26) requires `<span ... data-streamdown="strong" ...>important</span>` — matching the installed Streamdown contract, and still input-dependent (it also asserts `Alpha` present, `Bravo` absent, and the `em` case for the second input).
- **Type-only record**: `typescript-emission.json` shows byte-identical emitted JS before/after for all four files, 0 diagnostics. This is type-only evidence, not live-UI proof.

## Runs I requested (fresh, uncached)

| # | Arm | Result |
|---|-----|--------|
| 1 | candidate (clean) | 11/11 passed, typecheck exit 0, source unchanged |
| 2 | no-output mutant | 5 failed / 6 passed, exit 1, intended title failed |
| 3 | no-citations mutant | 2 failed / 9 passed, exit 1, intended title failed |
| 4 | wrong-page mutant | 1 failed / 10 passed, exit 1, intended title failed |
| 5 | full suite | 154/154 passed, 0 skipped, all 11 new titles discovered |
| 6 | candidate (clean, last) | 11/11 passed, typecheck exit 0, source unchanged |

## Fault-detection findings

- **no-output**: replacing `{content}` with `{""}` fails the input-dependent formatting assertion (`expected ... to contain 'Alpha'`) plus the highlighter, citation, missing-page and image/HTML assertions — 5 behavioural failures, not setup.
- **no-citations**: removing the plugin import and registration fails exactly the two citation assertions (`href="...right-page"` and `[9]`), proving the citation path is genuinely exercised.
- **wrong-page**: `pageStack[cursor]` → `pageStack[0]` fails only the cursor-selection assertion, proving the test distinguishes correct cursor indexing from a fixed page.
- All three compiled cleanly (typecheck exit 0) before failing, so each negative is a behavioural detection, not a compilation/setup artifact. Raw stderr for the mutants contains only the two known preserved warnings (stale Browserslist, missing rehype-harden sourcemap) — no load or collection errors.

## Type-only / SSR / browser boundary

The evidence proves real Node SSR rendering through real Streamdown, Shiki and remark, with the corrected strong-tag assertion matching the installed library. It does **not** prove browser layout, CSS appearance, packaged-app launch, or live services. The type-only emission record is byte-identical-JS evidence only.

## Unresolved issues (preserved, not waived)

- Full lint still fails at 122 errors / 11 warnings — not a clean whole-programme gate.
- Stale Browserslist database and missing dependency sourcemap warnings; separate Go linker warning; Keychain/root/package acceptance untouched.
- No model-weight immutability or independent-hardware claim.
- Whole-programme integration remains blocked.

All required arms were collected, completed, and behaved as intended; typecheck passed; no skips; source unchanged across every arm.

FRONTEND_RUNTIME_REVIEW=PASS