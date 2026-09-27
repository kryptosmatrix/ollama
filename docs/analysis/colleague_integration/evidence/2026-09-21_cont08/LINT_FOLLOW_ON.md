# Remaining lint: next implementation boundary

This is a reconnaissance note, not a promoted implementation blueprint or a lint-pass claim. Eko, 21 September 2026.

The current known remaining population is 104 generated-type errors, 18 handwritten errors and 11 warnings; the new continuation-08 lint run must supply the final count before closeout.

## Generated errors — do not replace classes with interfaces

The generator entry is app/ui/ui.go:42, `//go:generate tscriptify -package=github.com/ollama/ollama/app/ui/responses -target=./app/codegen/gotypes.gen.ts responses/types.go`. The module dependency is github.com/tkrajina/typescriptify-golang-structs v0.2.0 (go.mod:36). The ordinary shell lookup did not find tscriptify on PATH; that is not proof the binary or module source is absent elsewhere.

Generated classes are actual runtime dependencies, not just type declarations. A fresh Bridge search found `new Message(...)` in app/ui/app/src/hooks/useChats.ts at lines 272, 380, 396, 432, 446, 480, 515, 529, 564 and 704, and `new MessageType(...)` in components/Message.stories.tsx:25. Therefore changing generation to interface-only to eliminate constructor lint would remove a used runtime surface. Retain constructor/nested-conversion/date behaviour, inspect the pinned generator and its supported configuration, then repair generation reproducibly rather than hand-patching gotypes.gen.ts or suppressing ESLint.

The generated file also types Message.tool_result as number[] (gotypes.gen.ts:114); handwritten rendering reads structured fields under casts. Any correction to that wire-type contract needs the actual Go JSON producer and consumer trace, not an automatic `any` to `unknown` substitution.

## Handwritten and hook findings

From the retained continuation-07 lint output: remaining explicit-any sites are in Message.tsx (9), ChatForm.tsx (4), ThinkButton.tsx (1), ModelPicker.tsx (1) and useChats.ts (3). The 11 warnings concern Chat.tsx, ChatForm.tsx, ChatSidebar.tsx, SpeechSettings.tsx, ui/slider.tsx, contexts/StreamingContext.tsx and useSelectedModel.ts. Source must be freshly inspected before edits. Hook dependencies can change runtime subscription/cancellation behaviour; they are not merely annotation maintenance.

## Boundaries

Keep the history-preservation repair and completed frontend verification intact. Do not reopen design approval for already reviewed slices. No Keychain access or credential-fixture edits are required to investigate the above frontend changes. Main/package acceptance remains separate and incomplete; no installed feature is claimed by this note.
