# Persistent instructions for the Ollama colleague workspace

Status: [CONCEPT]. Owner: Eko. Date: 21 September 2026, Australia/Brisbane.
Commission: docs/OLLAMA_COLLEAGUE_INTEGRATION_WORK_LIST.md, OI-03/OI-04 and sections 0.6–0.7; the canonical commissioning copy is in the primary Ollama checkout.

## Intended observable behaviour

Ash can edit the colleague's standing instructions, see their scope, save them, restart the application, and have a newly opened conversation actually receive the saved instructions. Desktop and terminal use the same saved choice. An ongoing conversation keeps the instructions it started with until Ash explicitly applies a saved change at a safe point. Disabling or resetting the instructions is reversible. Ordinary chat remains available without a configured colleague.

Success is not a saved text box: the saved value reaches the real model request after restart, without losing the conversation, replacing model-specific requirements, multiplying instructions after compaction, or changing an active conversation behind Ash's back. Failure is visible when the selected configuration cannot be recovered or cannot fit; another identity is never substituted silently.

## Negative space

This is not founding the recurring colleague, proving LETHE memory, loading private autobiography, replacing the context compactor, changing inference weights, removing tool approvals, implementing autonomous goals, or transplanting Codex. It does not make Eko the new recurring Ollama colleague. It does not promise that every model obeys every instruction. The model receives the configured instructions; behavioural qualification remains separately attributable to the tested model and harness.

## Decisions assigned to the blueprint

Resolve one settings owner shared by desktop and terminal; revision and conversation identity; safe reload and concurrent work; exact request composition and model-template preservation; supported-context admission and honest budget diagnostics; inactive or unresolved colleague bindings; compatibility with existing conversations; reversible disable/reset; input validation; actual UI and terminal acceptance; failure controls and the division into implementation packs. These are delegated engineering decisions, not questions awaiting another approval from Ash.
