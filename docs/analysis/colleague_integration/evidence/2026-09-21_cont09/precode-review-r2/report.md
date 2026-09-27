## Verdict

**REVISE**

The revised contract is materially stronger than the prototype and closes the original public-type-inference objection. The remaining issues are finite and design-level, not a demand for a different architecture.

## Justified findings

**F1 — `--check` and normal generation are not yet specified as sharing one verified pipeline.**
R2 says `--check` regenerates, transforms, runs the emission guard, and byte-compares. Normal generation writes after "all upstream/transform/emission checks." That is close, but the contract does not state that both paths invoke the same internal function with the same inputs and the same failure ordering. Without that, `--check` can pass while normal generation takes a different path, or vice versa. This is exactly the drift class the contract is trying to prevent.

Required refinement: state that `--check` and normal generation are two modes of one pipeline function; the only difference is the final action (compare vs. atomic replace). The pipeline must be pure with respect to the target until the final action.

**F2 — The emission-equivalence claim is scoped to `Program.emit`, but the contract does not state what happens when `Program.emit` itself fails or emits diagnostics.**
G-02 requires byte-identical JavaScript. R2 says emission comparison uses `Program.emit` with parsed `tsconfig.app.json` options. It does not say that emit diagnostics are treated as failure, nor that a non-zero emit result aborts before the atomic replace. A compiler that emits partial output on error would violate G-04's "leave the existing output unchanged."

Required refinement: `Program.emit` must be treated as a checked step. Any emit diagnostic, non-zero emit result, or missing output file is a generation failure that preserves the target and returns non-zero.

**F3 — The `--output` flag is under-constrained relative to the atomic-replace and no-write guarantees.**
R2 introduces `--output` "primarily to make the actual CLI testable without touching the committed file." G-04 says failures leave "the existing output unchanged" and "never truncate the target before verification." With `--output`, "the target" is ambiguous: is it the committed file, the `--output` path, or both? A test that points `--output` at a scratch path must not be able to accidentally satisfy the no-write guarantee by writing somewhere else.

Required refinement: define `--output` as the sole write target for that invocation, with the same atomic-replace and no-write-on-failure rules applied to it. State that `--output` does not relax any verification step and that the committed target is untouched when `--output` is used.

**F4 — The public-`any` refusal is stated for public properties, but the contract does not state the refusal boundary for the JSONValue alias declaration itself.**
R2 says the adapter "provides the JSONValue alias once, rejecting a declaration/name collision." It also says the adapter "rejects a public AnyKeyword, including one in a generic or ts_type override." What is not stated is whether the adapter rejects an `any` that appears inside the JSONValue alias definition, or inside a non-public helper type that the alias depends on. If the alias is emitted by the adapter, its own body must be fixed and not derived from a public `any`. If it is derived from Go metadata, the metadata must not be able to produce `any` inside the alias.

Required refinement: state that the JSONValue alias body is a fixed adapter-owned declaration, not derived from any public Go type, and that the adapter rejects any attempt by Go metadata to redefine or shadow it. This is a one-sentence closure, not a new mechanism.

**F5 — The T-05 fault-copy permission is clear, but the contract does not state that fault copies are excluded from the accepted production pipeline's byte-identity invariant.**
R2 correctly permits T-05 edits in isolated fault copies and correctly says the no-hand-patching rule governs accepted production output. The remaining ambiguity is whether the emission-equivalence check (G-02) is expected to hold for fault copies. It must not: a fault copy is deliberately not byte-identical to raw upstream emission. If a reader applies G-02 to fault copies, the contract becomes internally inconsistent.

Required refinement: state explicitly that G-02's byte-identity invariant applies to the accepted production pipeline only, and that T-05 fault copies are exempt by construction and are never candidates for the committed target.

**F6 — The "no new findings elsewhere" lint requirement in G-03 is not paired with a defined comparison baseline.**
G-03 says "lint has zero findings in the generated target and no new findings elsewhere." G-07 says "full lint comparison." The contract does not state what the baseline is: the pre-change lint output, the committed baseline, or a recorded snapshot. Without a defined baseline, "no new findings elsewhere" is not reproducible.

Required refinement: name the baseline artifact (e.g., the pre-change lint output captured under the task directory) and state that the comparison is against that artifact, not against an unrecorded prior state.

**F7 — The contract does not state whether the Go metadata additions (`ts_type` tags) are themselves covered by a determinism check.**
G-01 requires running normal generation twice produces identical generated bytes. R2 adds `ts_type` metadata to two Go source files. If the metadata is read by the generator, then determinism of the generated output depends on determinism of the metadata extraction. The contract does not state that the metadata extraction is deterministic or that it is covered by the twice-run check. This is likely already true, but it is not stated.

Required refinement: state that the twice-run determinism check covers the full pipeline including Go metadata extraction, and that the metadata is read from the owning Go source, not from a separately maintained list.

## What is not a finding

- The rejection of public-field inference in favor of Go-owned `ts_type` metadata is sound and closes the original objection.
- The AST-bounded edit model with overlap guards and explicit supported grammar is a reasonable replacement for text searches.
- The separation of `test:codegen` from default Vitest is correct and does not need to be part of ordinary frontend tests.
- The T-05 fault-copy permission is consistent with the attached prompt and does not require forbidding an isolated fault in a generated file.
- The explicit unchecked-boundary statement and the refusal to claim input-validation safety are correct and should be preserved verbatim.

## Finite exit condition for this review

The contract can be approved once F1–F7 are addressed as stated refinements. No new architecture, no new dependency, and no expansion of scope is required. The revisions are clarifications of the existing design, not changes to it.