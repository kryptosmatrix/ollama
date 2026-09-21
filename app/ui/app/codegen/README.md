# Generated desktop models

Go owns the public field types in `app/store` and `app/ui/responses`. The existing
`typescriptify-golang-structs` dependency is pinned at v0.2.0; `generate.mjs` runs
its real Go model discovery, then applies erased annotations with the locked
TypeScript compiler. Classes remain classes because the desktop API and streaming
consumers call their constructors.

From `app/ui/app`, run:

```sh
npm run generate:types
npm run check:types
npm run test:codegen
```

The ordinary `go generate ./app/ui` route invokes the same generator before the
existing frontend build. Go and the installed frontend dependencies are required
for generation. Ordinary `npm test -- --run` does not require Go.

`--check` and write mode use the same fresh Go generation, annotation and compiler
verification function. They differ only at the final operation: compare, or atomic
replacement. `--output FILE` selects the sole target for that invocation, with the
same verification. Failed generation, unsupported templates and stale checks leave
existing target bytes untouched. The generator rejects symlink/non-regular targets
and inputs over 1 MiB. Temporary files are private to an invocation and removed
on normal exit. Abrupt process termination can leave disposable temporary files;
it is not a claim of crash-cleanup or power-loss durability.

## Type and compatibility boundary

The upstream materialisers are unchecked. Changing their parameters to `unknown`
does not turn them into JSON validators. The erased casts retain their prior
runtime behaviour, including missing/null values, malformed-JSON exceptions,
mutable map conversion, and existing time/raw-byte representation limitations.
This change does not correct those separate representation contracts.

For Go `any` fields that cross JSON, supply an explicit `ts_type:"JSONValue"` on
the owning field; for `map[string]any`, use `ts_type:"{[key: string]: JSONValue}"`.
The adapter never guesses a public property type. It rejects remaining public
`any` syntax and a collision with its fixed recursive `JSONValue` alias.

The adapter supports the pinned generator's ordinary exported classes, field
assignments, nested conversions and declared Date transforms. It rejects other
runtime shapes. The compiler guard then compares the raw and annotated module's
emitted JavaScript byte for byte. Both arms use parsed `tsconfig.app.json` options,
with `noEmit:false` and `allowImportingTsExtensions:false` solely to emit this
import-free module; comments and source maps are excluded equally. Parse/type/emit
diagnostics or a missing output fail the operation. Project configuration is not
changed. This is module-level compiler equivalence, not a browser or installed-app
acceptance claim. Receipts retain the compiler/options/source/output identities
and complete successful upstream subprocess output; unsuccessful child output is
included in the error and the child failure reaches the CLI exit status.

## Required candidate checks

For a change to this pipeline or its owning Go metadata, execute all of:

```sh
npm run check:types
npm run test:codegen
npx --no-install tsc -b --force
npm test -- --run
npm run lint
npm run build
```

The code-generation checks are required in addition to the default frontend suite;
they are not claimed to be installed in a new CI job. The existing frontend suite
includes `src/utils/generatedModels.test.tsx`: real generated models and Markdown
renderer, simulated HTTP transport, not browser interaction. Record each command's
true exit status, warnings and all discovered/completed tests. An inherited lint
or dependency warning is not a clean whole-application result.

For TECHNE acceptance, retain isolated compiled faults that bypass annotation,
discard nested chat messages, and replace message content with a constant. The
unchanged relevant consumer assertions must detect each fault after a passing clean
arm, followed by a clean rerun. These deliberately broken copies are excluded from
the accepted pipeline's emission invariant and never replace the committed target.
The source/tests/configuration/dependencies and exact candidate must remain bound
to those runs. Independent reviewer-directed reproduction is separate from these
local instructions and the implementer's self-audit.
