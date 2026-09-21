import ts from "typescript";
import fs from "node:fs";
import path from "node:path";
import { createHash } from "node:crypto";
import { jsonType } from "./annotate.mjs";

export const sha256 = value => createHash("sha256").update(value).digest("hex");
export function compareEmission(raw, refined, root) {
  const configPath = path.join(root, "tsconfig.app.json");
  const configBytes = fs.readFileSync(configPath, "utf8");
  const config = ts.parseConfigFileTextToJson(configPath, configBytes);
  if (config.error) throw new Error(ts.flattenDiagnosticMessageText(config.error.messageText, "\n"));
  const parsed = ts.parseJsonConfigFileContent(config.config, ts.sys, root);
  if (parsed.errors.length) throw new Error(ts.formatDiagnosticsWithColorAndContext(parsed.errors, {
    getCurrentDirectory: () => root, getCanonicalFileName: p => p, getNewLine: () => "\n",
  }));
  const overrides = { noEmit: false, allowImportingTsExtensions: false, removeComments: true,
    sourceMap: false, inlineSourceMap: false };
  const options = { ...parsed.options, ...overrides };
  const name = path.join(root, "codegen/gotypes.gen.ts");
  function emit(text) {
    const host = ts.createCompilerHost(options), original = host.readFile.bind(host);
    host.readFile = p => path.resolve(p) === name ? text : original(p);
    const program = ts.createProgram([name], options, host);
    const source = program.getSourceFile(name);
    if (!source || source.statements.some(s => ts.isImportDeclaration(s) || ts.isImportEqualsDeclaration(s) ||
      (ts.isExportDeclaration(s) && s.moduleSpecifier))) throw new Error("Generated module must be import-free");
    const diagnostics = ts.getPreEmitDiagnostics(program);
    const outputs = [];
    const result = program.emit(source, (p, value) => {
      if (p.endsWith(".js")) outputs.push(value);
    });
    if (diagnostics.length || result.diagnostics.length || result.emitSkipped || outputs.length !== 1 || !outputs[0])
      throw new Error("Generated module emission failed: " + ts.formatDiagnostics([...diagnostics, ...result.diagnostics], {
        getCurrentDirectory: () => root, getCanonicalFileName: p => p, getNewLine: () => "\n",
      }));
    return outputs[0];
  }
  const before = emit(jsonType + raw), after = emit(refined);
  if (before !== after) throw new Error("Generated JavaScript changed after annotation");
  return { compiler: ts.version, config_sha256: sha256(configBytes), options, overrides,
    javascript_sha256: sha256(before), javascript_bytes: Buffer.byteLength(before), identical: true };
}
