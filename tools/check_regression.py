#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Regression corpus guard for historical examples (M6/S6.4)."""

from __future__ import annotations

import json
import pathlib
import re
import subprocess
import sys
import tempfile

ROOT = pathlib.Path(__file__).resolve().parent.parent
SERICA = ROOT / "build" / "stage0" / "serica"
MANIFEST = ROOT / "tools" / "regression-corpus.json"


def main() -> int:
    if not SERICA.is_file():
        print(f"缺少编译器 {SERICA}，先运行 make stage0", file=sys.stderr)
        return 1
    manifest = json.loads(MANIFEST.read_text(encoding="utf-8"))
    entries = manifest["cases"]
    baseline = int(manifest["baseline"])
    if len(entries) < baseline:
        print(f"回归语料少于基线 {baseline}：{len(entries)}", file=sys.stderr)
        return 1
    if len(entries) != len({entry["id"] for entry in entries}):
        print("回归语料存在重复 id", file=sys.stderr)
        return 1

    cases = json.loads((ROOT / "conformance" / "cases.json").read_text(encoding="utf-8"))["cases"]
    by_id = {case["id"]: case for case in cases}
    checked = 0
    with tempfile.TemporaryDirectory(prefix="serica-regression-") as directory:
        for entry in entries:
            case = by_id.get(entry["case"])
            if case is None:
                print(f"回归用例未进入一致性套件：{entry['id']}", file=sys.stderr)
                return 1
            source = (ROOT / "conformance" / "sources" / case["source"]).resolve()
            output = pathlib.Path(directory) / f"{entry['id']}"
            if case["kind"] == "positive":
                build = subprocess.run(
                    [str(SERICA), "build", str(source), "-o", str(output)],
                    text=True, capture_output=True, check=False,
                )
                if build.returncode:
                    print(f"回归构建失败：{entry['id']}\n{build.stdout}{build.stderr}", file=sys.stderr)
                    return 1
                run = subprocess.run(
                    [str(output)],
                    input=case.get("stdin", ""), text=True, capture_output=True, check=False,
                )
                if run.returncode != case["exit"] or run.stdout != case["stdout"] or run.stderr:
                    print(
                        f"回归运行不符：{entry['id']}，期望 {case['stdout']!r}/{case['exit']}，"
                        f"实际 {run.stdout!r}/{run.returncode}",
                        file=sys.stderr,
                    )
                    return 1
            else:
                check = subprocess.run(
                    [str(SERICA), "check", str(source)], text=True, capture_output=True, check=False
                )
                diagnostic = check.stdout + check.stderr
                if check.returncode == 0 or f"错误 {case['diagnostic']}：" not in diagnostic:
                    print(f"回归反例诊断不符：{entry['id']}\n{diagnostic}", file=sys.stderr)
                    return 1
                if not re.search(r":\d+:\d+（显示第 \d+ 格）：", diagnostic):
                    print(f"回归反例缺少位置：{entry['id']}\n{diagnostic}", file=sys.stderr)
                    return 1
            checked += 1
    print(f"回归语料门通过：{checked} 例，基线 {baseline}，全部来自一致性套件的历史样例")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
