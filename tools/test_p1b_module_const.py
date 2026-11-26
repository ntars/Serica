# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
"""Read constant results from the native Serica compiler front end."""

from pathlib import Path
from tempfile import TemporaryDirectory

from native_artifact import verify_native_console_artifact
from test_p1b_import_paths import OUTPUT, source_text
from test_p1b_module_effect import masks
from test_p1b_module_graph import AST, EXPORT, read_graph, write
from test_p1b_module_type import assert_diagnostic
from test_p1b_parse import PARSER_IMPORTS
from test_p1b_scan import ROOT, compile_source, expect_run
from test_p1b_static import ACCEPTED


MODE = "module-const"
VALUE = -(2 << 60)
POOL = -(3 << 60)


def result(read, node):
    return read(VALUE - node * 2), read(VALUE - node * 2 - 1)


def root_fields(read):
    declaration = read(AST + 2)
    while read(AST + declaration) & 255 != 112:
        declaration = read(AST + declaration + 3)
        assert declaration
    name = read(AST + declaration + 1)
    record = read(AST + name + 3)
    first = read(AST + record + 1)
    return first, read(AST + first + 3)


def text(read, node):
    kind, handle = result(read, node)
    assert kind == 2 and handle > 0, (node, kind, handle)
    return decode_text(read, handle)


def decode_text(read, handle):
    size = read(POOL - handle)
    assert read(POOL - handle - 1) == 0 and 0 <= size < 1024
    return bytes(read(POOL - handle - 2 - i) for i in range(size)).decode("utf-8")


def array(read, node):
    kind, handle = result(read, node)
    size = read(POOL - handle)
    assert kind in (4, 5, 6) and handle > 0 and 0 < size < 64
    assert read(POOL - handle - 1) == kind - 3
    return kind, [read(POOL - handle - 2 - i) for i in range(size)]


def check_windows(parser):
    for name, expected in ACCEPTED.items():
        path = ROOT / name
        read = read_graph(parser, path.parent, path.name, MODE)
        assert read(-1) == 1, path
        _, body = root_fields(read)
        assert text(read, body) == expected, path
    return len(ACCEPTED)


def verify_modules(parser, work, artifact):
    main = source_text(["A.serica"]) + (
        "常量 缀=substring（前，1，5）+「\\n」。\n"
        "常量 形=array（缀，『末』）。\n"
        "常量 总=textByteLength（缀）+textLength（缀）+arrayLength（数）+arrayGet（数，1）。\n"
        "程序=【能力=array（『stdout』），输出=ifElse（equals（arrayGet（形，0），缀），总，0）】。\n"
    )
    files = {
        "main.serica": main,
        "A.serica": source_text(["Z.serica"]) + r"常量 前=『甲\n\『𠮷\』\\』。" + "\n",
        "Z.serica": "常量 下=-9223372036854775808。\n常量 数=array（下，7，1+2）。\n",
    }
    write(work, files)
    read = read_graph(parser, work, mode=MODE)
    assert (read(-1), read(-3)) == (3, 6)
    names = [read(EXPORT + (rank << 32) + 3) for rank in range(6)]
    assert result(read, names[0]) == (1, -(1 << 63))
    assert array(read, names[1]) == (4, [-(1 << 63), 7, 3])
    assert text(read, names[2]) == "甲\n『𠮷』\\"
    expected_suffix = "\n『𠮷』\\n"
    assert text(read, names[3]) == expected_suffix
    kind, handles = array(read, names[4])
    assert kind == 5 and [decode_text(read, h) for h in handles] == [expected_suffix, "末"]
    _, field = root_fields(read)
    expected = len(expected_suffix.encode()) + len(expected_suffix) + 3 + 7
    assert result(read, names[5]) == (1, expected)
    assert result(read, field) == (1, expected)
    assert result(read, read(AST + field + 1)) == (1, expected)
    assert read(VALUE) == read(VALUE - 4) == read(POOL) == read(-(1 << 63)) == 0
    compile_source(work / "main.serica", artifact)
    expect_run(artifact, f"{expected}\n".encode())


