#!/usr/bin/env python3
"""Deliberate-break matrix for the round-2 repairs of the OQ-3 instrument (revision 3).

Each mutation reverts or disables one repair in a scratch copy of the instrument, compiles it into
package cmd through its own overlay, runs the controls, and records whether they failed and how.
A mutation marked expect=survive documents defence in depth: another layer covers it.
Nothing in the worktree is edited; go.mod's hash is checked before and after every run.
"""
import hashlib, json, os, shutil, subprocess, sys, time

REPO = "/Users/krypto/GitHub/ollama-eko-chat-read-integrity"
HERE = REPO + "/docs/analysis/colleague_integration/proof_cont14/oq3-instrument"
OUT = sys.argv[1]
os.makedirs(OUT, exist_ok=True)

M = [
    ("M01-leak-scan-raw-bytes", "oq3_runner_test.go",
     "\t\tvar v any\n\t\tif json.Unmarshal(c.Body, &v) == nil {\n\t\t\twalk(v, 0)\n\t\t} else {\n\t\t\tout = append(out, oq3Leaks(string(c.Body))...)\n\t\t}\n",
     "\t\tout = append(out, oq3Leaks(string(c.Body))...)\n\t\t_ = walk\n",
     "RunnerControls", "fail", "leak-escaped"),
    ("M02-no-header-scan", "oq3_runner_test.go",
     "\tfor _, h := range c.HeaderLeaks {\n\t\tout = append(out, \"header \"+h)\n\t}\n",
     "\t_ = c.HeaderLeaks\n",
     "RunnerControls", "fail", "leak-in-header"),
    ("M03-no-query-scan", "oq3_runner_test.go",
     "\tif c.Query != \"\" {\n",
     "\tif false && c.Query != \"\" {\n",
     "RunnerControls", "fail", "leak-in-query"),
    ("M04-fake-serves-any-method", "oq3_harness_test.go",
     "known, allowed := oq3MethodAllowed(r.URL.Path, r.Method); known && !allowed {",
     "known, allowed := oq3MethodAllowed(r.URL.Path, r.Method); false && known && !allowed {",
     "RunnerControls", "fail", "405"),
    ("M05-scorer-ignores-method", "oq3_runner_test.go",
     "\t\t\twrongMethod := c.Method != http.MethodPost\n",
     "\t\t\twrongMethod := false && c.Method != http.MethodPost\n",
     "RunnerControls", "fail", "PUT replay"),
    ("M06-probe-always-idle", "oq3_harness_test.go",
     "func (p *oq3Terminal) idle(timeout time.Duration) bool {\n",
     "func (p *oq3Terminal) idle(timeout time.Duration) bool {\n\tif true {\n\t\treturn !p.hasExited()\n\t}\n",
     "RunnerControls", "fail", "idle control"),
    ("M07-runner-trusts-quiet", "oq3_runner_test.go",
     "\tr.mustIdle(\"after the turn completed\")\n",
     "\tr.mustQuiet(\"after the turn completed\")\n",
     "RunnerControls", "fail", "treated as ended"),
    ("M08-no-startup-refusal-path", "oq3_runner_test.go",
     "\t\tif r.sc.NoCapability {\n\t\t\tr.terminalStartWithoutCapability(i, st, preloaded)\n",
     "\t\tif false && r.sc.NoCapability {\n\t\t\tr.terminalStartWithoutCapability(i, st, preloaded)\n",
     "RunnerControls", "fail", "permitted refusal"),
    ("M09-exit-without-report-accepted", "oq3_runner_test.go",
     "\t\tif !oq3WaitFor(3*time.Second, reported) {\n",
     "\t\tif false && !oq3WaitFor(3*time.Second, reported) {\n",
     "RunnerControls", "fail", "passed as a refusal"),
    ("M10-t8-inserted-turn-expects-reload", "oq3_scenarios_test.go",
     "the refused reload must not have taken effect.\", cT(1, 1, \"A\")),",
     "the refused reload must not have taken effect.\", cT(2, 2, \"B\")),",
     "ScorerControls", "fail", "T8 control"),
    ("M11-probe-keeps-input", "oq3_harness_test.go",
     "\tfor time.Now().Before(deadline) {\n\t\tif !write(\"\\x15\") {\n\t\t\treturn false\n\t\t}\n",
     "\tfor time.Now().Before(deadline) {\n",
     "RunnerControls", "survive", ""),
    ("M12-typeline-keeps-input", "oq3_harness_test.go",
     "\tif _, err := p.master.Write([]byte(\"\\x15\")); err != nil {\n\t\tp.fail(\"pty write: %v\", err)\n\t}\n\ttime.Sleep(60 * time.Millisecond)\n\tif _, err := p.master.Write([]byte(text)); err != nil {\n",
     "\tif _, err := p.master.Write([]byte(text)); err != nil {\n",
     "RunnerControls", "survive", ""),
]

