#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Run the language conformance corpus against a Serica compiler.

The checker works both in the source tree (``build/stage0/serica``) and in a
release bundle (``bin/serica``). Set ``SERICA`` to override the executable.
"""

from __future__ import annotations

import json
import os
import pathlib
import re
import subprocess
import sys
import tempfile

ROOT = pathlib.Path(__file__).resolve().parent.parent
CORPUS = ROOT / "conformance" / "cases.json"
DIAGNOSTIC = re.compile(
    r"^.+:\d+:\d+（显示第 \d+ 格）：错误 S-\d{4}：\n"
    r"  期望：.+\n  实际：.+\n  建议：.+",
    re.MULTILINE,
)


def compiler_path() -> pathlib.Path:
    override = os.environ.get("SERICA")
    if override:
        return pathlib.Path(override)
    for candidate in (ROOT / "build" / "stage0" / "serica", ROOT / "bin" / "serica"):
        if candidate.is_file():
            return candidate
    return ROOT / "build" / "stage0" / "serica"


def main() -> int:
    serica = compiler_path()
    if not serica.is_file():
        print(f"缺少编译器 {serica}，先运行 make stage0", file=sys.stderr)
        return 1
    cases = json.loads(CORPUS.read_text(encoding="utf-8"))["cases"]
    failures = []
    with tempfile.TemporaryDirectory(prefix="serica-conformance-") as directory:
        work = pathlib.Path(directory)
        for case in cases:
            source = (ROOT / "conformance" / "sources" / case["source"]).resolve()
            checked = subprocess.run([str(serica), "check", str(source)], text=True, capture_output=True, timeout=8)
            output = checked.stdout + checked.stderr
            if case["kind"] == "negative":
                expected = case["diagnostic"]
                diagnostic = DIAGNOSTIC.search(output)
                if checked.returncode == 0 or expected not in output or diagnostic is None:
                    failures.append(f"{case['id']}: 期望 {expected}，实际退出 {checked.returncode}：{output.strip()}")
                continue
            artifact = work / f"{case['id']}.bin"
            built = subprocess.run([str(serica), "build", str(source), "-o", str(artifact)], text=True, capture_output=True, timeout=8)
            if built.returncode != 0:
                failures.append(f"{case['id']}: 构建失败：{built.stdout}{built.stderr}")
                continue
            run = subprocess.run([str(artifact)], input=case["stdin"], text=True, capture_output=True, timeout=3)
            if run.returncode != case["exit"] or run.stdout != case["stdout"]:
                failures.append(
                    f"{case['id']}: 期望 stdout={case['stdout']!r} exit={case['exit']}，"
                    f"实际 stdout={run.stdout!r} exit={run.returncode} stderr={run.stderr!r}"
                )
    if failures:
        print("一致性套件失败：", file=sys.stderr)
        print("\n".join(failures), file=sys.stderr)
        return 1
    positive = sum(1 for case in cases if case["kind"] == "positive")
    print(f"一致性套件通过：{len(cases)} 例（{positive} 个正例、{len(cases)-positive} 个反例）")
    return 0


if __name__ == "__main__":
    sys.exit(main())
