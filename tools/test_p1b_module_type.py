# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
"""Compare bound type shapes and diagnostics with native stage0 programs."""

import re
from pathlib import Path
from subprocess import run
from tempfile import TemporaryDirectory

from native_artifact import verify_native_console_artifact
from test_p1b_import_paths import OUTPUT, source_text
from test_p1b_module_effect import function_names
from test_p1b_module_graph import AST, read_graph, write
from test_p1b_parse import PARSER_IMPORTS
from test_p1b_scan import ROOT, SERICA, compile_source, expect_run


TYPE = -(1 << 60)


def check_types(parser, project, files, expected):
    write(project, files)
    read = read_graph(parser, project, mode="module-type")
    assert read(-1) == len(files)
    roots = sorted(read((rank << 32) + 2) for rank in range(len(files)))
    graph = read_graph(parser, project)
    descriptors = {}
    for node in range(roots[0], roots[1] if len(roots) > 1 else 1 << 60, 4):
        header = graph(AST + node)
        if not header:
            break
        descriptors[header >> 32, header & 255] = node
    source = files["main.serica"].encode()
    for marker, kind, shape in expected:
        offset = source.index(marker.encode())
        node = descriptors[offset, kind]
        assert read(TYPE - node) == shape, (marker, kind, shape, read(TYPE - node))
    assert read(TYPE) == read(TYPE - 2) == read(-(1 << 63)) == 0
    return read


def verify_types(parser, project, artifact):
    project.mkdir(parents=True, exist_ok=True)
    append_path = project / "type-append.txt"
    remove_path = project / "type-remove.txt"
    rename_from = project / "type-from.txt"
    rename_to = project / "type-to.txt"
    remove_path.write_text("删", encoding="utf-8")
    rename_from.write_text("移", encoding="utf-8")
    files = {
        "main.serica": source_text(["A.serica"])
        + "常量 合=先（）+2。\n"
        + "变量 标=equals（恒（文），『𠮷』）。\n"
        + "变量 列=array（1，arrayGet（底，1））。\n"
        + "变量 旗=array（标，equals（0，2））。\n"
        + "变量 文列=array（文，『𠮷』）。\n"
        + "变量 值=成功（textLength（恒（文）））。\n"
        + "变量 字=成功（文）。\n"
        + "变量 环境=readEnvironment（『SERICA_MODULE_TYPE_PROBE』）。\n"
        + "变量 对=映射（）。\n"
        + f"变量 追=appendFile（『{append_path}』，文）。\n"
        + f"变量 删=removeFile（『{remove_path}』）。\n"
        + f"变量 改=renameFile（『{rename_from}』，『{rename_to}』）。\n"
        + "若（标）【赋值 列=arrayPush（列，合）。】。\n"
        + "程序=【能力=array（『stdout』，『args』，『array.write』，『filesystem.write』），"
        "输出=ifElse（arrayGet（旗，0），arrayLength（列）+结果值（值）+累（3），0）】。\n",
        "A.serica": source_text(["Z.serica"])
        + "函数 先（）=后（）。\n"
        + "函数 恒（甲）=甲。\n"
        + "函数 累（甲）【变量 和=0。凡（和<甲）【赋值 和=和+1。】。返回 和。】。\n",
        "Z.serica": "常量 文=『𠮷』。\n常量 底=array（1，2）。\n函数 后（）=arrayGet（底，0）。\n",
    }
    expected = (
        ("合=", 3, 1), ("标=", 3, 3), ("列=", 3, 4), ("旗=", 3, 6),
        ("文列=", 3, 5), ("值=", 3, 10), ("字=", 3, 11), ("对=", 3, 7),
        ("追=", 3, 1), ("删=", 3, 1), ("改=", 3, 1),
        ("先（）+2", 4, 1), ("恒（文）", 4, 2), ("底，1", 3, 4),
        ("结果值（值）", 4, 1), ("ifElse（arrayGet", 4, 1), ("【能力=", 5, 8),
    )
    read = check_types(parser, project, files, expected)
    functions = function_names(read)
    for name, shape in (("先", 1), ("后", 1), ("累", 1), ("恒", 0)):
        assert read(TYPE - functions[name]) == shape
    compile_source(project / "main.serica", artifact)
    expect_run(artifact, b"7\n")
    assert append_path.read_text(encoding="utf-8") == "𠮷"
    assert not remove_path.exists() and rename_to.read_text(encoding="utf-8") == "移"
    return len(expected) + len(functions)