def gomod():
    return hashlib.sha256(open(REPO + "/go.mod", "rb").read()).hexdigest()

env = dict(os.environ, GOPROXY="off", GOTOOLCHAIN="local", GOFLAGS="")
results = []
only = set(sys.argv[2:])
for name, fname, old, new, test, expect, needle in M:
    if only and name not in only:
        continue
    d = os.path.join(OUT, name)
    shutil.rmtree(d, ignore_errors=True)
    os.makedirs(d)
    files = sorted(f for f in os.listdir(HERE) if f.startswith("oq3_") and f.endswith("_test.go"))
    for f in files:
        shutil.copy2(os.path.join(HERE, f), os.path.join(d, f))
    src = open(os.path.join(d, fname)).read()
    n = src.count(old)
    if n != 1:
        results.append({"mutation": name, "error": f"target occurs {n} times in {fname}"})
        print(name, "TARGET COUNT", n, flush=True)
        continue
    open(os.path.join(d, fname), "w").write(src.replace(old, new, 1))
    overlay = os.path.join(d, "overlay.json")
    json.dump({"Replace": {f"{REPO}/cmd/zz_{f}": os.path.join(d, f) for f in files}}, open(overlay, "w"), indent=1)
    before = gomod()
    t0 = time.time()
    p = subprocess.run(["go", "test", "-count=1", "-overlay", overlay, "-run", f"^TestOQ3{test}$", "-v", "-timeout", "900s", "./cmd"],
                       cwd=REPO, env=env, capture_output=True, text=True)
    after = gomod()
    log = p.stdout + p.stderr
    open(os.path.join(d, "test.log"), "w").write(log)
    lines = log.splitlines()
    fatal_line = ""
    for idx, l in enumerate(lines):
        if l.startswith("--- FAIL"):
            for back in range(idx - 1, -1, -1):
                if "zz_oq3_" in lines[back]:
                    fatal_line = lines[back].strip()
                    break
            break
    fatal = [fatal_line] if needle and needle in fatal_line else []
    compiled = "build failed" not in log and "[setup failed]" not in log
    detected = p.returncode != 0 and compiled
    ok = (expect == "fail" and detected and bool(fatal)) or (expect == "survive" and p.returncode == 0)
    r = {"mutation": name, "file": fname, "test": test, "expect": expect, "go_test_exit": p.returncode, "compiled": compiled,
         "expected_message_seen": bool(fatal), "verdict": "AS-EXPECTED" if ok else "UNEXPECTED",
         "fatal_line": fatal_line[:600],
         "seconds": round(time.time() - t0, 1), "go_mod_unchanged": before == after}
    results.append(r)
    print(name, r["verdict"], "exit", p.returncode, "msg", bool(fatal), f"{r['seconds']}s", "gomod-ok" if before == after else "GOMOD-CHANGED", flush=True)
    if before != after:
        sys.exit("go.mod changed during a mutation run; stopping")
json.dump(results, open(os.path.join(OUT, "matrix.json"), "w"), indent=1)
bad = [r for r in results if r.get("verdict") != "AS-EXPECTED"]
print("MATRIX", "ALL AS EXPECTED" if not bad else f"{len(bad)} UNEXPECTED", f"({len(results)} mutations)")
sys.exit(1 if bad else 0)
