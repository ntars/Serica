# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
"""Check capability declarations and provenance in the bound native module graph."""

import re
from pathlib import Path
from subprocess import run
from tempfile import TemporaryDirectory

from native_artifact import verify_native_console_artifact
from test_p1b_import_paths import OUTPUT, source_text
from test_p1b_module_graph import EXPORT, SOURCE, AST, read_graph, write
from test_p1b_parse import PARSER_IMPORTS
from test_p1b_scan import ROOT, SERICA, compile_source, expect_run


CAPABILITIES = 7 << 60


def masks(read, name_node=0):
    key = CAPABILITIES + (name_node << 1)
    return read(key), read(key + 1)


def function_names(read):
    functions = {}
    for index in range(read(-3)):
        key = EXPORT + (index << 32)
        if read(key + 1) != 104:
            continue
        owner, name = read(key), read(key + 3)
        head = read(AST + name)
        start, width = head >> 32, (head >> 8) & 0xffffff
        location = SOURCE + ((owner - 1) << 32) + start
        spelling = bytes(read(location + i) for i in range(width)).decode()
        functions[spelling] = name
    return functions


def verify_chain(parser, project, artifact):
    files = {
        "main.serica": source_text(["A.serica"]) + "程序=【能力=权限，输出=包（）】。\n",
        "A.serica": source_text(["Z.serica"])
        + "常量 屏=『stdout』。\n常量 权限=array（屏，『stdin』）。\n"
        + "函数 包（）能力=array（『stdin』，『filesystem.write』）=取（）。\n"
        + "函数 未用（）能力=array（『args』）=argumentCount（）。\n",
        "Z.serica": "函数 取（）能力=array（『stdin』，『filesystem.write』）=readInteger（）。\n",
    }
    write(project, files)
    read = read_graph(parser, project, mode="module-effect")
    assert read(-1) == 3
    functions = function_names(read)
    assert masks(read) == (3, 3)
    assert masks(read, functions["取"]) == (10, 2)
    assert masks(read, functions["包"]) == (10, 2)
    assert masks(read, functions["未用"]) == (64, 64)
    compile_source(project / "main.serica", artifact)
    expect_run(artifact, b"9\n", stdin=b"9\n")


def verify_builtins(parser, work):
    path = work / "all.serica"
    fixture, destination = work / "fixture.txt", work / "written.txt"
    fixture.write_text("测试", encoding="utf-8")
    program = (
        "变量 文=argumentText（1）。\n"
        f"变量 源=readFile（『{fixture}』）。\n"
        "变量 表=array（0）。\n赋值 表=arrayPush（表，1）。\n"
        "变量 对=映射（）。\n赋值 对=映射置（对，1，5）。\n"
        "程序=【能力=array（『stdout』，『stdin』，『filesystem.read』，『filesystem.write』"
        "，『array.write』，『map.write』，『args』），"
        f"输出=writeFile（『{destination}』，文）+textByteLength（源）+arrayGet（表，1）"
        "+映射取（对，1，0）+readInteger（）】。\n"
    )
    path.write_text(program, encoding="utf-8")
    read = read_graph(parser, work, path.name, "module-effect")
    assert masks(read) == (127, 127)
    artifact = work / "all"
    compile_source(path, artifact)
    expect_run(artifact, b"18\n", stdin=b"3\n", args=("字",))
    assert destination.read_text(encoding="utf-8") == "字"


def verify_pure_cycle(parser, project):
    files = {
        "main.serica": source_text(["A.serica"]) + OUTPUT.replace("输出=0", "输出=递（3）"),
        "A.serica": "函数 递（数）=ifElse（数==0，0，递（数-1））。\n",
    }
    write(project, files)
    read = read_graph(parser, project, mode="module-effect")
    assert masks(read) == (1, 1) and masks(read, function_names(read)["递"]) == (0, 0)


