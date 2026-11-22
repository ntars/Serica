# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
"""Verify project-scoped module reads through both native compiler entry points."""

import os
from pathlib import Path
from subprocess import run
from tempfile import TemporaryDirectory

from native_artifact import verify_native_console_artifact
from test_p1b_import_paths import OUTPUT, source_text
from test_p1b_parse import MODULE_READ_IMPORTS, PARSER_IMPORTS
from test_p1b_scan import ROOT, SERICA, compile_source, expect_run


def primitive_source(path, start, count, root='argumentText（1）', capabilities=None):
    if capabilities is None:
        capabilities = ('stdout', 'args', 'filesystem.read')
    values = "，".join(map(str, path))
    declared = "，".join(f"『{name}』" for name in capabilities)
    return (
        f"变量 字节=array（{values}）。\n"
        f"程序=【能力=array（{declared}），输出=readModule（{root}，字节，{start}，{count}）】。\n"
    )


def verify_queries(artifact, arguments, rank, expected):
    key = rank << 32
    expect_run(artifact, f"{len(expected)}\n".encode(), args=arguments, stdin=f"{key}\n".encode())
    for offset, byte in enumerate(expected, 1):
        expect_run(artifact, f"{byte}\n".encode(), args=arguments, stdin=f"{key + offset}\n".encode())
    expect_run(artifact, b"0\n", args=arguments, stdin=f"{key + len(expected) + 1}\n".encode())


def check_primitive(work, project, text):
    source, artifact = work / "direct.serica", work / "direct"
    normalized = b"lib/Exact.serica"
    guarded_range = (0, 255, *normalized, -1)
    source.write_text(primitive_source(guarded_range, 2, len(normalized)), encoding="utf-8")
    compile_source(source, artifact)
    verify_native_console_artifact(artifact, require_integer=False, required_imports=("_read", *MODULE_READ_IMPORTS))
    expect_run(artifact, text + b"\n", args=(project,))

    bad_ranges = (
        (normalized, -1, len(normalized)), (normalized, len(normalized) + 1, len(normalized)),
        (normalized, 1, len(normalized)), (normalized, 0, 0),
        (b"../Outside.serica", 0, 17), (b"lib/./Exact.serica", 0, 18),
        (b"lib//Exact.serica", 0, 17), (b"/Exact.serica", 0, 13),
        ((0, *b".serica"), 0, 8), ((256, *b".serica"), 0, 8),
        (b"lib/Exact.txt", 0, 13),
    )
    for data, start, count in bad_ranges:
        source.write_text(primitive_source(data, start, count), encoding="utf-8")
        compile_source(source, artifact)
        expect_run(artifact, b"", args=(project,), status=41)
    oversized = primitive_source(
        b"A", 0, 1024, capabilities=('stdout', 'args', 'filesystem.read', 'array.write'),
    ).replace(
        "程序=", "变量 长=1。\n凡（长<1024）【赋值 字节=arrayPush（字节，65）。赋值 长=长+1。】。\n程序=",
    )
    source.write_text(oversized, encoding="utf-8")
    compile_source(source, artifact)
    expect_run(artifact, b"", args=(project,), status=41)

    valid = primitive_source(normalized, 0, len(normalized))
    rejected = (
        (primitive_source(normalized, 0, len(normalized), capabilities=('stdout', 'args')), "S-4102"),
        (primitive_source(normalized, 0, len(normalized), capabilities=('stdout', 'filesystem.read')), "S-4102"),
        (valid.replace("readModule（argumentText（1），字节，0，16）", "readModule（）"), "S-3073"),
        (valid.replace("变量 字节=array（", "常量 字节=array（"), "S-3073"),
        (valid.replace("argumentText（1），字节", "『』，字节"), "S-3073"),
        (
            "函数 读（）=readModule（『.』，字节，0，16）。\n"
            + valid.replace("输出=readModule（argumentText（1），字节，0，16）", "输出=读（）"),
            "S-4106",
        ),
    )
    for body, code in rejected:
        source.write_text(body, encoding="utf-8")
        for command in ("check", "build"):
            argv = [SERICA, command, source]
            if command == "build":
                argv += ["-o", work / "rejected"]
            checked = run(argv, capture_output=True, text=True, timeout=8)
            if checked.returncode != 1 or f"错误 {code}：" not in checked.stderr:
                raise RuntimeError((command, code, body, checked.returncode, checked.stderr))
    return len(bad_ranges) + 1, len(rejected)


