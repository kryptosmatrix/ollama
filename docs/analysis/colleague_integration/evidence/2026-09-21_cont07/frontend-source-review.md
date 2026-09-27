SOURCE_REVIEW=REVISE

Static-only: yes. The diff is type/annotation-only; the emission hashes in `typescript-emission-final.stdout` are byte-identical for all four files, and no runtime statement, branch, value, or IO changed. I did not run anything; the platform block on `eko-cont07-frontend-baseline` stands, and the eleven regression tests remain unexecuted candidates.

HighlighterCore: reasonable. `createHighlighter({themes:[oneLightTheme,oneDarkTheme], langs:[...]})` returns a generic highlighter whose theme/lang key unions are the bundled names, not `"one-light"`/`"one-dark"`. `HighlighterCore = HighlighterGeneric<never,never>` is the correct widened instance type for pre-registered custom themes; `codeToTokensBase(code,{theme:"one-light"})` then accepts the string. The `highlighterPromise` return type is unchanged, and `highlighter = h` still assigns. No runtime import added. Fine.

mdast registration: does NOT match the runtime nodes. Concrete defects:

1. `CitationNode.data.hProperties.cursor` is typed `string`, but the runtime node in `remarkCitationParser.ts` is created with `cursor: match[1]` (string) — OK — yet the consumer in `StreamingMarkdownContent.tsx` line 240–246 declares `cursor: number` and compares `cursor < pageStack.length`. The registry augmentation does not fix that mismatch; the `@ts-expect-error` on line 239 is still present and is now masking a real type disagreement rather than a missing registry entry. Either the consumer prop type must be `string` (and converted) or the node must carry a number; the current diff leaves both.

2. `PhrasingContentMap`/`RootContentMap` keys must be the *node type string* used as the discriminant. mdast's convention is that the map key equals `type`. The augmentation uses key `customCitation` while the node's `type` is `"custom-citation"`. That is not how mdast's `RootContentMap`/`PhrasingContentMap` are keyed (see `mdast` `RootContentMap` entries like `paragraph`, `heading`). As written, `node.type === "custom-citation"` will not narrow through the registry, so the removed `@ts-expect-error` directives on lines 44 and 71 are only "passing" because `pieces: RootContent[]` accepts the object structurally — not because the registry narrowed. The augmentation is therefore decorative, not load-bearing. Fix: key both maps by `"custom-citation"` (quoted key) and ensure `CitationNode.type` matches.

3. `CitationNode extends Node` from `unist` while augmenting `mdast` maps: mdast's `RootContent`/`PhrasingContent` are `mdast`-flavored nodes. Extending `unist.Node` is acceptable structurally, but the `data` field on mdast nodes is `Data | undefined`; declaring `data` as required narrows the map entry in a way that will not unify with `RootContent` in `pieces.push`. Prefer `data?: {...}` or use `mdast`'s `Data` extension.

4. `hProperties: { cursor: string; start?: string; end?: string }` — the generic-citation branch (line 70–78) omits `start`/`end`, which is fine, but the range branch always sets all three. The optionality is correct; no change needed there.

Concrete test mistakes (unexecuted, but statically wrong):

- `remarkCitationParser.test.ts` line 45: `parse("【1†L1-L2】【1†L3-L4】【2†source】")` — the first pass runs the range regex over the whole text node, then the generic regex over `remaining = node.value.slice(last)`. After the two range matches, `last` is at the end of `【1†L3-L4】`, so `remaining` is `【2†source】` and the generic pass emits cursor `"2"`. The coalescing pass then sees `[1,1,2]` and removes the second `1`. Expected `[1,2]` is correct. OK.

- Line 53: `paragraph && "children" in paragraph && paragraph.children` — `paragraph` is `RootContent`; `"children" in paragraph` is a valid narrowing, but `toHaveLength(2)` on the union is fine. Not a mistake, just verbose.

- `StreamingMarkdownContent.integration.test.tsx` line 36: `expect(first).toContain("color:")` — `renderToStaticMarkup` emits inline `style="color:#..."`, not `color:`. This assertion will fail. Should be `toContain("color")` or `toContain("style=")`.

- Line 48: `expect(html).toContain('href="https://second.example/right-page"')` — the citation component renders `<a href={pageUrl}>`; `renderToStaticMarkup` will emit the href, but the surrounding `[1]` text is inside the anchor. The `not.toContain` on line 49 is fine. However line 50 `toContain("[1]")` — the component renders `[{cursor}]` where `cursor` is the *string* `"1"` from `hProperties`, so `[1]` is correct. OK.

- Line 56: `expect(html).not.toContain("href=")` — the page has no other links, but `Streamdown` may emit `href` for other reasons; this is fragile but not wrong per se.

- Line 63–67: `not.toContain("<img")` — the `img` component returns `<span>{alt}</span>`, so no `<img>` is emitted. OK. But `not.toContain('src="https://image.example')` is redundant given the previous assertion; harmless.

- `beforeAll` awaits `highlighterPromise` but `highlighter` is a module-level `let` assigned in `.then`; the test reads `highlighter` synchronously inside `CodeBlock`'s `useMemo`. Since `beforeAll` awaits the promise, the assignment has run. OK.

Required fixes before approval: (a) key the mdast maps by `"custom-citation"`; (b) reconcile `cursor` string vs number between node and consumer, and remove the now-misleading `@ts-expect-error` on line 239 only if the consumer type is corrected; (c) fix the `color:` assertion. Runtime tests and independent executable review remain outstanding; no GATE=PASS.