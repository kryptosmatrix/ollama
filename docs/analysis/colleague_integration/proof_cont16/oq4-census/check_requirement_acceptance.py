#!/usr/bin/env python3
from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[5]
BLUEPRINT = ROOT / "docs" / "_design" / "G1_PERSISTED_INSTRUCTIONS.md"

text = BLUEPRINT.read_text(encoding="utf-8")
requirements = set(re.findall(r"^(G1-R-\d{2}) MUST\b", text, flags=re.MULTILINE))
acceptance_rows: dict[str, set[str]] = {}
for line in text.splitlines():
    m = re.match(r"^\|\s*(I-\d{2})\s*\|", line)
    if not m:
        continue
    aid = m.group(1)
    acceptance_rows[aid] = set(re.findall(r"G1-R-\d{2}", line))

expected_requirements = {f"G1-R-{i:02d}" for i in range(1, 19)}
expected_acceptance = {f"I-{i:02d}" for i in range(1, 15)}

errors: list[str] = []
if requirements != expected_requirements:
    errors.append(
        "requirement set mismatch: "
        f"missing={sorted(expected_requirements - requirements)} "
        f"extra={sorted(requirements - expected_requirements)}"
    )
if set(acceptance_rows) != expected_acceptance:
    errors.append(
        "acceptance set mismatch: "
        f"missing={sorted(expected_acceptance - set(acceptance_rows))} "
        f"extra={sorted(set(acceptance_rows) - expected_acceptance)}"
    )

unknown_refs = set().union(*acceptance_rows.values()) - requirements if acceptance_rows else set()
if unknown_refs:
    errors.append(f"unknown requirement references: {sorted(unknown_refs)}")

unmapped_acceptance = sorted(a for a, refs in acceptance_rows.items() if not refs)
if unmapped_acceptance:
    errors.append(f"acceptance rows without requirement ids: {unmapped_acceptance}")

covered = set().union(*acceptance_rows.values()) if acceptance_rows else set()
uncovered = sorted(requirements - covered)
if uncovered:
    errors.append(f"requirements without acceptance coverage: {uncovered}")

if errors:
    print("G1_REQUIREMENT_ACCEPTANCE_BIJECTION=FAIL")
    for error in errors:
        print(error)
    sys.exit(1)

print(
    "G1_REQUIREMENT_ACCEPTANCE_BIJECTION=PASS "
    f"requirements={len(requirements)} acceptance={len(acceptance_rows)}"
)