def verify_diagnostics(parser, project):
    chain = source_text(["A.serica"]) + OUTPUT.replace("输出=0", "输出=包（）")
    leaf = "函数 取（）能力=array（『stdin』）=readInteger（）。\n"
    cases = (
        ({"main.serica": "程序=【输出=0】。\n"}, 1, "main.serica", "【", 4101),
        ({"main.serica": OUTPUT.replace("『stdout』", "『stdin』")}, 1, "main.serica", "输出", 4102),
        ({"main.serica": OUTPUT.replace("『stdout』", "『void』")}, 1, "main.serica", "array", 4103),
        ({"main.serica": OUTPUT.replace("『stdout』", "『stdout』，『stdout』")}, 1, "main.serica", "array", 4104),
        ({"main.serica": OUTPUT.replace("，输出", "，能力=array（『stdout』），输出")}, 1, "main.serica", "能力=array（『stdout』），输出", 4104),
        ({"main.serica": OUTPUT.replace("『stdout』", "1")}, 1, "main.serica", "array", 4105),
        ({"main.serica": source_text(["A.serica"]) + OUTPUT,
          "A.serica": "函数 取（）能力=array（『stdout』）=0。\n"}, 2, "A.serica", "『stdout』", 4103),
        ({"main.serica": source_text(["A.serica"]) + OUTPUT,
          "A.serica": "函数 取（）能力=array（『stdin』，『stdin』）=0。\n"}, 2, "A.serica", "『stdin』）", 4104),
        ({"main.serica": source_text(["A.serica"]) + OUTPUT,
          "A.serica": "函数 取（）能力=1=0。\n"}, 2, "A.serica", "1=", 4105),
        ({"main.serica": source_text(["A.serica"]) + OUTPUT, "A.serica": "函数 取（）=readInteger（）。\n"},
         2, "A.serica", "readInteger", 4106),
        ({"main.serica": chain, "A.serica": source_text(["Z.serica"]) + "函数 包（）=取（）。\n",
          "Z.serica": leaf}, 2, "A.serica", "取（）", 4106),
        ({"main.serica": chain, "A.serica": source_text(["Z.serica"])
          + "函数 包（）能力=array（『stdin』）=取（）。\n",
          "Z.serica": "函数 取（）能力=array（『stdin』，『filesystem.write』）=1。\n"},
         2, "A.serica", "取（）", 4106),
        ({"main.serica": chain, "A.serica": source_text(["Z.serica"])
          + "函数 包（）能力=array（『stdin』）=取（）。\n", "Z.serica": leaf},
         3, "Z.serica", "readInteger", 4102),
    )
    for files, module, name, marker, code in cases:
        write(project, files)
        before = files[name].split(marker, 1)[0]
        expected = -((module << 37) + (len(before.encode()) << 13) + code)
        read = read_graph(parser, project, mode="module-effect")
        actual = read(-1)
        assert actual == expected, (name, marker, code, actual, expected)
        seed = run([SERICA, "check", project / "main.serica"], capture_output=True, text=True, timeout=8)
        row, col = before.count("\n") + 1, len(before.rsplit("\n", 1)[-1]) + 1
        pattern = rf"{re.escape(name)}:{row}:{col}（显示第 \d+ 格）：错误 S-{code}："
        assert seed.returncode == 1 and re.search(pattern, seed.stderr), (name, marker, seed.stderr)
    return len(cases)


def main():
    with TemporaryDirectory(prefix="serica-effect-") as temp:
        work = Path(temp).resolve()
        parser = work / "module-effect"
        compile_source(ROOT / "compiler/parse-expression.serica", parser)
        verify_native_console_artifact(parser, require_runtime_input=True, required_imports=PARSER_IMPORTS)
        verify_chain(parser, work / "chain", work / "chain-reference")
        verify_builtins(parser, work)
        verify_pure_cycle(parser, work / "cycle")
        write(work, {"window.serica": "窗口=【标题=『窗口』，文本=『你好』】。\n"})
        assert masks(read_graph(parser, work, "window.serica", "module-effect")) == (0, 0)
        diagnostics = verify_diagnostics(parser, work / "diagnostics")
        own = read_graph(parser, ROOT / "compiler", "parse-expression.serica", "module-effect")
        assert own(-1) == 11 and masks(own) == (95, 95)
        print(f"P1b 模块效果通过：7 种能力、4 个正例图、{diagnostics} 个位置诊断、编译器自检")


if __name__ == "__main__":
    main()
