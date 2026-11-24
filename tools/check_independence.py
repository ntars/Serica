#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
"""Check source-level dependency, license, and cache isolation."""

import os
import pathlib
import subprocess
import sys


ROOT = pathlib.Path(__file__).resolve().parent.parent
MANIFESTS = {
    "go.mod",
    "go.sum",
    "Cargo.toml",
    "package.json",
    "requirements.txt",
    "CMakeLists.txt",
}
DEPENDENCIES = {
    "vendor",
    "node_modules",
    ".toolchain",
    "toolchains",
    "sdk",
    ".venv",
}


def fail(message: str) -> None:
    print(f"源码级：失败（{message}）", file=sys.stderr)
    raise SystemExit(1)


def check_source() -> None:
    for current, dirs, files in os.walk(ROOT):
        path = pathlib.Path(current)
        rel = path.relative_to(ROOT)
        if rel == pathlib.Path("."):
            dirs[:] = [name for name in dirs if name != ".git"]
        inside_bootstrap = rel == pathlib.Path("bootstrap") or rel.parts[:1] == ("bootstrap",)
        for name in dirs:
            if name in DEPENDENCIES:
                fail(f"发现依赖副本或工具链目录：{rel / name}")
            if not inside_bootstrap and name in {"vendor", "node_modules", "third_party"}:
                fail(f"发现第三方源码目录：{rel / name}")
        for name in files:
            file_path = path / name
            file_rel = file_path.relative_to(ROOT)
            if not inside_bootstrap and file_path.suffix == ".vcxproj":
                fail(f"发现跨语言构建描述：{file_rel}")
            if not inside_bootstrap and name in MANIFESTS:
                fail(f"发现跨语言构建描述：{file_rel}")
            if file_path.suffix not in {".go", ".serica", ".py"}:
                continue
            for line in file_path.read_text(encoding="utf-8", errors="replace").splitlines():
                if "Copyright (c)" in line and "行简（Serica）贡献者" not in line:
                    fail(f"版权归属需核对：{file_rel}")
    print("源码级：通过")


def check_caches() -> None:
    go = os.environ.get("GO", "go")
    result = subprocess.run(
        [go, "env", "GOMODCACHE", "GOCACHE", "GOBIN"],
        text=True,
        capture_output=True,
        check=False,
    )
    if result.returncode != 0:
        fail(f"读取 Go 缓存路径：{result.stderr.strip()}")
    values = result.stdout.split()
    values.append(os.environ.get("PIP_CACHE_DIR") or pathlib.Path.home() / "Library" / "Caches" / "pip")
    for value in values:
        try:
            path = pathlib.Path(value).resolve()
            path.relative_to(ROOT)
        except ValueError:
            continue
        fail(f"Go 缓存落在仓库内：{path}")


def main() -> int:
    if len(sys.argv) != 2 or sys.argv[1] != "source":
        print("用法：tools/check-independence source", file=sys.stderr)
        return 2
    check_source()
    check_caches()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
