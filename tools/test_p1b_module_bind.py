# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
"""Verify cross-file binding targets and located native read failures."""

import re
from pathlib import Path
from subprocess import run
from tempfile import TemporaryDirectory

from native_artifact import verify_native_console_artifact
from test_p1b_import_paths import OUTPUT, source_text
from test_p1b_module_graph import AST, file_record, read_graph, write
from test_p1b_module_read import primitive_source
from test_p1b_parse import PARSER_IMPORTS
from test_p1b_scan import ROOT, SERICA, compile_source, expect_run


BINDING = 5 << 60
TARGET_FILE = 6 << 60


def marker_offset(body, marker):
    if body.count(marker) != 1:
        raise ValueError(f"ambiguous marker {marker!r}")
    return len(body.split(marker, 1)[0].encode())


def descriptors(read):
    modules = sorted(file_record(read, rank) for rank in range(read(-1)))
    nodes = {}
    for index, (identifier, path, first, _, _) in enumerate(modules):
        last = modules[index + 1][2] if index + 1 < len(modules) else None
        nodes[path] = identifier, {}
        node = first
        while (last is None or node < last) and (value := read(AST + node)):
            coordinates = value >> 32, value & 255
            assert coordinates not in nodes[path][1]
            nodes[path][1][coordinates] = node
            node += 4
    return nodes


def verify_bindings(parser, project, artifact):
    files = {
        "main.serica": source_text(["B.serica", "A.serica"]) + "常量 结果=后+1。\n"
        + "变量 临=结果。\n程序=【能力=array（『stdout』），输出=转（临）+共（共）】。\n",
        "A.serica": "常量 先=40。\n常量 共=3。\n函数 外加（加数）=加数+先。\n"
        "函数 共（同）=同+1。\n",
        "B.serica": "常量 后=先+1。\n函数 转（输入）=外加（输入）。\n",
    }
    write(project, files)
    read = read_graph(parser, project, mode="module-bind")
    nodes = descriptors(read)

    def at(path, marker, kind=3, prefix=""):
        position = marker_offset(files[path], prefix + marker) + len(prefix.encode())
        return nodes[path][1][position, kind]

    bindings = (
        ("B.serica", "先+1", "", 3, "A.serica", "先=40"),
        ("B.serica", "外加（输入）", "", 4, "A.serica", "外加（加数）"),
        ("B.serica", "输入）。", "外加（", 3, "B.serica", "输入）=外加"),
        ("main.serica", "后+1", "", 3, "B.serica", "后=先"),
        ("main.serica", "结果。", "临=", 3, "main.serica", "结果=后"),
        ("main.serica", "转（临）", "", 4, "B.serica", "转（输入）"),
        ("main.serica", "临）+共", "转（", 3, "main.serica", "临=结果"),
        ("main.serica", "共（共）", "", 4, "A.serica", "共（同）"),
        ("main.serica", "共）】", "共（", 3, "A.serica", "共=3"),
        ("A.serica", "先。", "加数+", 3, "A.serica", "先=40"),
    )
    for source, marker, prefix, kind, target, declaration in bindings:
        occurrence = at(source, marker, kind, prefix)
        destination = at(target, declaration)
        assert (read(BINDING + occurrence), read(TARGET_FILE + occurrence)) == (destination, nodes[target][0])
    builtin = at("main.serica", "array（", 4)
    assert (read(BINDING + builtin), read(TARGET_FILE + builtin)) == (-9, 0)
    assert read(-4) == 6
    compile_source(project / "main.serica", artifact)
    expect_run(artifact, b"86\n")
    return len(bindings) + 1


def assert_diagnostic(parser, project, identifier, name, marker, code):
    text = (project / name).read_text(encoding="utf-8")
    prefix = text.split(marker, 1)[0]
    expected = -((identifier << 37) + (len(prefix.encode()) << 13) + code)
    read = read_graph(parser, project, mode="module-bind")
    actual = read(-4)
    assert actual == expected, (name, marker, code, actual, expected)
    seed = run([SERICA, "check", project / "main.serica"], capture_output=True, text=True, timeout=8)
    row, col = prefix.count("\n") + 1, len(prefix.rsplit("\n", 1)[-1]) + 1
    pattern = rf"{re.escape(name)}:{row}:{col}（显示第 \d+ 格）：错误 S-{code}："
    assert seed.returncode == 1 and re.search(pattern, seed.stderr), (name, marker, seed.stderr)


