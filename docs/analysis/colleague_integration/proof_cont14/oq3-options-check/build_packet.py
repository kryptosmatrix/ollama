#!/usr/bin/env python3
"""Build the OQ-3 instrument options packet from verbatim blueprint lines (no paraphrase of the contract).

The blueprint is read at the pinned commit through git, so the excerpt cannot drift with the working tree.
"""
import hashlib, os, subprocess

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = subprocess.check_output(["git", "-C", HERE, "rev-parse", "--show-toplevel"], text=True).strip()
COMMIT = "a7c35fd65e732cd378dc6d2e93ccc43f152efcef"
BP = "docs/_design/G1_PERSISTED_INSTRUCTIONS.md"
bp_bytes = subprocess.check_output(["git", "-C", REPO, "show", f"{COMMIT}:{BP}"])
assert hashlib.sha256(bp_bytes).hexdigest() == "209dedc67b1191f29bc757683a2a35a30a1c70ec0cbd912540ebe06acf629f45"
lines = bp_bytes.decode("utf-8").split("\n")

# (first, last) 1-based inclusive ranges, each quoted verbatim with its line numbers.
RANGES = [(18, 18), (35, 35), (79, 79), (175, 196), (198, 208), (305, 305), (309, 309), (331, 341), (348, 348), (354, 360)]

out = []
out.append("# Options check: an instrument that counts incorrect saved-instruction deliveries (Ollama fork, G1 OQ-3)\n")
out.append(
    "**What you are asked to do.** For each of questions Q1-Q7 in section 5, choose an option, give the strongest "
    "objection to every option (including the ones you choose), and say whether a materially better option is "
    "missing. Then answer Q8-Q10. Say whether anything in sections 3 and 4 is wrong at source, citing file:line. "
    "There is no preferred answer; the options are listed in no order of preference.\n")
out.append("## 1. The feature\n")
out.append(
    "G1 lets the operator save standing instructions once, in the desktop app or from the terminal. Every new "
    "conversation in the terminal agent and the desktop app then sends them to the model inside a leading system "
    "message (the \"carrier\"). An active conversation keeps the revision it started with until an explicit reload. "
    "Nothing of G1 is implemented at the commit under review: this is the pre-build baseline. The instrument asked "
    "for (OQ-3) must count incorrect deliveries through the real desktop HTTP and terminal entry points, be run on "
    "the unchanged candidate to give the baseline number, and be reusable as the final acceptance instrument once "
    "G1 is built.\n")
out.append(f"## 2. The contract, quoted verbatim from `{BP}` at commit `{COMMIT}`\n")
out.append("You may open this file and the concept `docs/_design/G1_PERSISTENT_INSTRUCTIONS_CONCEPT.md`. Each line "
           "below is prefixed with its line number.\n")
for a, b in RANGES:
    out.append("```text")
    for n in range(a, b + 1):
        out.append(f"{n}: {lines[n - 1]}")
    out.append("```\n")
out.append(open(os.path.join(HERE, "options.md"), encoding="utf-8").read())
packet = "\n".join(out)
with open(os.path.join(HERE, "PACKET.md"), "w", encoding="utf-8") as fh:
    fh.write(packet)
print("PACKET.md", hashlib.sha256(packet.encode("utf-8")).hexdigest(), len(packet.encode("utf-8")), "bytes")
