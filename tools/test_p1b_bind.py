# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
"""Verify name bindings, scope exits, and source diagnostics in the native front end."""

import re
from pathlib import Path
from subprocess import run
from tempfile import TemporaryDirectory

from native_artifact import verify_native_console_artifact
from test_p1b_parse import PARSER_IMPORTS, query
from test_p1b_scan import ROOT, SERICA, compile_source, expect_run


PROGRAM = """常量 基=3。
常量 阈=基+2。
函数 取（参）=加（参）+阈。
函数 加（甲）=甲+基。
函数 求（层）【
变量 局=层。
凡（局>0）【
变量 减=局。
凡（减>1）【
赋值 减=减-1。
若（减==2）【继续。】。
跳出。
】。
赋值 局=局-1。
若（局==2）【继续。】。
跳出。
】。
返回 局。
】。
变量 输入=readInteger（）。
变量 值=取（输入）。
程序=【能力=array（『stdout』，『stdin』），输出=值+求（输入）】。
"""
EXECUTABLES = {"lex-probe.serica", "scan-probe.serica", "parse-expression.serica", "emit-console.serica"}


def descriptors(parser, path, mode):
    by_position = {}
    index = 1
    while header := query(parser, path, index, mode):
        key = header >> 32, header & 255
        if key in by_position:
            raise RuntimeError((path.name, key, index))
        by_position[key] = index
        index += 4
    return by_position


def at(body, text, prefix=""):
    pattern = prefix + text
    if body.count(pattern) != 1:
        raise ValueError(f"Ambiguous source marker {pattern!r}")
    return len(body[:body.index(pattern) + len(prefix)].encode())


def assert_binding(parser, path, mode, node_id, expected):
    actual = query(parser, path, node_id, mode)
    if actual != expected:
        raise RuntimeError((path.name, node_id, actual, expected))


def check_program(parser, path):
    path.write_text(PROGRAM, encoding="utf-8")
    table = descriptors(parser, path, "program")

    def entry(text, prefix="", kind=3):
        return table[at(PROGRAM, text, prefix), kind]

    base = entry("基=3")
    outer = entry("凡（局>0）", kind=108)
    inner = entry("凡（减>1）", kind=108)
    cases = (
        (entry("基+2"), base),
        (entry("加（参）", kind=4), entry("加（甲）=")),
        (entry("参）+", "加（"), entry("参）=", "取（")),
        (entry("阈。", "加（参）+"), entry("阈=")),
        (entry("甲+基", "="), entry("甲）=", "加（")),
        (entry("基。", "甲+"), base),
        (entry("层。", "局="), entry("层）", "求（")),
        (entry("局。", "减="), entry("局=层")),
        (entry("减=减-1"), entry("减=局")),
        (entry("减-1", "减="), entry("减=局")),
        (entry("继续。", "若（减==2）【", 110), inner),
        (entry("跳出。", "若（减==2）【继续。】。\n", 109), inner),
        (entry("局=局-1"), entry("局=层")),
        (entry("继续。", "若（局==2）【", 110), outer),
        (entry("跳出。", "若（局==2）【继续。】。\n", 109), outer),
        (entry("局。", "返回 "), entry("局=层")),
        (entry("取（输入）", kind=4), entry("取（参）=")),
        (entry("输入）。", "取（"), entry("输入=readInteger")),
        (entry("值+求"), entry("值=取")),
        (entry("求（输入）", kind=4), entry("求（层）")),
        (entry("readInteger", kind=4), -24),
        (entry("array", kind=4), -9),
        (entry("输出", kind=6), 0),
        (entry("基=3"), 0),
    )
    for occurrence, target in cases:
        assert_binding(parser, path, "bind-program", occurrence, target)
    assert_binding(parser, path, "bind-program", -2, 0)
    return len(cases)


def check_imports(parser, path):
    (path.parent / "support.serica").write_text(
        "常量 外值=7。\n函数 外函（）=外值。\n", encoding="utf-8",
    )
    source = "导入 『support.serica』。\n函数 代（）=外函（）+外值。\n程序=【能力=array（『stdout』），输出=代（）】。\n"
    path.write_text(source, encoding="utf-8")
    table = descriptors(parser, path, "program")
    assert_binding(parser, path, "bind-program", table[at(source, "外函"), 4], -34)
    assert_binding(parser, path, "bind-program", table[at(source, "外值"), 3], -33)
    assert_binding(parser, path, "bind-program", -2, 2)
    artifact = path.parent / "seed-import"
    compile_source(path, artifact)
    expect_run(artifact, b"14\n")