def verify_binding_errors(parser, project):
    cases = (
        ({"main.serica": source_text(["A.serica", "B.serica"]) + OUTPUT,
          "A.serica": "常量 前=后。\n", "B.serica": "常量 后=2。\n"}, 2, "A.serica", "后。", 3022),
        ({"main.serica": source_text(["A.serica"]) + OUTPUT, "A.serica": "常量 前=后。\n常量 后=2。\n"},
         2, "A.serica", "后。", 3022),
        ({"main.serica": source_text(["A.serica"]) + OUTPUT.replace("输出=0", "输出=取（）"),
          "A.serica": "函数 取（甲）=甲。\n"}, 1, "main.serica", "取（）", 3061),
        ({"main.serica": source_text(["A.serica"]) + OUTPUT.replace("输出=0", "输出=缺（）"),
          "A.serica": "常量 前=1。\n"}, 1, "main.serica", "缺（）", 3023),
        ({"main.serica": source_text(["A.serica"]) + OUTPUT,
          "A.serica": "函数 取（）=缺（）。\n"}, 2, "A.serica", "缺（）", 3064),
        ({"main.serica": source_text(["A.serica"]) + "赋值 前=3。\n" + OUTPUT,
          "A.serica": "常量 前=1。\n"}, 1, "main.serica", "前=3", 3038),
    )
    for files, identifier, name, marker, code in cases:
        write(project, files)
        assert_diagnostic(parser, project, identifier, name, marker, code)
    return len(cases)


def verify_read_errors(parser, project):
    write(project, {"main.serica": source_text(["A.serica"]) + OUTPUT, "Valid.serica": ""})
    (project / "Alias.serica").symlink_to("Valid.serica")
    (project / "Folder.serica").mkdir()
    (project / "Bad.serica").write_bytes(b"\xff")
    cases = (("Missing.serica", 4002), ("Alias.serica", 4001), ("Folder.serica", 4002), ("Bad.serica", 4002))
    for name, code in cases:
        write(project, {"A.serica": "# 模块来源\n" + source_text([name])})
        assert_diagnostic(parser, project, 2, "A.serica", "导入", code)
    return len(cases)


def verify_result_recovery(work):
    project = work / "recover"
    leaf = b"Valid.serica"
    write(project, {"Valid.serica": "# 行简𠮷\n" * 5000})
    (project / "Broken.serica").write_bytes(b"\xff")
    combined = leaf + b"Broken.serica"
    source = primitive_source(combined, 0, len(leaf)).split("程序=", 1)[0]
    source += (
        "变量 读件=readModuleResult（argumentText（1），字节，0，12）。\n"
        "变量 保存=结果值（读件）。\n"
        "变量 初=textByteLength（保存）。\n"
        "赋值 读件=readModuleResult（argumentText（1），字节，12，13）。\n"
        "变量 码=错误码（读件）。\n"
        "赋值 读件=readModuleResult（argumentText（1），字节，0，12）。\n"
        "赋值 保存=结果值（读件）。\n"
        "程序=【能力=array（『stdout』，『args』，『filesystem.read』），"
        "输出=初+码+textByteLength（保存）】。\n"
    )
    write(work, {"recover.serica": source})
    output = work / "recover-artifact"
    compile_source(work / "recover.serica", output)
    expected = 2 * (project / "Valid.serica").stat().st_size + 42
    expect_run(output, f"{expected}\n".encode(), args=(project,))


def main():
    with TemporaryDirectory(prefix="serica-module-bind-") as directory:
        work = Path(directory).resolve()
        parser = work / "module-bind"
        compile_source(ROOT / "compiler/parse-expression.serica", parser)
        verify_native_console_artifact(parser, require_runtime_input=True, required_imports=PARSER_IMPORTS)
        bindings = verify_bindings(parser, work / "bind", work / "seed-bind")
        binding_errors = verify_binding_errors(parser, work / "binding-diags")
        read_errors = verify_read_errors(parser, work / "read-diags")
        verify_result_recovery(work)
        own = read_graph(parser, ROOT / "compiler", "parse-expression.serica", "module-bind")
        assert own(-1) == 11
        own_bindings = own(-4)
        assert own_bindings > 0
        print(f"P1b 跨文件绑定通过：{bindings} 个查询、{binding_errors} 个绑定诊断、"
              f"{read_errors} 个读取位置诊断、1 次读取恢复、编译器自身 {own_bindings} 个跨文件绑定")


if __name__ == "__main__":
    main()
