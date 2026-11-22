# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
"""Validate the native module graph, retained AST and declaration origins."""

import re
from pathlib import Path
from subprocess import run
from tempfile import TemporaryDirectory

from native_artifact import verify_native_console_artifact
from test_p1b_import_paths import OUTPUT, source_text
from test_p1b_parse import PARSER_IMPORTS
from test_p1b_scan import ROOT, SERICA, compile_source, expect_run


EDGE = 1 << 60
AST = 2 << 60
EXPORT = 3 << 60
SOURCE = 4 << 60


def read_graph(parser, project, entry="main.serica", mode="module-graph"):
    def read(index):
        result = run([parser, entry, mode, project], input=f"{index}\n".encode(),
                     capture_output=True, timeout=8)
        if result.returncode or result.stderr:
            raise RuntimeError((entry, index, result.returncode, result.stderr))
        return int(result.stdout)

    return read


def write(project, files):
    for name, content in files.items():
        path = project / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")


def file_record(read, rank):
    key = rank << 32
    module, width, root, edges, size = (read(key + field) for field in range(5))
    path = bytes(read(key + 5 + column) for column in range(width)).decode()
    return module, path, root, edges, size


def verify_graph_contents(parser, project, files, read_all_bytes=False):
    read = read_graph(parser, project)
    assert (read(-1), read(-2), read(-3)) == (5, 6, 5)
    ordered = [file_record(read, rank) for rank in range(5)]
    paths = [path for _, path, _, _, _ in ordered]
    assert paths == ["empty.serica", "common.serica", "A.serica", "lib/Z.serica", "main.serica"]
    assert [identifier for identifier, *_ in ordered] == [4, 3, 2, 5, 1]
    names = {identifier: path for identifier, path, _, _, _ in ordered}
    roots = set()
    for identifier, path, root, edges, size in ordered:
        dumped = files[path].encode()
        assert len(dumped) == size and root not in roots
        roots.add(root)
        assert read(AST + root) == (size << 8) + (100 if identifier == 1 else 101)
        assert read(AST + root + 3) == 0
        assert edges == files[path].count("导入 『")
        if read_all_bytes:
            source_key = SOURCE + ((identifier - 1) << 32)
            offsets = range(size) if size < 75 else (0, size // 2, size - 1)
            assert bytes(read(source_key + k) for k in offsets) == bytes(dumped[k] for k in offsets)
            assert read(source_key + size) == 0

    found_edges = []
    for rank in range(read(-2)):
        key = EDGE + (rank << 32)
        origin, destination, offset, node = (read(key + field) for field in range(4))
        assert read(AST + node) >> 32 == offset and read(AST + node) & 255 == 102
        assert files[names[origin]].encode()[offset:].startswith("导入".encode())
        found_edges.append((names[origin], names[destination]))
    assert found_edges == [
        ("main.serica", "A.serica"), ("main.serica", "A.serica"), ("main.serica", "lib/Z.serica"),
        ("A.serica", "common.serica"), ("common.serica", "empty.serica"), ("lib/Z.serica", "common.serica"),
    ]

    if read_all_bytes:
        expected = [
            ("common.serica", 103, "共享", -1), ("A.serica", 104, "加", 2),
            ("lib/Z.serica", 103, "吉", -1), ("lib/Z.serica", 103, "𠮷", -1),
            ("main.serica", 103, "根值", -1),
        ]
        for rank, (path, kind, spelling, count) in enumerate(expected):
            key = EXPORT + (rank << 32)
            unit, actual_kind, declaration, name, arity = (read(key + field) for field in range(5))
            assert names[unit] == path and (actual_kind, arity) == (kind, count)
            assert read(AST + declaration + 1) == name
            descriptor = read(AST + name)
            start, width = descriptor >> 32, (descriptor >> 8) & 0xffffff
            assert files[path].encode()[start:start + width].decode() == spelling


def verify_dependency_order(parser, project, artifact):
    entry_body = "常量 根值=42。\n" + OUTPUT.replace("输出=0", "输出=加（根值，𠮷）")
    files = {
        "main.serica": source_text(["lib/Z.serica", "./A.serica", "A.serica"]) + entry_body,
        "A.serica": source_text(["common.serica"]) + "函数 加（甲，乙）=甲+乙+共享。\n",
        "lib/Z.serica": source_text([".././common.serica"]) + "常量 吉=3。\n常量 𠮷=7。\n",
        "common.serica": source_text(["empty.serica"]) + "常量 共享=1。\n",
        "empty.serica": "",
    }
    write(project, files)
    verify_graph_contents(parser, project, files, read_all_bytes=True)
    compile_source(project / "main.serica", artifact)
    expect_run(artifact, b"50\n")
    expected_image = artifact.read_bytes()
    files["main.serica"] = source_text(["A.serica", "lib/./Z.serica", "temp/../A.serica"]) + entry_body
    write(project, {"main.serica": files["main.serica"]})
    verify_graph_contents(parser, project, files)
    compile_source(project / "main.serica", artifact)
    expect_run(artifact, b"50\n")
    assert artifact.read_bytes() == expected_image


def verify_diagnostics(parser, project):
    cases = (
        ({"main.serica": source_text(["A.serica"]) + OUTPUT,
          "A.serica": "# 环的位置\n" + source_text(["./main.serica"])}, "A.serica", 2, "导入", 4003),
        ({"main.serica": source_text(["A.serica"]) + OUTPUT,
          "A.serica": source_text(["B.serica"]), "B.serica": source_text(["./A.serica"])},
         "B.serica", 3, "导入", 4003),
        ({"main.serica": source_text(["A.serica"]) + OUTPUT, "A.serica": "# 模块\n" + OUTPUT},
         "A.serica", 2, "程序", 4004),
        ({"main.serica": source_text(["A.serica"]) + OUTPUT, "A.serica": "常量 甲=😀。\n"},
         "A.serica", 2, "😀", 1001),
        ({"main.serica": source_text(["lib/Z.serica"]) + OUTPUT,
          "lib/Z.serica": source_text(["../../escape.serica"])}, "lib/Z.serica", 2, "导入", 4001),
        ({"main.serica": source_text(["Z.serica", "A.serica"]) + OUTPUT,
          "A.serica": "常量 同=1。\n", "Z.serica": "常量 同=2。\n"}, "Z.serica", 3, "同=2", 4005),
        ({"main.serica": source_text(["Z.serica", "A.serica"]) + OUTPUT,
          "A.serica": "函数 同（）=1。\n", "Z.serica": "函数 同（）=2。\n"}, "Z.serica", 3, "同（）=2", 4006),
        ({"main.serica": source_text(["A.serica"]) + "常量 同=2。\n" + OUTPUT,
          "A.serica": "常量 同=1。\n"}, "main.serica", 1, "同=2", 4005),
        ({"main.serica": "常量 同=1。\n常量 同=2。\n" + OUTPUT}, "main.serica", 1, "同=2", 3007),
        ({"main.serica": "函数 同（）=1。\n函数 同（）=2。\n" + OUTPUT}, "main.serica", 1, "同（）=2", 3060),
    )
    for files, name, module, marker, code in cases:
        write(project, files)
        prefix = files[name].split(marker, 1)[0]
        expected = -((module << 37) + (len(prefix.encode()) << 13) + code)
        read = read_graph(parser, project)
        assert read(-1) == expected, (name, code, read(-1), expected)
        seed = run([SERICA, "check", project / "main.serica"], capture_output=True, text=True, timeout=8)
        row, col = prefix.count("\n") + 1, len(prefix.rsplit("\n", 1)[-1]) + 1
        pattern = rf"{re.escape(name)}:{row}:{col}（显示第 \d+ 格）：错误 S-{code}："
        assert seed.returncode == 1 and re.search(pattern, seed.stderr), (name, marker, seed.stderr)
    return len(cases)


def main():
    with TemporaryDirectory(prefix="serica-module-graph-") as d:
        work = Path(d).resolve()
        parser = work / "module-graph"
        compile_source(ROOT / "compiler/parse-expression.serica", parser)
        verify_native_console_artifact(parser, require_runtime_input=True, required_imports=PARSER_IMPORTS)
        verify_dependency_order(parser, work / "diamond", work / "seed-program")
        diag_count = verify_diagnostics(parser, work / "diagnostics")
        project = work / "chain"
        chain_depth = 72
        paths = [f"m{index:02}.serica" for index in range(chain_depth)]
        write(project, {"main.serica": source_text(paths[:1]) + OUTPUT})
        write(project, {name: source_text(paths[i + 1:i + 2]) for i, name in enumerate(paths)})
        read = read_graph(parser, project)
        assert (read(-1), read(-2), read(-3)) == (chain_depth + 1, chain_depth, 0)
        assert file_record(read, 0)[1] == paths[-1]
        assert file_record(read, chain_depth)[1] == "main.serica"
        for target, code in (("Missing.serica", 4002), ("Link.serica", 4001), ("Broken.serica", 4002)):
            write(project, {"main.serica": source_text([target]) + OUTPUT})
            if target == "Link.serica":
                (project / target).symlink_to(paths[0])
            if target == "Broken.serica":
                (project / target).write_bytes(b"\xff")
            assert read(-1) == -((1 << 37) + code)
        (project / "EntryAlias.serica").symlink_to("main.serica")
        assert read_graph(parser, project, "EntryAlias.serica")(-1) == -((1 << 37) + 4001)
        read = read_graph(parser, ROOT / "compiler", "parse-expression.serica")
        own_files, own_edges = read(-1), read(-2)
        assert (own_files, own_edges) == (11, 13)
        print(f"P1b 模块依赖图通过：5 节点共享图、{chain_depth} 层导入、{own_files} 份编译器源码、"
              f"{diag_count} 个位置诊断及 3 种读取失败")


if __name__ == "__main__":
    main()
