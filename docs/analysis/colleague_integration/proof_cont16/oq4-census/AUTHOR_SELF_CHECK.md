# G1 OQ-4 author self-check — 2026-10-01

Candidate source: `465b963e35f6afa46f15e6fdec043189a905d1d8`
Blueprint SHA-256 after OQ-4 amendment: `3476c85773ab3ecd6c7c9e5f70105f6a276cb9fb2f21b1d77c46c17b4094f51e`

Executed from `/Users/krypto/GitHub/ollama-eko-upstream-20261001`:

```text
G1_REQUIREMENT_ACCEPTANCE_BIJECTION=PASS requirements=18 acceptance=14
BLUEPRINT_BYTES=99555
MUST_LINES=18
ACCEPTANCE_ROWS=14
STALE_CHECKS
```

`git diff --check` exited 0 in the same command.

The active Method-16 R-3.2 ceiling is 111,880 bytes from the itemised 2,797-production-line estimate, so the 99,555-byte blueprint is within the ratio gate. The stale-string scan found none of:

- `once per connection`
- `field is defined only for /api/chat`
- the superseded OQ-4-open wording
- the superseded 2,181-line estimate

This is an author-side mechanical check only. It is not a Method-16 judge stamp and does not change `[SPEC-DRAFT]` / `PACKS_BLOCKED`.
