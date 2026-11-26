#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Assert that CI and verify include the conformance and format gates."""

from __future__ import annotations

import json
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent


def main() -> int:
    makefile = (ROOT / "Makefile").read_text(encoding="utf-8")
    workflow = (ROOT / ".github" / "workflows" / "preflight.yml").read_text(encoding="utf-8")
    cases = json.loads((ROOT / "conformance" / "cases.json").read_text(encoding="utf-8"))["cases"]
    verify = next(line for line in makefile.splitlines() if line.startswith("verify:"))
    failures = []
    for gate in ("conformance", "format", "regression"):
        if gate not in verify.split():
            failures.append(f"verify 目标缺少 {gate}")
        if gate not in workflow:
            failures.append(f"CI 原生门缺少 {gate}")
    if len(cases) < 41:
        failures.append(f"一致性用例少于基线 41：{len(cases)}")
    if "41" in workflow:
        failures.append("CI 不得硬编码一致性用例数，须从 cases.json 读取")
    if failures:
        print("CI 一致性门检查失败：", file=sys.stderr)
        print("\n".join(failures), file=sys.stderr)
        return 1
    print(f"CI 一致性门检查：通过（{len(cases)} 例，基线从 cases.json 读取）")
    return 0


if __name__ == "__main__":
    sys.exit(main())