def check_diagnostics(parser, source):
    root = "程序=【能力=array（『stdout』），输出=0】。"
    invalid = (
        ("常量 甲=乙。\n常量 乙=1。\n" + root, "乙。", "甲=", 3022),
        ("常量 甲=甲。\n" + root, "甲。", "甲=", 3022),
        ("常量 甲=0。\n常量 甲=1。\n" + root, "甲=1", "", 3007),
        ("函数 甲（）=0。\n函数 甲（）=1。\n" + root, "甲（）=1", "", 3060),
        ("常量 甲=0。\n变量 甲=1。\n" + root, "甲=1", "", 3037),
        ("变量 甲=0。\n变量 甲=1。\n" + root, "甲=1", "", 3037),
        ("常量 甲=1。\n赋值 甲=2。\n" + root, "甲=2", "", 3038),
        ("变量 甲=乙。\n变量 乙=2。\n" + root, "乙。", "甲=", 3022),
        ("跳出。\n" + root, "跳出", "", 3067),
        ("若（1==1）【继续。】。\n" + root, "继续", "", 3067),
        ("函数 函（甲）=甲。\n程序=【能力=array（『stdout』），输出=函（）】。", "函（）", "", 3061),
        ("程序=【能力=array（『stdout』），输出=缺名（）】。", "缺名", "", 3023),
        ("函数 函（甲）【变量 甲=1。返回 甲+1。】。\n" + root, "甲=1", "", 3068),
        ("函数 函（甲）【赋值 乙=1。返回 甲。】。\n" + root, "乙=1", "", 3068),
        ("函数 函（甲）【变量 乙=丙。变量 丙=甲。返回 乙。】。\n" + root, "丙。", "乙=", 3068),
        (
            "函数 函（甲）【若（甲>0）【变量 局=1。返回 局。】。返回 局。】。\n" + root,
            "局。", "】。返回 ", 3068,
        ),
        (
            "函数 函（甲）【变量 局=0。凡（甲>0）【变量 局=1。跳出。】。返回 局。】。\n" + root,
            "函（甲）", "", 3068,
        ),
        (
            "函数 函（甲）【凡（甲>0）【变量 局=甲。跳出。】。返回 局。】。\n"
            "程序=【能力=array（『stdin』，『stdout』），输出=函（readInteger（））】。",
            "局。", "返回 ", 3022,
        ),
        ("程序=【输出=😀】。", "😀", "", 1001),
    )
    for body, marker, prefix, code in invalid:
        source.write_text(body, encoding="utf-8")
        offset = at(body, marker, prefix)
        assert_binding(parser, source, "bind-program", -1, -((offset << 13) + code))
        start = body.encode()[:offset].decode()
        row = start.count("\n") + 1
        column = len(start.rsplit("\n", 1)[-1]) + 1
        result = run([SERICA, "check", source], capture_output=True, text=True, timeout=8)
        expected = rf":{row}:{column}（显示第 \d+ 格）：错误 S-{code}："
        if result.returncode != 1 or not re.search(expected, result.stderr):
            raise RuntimeError((body, expected, result.stdout, result.stderr))
    return len(invalid)


def check_examples(parser):
    accepted = 0
    with_imports = 0
    for source in sorted((ROOT / "examples").glob("*.serica")):
        seed = run([SERICA, "check", source], capture_output=True, timeout=8)
        if seed.returncode:
            continue
        pending = query(parser, source, -2, "bind-program")
        if pending < 0 or (pending > 0 and "导入 『" not in source.read_text(encoding="utf-8")):
            raise RuntimeError((source.name, pending))
        accepted += 1
        with_imports += pending > 0
    if accepted == 0:
        raise RuntimeError("stage0 did not accept any of the repository examples")
    return accepted, with_imports


def main():
    with TemporaryDirectory(prefix="serica-bind-") as directory:
        work = Path(directory)
        parser = work / "bind"
        compile_source(ROOT / "compiler/parse-expression.serica", parser)
        verify_native_console_artifact(
            parser, require_runtime_input=True,
            required_imports=PARSER_IMPORTS,
            required_instructions=("bl\t", "ret", "ldrb\tw0, [x1]"),
        )
        source = work / "input.serica"
        bindings = check_program(parser, source)
        artifact = work / "seed-program"
        compile_source(source, artifact)
        expect_run(artifact, b"12\n", stdin=b"3\n")
        check_imports(parser, source)
        diagnostics = check_diagnostics(parser, source)

        module = "常量 吉=1。\n常量 𠮷=2。\n函数 读（）=𠮷+吉。\n"
        source.write_text(module, encoding="utf-8")
        table = descriptors(parser, source, "module")
        for reference, declaration in (("𠮷+吉", "𠮷=2"), ("吉。", "吉=1")):
            assert_binding(parser, source, "bind-module",
                           table[at(module, reference), 3], table[at(module, declaration), 3])
        bindings += 2
        own_files = sorted((ROOT / "compiler").glob("*.serica"))
        for own in own_files:
            mode = "bind-program" if own.name in EXECUTABLES else "bind-module"
            assert_binding(parser, own, mode, -1, 1)
            pending = query(parser, own, -2, mode)
            if pending < 0 or (pending > 0 and "导入 『" not in own.read_text(encoding="utf-8")):
                raise RuntimeError((own.name, pending))
        examples, deferred_examples = check_examples(parser)
        print(f"P1b 名称绑定通过：{bindings} 个节点绑定、{diagnostics} 个诊断、"
              f"2 条待链接引用、{len(own_files)} 份编译器源码、"
              f"{examples} 个现有示例（{deferred_examples} 个有待链接引用）")


if __name__ == "__main__":
    main()
