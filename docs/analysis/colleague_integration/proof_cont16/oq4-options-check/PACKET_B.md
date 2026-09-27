# Options check B: composition, scope and test placement for G1 (Ollama fork)

**What you are asked to do.** For each question below, choose among the options or propose another, and say why, citing file:line for every claim you rest on. Say what evidence would change your choice and what risk your choice carries. Where you think a question is not an engineering choice at all (for example, it decides what the product does for its user rather than how), say so and why. You are advising on design; you are not reviewing code for defects. The options are listed in no order of preference.

## Context

G1 (`docs/_design/G1_PERSISTED_INSTRUCTIONS.md`) is a design at `[SPEC-DRAFT]`, not implemented: the operator's saved instructions (the "carrier") are composed into the system message of every new conversation in two adapters, the desktop app (`app/ui`) and the terminal agent chat (`cmd/agent_tui.go`, `cmd/tui/chat/`), by one pure function `Compose(base, selection)` (§5, line 200): disabled returns the base byte for byte; enabled returns the base, `\n\n` when the base is non-empty, the header `User-configured instructions; revision <revision>; selection <generation>:\n`, and the profile text. G1's failure number (line 18) is measured by an instrument (`docs/analysis/colleague_integration/proof_cont14/oq3-instrument/`, design in `DESIGN.md`), whose "Blueprint gaps this instrument exposed" (`DESIGN.md:58-64`) this packet partly covers (gaps 1, 5 and 4). Two further questions arose from reading the source for G1's closure census (OQ-4, blueprint line 35).

The concept (`docs/_design/G1_PERSISTENT_INSTRUCTIONS_CONCEPT.md`) says "Desktop and terminal use the same saved choice." The commission, which is not in this repository, says of OI-03 "Add or complete the terminal-instructions editor" and of OI-04: "Trace and extend the existing `SystemPrompt` path rather than introduce a parallel prompt assembly. Preserve model-template requirements, tool schemas and protocol messages", complete when "request-boundary capture proves the exact effective carrier reaches the selected model through the real terminal entry point".

## Questions

**B-1. How are the desktop's model-system text and MCP instructions joined into the base?** (Gap 1.) §5 (line 204): when enabled, "model-system text and existing MCP instructions form the base in that order, omitting only empty blocks; the shared composer appends the configured text. The builder adds one effective system message before the conversation"; disabled, "its old path remains exactly as before, including the current MCP instruction ordering". Today the builder sends MCP instructions as their own leading system message when `strings.TrimSpace(mcpInstructions) != ""` (`app/ui/ui.go:1909-1916`), and the daemon prepends the model's own system text only when the first request message is not a system message (`server/routes.go:2522-2524`, `:2634-2636`). The join between the two blocks, and what counts as empty, are unstated.
1. Keep MCP instructions as a separate system message after the effective one (two system messages), amending §5's "one effective system message".
2. Joined by a single `\n`; a block is empty when `strings.TrimSpace` of it is empty; non-empty blocks are otherwise byte-exact.
3. Each block labelled (for example a heading line per block) before joining.
4. Non-empty blocks joined by `\n\n`, the composer's own separator, with the same emptiness rule as 2.

**B-2. Does the compaction summariser carry the carrier?** (Gap 5.) The terminal's summariser sends its own fixed system prompt and the archived text (`agent/compactor.go:31`, `:230-251`); the conversation's effective system prompt reaches the compactor only for estimating the next request's size (`cmd/tui/chat/compaction.go:45`; `agent/compactor.go:383-388`). §4's compaction row (line 190): "Reattach carrier from selection, not from summarised text; do not store duplicate system messages." D3 (line 30): "Instruction bindings are metadata, never a second transcript." The instrument today expects a summariser request with no header or profile text (`DESIGN.md:33`).
1. Always: the summariser receives the carrier so the summary follows the operator's instructions.
2. Configurable per profile.
3. Never: the summariser request carries no carrier and no profile text; the carrier returns on the next request from the selection.

**B-3. Where does the failure-number instrument live in G1's final test suite?** (Gap 4.) The blueprint designates `cmd/tui/chat/instructions_test.go` (line 360), which sits below `GenerateAgentTUI`, where G1 creates the terminal's selection (§5, line 202; line 18). The instrument runs today as a `go test -overlay` into package `cmd`, driven by `run.sh`, with its own goldens (`DESIGN.md:3`, `:66-69`).
1. It stays an overlay under `docs/analysis/`, and `run.sh accept` is the gate's command; nothing is added to the default suite.
2. A separate test package behind a build tag or environment variable, invoked explicitly by the combined proof set.
3. A committed test in package `cmd` (a named file), run by the combined proof set with a stated command; `cmd/tui/chat/instructions_test.go` keeps the chat-model-level tests.

**B-4. Does G1 cover the terminal's classic `ollama run` paths?** The bare `ollama` command opens a launcher whose run-model action starts the agent chat (`cmd/cmd.go:2162-2188`, `:2301`, `:2130-2158`), the path G1 names. `ollama run <model>` is a separate production path: interactive, it enters the classic chat, `generateInteractive` (`cmd/cmd.go:899-925`, `cmd/interactive.go:34`), whose turns go through `chat` (`cmd/cmd.go:1758`, request at `:1840-1852`) with its own `/set system` command (`cmd/interactive.go:59`, `:183-184`, `:281-282`); with a prompt argument it sends one `/api/generate` request with `System: opts.System` (`cmd/cmd.go:1972-1984`). None of G1's records (the blueprint, the concept, the ADR `docs/_design/ADR_2026-09-26_G1_INSTRUCTION_ADMISSION.md`) mentions either path.
1. Include both.
2. Include the classic interactive chat (it composes the carrier, carries the admission contract and refuses as the agent chat does); exclude the one-shot prompt.
3. Exclude both, and have the classic interactive chat show a one-line notice when the saved profile is enabled that saved instructions do not apply there.
4. Exclude both, stated in the blueprint's scope and negative space with the operator-visible consequence (saved instructions do not apply to `ollama run`).

**B-5. What do the terminal's `/prompt` and `/save` show and write?** `/prompt` displays the request the chat would send; `/save <name>` writes it as JSON to a file in the working directory with mode 0644 (`cmd/tui/chat/input.go:65-66`; `cmd/tui/chat/debug.go:28-49`, `:51-64`, `:160-186`). Both build it from the effective system prompt (`m.systemPrompt("")`, `debug.go:181`) through `previewChatRequest` (`:196-222`), which prepends the system prompt and sets model, format, options, think, keep-alive and tools, but no admission field. Under G1 the effective system prompt includes the profile text. §5 (line 208) says actual-request diagnostics are recorded "without storing private instruction text in public logs or repository evidence".
1. Both omit the profile text (show the base and a placeholder naming revision and generation), so that `/save` cannot write private instruction text into a working directory.
2. Both show and write the composed carrier, because they read the effective system prompt, but no admission field; the view is labelled a preview, not proof of dispatch.
3. Both show and write the request exactly as it would be sent, including the carrier and the admission field (one construction path for preview and dispatch).
4. As 3, and `/save` warns when the file it writes contains saved instructions.

## For every answer

Choose, give reasons with citations, name what would change your choice, and name the risk. If two questions constrain each other, say how.
