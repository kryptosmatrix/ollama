import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { spawnSync } from "node:child_process";
import ts from "typescript";
import { annotate } from "./annotate.mjs";
import { compareEmission, sha256 } from "./emission.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const maximumBytes = 1024 * 1024;
function snapshot(target) {
  try {
    const stat = fs.lstatSync(target);
    if (!stat.isFile() || stat.isSymbolicLink() || stat.size > maximumBytes)
      throw new Error("Generation target must be a regular file of at most 1 MiB");
    return { hash: sha256(fs.readFileSync(target)), mode: stat.mode & 0o777 };
  } catch (error) {
    if (error.code === "ENOENT") return null;
    throw error;
  }
}

// Both modes verify the same fresh output; neither touches the target during verification.
function verifiedGeneration() {
  const lock = JSON.parse(fs.readFileSync(path.join(root, "package-lock.json"), "utf8"));
  if (lock.packages?.["node_modules/typescript"]?.version !== ts.version)
    throw new Error("Installed TypeScript differs from package-lock.json");
  const scratch = fs.mkdtempSync(path.join(os.tmpdir(), "ollama-gotypes-"));
  try {
    const ui = path.dirname(root);
    const env = { ...process.env, TMPDIR: scratch, TMP: scratch, TEMP: scratch };
    const upstreamRuns = [];
    const run = args => {
      const result = spawnSync("go", args, { cwd: ui, env, encoding: "utf8", timeout: 120000, maxBuffer: 4 * maximumBytes });
      if (result.error || result.status !== 0)
        throw new Error(`Go generation failed (exit ${result.status}): ${result.error?.message ?? ""}\n${result.stdout ?? ""}${result.stderr ?? ""}`);
      upstreamRuns.push({ argv: ["go", ...args], exit_code: result.status, stdout: result.stdout, stderr: result.stderr });
      if (result.stderr) process.stderr.write(result.stderr);
      return result.stdout;
    };
    const module = JSON.parse(run(["list", "-m", "-json", "github.com/tkrajina/typescriptify-golang-structs"]));
    if (module.Version !== "v0.2.0" || module.Replace)
      throw new Error("Unsupported typescriptify version or module replacement");
    const rawPath = path.join(scratch, "gotypes.raw.ts");
    run(["run", "github.com/tkrajina/typescriptify-golang-structs/tscriptify",
      "-package=github.com/ollama/ollama/app/ui/responses", `-target=${rawPath}`, "responses/types.go"]);
    if (fs.statSync(rawPath).size > maximumBytes) throw new Error("Generated source exceeds 1 MiB");
    const raw = fs.readFileSync(rawPath, "utf8");
    const result = annotate(raw);
    if (Buffer.byteLength(result.text) > maximumBytes) throw new Error("Annotated source exceeds 1 MiB");
    const emission = compareEmission(raw, result.text, root);
    return { ...result, raw_sha256: sha256(raw), emission, generator_version: module.Version, upstream_runs: upstreamRuns };
  } finally {
    // Only the private directory just created by this invocation is disposable.
    fs.rmSync(scratch, { recursive: true, force: true });
  }
}

export function generate({ check = false, output = path.join(root, "codegen/gotypes.gen.ts") } = {}) {
  const target = path.resolve(output), before = snapshot(target);
  const verified = verifiedGeneration();
  const after = snapshot(target);
  if (JSON.stringify(before) !== JSON.stringify(after)) throw new Error("Generation target changed concurrently");
  if (check) {
    if (after?.hash !== sha256(verified.text)) throw new Error("Generated types are stale; run npm run generate:types");
  } else {
    const stage = fs.mkdtempSync(path.join(path.dirname(target), ".gotypes-"));
    try {
      const temporary = path.join(stage, "output.ts");
      fs.writeFileSync(temporary, verified.text, { flag: "wx", mode: before?.mode ?? 0o644 });
      fs.renameSync(temporary, target);
    } finally {
      fs.rmSync(stage, { recursive: true, force: true });
    }
  }
  const { text, ...receipt } = verified;
  return { ...receipt, output: target, output_sha256: sha256(text), mode: check ? "check" : "write" };
}

function main(args) {
  let check = false, output;
  for (let i = 0; i < args.length; i++) {
    if (args[i] === "--check" && !check) check = true;
    else if (args[i] === "--output" && output === undefined && args[i + 1] && !args[i + 1].startsWith("--")) output = args[++i];
    else throw new Error("Usage: node codegen/generate.mjs [--check] [--output FILE]");
  }
  console.log(JSON.stringify(generate({ check, output }), null, 2));
}
if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  try { main(process.argv.slice(2)); }
  catch (error) { console.error(error.message); process.exitCode = 1; }
}