def verify_control(parser, work, output):
    cases = (
        ("ifElse（1==0 且 1/0==0，1/0，ifElse（1==1 故 1/0==2，7，9））", 1, 7, b"7\n", b""),
        ("notEquals（1<2，4<3）", 3, 1, b"1\n", b""),
        ("readInteger（）+arrayLength（数）", 0, 0, b"12\n", b"9\n"),
    )
    for index, (expression, kind, scalar, expected, stdin) in enumerate(cases):
        src = OUTPUT.replace("输出=0", "输出=" + expression)
        if stdin:
            src = "常量 数=array（3，6，9）。\n" + src.replace("『stdout』", "『stdout』，『stdin』")
        write(work, {"main.serica": src})
        read = read_graph(parser, work, mode=MODE)
        assert read(-1) == 1, expression
        assert result(read, root_fields(read)[1]) == (kind, scalar), expression
        artifact = output / f"control-{index}"
        compile_source(work / "main.serica", artifact)
        expect_run(artifact, expected, stdin=stdin)
    return len(cases)


def verify_integer_edges(parser, work, artifact):
    values = {
        "-1<<63": -(1 << 63),
        "-1>>63": -1,
        "-9223372036854775808%-1": 0,
        "-3037000499*-3037000499": 3037000499 ** 2,
        "-4611686018427387904*2": -(1 << 63),
        "-9223372036854775808+9223372036854775807": -1,
        "textByteAt（『𠮷』，3）": "𠮷".encode()[3],
        "textByteLength（『é』）+textLength（『é』）": 3,
        "arrayLength（array（1/0））": 1,
    }
    source = "常量 数=array（" + "，".join(values) + "）。\n" + OUTPUT.replace("输出=0", "输出=arrayGet（数，0）")
    write(work, {"main.serica": source})
    read = read_graph(parser, work, mode=MODE)
    assert read(-1) == 1
    assert array(read, read(EXPORT + 3)) == (4, list(values.values()))
    compile_source(work / "main.serica", artifact)
    expect_run(artifact, b"-9223372036854775808\n")
    return len(values)


def verify_errors(parser, work):
    cases = (
        ("9223372036854775807+1", "+", 3018),
        ("-9223372036854775808-1", "-1", 3018),
        ("-（-9223372036854775808）", "-（", 3018),
        ("-3037000500*-3037000500", "*", 3018),
        ("-4611686018427387905*2", "*", 3018),
        ("2*-4611686018427387905", "*", 3018),
        ("4611686018427387904*2", "*", 3018),
        ("-9223372036854775808/-1", "/", 3018),
        ("1/0", "/", 3024),
        ("1%0", "%", 3024),
        ("2<<62", "<<", 3018),
        ("1>>64", ">>", 3041),
        ("1<<-1", "<<", 3041),
        ("arrayGet（array（1，2），2）", "arrayGet", 3032),
        ("arrayGet（array（1），-1）", "arrayGet", 3032),
        ("arrayGet（array（1，1/0），0）", "/", 3024),
        ("textByteAt（『字』，3）", "3）", 3072),
        ("ifElse（1==1，『行简』，substring（『字』，0，2））", "substring", 3013),
    )
    for expression, marker, code in cases:
        source = OUTPUT.replace("输出=0", "输出=" + expression)
        assert_diagnostic(parser, work, {"main.serica": source}, 1, "main.serica", marker, code, MODE, VALUE - 2)
    files = {
        "main.serica": source_text(["A.serica"]) + OUTPUT,
        "A.serica": "常量 溢=9223372036854775807+1。\n",
    }
    assert_diagnostic(parser, work, files, 2, "A.serica", "+", 3018, MODE, VALUE - 2)
    return len(cases) + 1


def verify_expansion(parser, work, artifact):
    files = {
        "main.serica": source_text(["A.serica"]) + (
            "常量 数=裹（21）。\n"
            "常量 文=裹（『𠮷』）。\n"
            "常量 惰=挑（数，1/0，1==1）。\n"
            "常量 长=求长（array（1/0））。\n"
            "程序=【能力=array（『stdout』），输出="
            "ifElse（equals（文，『𠮷𠮷』），惰+长+裹（裹（2）），0）】。\n"
        ),
        "A.serica": source_text(["Z.serica"]) + "函数 裹（甲）=折（甲）。\n",
        "Z.serica": (
            "函数 折（甲）=甲+甲。\n"
            "函数 挑（甲，乙，条件）=ifElse（条件，甲，乙）。\n"
            "函数 求长（甲）=arrayLength（甲）。\n"
        ),
    }
    write(work, files)
    read = read_graph(parser, work, mode=MODE)
    assert read(-1) == 3
    exported = [read(EXPORT + (rank << 32) + 3) for rank in range(4, 8)]
    assert result(read, exported[0]) == result(read, exported[2]) == (1, 42)
    assert text(read, exported[1]) == "𠮷𠮷"
    assert result(read, exported[3]) == (1, 1)
    assert result(read, root_fields(read)[1]) == (1, 51)
    compile_source(work / "main.serica", artifact)
    expect_run(artifact, b"51\n")