def verify_forward_and_polymorphic(parser, project, artifact):
    # Every earlier declaration initially depends on the later one.
    chain = "".join(f"函数 接{i}（）=接{i + 1}（）。\n" for i in range(24))
    source = (
        chain + "函数 接24（）=40。\n函数 未用（）=1+『字』。\n函数 恒（甲）=甲。\n"
        "函数 选择结果（数）=ifElse（数<0，失败（17），后果（））。\n"
        "函数 后果（）=成功（2）。\n"
        "常量 文=恒（『𠮷』）。\n变量 数=恒（接0（））。\n"
        "变量 选=ifElse（equals（文，『𠮷』），数，0）。\n"
        "变量 结果=选择结果（选）。\n"
        + OUTPUT.replace("输出=0", "输出=选+结果值（结果）")
    )
    expected = (("接0（）=接1", 3, 1), ("数=", 3, 1), ("文=", 3, 2), ("选=", 3, 1), ("结果=选择", 3, 10))
    check_types(parser, project, {"main.serica": source}, expected)
    compile_source(project / "main.serica", artifact)
    expect_run(artifact, b"42\n")
    return len(expected)


def verify_recursive(parser, project, artifact):
    source = (
        "函数 和（数）=ifElse（数<=0，0，和（数-1）+数）。\n"
        "变量 输入=readInteger（）。\n"
        "程序=【能力=array（『stdout』，『stdin』），输出=和（输入）】。\n"
    )
    expected = (("和（数）=", 3, 1), ("和（数-1）", 4, 1), ("和（输入）", 4, 1))
    check_types(parser, project, {"main.serica": source}, expected)
    compile_source(project / "main.serica", artifact)
    expect_run(artifact, b"15\n", stdin=b"5\n")
    return len(expected)