def check_frontend(work, project, text):
    parser = work / "read-import"
    compile_source(ROOT / "compiler/parse-expression.serica", parser)
    verify_native_console_artifact(parser, require_runtime_input=True, required_imports=PARSER_IMPORTS)
    root = project / "main.serica"
    root.write_text(source_text(["z/../Second.serica", "lib/./Exact.serica"]) + OUTPUT, encoding="utf-8")
    (project / "Second.serica").write_bytes(b"")
    verify_queries(parser, (root, "read-import", project), 0, b"")
    verify_queries(parser, (root, "read-import", project), 1, text)

    sub = project / "lib" / "module.serica"
    sub.write_text(source_text(["../Second.serica", "./Exact.serica"]), encoding="utf-8")
    verify_queries(parser, (sub, "read-module-import", "lib/module.serica", project), 1, text)

    other = work / "outside"
    other.mkdir()
    (other / "Outside.serica").write_bytes(b"outside")
    (project / "Foreign.serica").symlink_to(other / "Outside.serica")
    (project / "Alias.serica").symlink_to("lib/Exact.serica")
    (project / "alias-dir").symlink_to("lib", target_is_directory=True)
    (project / "Folder.serica").mkdir()
    (project / "Broken.serica").write_bytes(b"\xf0\x9f")
    os.mkfifo(project / "Pipe.serica")
    with (project / "Large.serica").open("wb") as large:
        large.truncate(16 * 1024 * 1024)
    with (project / "Limit.serica").open("wb") as limit:
        limit.truncate(16 * 1024 * 1024 - 1)
    root.write_text(source_text(["Limit.serica"]) + OUTPUT, encoding="utf-8")
    expect_run(parser, b"16777215\n", args=(root, "read-import", project), stdin=b"0\n")
    wrong_case_status = 41 if (project / "lib" / "exact.serica").exists() else 42
    wrong_directory_case = 41 if (project / "LIB" / "Exact.serica").exists() else 42
    unreadable = (
        ("Foreign.serica", 41), ("Alias.serica", 41), ("alias-dir/Exact.serica", 41),
        ("lib/exact.serica", wrong_case_status), ("LIB/Exact.serica", wrong_directory_case),
        ("Folder.serica", 42), ("Broken.serica", 42), ("Missing.serica", 42),
        ("Pipe.serica", 42), ("Large.serica", 42),
    )
    for target, status in unreadable:
        root.write_text(source_text([target]) + OUTPUT, encoding="utf-8")
        expect_run(parser, b"", args=(root, "read-import", project), stdin=b"0\n", status=status)

    root.write_text(source_text(["lib/Exact.serica"]) + OUTPUT, encoding="utf-8")
    alias = work / "project-link"
    alias.symlink_to(project, target_is_directory=True)
    for spelling in (str(alias), str(alias) + "/", str(alias) + "/."):
        expect_run(parser, b"", args=(root, "read-import", spelling), stdin=b"0\n", status=41)
    return len(unreadable) + 3


def main():
    with TemporaryDirectory(prefix="serica-module-read-") as temp:
        work = Path(temp).resolve()
        project = work / "project"
        (project / "lib").mkdir(parents=True)
        text = "常量 𠮷=42。\n".encode()
        (project / "lib" / "Exact.serica").write_bytes(text)
        ranges, rejected = check_primitive(work, project, text)
        unreadable = check_frontend(work, project, text)
        print(f"P1b 受限模块读取通过：5 个读取正例、{ranges + unreadable} 个运行时拒绝、{rejected} 类编译诊断")


if __name__ == "__main__":
    main()
