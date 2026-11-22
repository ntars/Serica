# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
"""Exercise import path validation and sorted module edges in the native front end."""

import posixpath
import re
from pathlib import Path
from random import Random
from subprocess import run
from tempfile import TemporaryDirectory

from native_artifact import verify_native_console_artifact
from test_p1b_bind import descriptors
from test_p1b_parse import PARSER_IMPORTS, query
from test_p1b_scan import ROOT, SERICA, compile_source, expect_run


OUTPUT = "程序=【能力=array（『stdout』），输出=0】。\n"


def source_text(paths):
    return "".join(f"导入 『{path}』。\n" for path in paths)


def reader(parser, source, mode, module_path=None):
    arguments = [parser, source, mode]
    if module_path is not None:
        arguments.append(module_path)

    def read(index):
        result = run(arguments, input=f"{index}\n".encode(), capture_output=True, timeout=8)
        if result.returncode or result.stderr:
            raise RuntimeError((arguments, result.returncode, result.stdout, result.stderr))
        return int(result.stdout)

    return read


def check_paths(parser, source, paths, module_path=None):
    imports = source_text(paths)
    source.write_text(imports + (OUTPUT if module_path is None else ""), encoding="utf-8")
    ast_mode, mode = ("program", "imports") if module_path is None else ("module", "module-imports")
    read = reader(parser, source, mode, module_path)
    if read(-1) != len(paths) or read(len(paths) << 32) != 0:
        raise RuntimeError((paths, read(-1)))
    table = descriptors(parser, source, ast_mode)
    parent = posixpath.dirname(module_path or "")
    starts = [0]
    for line in imports.splitlines(keepends=True):
        starts.append(starts[-1] + len(line.encode()))

    for rank, j in enumerate(sorted(range(len(paths)), key=lambda i: posixpath.normpath(paths[i]))):
        start = rank << 32
        expected = posixpath.normpath(posixpath.join(parent, paths[j])).encode()
        length = read(start + 1)
        if length != len(expected):
            raise RuntimeError((paths[j], expected, length))
        actual = bytes(read(start + 2 + offset) for offset in range(length))
        if actual != expected or read(start) != table[starts[j], 102] or read(start + 2 + length) != 0:
            raise RuntimeError((paths[j], expected, actual, rank))
    return len(paths)


def check_invalid(parser, source):
    invalid = (
        "", "/absolute.serica", "a//b.serica", "a.serica/", "../out.serica",
        "sub/../../out.serica", "a.SERICA", "a.seric", "a:b.serica", "汉.serica",
        "a\\\\b.serica", "a\\nb.serica", "a b.serica",
    )
    prefix = "# 位置按 UTF-8 字节计算\n"
    expected = -((len(prefix.encode()) << 13) + 4001)
    for path in invalid:
        source.write_text(prefix + source_text([path]) + OUTPUT, encoding="utf-8")
        actual = reader(parser, source, "imports")(-1)
        if actual != expected or query(parser, source, -1, "bind-program") != expected:
            raise RuntimeError((path, actual, expected))
        seed = run([SERICA, "check", source], capture_output=True, text=True, timeout=8)
        if seed.returncode != 1 or not re.search(r":2:1（显示第 1 格）：错误 S-4001：", seed.stderr):
            raise RuntimeError((path, seed.returncode, seed.stderr))

    source.write_text(source_text(["../../out.serica"]), encoding="utf-8")
    if reader(parser, source, "module-imports", "child/input.serica")(-1) != -4001:
        raise RuntimeError("nested module escaped the entry directory")
    if query(parser, source, -1, "bind-module") != 1:
        raise RuntimeError("binding without a project-relative origin must defer the depth check")
    source.write_text(source_text(["target.serica"]), encoding="utf-8")
    for origin in (None, "/root/child.serica", "dir/./child.serica", "dir/../child.serica"):
        if reader(parser, source, "module-imports", origin)(-1) != -4001:
            raise RuntimeError((origin, "accepted invalid module origin"))
    return len(invalid) + 5


def check_project(parser, directory):
    root = directory / "main.serica"
    nested = directory / "lib" / "source.serica"
    nested.parent.mkdir()
    (directory / "shared.serica").write_text("常量 共享=42。\n", encoding="utf-8")
    nested.write_text('导入 『../shared.serica』。\n函数 值（）=共享。\n', encoding="utf-8")
    program = '程序=【能力=array（『stdout』），输出=值（）】。\n'
    artifacts = []
    for paths in (("lib/source.serica", "./shared.serica"), ("shared.serica", "lib/./source.serica")):
        root.write_text(source_text(paths) + program, encoding="utf-8")
        artifact = directory / "seed"
        compile_source(root, artifact)
        expect_run(artifact, b"42\n")
        artifacts.append(artifact.read_bytes())
        if reader(parser, root, "imports")(-1) != 2:
            raise RuntimeError("entry did not retain both direct import edges")
    if artifacts[0] != artifacts[1]:
        raise RuntimeError("stage0 output changed across equivalent import orderings")
    if reader(parser, nested, "module-imports", "lib/source.serica")(-1) != 1:
        raise RuntimeError("nested import edge is missing")

    root.write_text(source_text(["absent.serica"]) + OUTPUT, encoding="utf-8")
    seed = run([SERICA, "check", root], capture_output=True, text=True, timeout=8)
    if reader(parser, root, "imports")(-1) != 1 or seed.returncode != 1 or "S-4002" not in seed.stderr:
        raise RuntimeError("direct edge extraction should leave missing-file diagnostics to the loader")


def main():
    with TemporaryDirectory(prefix="serica-import-paths-") as temporary:
        work = Path(temporary)
        parser = work / "imports"
        compile_source(ROOT / "compiler/parse-expression.serica", parser)
        verify_native_console_artifact(
            parser, require_runtime_input=True,
            required_imports=PARSER_IMPORTS,
            required_instructions=("bl\t", "ret", "ldrb\tw0, [x1]"),
        )
        source = work / "input.serica"
        edges = check_paths(parser, source, [])
        edges += check_paths(parser, source, [
            "z.serica", "sub/./data.serica", "a/../b.serica", "./b.serica", ".serica", "B.serica",
            "folder-" + "x" * 90 + "/_file-2.serica", "prefix.serica", "prefix.serica/nested.serica",
        ])
        edges += check_paths(parser, source, [
            "../../shared.serica", "../lib.serica", "./sub/../local.serica",
            "z.serica", "../deep/local.serica", "../deep/./z.serica",
        ], "lib/deep/self.serica")
        rng = Random(121)
        for _ in range(3):
            paths = [rng.choice(("./", "x/../", "a/./", "../../", "../b/../")) + f"{j}.serica" for j in range(5)]
            rng.shuffle(paths)
            edges += check_paths(parser, source, paths, "first/second/third/unit.serica")
        rejected = check_invalid(parser, source)
        check_project(parser, work)
        print(f"P1b 导入路径通过：{edges} 条排序及规范化记录、{rejected} 个路径拒绝场景、2 个原生参照产物")


if __name__ == "__main__":
    main()
