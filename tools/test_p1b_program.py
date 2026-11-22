# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
"""Exercise real compilation-unit ASTs emitted by the Serica parser."""

import re
from pathlib import Path
from subprocess import run
from tempfile import TemporaryDirectory

from native_artifact import verify_native_console_artifact
from test_p1b_parse import PARSER_IMPORTS, node, number, query, read_tree, reference
from test_p1b_scan import ROOT, SERICA, compile_source, expect_run


def check_tree(parser, path, mode, expected):
    root = query(parser, path, -1, mode)
    seen = set()
    actual, sibling = read_tree(parser, path, path.read_bytes(), root, seen, mode)
    if sibling or actual != expected:
        raise RuntimeError((path.name, actual, expected))
    return len(seen)


def program_fixture():
    imported = "导入 『support.serica』。"
    constant = "常量 基=2。"
    parameters = "（甲，乙）"
    effects = "能力=array（『stdin』）"
    first = "【返回 和。】"
    second = "【返回 0。】"
    guard = "若（和>0）" + first + "否" + second + "。"
    block = "【变量 和=甲+乙。" + guard + "】"
    function = "函数 算" + parameters + effects + block + "。"
    variable = "变量 数=0。"
    loop_guard = "若（数<基）【继续。】。"
    loop_block = "【赋值 数=数+1。" + loop_guard + "跳出。】"
    loop = "凡（数<基）" + loop_block + "。"
    record = "【能力=array（『stdout』），输出=数】"
    root = "程序=" + record + "。"
    source = "# 完整入口\n" + "\n".join((imported, constant, function, variable, loop, root)) + "\n"
    expected = node(
        100, source,
        node(102, imported, node(2, "『support.serica』")),
        node(103, constant, reference("基"), number("2")),
        node(
            104, function, reference("算"),
            node(114, parameters, reference("甲"), reference("乙")),
            node(115, effects, node(4, "array（『stdin』）", node(2, "『stdin』"))),
            node(
                113, block,
                node(105, "变量 和=甲+乙。", reference("和"), node(60, "+", reference("甲"), reference("乙"))),
                node(
                    107, guard, node(41, ">", reference("和"), number("0")),
                    node(113, first, node(111, "返回 和。", reference("和"))),
                    node(113, second, node(111, "返回 0。", number("0"))),
                ),
            ),
        ),
        node(105, variable, reference("数"), number("0")),
        node(
            108, loop, node(40, "<", reference("数"), reference("基")),
            node(
                113, loop_block,
                node(106, "赋值 数=数+1。", reference("数"), node(60, "+", reference("数"), number("1"))),
                node(107, loop_guard, node(40, "<", reference("数"), reference("基")),
                     node(113, "【继续。】", node(110, "继续。"))),
                node(109, "跳出。"),
            ),
        ),
        node(
            112, root, reference("程序"),
            node(5, record, node(6, "能力", node(4, "array（『stdout』）", node(2, "『stdout』"))),
                 node(6, "输出", reference("数"))),
        ),
    )
    return source, expected