def verify_computed_capabilities(parser, work, artifact):
    files = {
        "main.serica": source_text(["A.serica"])
        + "常量 权限=合成（标识）。\n程序=【能力=权限，输出=readInteger（）+1】。\n",
        "A.serica": (
            "常量 标识=array（『std』+『out』，『std』+『in』）。\n"
            "函数 合成（组）=array（arrayGet（组，0），arrayGet（组，1），"
            "『filesystem.』+『read』，『filesystem.』+『write』，"
            "『array.』+『write』，『map.』+『write』，substring（『xargs』，1，5））。\n"
        ),
    }
    write(work, files)
    read = read_graph(parser, work, mode=MODE)
    assert read(-1) == 2 and masks(read) == (127, 3)
    capability = root_fields(read)[0]
    kind, handles = array(read, capability)
    assert kind == 5 and [decode_text(read, h) for h in handles] == [
        "stdout", "stdin", "filesystem.read", "filesystem.write", "array.write", "map.write", "args",
    ]
    compile_source(work / "main.serica", artifact)
    expect_run(artifact, b"42\n", stdin=b"41\n")


def verify_shared_arguments(parser, work):
    depth = 45
    source = "函数 双（甲）=甲+甲。\n" + OUTPUT.replace(
        "输出=0", "输出=" + "双（" * depth + "1" + "）" * depth
    )
    write(work, {"main.serica": source})
    read = read_graph(parser, work, mode=MODE)
    assert read(-1) == 1 and result(read, root_fields(read)[1]) == (1, 1 << depth)


def verify_deferred_calls_and_large_frame(parser, work, output):
    cases = (
        (
            "函数 和（数）=ifElse（数<=0，0，和（数-1）+数）。\n"
            "变量 初=readInteger（）。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=和（初）】。\n",
            b"5\n", b"15\n",
        ),
        (
            "函数 加一（数值）能力=array（『stdin』）=数值+1。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=加一（readInteger（））】。\n",
            b"41\n", b"42\n",
        ),
        (
            "函数 加一（数值）能力=array（『stdin』）=数值+1。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=加一（1+readInteger（））】。\n",
            b"40\n", b"42\n",
        ),
        (
            "".join(f"变量 数值{index}={index}。\n" for index in range(501))
            + "变量 表=array（4）。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=数值500+arrayGet（表，0）+readInteger（）】。\n",
            b"3\n", b"507\n",
        ),
        (
            "".join(f"变量 数值{index}=0。\n" for index in range(480))
            + (ROOT / "examples/p1b-runtime-text-concat-five-levels.serica").read_text(encoding="utf-8"),
            "行简\n".encode(), "行简甲乙丙丁戊\n".encode(),
        ),
    )
    for index, (source, stdin, expected) in enumerate(cases):
        write(work, {"main.serica": source})
        read = read_graph(parser, work, mode=MODE)
        assert read(-1) == 1 and result(read, root_fields(read)[1]) == (0, 0)
        artifact = output / f"deferred-{index}"
        compile_source(work / "main.serica", artifact)
        expect_run(artifact, expected, stdin=stdin)
    return len(cases)


def verify_expansion_errors(parser, work):
    allowed = OUTPUT.replace("『stdout』", "『stdout』，『stdin』")
    cases = (
        ("函数 忽（甲）=1。\n", "忽（2）", "甲", 3065),
        ("函数 续（甲）=甲+甲。\n", "续（readInteger（））", "readInteger", 4106),
        ("函数 忽（甲）=1。\n", "忽（readInteger（））", "甲", 3065),
        (
            "函数 挑（甲，乙，条件）=ifElse（条件，甲，乙）。\n",
            "挑（『行』，substring（『简』，0，7），1==1）", "substring", 3013,
        ),
    )
    for prefix, expression, marker, code in cases:
        source = prefix + allowed.replace("输出=0", "输出=" + expression)
        assert_diagnostic(parser, work, {"main.serica": source}, 1, "main.serica", marker, code, MODE, VALUE - 2)
    modules = {
        "main.serica": source_text(["A.serica"]) + OUTPUT.replace("输出=0", "输出=折（1<2）"),
        "A.serica": "函数 折（甲）=甲+甲。\n",
    }
    assert_diagnostic(parser, work, modules, 2, "A.serica", "+", 3023, MODE, VALUE - 2)
    return len(cases) + 1


