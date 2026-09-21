import assert from "node:assert/strict";
import test from "node:test";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";
import ts from "typescript";
import { annotate, jsonType } from "./annotate.mjs";
import { compareEmission, sha256 } from "./emission.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const raw = `export class Example {
 value: string;
 constructor(source: any = {}) {
  if ('string' === typeof source) source = JSON.parse(source);
  this.value = source["value"];
 }
}`;
const refine = () => annotate(raw).text;

test("annotation is deterministic and contains no explicit-any", () => {
  const first = refine();
  assert.equal(first, refine());
  assert.match(first, /constructor\(source: unknown = \{\}\)/);
  assert.match(first, /value: string;/);
  const source = ts.createSourceFile("models.ts", first, ts.ScriptTarget.Latest, true);
  function visit(node) {
    assert.notEqual(node.kind, ts.SyntaxKind.AnyKeyword);
    ts.forEachChild(node, visit);
  }
  visit(source);
});
test("unknown public types require their owning Go metadata", () => {
  for (const type of ["any", "Array<any>", "{ [key: string]: any }"])
    assert.throws(() => annotate(raw.replace("value: string;", `value: ${type};`)), /Go-owned ts_type/);
});
test("JSONValue metadata is retained, not inferred by the adapter", () => {
  const tagged = raw.replace("value: string;", "value: JSONValue;");
  assert.match(annotate(tagged).text, /value: JSONValue;/);
  assert.equal(annotate(tagged).text.split("export type JSONValue =").length, 2);
});
test("alias collision is refused", () => {
  assert.throws(() => annotate(raw.replace("class Example", "class JSONValue")), /name collision/);
  assert.throws(() => annotate("type JSONValue = string;\n" + raw), /only named classes/);
});
test("empty and syntactically invalid upstream output are refused", () => {
  assert.throws(() => annotate(""), /empty module/);
  assert.throws(() => annotate("export class {"), /parse failure/);
});
test("new runtime declarations and constructor statements are refused", () => {
  assert.throws(() => annotate("console.log('changed');\n" + raw), /only named classes/);
  assert.throws(() => annotate(raw.replace('this.value =', 'console.log(source); this.value =')), /field count/);
  assert.throws(() => annotate(raw.replace('this.value = source["value"]', 'this.value = String(source)')), /nested conversion/);
});
test("missing or duplicate constructor writes are refused", () => {
  assert.throws(() => annotate(raw.replace('this.value = source["value"];', "")), /field count/);
  assert.throws(() => annotate(raw.replace('this.value = source["value"];', 'this.other = source["value"];')), /unknown or repeated/);
});
test("upstream helper-template drift is refused", () => {
  assert.throws(() => annotate(raw.replace(/}\s*$/, "convertValues(a: any): any { return a; }\n}")), /convertValues template/);
});
test("new transform operators are refused rather than guessed", () => {
  assert.throws(() => annotate(raw.replace('source["value"];', 'source["value"] || new Date(source["value"]);')), /guard operator/);
});
test("computed public field names are refused", () => {
  assert.throws(() => annotate(raw.replace("value: string;", '["value"]: string;')), /public field/);
});
test("compiler guard proves the unchanged emitted JavaScript", () => {
  const result = compareEmission(raw, refine(), root);
  assert.equal(result.identical, true);
  assert.ok(result.javascript_bytes > 0);
  assert.equal(result.overrides.noEmit, false);
  assert.equal(result.overrides.allowImportingTsExtensions, false);
});
test("compiler guard rejects a runtime change that still compiles", () => {
  const broken = refine().replace('(source as Record<string, unknown>)["value"] as Example["value"]', '"constant"');
  assert.notEqual(broken, refine());
  assert.throws(() => compareEmission(raw, broken, root), /JavaScript changed/);
});
test("compiler guard rejects type errors rather than emitting partial output", () => {
  assert.throws(() => compareEmission(raw, refine().replace("value: string;", "value: MissingType;"), root), /emission failed/);
});
test("compiler guard refuses runtime imports", () => {
  assert.throws(() => compareEmission(raw, 'import "missing-module";\n' + refine(), root), /import-free/);
});
test("Go-owned JSON union changes no runtime emission", () => {
  const tagged = raw.replace("value: string;", "value: JSONValue;");
  assert.equal(compareEmission(tagged, annotate(tagged).text, root).javascript_sha256,
    compareEmission(raw, refine(), root).javascript_sha256);
  assert.match(jsonType, /null/);
});

test("real generation CLI is reproducible, detects stale output and preserves failures", { timeout: 240000 }, async t => {
  const scratch = fs.mkdtempSync(path.join(os.tmpdir(), "ollama-codegen-check-"));
  const target = path.join(scratch, "generated.ts");
  const committed = path.join(root, "codegen/gotypes.gen.ts");
  const original = fs.readFileSync(committed);
  const cli = (args, env = process.env) => {
    const argv = [path.join(root, "codegen/generate.mjs"), "--output", target, ...args];
    const result = spawnSync(process.execPath, argv, { cwd: root, env, encoding: "utf8", timeout: 180000, maxBuffer: 8 * 1024 * 1024 });
    console.log(JSON.stringify({ argv, exit_code: result.status, error: result.error?.message,
      stdout: result.stdout, stderr: result.stderr }));
    assert.equal(result.error, undefined);
    return result;
  };
  try {
    let first;
    await t.test("fresh Go generation twice produces identical bytes", () => {
      assert.equal(cli([]).status, 0);
      first = fs.readFileSync(target);
      assert.equal(cli([]).status, 0);
      assert.deepEqual(fs.readFileSync(target), first);
    });
    await t.test("check verifies the final transformed bytes without writing", () => {
      assert.equal(cli(["--check"]).status, 0);
      assert.deepEqual(fs.readFileSync(target), first);
    });
    await t.test("check rejects stale bytes without replacing them", () => {
      fs.appendFileSync(target, "\n// deliberate stale-file probe\n");
      const stale = fs.readFileSync(target);
      const result = cli(["--check"]);
      assert.equal(result.status, 1);
      assert.match(result.stderr, /types are stale/);
      assert.deepEqual(fs.readFileSync(target), stale);
    });
    await t.test("unavailable Go returns nonzero and preserves the existing target", () => {
      const stale = fs.readFileSync(target);
      const result = cli([], { ...process.env, PATH: scratch });
      assert.equal(result.status, 1);
      assert.match(result.stderr, /Go generation failed/);
      assert.deepEqual(fs.readFileSync(target), stale);
    });
    await t.test("malformed command options fail before touching output", () => {
      const before = fs.readFileSync(target);
      const result = cli(["--unsupported"]);
      assert.equal(result.status, 1);
      assert.match(result.stderr, /Usage:/);
      assert.deepEqual(fs.readFileSync(target), before);
    });
    await t.test("the alternative target never changes the committed output", () => {
      assert.equal(sha256(fs.readFileSync(committed)), sha256(original));
    });
  } finally {
    fs.rmSync(scratch, { recursive: true, force: true });
  }
});
