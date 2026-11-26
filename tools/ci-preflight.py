#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Cross-platform checks that can run before the native macOS stage gates."""

from pathlib import Path
import json
import subprocess
import sys

ROOT = Path(__file__).resolve().parent.parent


def run(*command: str) -> None:
    result = subprocess.run(command, cwd=ROOT, text=True, capture_output=True)
    if result.returncode:
        sys.stdout.write(result.stdout)
        sys.stderr.write(result.stderr)
        raise SystemExit(f"CI 前置检查失败：{' '.join(command)}")


def main() -> int:
    run("python3", "tools/check_decisions.py")
    data = json.loads((ROOT / "DECISIONS.json").read_text(encoding="utf-8"))
    decisions = data["decisions"]
    if not decisions:
        raise SystemExit("DECISIONS.json 缺少决策记录")
    run("python3", "tools/check_seed_subset.py")
    run("python3", "tools/check_independence.py", "source")
    print("CI 前置检查通过：决策索引、种子边界与源码独立性")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
