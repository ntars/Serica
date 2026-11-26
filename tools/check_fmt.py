#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Check that maintained examples are already in canonical Serica source form."""

from __future__ import annotations

import pathlib
import subprocess
import sys
import tempfile

ROOT = pathlib.Path(__file__).resolve().parent.parent
SERICA = ROOT / "build" / "stage0" / "serica"
SOURCES = [ROOT / "conformance" / "sources"] + [ROOT / "examples"]


def main() -> int:
    if not SERICA.is_file():
        print(f"缺少编译器 {SERICA}，先运行 make stage0", file=sys.stderr)
        return 1
    files = []
    for directory in SOURCES:
        files.extend(sorted(directory.rglob("*.serica")))
    files = [path for path in files if path.name != "bad-nfc.serica"]
    failures = []
    with tempfile.TemporaryDirectory(prefix="serica-fmt-") as directory:
        copy_root = pathlib.Path(directory)
        for index, source in enumerate(files):
            copy = copy_root / f"{index}.serica"
            copy.write_text(source.read_text(encoding="utf-8"), encoding="utf-8")
            checked = subprocess.run(
                [str(SERICA), "fmt", "--check", str(copy)], text=True, capture_output=True, timeout=8
            )
            if checked.returncode != 0:
                failures.append(f"{source.relative_to(ROOT)}: {checked.stdout}{checked.stderr}".strip())
    if failures:
        print("格式门失败：", file=sys.stderr)
        print("\n".join(failures), file=sys.stderr)
        return 1
    print(f"格式门通过：{len(files)} 份源码无需改写")
    return 0


if __name__ == "__main__":
    sys.exit(main())