def verify_capability_errors(parser, work):
    cases = (
        ("array（『std』+『zzz』）", "0", "array", 4103),
        ("array（『stdout』，『std』+『out』）", "0", "array", 4104),
        ("array（1）", "0", "array", 4105),
        ("array（readText（））", "0", "array", 4105),
        ("array（『std』+『out』）", "readInteger（）", "readInteger", 4102),
    )
    for capabilities, expression, marker, code in cases:
        source = f"程序=【能力={capabilities}，输出={expression}】。\n"
        assert_diagnostic(parser, work, {"main.serica": source}, 1, "main.serica", marker, code, MODE, VALUE - 2)
    return len(cases)


def verify_pure_blocks(parser, work, output):
    project = work / "modules"
    files = {
        "main.serica": source_text(["state.serica"]) + (
            "常量 主=计算（5）+计算（2）。\n"
            "常量 双=分支（1）+分支（4）。\n"
            "常量 换=更新（5）。\n"
            "常量 类型=转型（40）。\n"
            "常量 空=跳过（）。\n"
            "常量 守=保护（3）。\n"
            + OUTPUT.replace("输出=0", "输出=主+双+换+类型+空+守")
        ),
        "state.serica": (
            "函数 计算（料）【变量 原=料+1。变量 副=原。赋值 原=原*2。"
            "若（原>10）【变量 影=副+3。赋值 影=影*2。返回 原+影。】。返回 原+副。】。\n"
            "函数 分支（数）【变量 局=数+1。若（局>3）【变量 同=7。赋值 局=局+同。返回 局。】"
            "否【变量 同=3。赋值 局=局*同。返回 局。】。】。\n"
            "函数 更新（甲）【变量 最初=甲+0。赋值 甲=甲+1。返回 最初+甲。】。\n"
            "函数 转型（料）【变量 形=『文本』。赋值 形=料+1。返回 形+1。】。\n"
            "函数 跳过（）【变量 未用=1/0。返回 42。】。\n"
            "函数 保护（数）【若（数>0）【返回 数。】。返回 0。】。\n"
        ),
    }
    write(project, files)
    read = read_graph(parser, project, mode=MODE)
    assert read(-1) == 2
    names = [read(EXPORT + (rank << 32) + 3) for rank in range(6, 12)]
    assert [result(read, n) for n in names] == [(1, n) for n in (39, 18, 11, 42, 42, 3)]
    assert result(read, root_fields(read)[1]) == (1, 155)
    artifact = output / "blocks-reference"
    compile_source(project / "main.serica", artifact)
    expect_run(artifact, b"155\n")

    project = work / "text"
    source = (
        "函数 选文（数）【变量 前=『行简』。若（数>0）【变量 支=『𠮷』。返回 前+支。】"
        "否【变量 支=『乙』。返回 前+支。】。】。\n"
        + OUTPUT.replace("输出=0", "输出=选文（1）+选文（0）")
    )
    write(project, {"main.serica": source})
    read = read_graph(parser, project, mode=MODE)
    assert read(-1) == 1 and text(read, root_fields(read)[1]) == "行简𠮷行简乙"
    artifact = output / "block-text-reference"
    compile_source(project / "main.serica", artifact)
    expect_run(artifact, "行简𠮷行简乙\n".encode())

    project = work / "runtime"
    source = (
        "函数 码长（源）【变量 副=源。返回 textLength（副）。】。\n"
        "函数 累（限）【变量 数=0。凡（数<限）【赋值 数=数+1。】。返回 数。】。\n"
        "函数 和（数）【若（数<=0）【返回 0。】。返回 和（数-1）+数。】。\n"
        + OUTPUT.replace("输出=0", "输出=码长（『𠮷字』）+累（3）+和（3）")
    )
    write(project, {"main.serica": source})
    read = read_graph(parser, project, mode=MODE)
    assert read(-1) == 1 and result(read, root_fields(read)[1]) == (0, 0)
    artifact = output / "block-runtime-reference"
    compile_source(project / "main.serica", artifact)
    expect_run(artifact, b"11\n")