def main():
    with TemporaryDirectory(prefix="serica-program-") as directory:
        work = Path(directory)
        entry = ROOT / "compiler/parse-expression.serica"
        parser = work / "parse"
        compile_source(entry, parser)
        verify_native_console_artifact(
            parser, require_runtime_input=True,
            required_imports=PARSER_IMPORTS,
            required_instructions=("bl\t", "ret", "ldrb\tw0, [x1]"),
        )
        path = work / "input.serica"
        (work / "support.serica").write_text("常量 支=7。\n", encoding="utf-8")
        source, expected = program_fixture()
        path.write_text(source, encoding="utf-8")
        nodes = check_tree(parser, path, "program", expected)
        reference_artifact = work / "seed-program"
        compile_source(path, reference_artifact)
        expect_run(reference_artifact, b"2\n")

        module = "常量 值=42。\n函数 求（）=值。\n"
        path.write_text(module, encoding="utf-8")
        nodes += check_tree(
            parser, path, "module",
            node(101, module, node(103, "常量 值=42。", reference("值"), number("42")),
                 node(104, "函数 求（）=值。", reference("求"), node(114, "（）"), reference("值"))),
        )
        path.write_bytes(b"")
        nodes += check_tree(parser, path, "module", node(101, ""))
        window = "窗口=【标题=『行简』，文本=『你好』】。"
        path.write_text(window, encoding="utf-8")
        nodes += check_tree(
            parser, path, "program",
            node(100, window, node(112, window, reference("窗口"),
                 node(5, "【标题=『行简』，文本=『你好』】",
                      node(6, "标题", node(2, "『行简』")), node(6, "文本", node(2, "『你好』"))))),
        )

        # Stage0 reaches a module through an import. The embedded parser reads it directly.
        wrapper = work / "wrap.serica"
        wrapper.write_text("导入 『input.serica』。\n程序=【能力=array（『stdout』），输出=0】。\n", encoding="utf-8")
        invalid = (
            ("program", "【】", "【", 1, "S-2001"),
            ("program", "错误=0。", "错误", 5, "S-2002"),
            ("program", "常量 普=0。\n导入 『support.serica』。", "导入", 1, "S-2001"),
            ("program", "变量 甲=0。\n函数 普（）=0。", "函数", 1, "S-2001"),
            ("program", "常量 Program=0。", "Program", 6, "S-2003"),
            ("program", "函数 普（甲，甲）=0。", "甲）", 14, "S-3060"),
            ("program", "函数 普（甲，）=0。", "）", 1, "S-2001"),
            ("program", "函数 普（）【数字。】。", "数字", 12, "S-3068"),
            ("program", "若（1==1）【变量 甲=0。】。", "变量", 13, "S-3037"),
            ("program", "函数 普（）能力 array（『stdin』）=0。", "array", 1, "S-2001"),
            ("program", "程序=0。\n常量 后=1。", "常量", 1, "S-2001"),
            ("program", "程序=【输出=1+】。", "】", 1, "S-2001"),
            ("program", "变量 甲=1。\n", "", 1, "S-2001"),
            ("module", "变量 甲=1。", "变量", 4, "S-4004"),
            ("module", "窗口=【】。", "窗口", 4, "S-4004"),
        )
        for mode, body, marker, tag, code in invalid:
            path.write_text(body, encoding="utf-8")
            index = body.index(marker) if marker else len(body)
            before = body[:index]
            expected = -(len(before.encode()) * 16 + tag)
            actual = query(parser, path, -1, mode)
            if actual != expected:
                raise RuntimeError((mode, body, actual, expected))
            result = run([SERICA, "check", wrapper if mode == "module" else path],
                         capture_output=True, text=True, timeout=8)
            row, column = before.count("\n") + 1, len(before.rsplit("\n", 1)[-1]) + 1
            accepted = rf":{row}:{column}（显示第 \d+ 格）：错误 {code}："
            if result.returncode != 1 or not re.search(accepted, result.stderr):
                raise RuntimeError((body, accepted, result.stdout, result.stderr))

        own_files = tuple(sorted((ROOT / "compiler").glob("*.serica")))
        executables = {"lex-probe.serica", "scan-probe.serica", "parse-expression.serica", "emit-console.serica"}
        for own in own_files:
            mode = "program" if own.name in executables else "module"
            root = query(parser, own, -1, mode)
            if root <= 0:
                raise RuntimeError((own.name, root))
            head = query(parser, own, root, mode)
            if head != (len(own.read_bytes()) << 8) + (100 if mode == "program" else 101):
                raise RuntimeError((own.name, head))
        print(f"P1b 程序解析通过：4 棵完整树、{nodes} 个节点、{len(invalid)} 个诊断及 {len(own_files)} 份编译器源码")


if __name__ == "__main__":
    main()
