# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
import pathlib
import subprocess
import sys


ROOT = pathlib.Path(__file__).resolve().parent.parent
SERICA = ROOT / "build" / "stage0" / "serica"


def check(path: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [str(SERICA), "check", str(ROOT / path)],
        cwd=ROOT,
        text=True,
        capture_output=True,
        check=False,
    )


def main() -> int:
    valid = check("examples/p0b-window.serica")
    if valid.returncode != 0 or "源程序结构：通过" not in valid.stdout:
        sys.stderr.write(valid.stdout + valid.stderr)
        print("P0b 源程序未通过 stage0", file=sys.stderr)
        return 1

    missing_field = check("examples/diagnostics/p0b-missing-text.serica")
    missing_output = missing_field.stdout + missing_field.stderr
    if missing_field.returncode == 0 or "错误 S-3006：" not in missing_output:
        sys.stderr.write(missing_output)
        print("P0b 缺少必填字段时没有返回 S-3006", file=sys.stderr)
        return 1

    print("P0b 正例与缺字段诊断通过")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
