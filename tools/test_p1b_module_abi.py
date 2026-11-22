# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
"""Inspect compiler ABI descriptors and compare them with executed ARM64 artifacts."""

from pathlib import Path
from tempfile import TemporaryDirectory

from native_artifact import verify_native_console_artifact
from test_p1b_import_paths import OUTPUT, source_text
from test_p1b_module_effect import function_names
from test_p1b_module_graph import AST, read_graph, write
from test_p1b_module_type import assert_diagnostic
from test_p1b_parse import PARSER_IMPORTS
from test_p1b_scan import ROOT, compile_source, expect_run


MODE = "module-abi"
ABI = -(4 << 60)
CALL = -(5 << 60)
LOCAL = -(6 << 60)


def signature(read, node):
    base = ABI - (node << 6)
    return tuple(read(base - field) for field in range(8 + 2 * read(base - 1)))


def root_ast_node(read, source, marker, kind):
    offset = source.encode().index(marker.encode())
    for slot in range(1, len(source.encode()) * 4 + 1, 4):
        head = read(AST + slot)
        if not head:
            break
        if head >> 32 == offset and head & 255 == kind:
            return slot
    raise AssertionError((marker, kind, offset))


def reference_artifact(project, output, expected, stdin=b"", required=(), imports=()):
    compile_source(project / "main.serica", output)
    verify_native_console_artifact(
        output, require_integer=False, require_runtime_input=bool(stdin),
        required_instructions=("bl\t", "ret", *required), required_imports=imports,
    )
    expect_run(output, expected, stdin=stdin)


def verify_recursive_group(parser, project, output):
    main = source_text(["recursive.serica"]) + (
        "函数 包（值）=甲（值）+2。\n"
        "函数 整形（数）【变量 局=数。若（数<=0）【返回 0。】。返回 局。】。\n"
        "变量 初=readInteger（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=包（初）+整形（2）+降（3）】。\n"
    )
    write(project, {
        "main.serica": main,
        "recursive.serica": (
            "函数 甲（数）=ifElse（数<=0，0，乙（数-1）+1）。\n"
            "函数 乙（数）=ifElse（数<=0，0，甲（数-1）+1）。\n"
            "函数 降（数）【变量 此=数+1。若（数<=0）【返回 0。】。返回 此+降（数-1）。】。\n"
        ),
    })
    read = read_graph(parser, project, mode=MODE)
    assert read(-1) == 2
    names = function_names(read)
    expected = (1, 1, 1, 1, 0, 16, 1, -1, 1, 0)
    for name in ("甲", "乙", "降"):
        assert signature(read, names[name]) == expected
    assert signature(read, names["包"]) == signature(read, names["整形"]) == (0,) * 8
    graph = read_graph(parser, project)
    inner = root_ast_node(graph, main, "甲（值）", 4)
    outer = root_ast_node(graph, main, "包（初）", 4)
    assert read(CALL - inner) == names["甲"] and read(CALL - outer) == 0
    reference_artifact(project, output, b"18\n", b"5\n", required=("str\tx30, [sp, #0x8]",))


def verify_mixed_parameters(parser, project, output):
    source = (
        "函数 合并（甲，乙，丙，丁，戊，己）【"
        "变量 合=textLength（甲）+textLength（丁）。返回 合+乙+丙+戊+己。】。\n"
        + OUTPUT.replace("输出=0", "输出=合并（『𠮷x』，2，3，『简』，4，5）")
    )
    write(project, {"main.serica": source})
    read = read_graph(parser, project, mode=MODE)
    assert read(-1) == 1
    name = function_names(read)["合并"]
    assert signature(read, name) == (2, 6, 8, 1, 1, 80, 1, -1, 2, 0, 1, 2, 1, 3, 2, 4, 1, 6, 1, 7)
    graph = read_graph(parser, project)
    local = root_ast_node(graph, source, "合=textLength", 3)
    call = root_ast_node(graph, source, "合并（『", 4)
    assert read(LOCAL - local) == (64 << 3) + 1 and read(CALL - call) == name
    reference_artifact(
        project, output, b"17\n",
        required=("sub\tsp, sp, #0x50", "str\tx1, [x19, #0x8]", "str\tx7, [x19, #0x38]"),
    )


def verify_callsite_text(parser, project, output):
    source = (
        "函数 比（甲，乙，轮）【变量 状=0。凡（状<轮）【赋值 状=状+1。】。"
        "若（equals（甲，乙））【返回 状+1。】。返回 状。】。\n"
        + OUTPUT.replace("输出=0", "输出=比（『同』，『同』，2）")
    )
    write(project, {"main.serica": source})
    read = read_graph(parser, project, mode=MODE)
    assert read(-1) == 1
    assert signature(read, function_names(read)["比"]) == (2, 3, 5, 1, 1, 48, 1, -1, 2, 0, 2, 2, 1, 4)
    reference_artifact(project, output, b"3\n", required=("str\tx4, [x19, #0x20]",))


