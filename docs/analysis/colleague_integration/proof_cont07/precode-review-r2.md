**APPROVE**

The structural-typing objection is correctly rejected. TypeScript object types are structural: `{ page_stack: string[] }` declared independently in `StreamingMarkdownContent.tsx` and derived via `React.ComponentProps<typeof Message>["browserToolResult"]` in `MessageList.tsx` are mutually assignable without a shared nominal export. My earlier "shared exported type" demand was wrong; no import cycle or duplicate component is needed. The plan's narrower choice — StreamingMarkdownContent declares only the field it consumes — is the better boundary.

Point 1 is now falsifiable: identical `clang++` invocation, only `-o` differs, three-space delta, `-fsyntax-only` control with `-Werror=deprecated-literal-operator`, object equality measured not presumed. Good.

Point 3 is the right disposition. The pre-existing `cursor: string` (parser) vs `cursor: number` (renderer annotation) mismatch is real, but the plan correctly refuses to fix it inside a type-only slice and correctly refuses to silently coerce. The instruction "if type checking demands an algorithm/value change, stop and revise the boundary" is the correct stop condition. One residual risk to name: if mdast augmentation causes the renderer's `cursor: number` annotation to be *checked* against the node's `hProperties.cursor: string`, the slice will fail typecheck and must halt — that is the intended outcome, not a defect.

Point 4 keeps the production surface to four files plus additive tests. Generated types, Message.tsx, runtime IO untouched.

No tests or implementation are claimed here. Whole-programme proof remains separately blocked.