def verify_diagnostics(parser, project):
    ordinary = OUTPUT.replace("『stdout』", "『stdout』，『stdin』，『filesystem.write』")
    cases = (
        ("", "1+『字』", "+", 3023),
        ("", "-（『字』）", "-", 3023),
        ("", "『字』==『字』", "==", 3023),
        ("", "1 且 0<2", "且", 3015),
        ("", "ifElse（1，2，3）", "ifElse", 3015),
        ("", "ifElse（1==1，2，『字』）", "ifElse", 3023),
        ("", "textByteAt（『字』，『字』）", "『字』）", 3072),
        ("", "substring（『字』，『字』，1）", "『字』，1", 3012),
        ("", "arrayLength（1）", "arrayLength", 3036),
        ("", "arrayGet（array（1），『字』）", "arrayGet", 3034),
        ("", "arrayLength（array（1，『字』））", "array（1", 3031),
        ("", "arrayLength（array（））", "array（）", 3030),
        ("", "是否失败（1）", "是否失败", 3013),
        ("", "映射长（1）", "映射长", 3011),
        ("", "bitAnd（1，『字』）", "bitAnd", 3021),
        ("", "readInteger（1）", "readInteger", 3043),
        ("", "appendFile（『x』）", "appendFile", 3071),
        ("", "appendFile（1，『x』）", "1，『x』", 3071),
        ("", "appendFile（『x』，1）", "1）", 3071),
        ("", "removeFile（『x』，『y』）", "removeFile", 3075),
        ("", "removeFile（1）", "1", 3075),
                ("", "renameFile（『x』，1）", "1）", 3076),
        ("变量 数=1。\n赋值 数=『字』。\n", "数", "『字』", 3023),
        ("若（1）【】。\n", "0", "1）", 3015),
        ("", "array（1）", "array（1", 3023),
    )
    for prefix, expression, marker, code in cases:
        source = prefix + ordinary.replace("输出=0", "输出=" + expression)
        assert_diagnostic(parser, project, {"main.serica": source}, 1, "main.serica", marker, code)
    files = {
        "main.serica": source_text(["A.serica"]) + OUTPUT.replace("输出=0", "输出=外（）"),
        "A.serica": source_text(["Z.serica"]) + "函数 外（）=内（）。\n",
        "Z.serica": "函数 内（）=bitXor（1，『字』）。\n",
    }
    assert_diagnostic(parser, project, files, 3, "Z.serica", "bitXor", 3021)
    file_size_source = ordinary.replace("输出=0", "输出=fileSize（1）")
    write(project, {"main.serica": file_size_source})
    read = read_graph(parser, project, mode="module-type")
    position = file_size_source.index("fileSize（1）") + len("fileSize（1）") + 25
    expected = -((1 << 37) + (position << 13) + 3080)
    assert read(-1) == expected, (read(-1), expected)
    exists_source = ordinary.replace("输出=0", "输出=exists（1）")
    write(project, {"main.serica": exists_source})
    read = read_graph(parser, project, mode="module-type")
    position = exists_source.index("exists（1）") + len("exists（1）") + 36
    expected = -((1 << 37) + (position << 13) + 3081)
    assert read(-1) == expected, (read(-1), expected)
    directory_source = ordinary.replace("输出=0", "输出=isDirectory（1）")
    write(project, {"main.serica": directory_source})
    read = read_graph(parser, project, mode="module-type")
    position = directory_source.index("isDirectory（1）") + len("isDirectory（1）") + 36
    expected = -((1 << 37) + (position << 13) + 3081)
    assert read(-1) == expected, (read(-1), expected)
    return len(cases) + 4


def assert_diagnostic(parser, project, files, module, name, marker, code, mode="module-type", query=TYPE - 1):
    write(project, files)
    before = files[name].split(marker, 1)[0]
    expected = -((module << 37) + (len(before.encode()) << 13) + code)
    read = read_graph(parser, project, mode=mode)
    actual = read(-1)
    assert actual == expected, (name, marker, code, actual, expected)
    assert read(query) == expected
    seed = run([SERICA, "check", project / "main.serica"], capture_output=True, text=True, timeout=8)
    row, col = before.count("\n") + 1, len(before.rsplit("\n", 1)[-1]) + 1
    pattern = rf"{re.escape(name)}:{row}:{col}（显示第 \d+ 格）：错误 S-{code}："
    assert seed.returncode == 1 and re.search(pattern, seed.stderr), (name, marker, seed.stderr)


def main():
    with TemporaryDirectory(prefix="serica-type-") as temp:
        work = Path(temp).resolve()
        parser = work / "module-type"
        compile_source(ROOT / "compiler/parse-expression.serica", parser)
        verify_native_console_artifact(parser, require_runtime_input=True, required_imports=PARSER_IMPORTS)
        queries = verify_types(parser, work / "types", work / "types-reference")
        queries += verify_forward_and_polymorphic(parser, work / "forward", work / "forward-reference")
        queries += verify_recursive(parser, work / "recursive", work / "recursive-reference")
        diagnostics = verify_diagnostics(parser, work / "diagnostics")
        own = read_graph(parser, ROOT / "compiler", "parse-expression.serica", "module-type")
        assert own(-1) == 11
        print(f"P1b 模块类型通过：{queries} 个类型查询、{diagnostics} 个位置诊断、3 个原生参照产物、编译器自检")


if __name__ == "__main__":
    main()
