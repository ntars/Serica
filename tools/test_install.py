#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Verify local install/uninstall of the release bundle in an isolated prefix."""

from __future__ import annotations

import os
import pathlib
import shutil
import subprocess
import sys
import tempfile


def run(command, *, cwd, env=None, stdin=None):
    result = subprocess.run(command, cwd=cwd, env=env, input=stdin,
                            text=True, capture_output=True, check=False)
    if result.returncode != 0:
        raise RuntimeError(
            f"命令失败：{' '.join(str(item) for item in command)}\n"
            f"{result.stdout}{result.stderr}"
        )
    return result


def verify(release: pathlib.Path) -> None:
    for name in ("tools/install.sh", "tools/uninstall.sh", "bin/hello.serica"):
        if not (release / name).is_file():
            raise RuntimeError(f"发布目录缺少安装文件：{name}")

    with tempfile.TemporaryDirectory(prefix="serica-install-") as tmp:
        home = pathlib.Path(tmp) / "home"
        prefix = home / ".local"
        home.mkdir()
        env = dict(os.environ)
        env["HOME"] = str(home)
        env["SERICA_PREFIX"] = str(prefix)
        env["PATH"] = f"{prefix / 'bin'}:{env.get('PATH', '')}"

        run(["/bin/sh", str(release / "tools/install.sh")], cwd=release, env=env)
        serica = shutil.which("serica", path=env["PATH"])
        if serica is None:
            raise RuntimeError("安装后 PATH 未找到 serica")
        expected = subprocess.run(["readlink", serica], capture_output=True,
                                  text=True, check=True).stdout.strip()
        if not expected.startswith(str(prefix / "lib/serica/")):
            raise RuntimeError(f"serica 链接目标无效：{expected}")
        install_dir = pathlib.Path(expected).parent.parent
        if not (install_dir / "RELEASE.md").is_file():
            raise RuntimeError("安装目录缺少发布文档")

        sample = pathlib.Path(tmp) / "hello.serica"
        shutil.copy(release / "bin/hello.serica", sample)
        run(["serica", "check", str(sample)], cwd=tmp, env=env)
        program = pathlib.Path(tmp) / "console"
        run(["serica", "build", str(sample), "-o", str(program)], cwd=tmp, env=env)
        output = run([str(program)], cwd=tmp, env=env, stdin="行简\n").stdout
        if output != "行简\n":
            raise RuntimeError(f"安装后控制台输出不符：{output!r}")

        run(["/bin/sh", str(install_dir / "tools/uninstall.sh")], cwd=tmp, env=env)
        if (prefix / "bin/serica").exists():
            raise RuntimeError("卸载后命令链接仍存在")
        if (prefix / "lib/serica").exists():
            raise RuntimeError("卸载后安装目录仍存在")
        if not (home / ".local").is_dir():
            raise RuntimeError("卸载误删了调用方 prefix")

    print("本地安装与卸载验收通过")


def main() -> int:
    if len(sys.argv) != 2:
        print("用法：tools/test_install.py <发布目录>", file=sys.stderr)
        return 2
    try:
        verify(pathlib.Path(sys.argv[1]).resolve())
    except (RuntimeError, OSError) as error:
        print(str(error), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
