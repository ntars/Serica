#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Verify a Serica release directory is runnable, complete, and checksummed."""

from __future__ import annotations

import hashlib
import json
import pathlib
import re
import subprocess
import sys
import tarfile
import tempfile


ROOT = pathlib.Path(__file__).resolve().parent.parent
REQUIRED_FILES = (
    "README.md",
    "README_EN.md",
    "RELEASE.md",
    "ACCEPTANCE.md",
    "CHANGELOG.md",
    "DEPENDENCIES.md",
    "LICENSE",
    "LICENSE-MIT",
    "NOTICE",
    "Makefile",
    "VERSION",
    "bin/serica",
    "bin/hello.serica",
    "examples/p0b-window.serica",
    "examples/p1b-runtime-text-return.serica",
    "examples/community/fizzbuzz.serica",
    "examples/community/gcd.serica",
    "examples/community/text-counter.serica",
    "conformance/cases.json",
    "tools/check_conformance.py",
    "tools/migrate",
    "compiler/parse-expression.serica",
    "compiler/emit-console.serica",
)


def run(command, *, cwd, stdin=None):
    result = subprocess.run(
        command,
        cwd=cwd,
        input=stdin,
        text=True,
        capture_output=True,
        check=False,
    )
    if result.returncode != 0:
        raise RuntimeError(
            f"命令失败：{' '.join(str(item) for item in command)}\n"
            f"{result.stdout}{result.stderr}"
        )
    return result


def digest(path: pathlib.Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def verify(release: pathlib.Path) -> dict:
    if not release.is_dir():
        raise RuntimeError(f"发布目录不存在：{release}")
    missing = [name for name in REQUIRED_FILES if not (release / name).is_file()]
    if missing:
        raise RuntimeError(f"发布目录缺少文件：{', '.join(missing)}")

    version = (release / "VERSION").read_text(encoding="utf-8").strip()
    if not re.fullmatch(r"v\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?", version):
        raise RuntimeError(f"版本号无效：{version}")

    checksum_file = release / "SHA256SUMS"
    checksums = {}
    for line in checksum_file.read_text(encoding="utf-8").splitlines():
        value, name = line.split(None, 1)
        checksums[name.strip()] = value
    entries = sorted(path.relative_to(release).as_posix()
                     for path in release.rglob("*")
                     if path.is_file() and path.name != "SHA256SUMS"
                     and not path.name.endswith(".uploading.cfg"))
    if not entries or set(checksums) != set(entries):
        raise RuntimeError("SHA256SUMS 与发布目录文件集合不一致")
    for name in entries:
        if digest(release / name) != checksums[name]:
            raise RuntimeError(f"校验和不匹配：{name}")

    manifest = json.loads((release / "sbom.json").read_text(encoding="utf-8"))
    if manifest.get("version") != version or manifest.get("component", "").startswith("/"):
        raise RuntimeError("sbom.json 的版本或组件字段无效")

    serica = release / "bin" / "serica"
    if not serica.stat().st_mode & 0o111:
        raise RuntimeError("bin/serica 不可执行")
    with tempfile.TemporaryDirectory(prefix="serica-release-") as temporary:
        console = pathlib.Path(temporary) / "console"
        run([str(serica), "build", str(release / "bin" / "hello.serica"), "-o", str(console)], cwd=release)
        window = pathlib.Path(temporary) / "window"
        run([str(serica), "build", str(release / "examples" / "p0b-window.serica"), "-o", str(window)], cwd=release)
        for path in (window, console):
            info = run(["file", str(path)], cwd=release).stdout
            if "Mach-O 64-bit executable arm64" not in info:
                raise RuntimeError(f"产物架构无效：{path}")
            verify_result = run(["codesign", "--verify", "--strict", str(path)], cwd=release)
            if verify_result.returncode != 0:
                raise RuntimeError(f"签名验证失败：{path}")

        console_output = run([str(console)], cwd=release, stdin="行简\n").stdout
        if console_output != "行简\n":
            raise RuntimeError(f"控制台运行输出不符：{console_output!r}")
    archive = release.parent / f"{release.name}.tar.gz"
    if archive.is_file() and digest(archive) not in {checksums.get(archive.name), ""}:
        # The archive is intentionally not listed inside itself; verify a
        # companion checksum when present.
        companion = release.parent / "SHA256SUMS.archives"
        if companion.is_file():
            expected = next(
                (line.split(None, 1)[0] for line in companion.read_text(encoding="utf-8").splitlines()
                 if line.split(None, 1)[-1].strip() == archive.name),
                None,
            )
            if expected and digest(archive) != expected:
                raise RuntimeError(f"压缩包校验和不匹配：{archive}")

    if archive.is_file():
        with tarfile.open(archive, "r:gz") as tar:
            names = {member.name.split("/", 1)[-1] for member in tar.getmembers() if member.isfile()}
        if "VERSION" not in names or "bin/serica" not in names:
            raise RuntimeError("发布压缩包内容不完整")

    print(f"发布验收通过：{release.name}（{version}）")
    return {
        "version": version,
        "files": len(entries),
        "console_output": console_output.strip(),
    }


def main() -> int:
    if len(sys.argv) != 2:
        print("用法：tools/test_release.py <发布目录>", file=sys.stderr)
        return 2
    try:
        verify(pathlib.Path(sys.argv[1]).resolve())
    except (RuntimeError, OSError, ValueError, json.JSONDecodeError, tarfile.TarError) as error:
        print(str(error), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