def verify_block_errors(parser, work):
    capabilities = OUTPUT.replace("『stdout』", "『stdout』，『stdin』")
    cases = (
        ("函数 无（）【变量 终=1。】。\n", "无（）", "无（）", 3068),
        ("函数 无（甲）【若（甲>0）【返回 1。】。】。\n", "无（1）", "若", 3068),
        ("函数 无（）【返回 1。变量 后=2。】。\n", "无（）", "后", 3068),
        ("函数 无（）【若（1==1）【返回 1。】否【返回 2。】。变量 尾=3。】。\n", "无（）", "尾", 3068),
        ("函数 忘（甲）【变量 未用=甲+1。返回 0。】。\n", "忘（1）", "甲", 3065),
        ("函数 溢（数）【返回 数+1。】。\n", "溢（9223372036854775807）", "+", 3018),
        ("函数 错（值）【若（值>0）【返回 1。】否【返回 『错』。】。】。\n", "错（1）", "若", 3023),
        ("函数 错（值）【若（值）【返回 1。】。返回 0。】。\n", "错（1）", "若", 3015),
        (
            "函数 错（）能力=array（『stdin』）【变量 局=readInteger（）。返回 局。】。\n",
            "错（）", "readInteger", 3068,
        ),
        (
            "函数 错（）能力=array（『stdin』）【变量 局=0。赋值 局=readInteger（）。返回 局。】。\n",
            "错（）", "readInteger", 3068,
        ),
    )
    for prefix, expression, marker, code in cases:
        source = prefix + capabilities.replace("输出=0", "输出=" + expression)
        assert_diagnostic(parser, work, {"main.serica": source}, 1, "main.serica", marker, code, MODE, VALUE - 2)
    files = {
        "main.serica": source_text(["invalid.serica"]) + OUTPUT.replace("输出=0", "输出=错（1）"),
        "invalid.serica": "函数 错（值）【若（值>0）【返回 『行简』。】。返回 substring（『字』，0，2）。】。\n",
    }
    assert_diagnostic(parser, work, files, 2, "invalid.serica", "substring", 3013, MODE, VALUE - 2)
    return len(cases) + 1


def main():
    with TemporaryDirectory(prefix="serica-const-") as directory:
        work = Path(directory).resolve()
        parser = work / "module-const"
        compile_source(ROOT / "compiler/parse-expression.serica", parser)
        verify_native_console_artifact(parser, require_runtime_input=True, required_imports=PARSER_IMPORTS)
        windows = check_windows(parser)
        verify_modules(parser, work / "modules", work / "modules-reference")
        edges = verify_integer_edges(parser, work / "integer", work / "integer-reference")
        console = verify_control(parser, work / "control", work)
        errors = verify_errors(parser, work / "errors")
        verify_expansion(parser, work / "expansion", work / "expansion-reference")
        verify_computed_capabilities(parser, work / "capabilities", work / "capabilities-reference")
        verify_shared_arguments(parser, work / "shared-arguments")
        deferred = verify_deferred_calls_and_large_frame(parser, work / "deferred", work)
        expansion_errors = verify_expansion_errors(parser, work / "expansion-errors")
        capability_errors = verify_capability_errors(parser, work / "capability-errors")
        verify_pure_blocks(parser, work / "blocks", work)
        block_errors = verify_block_errors(parser, work / "block-errors")
        own = read_graph(parser, ROOT / "compiler", "parse-expression.serica", MODE)
        assert own(-1) == 11
        print(f"P1b 模块静态求值通过：{windows} 个窗口文本、{edges} 个边界值、"
              f"{console + 2} 个运行参照产物、{errors} 个位置诊断、跨文件常量及编译器自检")
        print(f"P1b 静态调用通过：{deferred + 2} 个运行参照产物、{expansion_errors} 个调用位置诊断、"
              f"{capability_errors} 个能力位置诊断、7 种计算得出的能力、45 层共享实参")
        print(f"P1b 块函数降级通过：3 个运行参照产物、{block_errors} 个位置诊断、"
              "赋值版本与分支隔离、延迟初值、循环/递归/借用文本留给运行时")


if __name__ == "__main__":
    main()