def verify_returns(parser, root, output):
    cases = (
        ("recursive-result", "累计", (1, 2, 2, 2, 0, 16, 2, -1, 1, 0, 1, 1), b"5\n", b"15\n", ("str\tx1, [x19, #0x8]",)),
        (
            "recursive-text-result", "倒数", (1, 1, 1, 3, 0, 16, 4, -1, 1, 0),
            b"3\n", "到达零\n".encode(), ("adr\tx2,", "mov\tx3, #0x9"),
        ),
        (
            "text-return", "欢迎", (2, 1, 2, 4, 3, 48, 4, 16, 2, 0),
            "行简𠮷\n".encode(), "行简𠮷\n".encode(),
            ("ldr\tx1, [x19, #0x10]", "ldr\tx2, [x19, #0x18]", "ldr\tx3, [x19, #0x20]"),
        ),
    )
    for fixture, name, expected, stdin, returned, required in cases:
        source = (ROOT / f"examples/p1b-runtime-{fixture}.serica").read_text(encoding="utf-8")
        project = root / fixture
        write(project, {"main.serica": source})
        read = read_graph(parser, project, mode=MODE)
        assert read(-1) == 1
        assert signature(read, function_names(read)[name]) == expected, fixture
        if fixture == "text-return":
            local = root_ast_node(read_graph(parser, project), source, "副本=来源", 3)
            assert read(LOCAL - local) == (16 << 3) + 2
        imports = ("_mmap", "_munmap") if fixture == "text-return" else ()
        reference_artifact(project, output / fixture, returned, stdin, required, imports)
        if fixture == "recursive-result":
            expect_run(output / fixture, b"", status=31, stdin=b"-1\n")
    return len(cases)


def verify_diagnostics(parser, project):
    cases = (
        ("函数 长（源）【返回 textLength（源）。】。\n", "长（1）", "1）", 3062),
        (
            "函数 宽（甲，乙，丙，丁，戊）【返回 textLength（甲）+textLength（乙）+textLength（丙）"
            "+textLength（丁）+textLength（戊）。】。\n",
            "宽（『甲』，『乙』，『丙』，『丁』，『戊』）", "宽（", 3062,
        ),
        (
            "函数 宽（甲，乙，丙，丁，戊，己，庚，辛，壬）=ifElse（甲<=0，乙+丙+丁+戊+己+庚+辛+壬，宽（甲-1，乙，丙，丁，戊，己，庚，辛，壬））。\n",
            "宽（1，2，3，4，5，6，7，8，9）", "宽（", 3062,
        ),
        (
            "函数 错（数）=ifElse（数<=0，成功（textConcat（『行』，『简』）），错（数-1））。\n"
            "变量 终=错（1）。\n",
            "结果值（终）", "错（", 3023,
        ),
        (
            "函数 错（源）【赋值 源=textConcat（源，『字』）。返回 textLength（源）。】。\n",
            "错（『行』）", "源=textConcat", 3068,
        ),
        (
            "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n",
            "textLength（欢迎（『行简』））", "textLength", 3023,
        ),
        (
            "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n",
            "textLength（欢迎（『行简』））", "textLength（欢迎（『行简』））", 3023,
        ),
        (
            "函数 欢迎（来源）【变量 副本=来源。返回 副本。】。\n函数 包（）=欢迎（『行简』）。\n",
            "包（）", "包（）", 3023,
        ),
    )
    for prefix, expression, marker, code in cases:
        source = prefix + OUTPUT.replace("输出=0", "输出=" + expression)
        assert_diagnostic(parser, project, {"main.serica": source}, 1, "main.serica", marker, code, MODE, ABI - 64)
    return len(cases)


def verify_examples(parser):
    sources = sorted((ROOT / "examples").glob("p1b-runtime*.serica"))
    for source in sources:
        actual = read_graph(parser, source.parent, source.name, MODE)(-1)
        assert actual > 0, (source.name, actual)
    return len(sources)


def main():
    with TemporaryDirectory(prefix="serica-abi-") as directory:
        work = Path(directory).resolve()
        parser = work / "module-abi"
        compile_source(ROOT / "compiler/parse-expression.serica", parser)
        verify_native_console_artifact(parser, require_runtime_input=True, required_imports=PARSER_IMPORTS)
        verify_recursive_group(parser, work / "recursion", work / "recursion-reference")
        verify_mixed_parameters(parser, work / "mixed", work / "mixed-reference")
        verify_callsite_text(parser, work / "inferred", work / "inferred-reference")
        references = 3 + verify_returns(parser, work / "returns", work)
        diagnostics = verify_diagnostics(parser, work / "diagnostics")
        examples = verify_examples(parser)
        own = read_graph(parser, ROOT / "compiler", "parse-expression.serica", MODE)
        assert own(-1) == 11 and own(ABI) == own(CALL) == own(LOCAL) == own(-(1 << 63)) == 0
        print(f"P1b 模块 ABI 通过：{references} 个原生参照产物、4 种返回约定、"
              f"{diagnostics} 个位置诊断、混合 8 寄存器、{examples} 个运行时示例及编译器自检")


if __name__ == "__main__":
    main()
