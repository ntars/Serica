# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
"""Build signed CLI executables using the Serica byte emitter."""

import os
import signal
import struct
from pathlib import Path
from random import Random
from resource import RLIMIT_FSIZE, RLIMIT_NOFILE, getrlimit, setrlimit
from subprocess import DEVNULL, PIPE, Popen, run
from tempfile import TemporaryDirectory

from native_artifact import verify_native_console_artifact
from test_p1b_import_paths import source_text
from test_p1b_module_graph import write
from test_p1b_module_read import primitive_source
from test_p1b_scan import ROOT, SERICA, compile_source, expect_run


BUILD = ROOT / "tools/build-serica-native"
TOOLS = ROOT / "build/serica-native"


def program(expression, capabilities="array（『stdout』）"):
    return f"程序=【能力={capabilities}，输出={expression}】。\n"


def build(project, output):
    return run([BUILD, project / "main.serica", "-o", output], capture_output=True, text=True, timeout=20)


def verify_layout(artifact, constant_bytes, *, runtime=False, reads=False, instructions=(), imports=()):
    required = instructions if runtime else ("adr\tx19,", "blr\tx16", "subs\tx20, x20, x0")
    verify_native_console_artifact(
        artifact, require_integer=runtime, require_runtime_input=reads,
        required_instructions=required, required_imports=imports,
    )
    image = artifact.read_bytes()
    magic, cpu, subtype, kind, count, length, flags, _ = struct.unpack_from("<8I", image)
    assert (magic, cpu, subtype, kind, flags) == (0xFEEDFACF, 0x0100000C, 0, 2, 0x200084)
    assert count == 13 and length < 2048 - 32  # Twelve emitted commands plus the system signature.
    cursor = 32
    segments = {}
    entry = None
    for _ in range(count):
        command, size = struct.unpack_from("<II", image, cursor)
        assert size >= 8 and size % 8 == 0 and cursor + size <= 32 + length
        if command == 0x19:
            name, va, vs, fo, fs, mp, ip, ns, _ = struct.unpack_from("<16s4Q4I", image, cursor + 8)
            sections = {}
            for index in range(ns):
                at = cursor + 72 + index * 80
                label, owner, address, width, offset, alignment = struct.unpack_from("<16s16sQQII", image, at)
                assert owner == name and offset % (1 << alignment) == 0
                assert va <= address and address + width <= va + vs and fo <= offset and offset + width <= fo + fs
                sections[label.rstrip(b"\0")] = (offset, width)
            segments[name.rstrip(b"\0")] = (fo, fs, mp, ip, sections)
        if command == 0x80000028:
            entry = struct.unpack_from("<Q", image, cursor + 8)[0]
        cursor += size
    assert cursor == 32 + length and entry == 2048
    assert set(segments) == {b"__PAGEZERO", b"__TEXT", b"__DATA", b"__LINKEDIT"}
    assert [segments[seg][2:4] for seg in (b"__PAGEZERO", b"__TEXT", b"__DATA", b"__LINKEDIT")] == [
        (0, 0), (5, 5), (3, 3), (1, 1),
    ]
    header, text_size, _, _, sections = segments[b"__TEXT"]
    assert header == 0 and text_size % 16384 == 0
    code_offset, code_width = sections[b"__text"]
    assert code_offset == entry and code_width % 4 == 0
    assert code_width > 84 if runtime else code_width == 84
    offset, width = sections[b"__const"]
    assert image[offset:offset + width] == constant_bytes
    _, got_width = segments[b"__DATA"][4][b"__got"]
    assert got_width == 8 * (2 + int(reads) + len(imports))
    assert segments[b"__DATA"][0] == text_size
    assert segments[b"__LINKEDIT"][0] == text_size + 16384
    if not runtime:
        expect_run(artifact, constant_bytes)
    return text_size


def verify_programs(work):
    cases = (
        ("-9223372036854775808", b"-9223372036854775808\n"),
        ("9223372036854775807", b"9223372036854775807\n"),
        ("-17+17", b"0\n"),
        ("equals（『行』，『行』）", b"1\n"),
        ("notEquals（0，0）", b"0\n"),
        ("『』", b"\n"),
        ("『é𠮷\\n简』+「\\n」", "é𠮷\n简\\n\n".encode()),
    )
    for index, (expression, expected) in enumerate(cases):
        project = work / f"static {index}"
        write(project, {"main.serica": program(expression)})
        output = project / "native artifact"
        result = build(project, output)
        assert result.returncode == 0, (expression, result.stdout, result.stderr)
        verify_layout(output, expected)
        reference = project / "stage0"
        compile_source(project / "main.serica", reference)
        expect_run(reference, expected)
    return len(cases)


def verify_modules(work):
    project = work / "modules"
    write(project, {
        "main.serica": source_text(["A.serica"]) + program("包（21）", "许可"),
        "A.serica": "常量 许可=array（『std』+『out』）。\n函数 包（数）=数+数。\n",
    })
    artifact = project / "program"
    first = build(project, artifact)
    assert first.returncode == 0, first.stderr
    verify_layout(artifact, b"42\n")
    snapshot = artifact.read_bytes()
    second = build(project, artifact)
    assert second.returncode == 0 and snapshot == artifact.read_bytes(), second.stderr
    reference = project / "stage0"
    compile_source(project / "main.serica", reference)
    expect_run(reference, b"42\n")


def verify_multipage(work):
    project = work / "large"
    payload = "行简𠮷abc" * 5600
    write(project, {"main.serica": program(f"『{payload}』")})
    output = project / "program"
    result = build(project, output)
    assert result.returncode == 0, (result.stdout, result.stderr)
    expected = (payload + "\n").encode()
    assert len(expected) > 65536 and verify_layout(output, expected) >= 81920
    return len(expected)


def verify_rejections(work):
    project = work / "rejected"
    output = project / "existing"
    cases = (
        (program("1/0"), "S-3024"),
        ("程序=【输出=0】。\n", "S-4101"),
        ("程序=【能力=array（『stdout』）】。\n", "S-3006"),
        (program("0", "array（『stdin』）"), "S-4102"),
        ("变量 字=array（『行简』）。\n变量 借=字。\n" + program("arrayLength（借）"), "S-3033"),
        (program("textConcat（readText（），『尾』）", "array（『stdout』，『stdin』）"), "S-3080"),
        ("窗口=【标题=『窗口』，文本=『行简』】。\n", "S-3080"),
        ("程序=【能力=array（『stdout』），输出=0，额外=1】。\n", "S-3080"),
    )
    for source, code in cases:
        write(project, {"main.serica": source})
        output.write_bytes(b"existing\0binary")
        completed = build(project, output)
        assert completed.returncode == 1 and code in completed.stderr, (source, completed)
        assert output.read_bytes() == b"existing\0binary"
        assert not list(project.glob(".serica-native.*"))
    return len(cases)


def runtime_pair(work, name, files, runs, *, reads=True, instructions=(), imports=(), rebuild=False, pool=b""):
    project = work / name
    write(project, files)
    output = project / "native"
    result = build(project, output)
    assert result.returncode == 0, (name, result.stdout, result.stderr)
    size = verify_layout(output, pool, runtime=True, reads=reads, instructions=instructions, imports=imports)
    if rebuild:
        previous = output.read_bytes()
        result = build(project, output)
        assert result.returncode == 0 and output.read_bytes() == previous, result
    reference = project / "stage0"
    compile_source(project / "main.serica", reference)
    for stdin, expected, status, *arguments in runs:
        args = arguments[0] if arguments else ()
        expect_run(output, expected, stdin=stdin, status=status, args=args)
        expect_run(reference, expected, stdin=stdin, status=status, args=args)
    return size


def verify_runtime(work):
    stdin_caps = "array（『stdout』，『stdin』）"
    sample_cases = (
        ("integer-control-flow", ((b"", b"8\n", 0),), False, ("b.eq\t", "str\tx0, [x28]")),
        ("loop-control", ((b"", b"112233\n", 0),), False, ("b.eq\t", "b\t")),
        ("block-loop-control", ((b"9\n", b"9\n", 0), (b"1\n", b"1\n", 0)), True,
         ("bl\t", "ret", "ldp\tx29, x30")),
        ("recursive-function", ((b"5\n", b"120\n", 0), (b"0\n", b"1\n", 0), (b"21\n", b"", 1)), True,
         ("smulh\t", "bl\t", "stp\tx29, x30")),
        ("read-multiple", ((b"2\n-3\r\n", b"-3\n", 0), (b"2\n", b"", 1)), True, ("blr\tx16",)),
    )
    for name, runs, reads, instructions in sample_cases:
        source = ROOT / "examples" / f"p1b-runtime-{name}.serica"
        runtime_pair(work, name, {"main.serica": source.read_text(encoding="utf-8")}, runs,
                     reads=reads, instructions=instructions)

    echo = program("readInteger（）", stdin_caps)
    input_runs = tuple((value, expected, 0) for value, expected in (
        (b"9223372036854775807", b"9223372036854775807\n"),
        (b"-9223372036854775808\r\n", b"-9223372036854775808\n"),
        (b"0" * 63 + b"\n", b"0\n"),
    )) + tuple((value, b"", 1) for value in (
        b"", b"\n", b"\r\n", b"-\n", b"+1\n", b"1\r", b"1x\n", b"0" * 64,
        b"9223372036854775808", b"-9223372036854775809",
    ))
    runtime_pair(work, "integer-input", {"main.serica": echo}, input_runs, instructions=("ldrb\t", "udiv\t"))

    operations = ("数+1", "数-1", "数*2", "-数", "数/（号-5）", "数/（号-5）",
                  "数%（号-9）", "数<<（数-1）")
    expression = "数>>（数-1）"
    for index, operation in reversed(tuple(enumerate(operations))):
        expression = f"ifElse（号=={index}，{operation}，{expression}）"
    checked = "变量 号=readInteger（）。\n变量 数=readInteger（）。\n" + program(expression, stdin_caps)
    arithmetic_success = ((0, 41, 42), (1, 4, 3), (2, -2, -4), (3, -42, 42), (4, 9, -9),
                          (6, -7, -1), (7, 4, 32), (8, 4, 0))
    arithmetic_failures = ((0, 2**63 - 1), (1, -2**63), (2, 2**62), (3, -2**63), (4, -2**63),
                           (5, 0), (7, 0), (7, 64), (8, 65))
    runtime_pair(
        work, "arithmetic", {"main.serica": checked},
        tuple((f"{mode}\n{value}\n".encode(), f"{expected}\n".encode(), 0)
              for mode, value, expected in arithmetic_success)
        + tuple((f"{mode}\n{value}\n".encode(), b"", 1) for mode, value in arithmetic_failures),
        instructions=("sdiv\t", "msub\t", "smulh\t", "lsl\t", "asr\t", "b.vc\t"),
    )

    lazy = "变量 数=readInteger（）。\n" + program(
        "数==0 故 数>0 且 100/数>1", stdin_caps,
    )
    runtime_pair(work, "logical", {"main.serica": lazy},
                 ((b"0\n", b"1\n", 0), (b"-9\n", b"0\n", 0),
                  (b"5\n", b"1\n", 0), (b"50\n", b"1\n", 0), (b"100\n", b"0\n", 0)),
                 instructions=("cset\t", "b.ne\t"))

    main = (
        source_text(["functions.serica"]) + "变量 数=readInteger（）。\n"
        + program("八（基数，2，3，4，5，6，7，奇（数））", stdin_caps)
    )
    functions = (
        "常量 基数=1。\n"
        "函数 奇（数）=ifElse（数<=0，0，偶（数-1））。\n"
        "函数 偶（数）=ifElse（数<=0，1，奇（数-1））。\n"
        "函数 八（甲，乙，丙，丁，戊，己，庚，辛）【变量 合=甲+乙+丙+丁+戊+己+庚。返回 合+辛。】。\n"
    )
    runtime_pair(work, "runtime-module", {"main.serica": main, "functions.serica": functions},
                 ((b"5\n", b"29\n", 0), (b"6\n", b"28\n", 0)),
                 instructions=("ldr\tx7, [sp]", "bl\t", "ret"), rebuild=True)

    large = "变量 数=0。\n" + "赋值 数=数+1。\n" * 400 + program("数")
    assert runtime_pair(work, "large-code", {"main.serica": large}, ((b"", b"400\n", 0),),
                        reads=False) >= 32768
    return len(sample_cases) + 5


def verify_arrays(work):
    memory = ("_mmap", "_munmap")
    changed = "array（『stdout』，『stdin』，『array.write』）"
    growth = (
        "变量 甲=array（7，11，13）。\n变量 乙=array（101）。\n"
        "变量 总=readInteger（）。\n变量 位=0。\n"
        "凡（位<总）【赋值 甲=arrayPush（甲，位*位+3）。赋值 位=位+1。】。\n"
        "赋值 甲=arraySet（甲，1，arrayGet（乙，0））。\n"
        "赋值 乙=arrayPush（乙，arrayLength（甲））。\n"
        + program("arrayGet（甲，0）*1000000+arrayGet（甲，1）*10000+arrayGet（甲，2）*1000+"
                  "arrayGet（甲，arrayLength（甲）-1）+arrayGet（乙，1）", changed)
    )
    runtime_pair(
        work, "array-growth", {"main.serica": growth},
        tuple((f"{n}\n".encode(), f"{8023000 + (13 if n == 0 else (n - 1)**2 + 3) + 3 + n}\n".encode(), 0)
              for n in (0, 1, 4, 100, 20000)),
        imports=memory, instructions=("ldr\tx4, [x3, x2, lsl #3]", "str\tx4, [x23, x2, lsl #3]", "bl\t"),
        rebuild=True,
    )
    source = ROOT / "examples/p1b-runtime-integer-array.serica"
    runtime_pair(work, "array-set-order", {"main.serica": source.read_text(encoding="utf-8")},
                 ((b"5\n1\n42\n", b"42\n", 0), (b"5\n0\n9\n", b"20\n", 0),
                  (b"5\n-1\n0\n", b"", 1), (b"5\n3\n0\n", b"", 1)),
                 imports=memory, instructions=("b.lo\t", "str\tx0, [x1, x2, lsl #3]"))
    for artifact in ("native", "stage0"):
        for invalid in (b"-1", b"3"):
            process = Popen([work / "array-set-order" / artifact], stdin=PIPE, stdout=PIPE, stderr=PIPE)
            try:
                process.stdin.write(b"5\n" + invalid + b"\n")
                process.stdin.flush()
                # Keep stdin open. Reaching the RHS read would block.
                assert process.wait(timeout=4) == 1
                assert process.stdout.read() == b"" and process.stderr.read() == b""
            finally:
                if process.poll() is None:
                    process.kill()
                process.communicate()

    source = ROOT / "examples/p1b-runtime-boolean-array.serica"
    runtime_pair(work, "array-boolean", {"main.serica": source.read_text(encoding="utf-8")},
                 ((b"3\n1\n1\n2\n", b"4203\n", 0), (b"-1\n1\n0\n1\n", b"-97\n", 0),
                  (b"0\n1\n1\n3\n", b"", 1), (b"1\n0\n0\n-1\n", b"", 1)),
                 imports=memory, instructions=("ldr\tx0, [x1, x0, lsl #3]",))

    readonly = "变量 表=array（17，20，22）。\n" + program("arrayGet（表，1）+arrayGet（表，2）")
    runtime_pair(work, "array-readonly", {"main.serica": readonly}, ((b"", b"42\n", 0),),
                 reads=False, imports=memory)
    folded = (
        "常量 表=array（21，6）。\n变量 初=readInteger（）。\n"
        + program("初+ifElse（1==1，arrayGet（表，1-1），arrayGet（表，99））"
                  "+arrayLength（array（1，2，3））"
                  "+ifElse（0==0 故 arrayGet（表，99）==0，0，1）"
                  "+ifElse（0==1 且 arrayGet（表，99）==0，1，0）", "array（『stdout』，『stdin』）")
    )
    runtime_pair(work, "array-constant", {"main.serica": folded}, ((b"18\n", b"42\n", 0),))

    # Array ownership in real functions extends beyond the seed's scalar block frames.
    local = (
        "函数 局（限）【变量 回=0。变量 次=0。凡（次<限）【变量 临=array（次，次+1）。"
        "若（次==2）【返回 arrayGet（临，1）+回。】。"
        "赋值 回=回+arrayGet（临，0）。赋值 次=次+1。】。返回 回。】。\n"
        "函数 小（数）【变量 对=array（数，2）。返回 arrayGet（对，0）+arrayLength（对）。】。\n"
        "变量 输入=readInteger（）。\n"
        + program("局（输入）+局（输入+1）*10+小（输入）", "array（『stdout』，『stdin』）")
    )
    project = work / "array-local"
    write(project, {"main.serica": local})
    output = project / "native"
    result = build(project, output)
    assert result.returncode == 0, (result.stdout, result.stderr)
    verify_layout(output, b"", runtime=True, reads=True, imports=memory,
                  instructions=("bl\t", "ldp\tx29, x30", "str\txzr, [x28"))
    for value, expected in ((-1, 1), (0, 2), (1, 13), (2, 45), (3, 49)):
        expect_run(output, f"{expected}\n".encode(), stdin=f"{value}\n".encode())

    project = work / "array-rejected"
    output = project / "existing"
    mutation_caps = "array（『stdout』，『array.write』）"
    cases = (
        ("常量 坏=0-1。\n变量 表=array（3）。\n赋值 表=arraySet（表，坏，0）。\n"
         + program("0", mutation_caps), "S-3032"),
        ("变量 甲=array（1）。\n变量 乙=array（2）。\n赋值 甲=arrayPush（乙，3）。\n"
         + program("0", mutation_caps), "S-3033"),
        ("变量 表=array（0==0）。\n赋值 表=arraySet（表，0，1）。\n"
         + program("0", mutation_caps), "S-3031"),
        ("变量 表=array（1）。\n赋值 表=arrayPush（表，2）。\n" + program("0"), "S-4102"),
    )
    for source, code in cases:
        write(project, {"main.serica": source})
        output.write_bytes(b"existing\0binary")
        native = build(project, output)
        seed = run([SERICA, "check", project / "main.serica"], capture_output=True, text=True, timeout=10)
        assert native.returncode == seed.returncode == 1 and code in native.stderr and code in seed.stderr, (
            code, native.stderr, seed.stderr,
        )
        assert output.read_bytes() == b"existing\0binary"
        assert not list(project.glob(".serica-native.*"))
    return len(cases)


def verify_utf8_decoder(artifact):
    random = Random(134)
    cases = [bytes((value,)) for value in range(256)]
    cases.extend(random.randbytes(random.randrange(1, 19)) for _ in range(128))
    for _ in range(128):
        scalars = []
        for _ in range(random.randrange(1, 13)):
            value = random.randrange(0x110000 - 0x800)
            scalars.append(chr(value + (0x800 if value >= 0xD800 else 0)))
        cases.append("".join(scalars).encode())
    for content in cases:
        line = content.split(b"\n", 1)[0].removesuffix(b"\r")
        try:
            line.decode("utf-8", errors="strict")
        except UnicodeDecodeError:
            expect_run(artifact, b"", stdin=content + b"\n", status=1)
        else:
            expect_run(artifact, line + b"\n", stdin=content + b"\n")
    return len(cases)


def verify_runtime_text(work):
    memory = ("_mmap", "_munmap")
    caps = "array（『stdout』，『stdin』）"
    malformed = (
        b"\x80", b"\xc0\xaf", b"\xc2", b"\xc2A", b"\xe0\x9f\x80",
        b"\xed\xa0\x80", b"\xe2\x82A", b"\xf0\x8f\x80\x80",
        b"\xf4\x90\x80\x80", b"\xf5\x80\x80\x80", b"\xf0\xa0\xae",
    )
    exact = b"a" * 59 + "𠮷".encode()
    valid = (
        (b"", b"\n"), (b"\r\n", b"\n"), (b"\r", b"\n"), (b"\x00\n", b"\x00\n"),
        (b"\xf4\x8f\xbf\xbf\n", b"\xf4\x8f\xbf\xbf\n"),
        ("行简é𠮷\r\n".encode(), "行简é𠮷\n".encode()),
        (exact + b"\n", exact + b"\n"), (exact, exact + b"\n"),
    )
    direct = tuple((value, expected, 0) for value, expected in valid)
    direct += tuple((value + b"\n", b"", 1) for value in (*malformed, exact + b"x"))
    runtime_pair(work, "text-direct", {"main.serica": program("readText（）", caps)}, direct,
                 imports=memory, instructions=("ldrb\t", "blr\tx16", "subs\tx20, x20, x0"),
                 rebuild=True)
    checked_byte_sequences = verify_utf8_decoder(work / "text-direct" / "native")

    large = "甲𠮷éé\x00" * 10000
    wire = large.encode()
    text = "变量 文=readText（）。\n" + program("文", caps)
    runtime_pair(work, "text-growth", {"main.serica": text},
                 ((wire + b"\r\n", wire + b"\n", 0), (wire + b"\xf0\xa0\n", b"", 1)),
                 imports=memory, instructions=("strb\t", "add\tx22, x22, x22"))
    limit = 16 * 1024 * 1024
    counted = "变量 文=readText（）。\n" + program("textByteLength（文）", caps)
    runtime_pair(work, "text-limit", {"main.serica": counted},
                 ((b"a" * (limit - 1) + b"\n", f"{limit - 1}\n".encode(), 0),
                  (b"a" * limit + b"\n", b"", 1)), imports=memory)

    replace = (
        "变量 文=readText（）。\n变量 备=readText（）。\n赋值 文=readText（）。\n"
        + program("textByteLength（文）*10000+textLength（文）*100+textByteLength（备）"
                  "+textByteAt（『𠮷』，0）+textByteLength（『é』）", caps)
    )
    runtime_pair(work, "text-replace", {"main.serica": replace},
                 ((wire + "\n核𠮷\né𠮷\n".encode(), b"60449\n", 0),
                  (b"old\nbackup\n\n", b"248\n", 0),
                  (b"valid\nbackup\n\xed\xa0\x80\n", b"", 1)), imports=memory)

    indexed = "变量 文=readText（）。\n" + program("textByteAt（文，readInteger（））", caps)
    runtime_pair(work, "text-index", {"main.serica": indexed},
                 (("行𠮷\n1\n".encode(), b"161\n", 0),
                  ("行𠮷\n6\n".encode(), b"183\n", 0),
                  ("行𠮷\n7\n".encode(), b"", 1),
                  ("行𠮷\n-1\n".encode(), b"", 1), (b"\n0\n", b"", 1)),
                 imports=memory, instructions=("cmp\tx0, x2", "ldrb\tw0, [x1]"))

    mixed = (
        "变量 文=readText（）。\n变量 另=readText（）。\n变量 表=array（textByteAt（另，1），42）。\n"
        + program("arrayGet（表，0）+textByteLength（文）", caps)
    )
    runtime_pair(work, "text-array", {"main.serica": mixed},
                 (("你\n行\n".encode(), b"164\n", 0), (b"ok\nx\n", b"", 1)),
                 imports=memory, instructions=("ldr\tx0, [x1, x0, lsl #3]", "ldrb\tw0, [x1]"))

    function = (
        "函数 长（）能力=array（『stdin』）=textLength（readText（））。\n"
        + program("长（）+长（）", caps)
    )
    runtime_pair(work, "text-function", {"main.serica": function},
                 (("𠮷才\né\n".encode(), b"3\n", 0), (b"\n\n", b"0\n", 0),
                  (b"abc\n\xf4\x90\x80\x80\n", b"", 1)),
                 imports=memory, instructions=("bl\t", "ldp\tx29, x30", "str\txzr, [x28"))

    # The native path assigns stable slots to direct input expressions.
    project = work / "text-temporary"
    write(project, {"main.serica":
                    "变量 表=array（textByteAt（readText（），textLength（readText（））），42）。\n"
                    + program("arrayGet（表，0）", caps)})
    output = project / "native"
    result = build(project, output)
    assert result.returncode == 0, (result.stdout, result.stderr)
    verify_layout(output, b"", runtime=True, reads=True, imports=memory, instructions=("str\txzr, [x28",))
    expect_run(output, b"161\n", stdin="行𠮷\né\n".encode())
    expect_run(output, b"", stdin="x\né𠮷\n".encode(), status=1)

    project = work / "text-rejections"
    output = project / "existing"
    for source, code in (
        (program("readText（）"), "S-4102"),
        ("变量 文=readText（）。\n" + program("textByteAt（文，0-1）", caps), "S-3072"),
        ("变量 文=readText（）。\n" + program("equals（文，1）", caps), "S-3015"),
    ):
        write(project, {"main.serica": source})
        output.write_bytes(b"existing\0binary")
        result = build(project, output)
        assert result.returncode == 1 and code in result.stderr, (source, result)
        assert output.read_bytes() == b"existing\0binary"
        assert not list(project.glob(".serica-native.*"))
    return len(direct) + 17, checked_byte_sequences


def verify_borrowed_text(work):
    memory = ("_mmap", "_munmap")
    caps = "array（『stdout』，『stdin』）"
    # A named constant can hold folded expressions, while a direct literal keeps its own bytes.
    literal = "常量 文=『行简𠮷\\n』+「é」。\n变量 位=0。\n" + program("文")
    output = "行简𠮷\né".encode()
    runtime_pair(work, "literal-rodata", {"main.serica": literal}, ((b"", output + b"\n", 0),),
                 reads=False, instructions=("adr\tx0,", "strb\tw9, [sp"), pool=output + b"\0")
    empty = "变量 位=0。\n" + program("『』")
    runtime_pair(work, "literal-empty", {"main.serica": empty}, ((b"", b"\n", 0),),
                 reads=False, pool=b"\0")

    large = "行𠮷a" * 9000
    payload = large.encode()
    indexed = (
        f"常量 前=『{large}』。\n常量 尾=『𠮷』。\n变量 位=readInteger（）。\n"
        + program("textByteAt（前，位）+textByteAt（尾，位%1）", caps)
    )
    assert runtime_pair(
        work, "literal-multipage", {"main.serica": indexed},
        ((b"0\n", b"472\n", 0), (f"{len(payload) - 1}\n".encode(), b"337\n", 0),
         (f"{len(payload)}\n".encode(), b"", 1), (b"-1\n", b"", 1)),
        instructions=("adr\tx0,", "ldrb\tw0, [x1]"), pool=payload + b"\0" + "𠮷".encode() + b"\0",
        rebuild=True,
    ) > 4 * 16384

    mixed = "函数 合并（甲，乙，丙，丁，戊，己）【变量 合=textLength（甲）+textLength（丁）。返回 合+乙+丙+戊+己。】。\n"
    runtime_pair(
        work, "borrow-readonly", {"main.serica": mixed + program("合并（『𠮷x』，2，3，『简』，4，5）")},
        ((b"", b"17\n", 0),), reads=False,
        instructions=("ldr\tx1, [x0, #0x8]", "str\tx7, [x28, #0x38]", "bl\t"),
        pool="𠮷x\0简\0".encode(),
    )
    mixed += "变量 文=readText（）。\n" + program("合并（文，2，3，『简』，4，5）", caps)
    runtime_pair(
        work, "borrow-owned", {"main.serica": mixed},
        (("𠮷x\n".encode(), b"17\n", 0), (b"\n", b"15\n", 0), (b"\xff\n", b"", 1)),
        imports=memory, instructions=("ldr\tx5, [x4, #0x8]", "str\tx7, [x28, #0x38]", "ret"),
        pool="简\0".encode(),
    )

    # Drive the existing lexer and UTF-8 module with stdin until native file input is wired.
    lexer = {
        name: (ROOT / "compiler" / name).read_text(encoding="utf-8")
        for name in ("lexer.serica", "utf8.serica")
    }
    probe = (ROOT / "compiler/lex-probe.serica").read_text(encoding="utf-8")
    lexer["main.serica"] = probe.replace("readFile（argumentText（1））", "readText（）").replace(
        "，『args』，『filesystem.read』", "",
    )
    data = "𠮷甲=12\n".encode()
    spans = ((0, 7, 1), (7, 1, 6), (8, 2, 3), (10, 0, 7), (0, 0, 0))
    runs = tuple((data + f"{i}\n".encode(), f"{(offset << 28) + (width << 4) + kind}\n".encode(), 0)
                 for i, (offset, width, kind) in enumerate(spans))
    runtime_pair(work, "borrow-lexer", lexer, runs + ((b"\xf4\x90\x80\x80\n0\n", b"", 1),),
                 imports=memory, instructions=("bl\t", "ldrb\tw0, [x1]", "ret"))

    project = work / "borrow-rejections"
    destination = project / "existing"
    cases = (
        ("函数 长（甲）【变量 宽=textLength（甲）。返回 宽。】。\n" + program("长（1）"), "S-3062"),
        ("函数 长（甲，乙，丙，丁，戊）【返回 textLength（甲）+textLength（乙）+textLength（丙）+textLength（丁）+戊。】。\n"
         + program("长（『一』，『二』，『三』，『四』，5）"), "S-3062"),
        ("函数 长（甲，乙）【变量 宽=textLength（甲）。赋值 甲=乙。返回 宽。】。\n"
         + "变量 文=readText（）。\n" + program("长（文，『重写』）", caps), "S-3068"),
    )
    for source, code in cases:
        write(project, {"main.serica": source})
        destination.write_bytes(b"keep the prior artifact")
        result = build(project, destination)
        assert result.returncode == 1 and code in result.stderr, (source, result)
        assert destination.read_bytes() == b"keep the prior artifact"
        assert not list(project.glob(".serica-native.*"))
    payload_file = project / "payload"
    malformed = ("a0d65f03c0!0", "a0d65f03c0!xx", "a200000001!00")
    for content in malformed:
        payload_file.write_text(content, encoding="ascii")
        result = run([TOOLS / "emit-console", payload_file, destination, "runtime"],
                     capture_output=True, timeout=10)
        assert (result.returncode, result.stdout, result.stderr) == (0, b"-3080\n", b""), result
        assert destination.read_bytes() == b"keep the prior artifact"
    return len(runs) + 11, len(cases), len(malformed)


def verify_arguments_and_files(work):
    args = "array（『stdout』，『args』）"
    args_stdin = "array（『stdout』，『args』，『stdin』）"
    files = "array（『stdout』，『args』，『filesystem.read』）"
    files_stdin = "array（『stdout』，『args』，『filesystem.read』，『stdin』）"
    file_imports = ("_mmap", "_munmap", "_open", "_close")
    document = work / "补充平面 源码.serica"
    empty = work / "empty"
    invalid = work / "invalid"
    maximum = work / "maximum"
    oversized = work / "oversized"
    missing = work / "missing"
    document.write_bytes("𠮷甲=12\n".encode())
    empty.write_bytes(b"")
    invalid.write_bytes(b"valid\xf4\x90\x80\x80")
    maximum.write_bytes(b"a" * ((1 << 24) - 1))
    oversized.write_bytes(b"a" * (1 << 24))

    runtime_pair(
        work, "argc", {"main.serica": program("argumentCount（）", args)},
        ((b"", b"1\n", 0), (b"", b"3\n", 0, ("甲", ""))),
        reads=False, instructions=("mov\tx27, x0", "mov\tx0, x27"),
    )
    runtime_pair(
        work, "argv", {"main.serica": "变量 下标=readInteger（）。\n" + program("argumentText（下标）", args_stdin)},
        (
            (b"1\n", "𠮷甲\n".encode(), 0, ("𠮷甲",)),
            (b"2\n", b"\n", 0, ("未读", "")),
            (b"1\n", b"x" * 4095 + b"\n", 0, ("x" * 4095,)),
            (b"1\n", b"", 3, ("x" * 4096,)),
            (b"-1\n", b"", 2, ("甲",)),
            (b"1\n", b"", 2),
            (b"1\n", b"", 1, (b"\xc0\xaf",)),
        ),
        instructions=("ldr\tx9, [x26, x0, lsl #3]", "bl\t"),
    )

    lexer = {name: (ROOT / "compiler" / name).read_text(encoding="utf-8")
             for name in ("lexer.serica", "utf8.serica")}
    lexer["main.serica"] = (ROOT / "compiler/lex-probe.serica").read_text(encoding="utf-8")
    spans = ((0, 7, 1), (7, 1, 6), (8, 2, 3), (11, 0, 7), (0, 0, 0))
    lex_runs = tuple(
        (f"{index}\n".encode(), f"{(offset << 28) + (width << 4) + kind}\n".encode(), 0, (document,))
        for index, (offset, width, kind) in enumerate(spans)
    ) + ((b"0\n", b"7\n", 0, (empty,)), (b"0\n", b"", 1, (invalid,)), (b"0\n", b"", 2))
    runtime_pair(work, "lex-file", lexer, lex_runs, imports=file_imports, rebuild=True)

    io_runs = (
        (b"", b"16777215\n", 0, (maximum,)), (b"", b"", 1, (oversized,)),
        (b"", b"0\n", 0, (empty,)), (b"", b"", 1, (invalid,)),
        (b"", b"", 1, (missing,)), (b"", b"", 1, (work,)),
        (b"", b"", 1, ("",)), (b"", b"", 1, ("x" * 4095,)), (b"", b"", 3, ("x" * 4096,)),
    )
    runtime_pair(
        work, "file-bounds",
        {"main.serica": "变量 字=readFile（argumentText（1））。\n" + program("textByteLength（字）", files)},
        io_runs, imports=file_imports, instructions=("cmp\tw0, #0x0",),
    )

    raw = work / "raw file"
    raw.write_bytes("甲\0乙\r\n".encode())
    runtime_pair(
        work, "static-path", {"main.serica": program(f"readFile（『{raw}』）",
                                                     "array（『stdout』，『filesystem.read』）")},
        ((b"", raw.read_bytes() + b"\n", 0),),
        imports=file_imports, pool=str(raw).encode() + b"\0",
    )

    source = (
        "变量 文=argumentText（1）。\n"
        "变量 次=0。\n"
        "凡（次<80）【\n"
        "赋值 文=readFile（argumentText（2））。\n"
        "赋值 文=argumentText（1）。\n"
        "赋值 次=次+1。\n"
        "】。\n"
        "赋值 文=readText（）。\n"
    ) + program("文", files_stdin)
    runtime_pair(
        work, "file-reassignment", {"main.serica": source},
        (("尾𠮷\n".encode(), "尾𠮷\n".encode(), 0, ("借用", document)),), imports=file_imports,
    )
    prior_limit, hard_limit = getrlimit(RLIMIT_NOFILE)
    assert prior_limit >= 32 or prior_limit < 0

    def limit_descriptors():
        setrlimit(RLIMIT_NOFILE, (32, hard_limit))

    for variant in ("native", "stage0"):
        result = run([work / "file-reassignment" / variant, "借用", document], input=b"last\n",
                     capture_output=True, timeout=10, preexec_fn=limit_descriptors)
        assert (result.returncode, result.stdout, result.stderr) == (0, b"last\n", b""), result

    project = work / "argument-file-rejections"
    destination = project / "existing"
    rejections = (
        (program("argumentCount（）"), "S-4102"),
        (program("argumentText（-1）", args), "S-3069"),
        (program("readFile（argumentText（1））", args), "S-4102"),
        (program("readFile（argumentText（1））", "array（『stdout』，『filesystem.read』）"), "S-4102"),
        (program("readFile（readText（））", files_stdin), "S-3070"),
        (program("readFile（『』）", files), "S-3070"),
        (program("readFile（『a\0b』）", files), "S-3070"),
    )
    for source, code in rejections:
        write(project, {"main.serica": source})
        destination.write_bytes(b"keep the prior artifact")
        result = build(project, destination)
        assert result.returncode == 1 and code in result.stderr, (source, result)
        assert destination.read_bytes() == b"keep the prior artifact"
    payload_file = project / "payload"
    malformed = ("V800d65f03c0", "V200d65f03c0", "V1010000000d", "V1g0d65f03c0")
    for content in malformed:
        payload_file.write_text(content, encoding="ascii")
        result = run([TOOLS / "emit-console", payload_file, destination, "runtime"],
                     capture_output=True, timeout=10)
        assert (result.returncode, result.stdout, result.stderr) == (0, b"-3080\n", b""), result
        assert destination.read_bytes() == b"keep the prior artifact"
    return 12 + len(lex_runs) + len(io_runs), len(rejections), len(malformed)


MODULE_IMPORTS = ("_mmap", "_munmap", "_open", "_close", "_openat", "_fcntl", "_fstat", "___error")
MODULE_CAPS = "array（『stdout』，『args』，『filesystem.read』，『array.write』）"


def verify_module_chain(work):
    # Read each import from the preceding file with the actual lexer. This fixture has one edge per importer.
    entry = "Entry.serica"
    path = "，".join(map(str, entry.encode()))
    source = f"""导入 『lexer.serica』。
函数 跳空（文，起）【
变量 序=起。
若（序==textByteLength（文））【返回 序。】。
变量 词=读取词元（文，序）。
凡（bitAnd（词，15）==0）【
赋值 序=词>>4。
赋值 词=读取词元（文，序）。
】。
返回 序。
】。
变量 路=array（{path}）。
变量 路首=0。
变量 路宽={len(entry)}。
变量 源=argumentText（1）。
变量 结果=readModuleResult（argumentText（1），路，路首，路宽）。
变量 模块数=0。
变量 边数=0。
变量 偏移=0。
变量 输出值=0。
变量 词首=0。
变量 词尾=0。
变量 词=0。
凡（模块数<8）【
若（是否失败（结果））【
赋值 输出值=-（（模块数<<37）+（偏移<<13）+ifElse（错误码（结果）==41，4001，4002））。
跳出。
】。
赋值 源=结果值（结果）。
赋值 模块数=模块数+1。
赋值 输出值=（模块数<<32）+边数。
赋值 词首=跳空（源，0）。
赋值 词=读取词元（源，词首）。
若（notEquals（bitAnd（词，15），2））【跳出。】。
赋值 偏移=词首。
赋值 词首=跳空（源，词>>4）。
赋值 词=读取词元（源，词首）。
若（notEquals（bitAnd（词，15），4））【跳出。】。
赋值 词尾=（词>>4）-3。
赋值 词首=词首+3。
赋值 路首=arrayLength（路）。
赋值 路宽=词尾-词首。
凡（词首<词尾）【
赋值 路=arrayPush（路，textByteAt（源，词首））。
赋值 词首=词首+1。
】。
赋值 边数=边数+1。
赋值 结果=readModuleResult（argumentText（1），路，路首，路宽）。
】。
""" + program("输出值", MODULE_CAPS)
    compiler = work / "module-chain"
    modules = {name: (ROOT / "compiler" / name).read_text(encoding="utf-8") for name in ("lexer.serica", "utf8.serica")}
    write(compiler, {"main.serica": source, **modules})
    output, reference = compiler / "native", compiler / "stage0"
    built = build(compiler, output)
    assert built.returncode == 0, (built.stdout, built.stderr)
    verify_layout(output, b"", runtime=True, reads=True, imports=MODULE_IMPORTS)
    compile_source(compiler / "main.serica", reference)

    project = work / "chain-input"
    first_comment = "# 起点\n"
    next_comment = "# 中间𠮷\n"
    write(project, {entry: first_comment + "导入 『Next.serica』。\n",
                    "Next.serica": next_comment + "导入 『Leaf.serica』。\n"})
    leaf = project / "Leaf.serica"
    outside = work / "Outside.serica"
    outside.write_text("常量 𠮷=42。\n", encoding="utf-8")
    diagnostic = (2 << 37) + (len(next_comment.encode()) << 13)
    snapshots = (
        (None, f"{-(diagnostic+4002)}\n".encode()),
        (b"\xf0\x9f", f"{-(diagnostic+4002)}\n".encode()),
        (outside.read_bytes(), f"{(3 << 32) + 2}\n".encode()),
    )
    for content, expected in snapshots:
        if content is not None:
            leaf.write_bytes(content)
        for runner in (output, reference):
            expect_run(runner, expected, args=(project,))
    (project / "Next.serica").write_text(next_comment + "导入 『Alias.serica』。\n", encoding="utf-8")
    (project / "Alias.serica").symlink_to(outside)
    for runner in (output, reference):
        expect_run(runner, f"{-(diagnostic+4001)}\n".encode(), args=(project,))
    return len(snapshots) + 1


def verify_native_module_reads(work):
    project = work / "modules-input"
    (project / "lib").mkdir(parents=True)
    body = "常量 𠮷=42。\n".encode()
    (project / "lib/Exact.serica").write_bytes(body)
    (project / "Empty.serica").write_bytes(b"")
    (project / "Broken.serica").write_bytes(b"\xed\xa0\x80")
    (project / "Folder.serica").mkdir()
    os.mkfifo(project / "Pipe.serica")
    (project / "Alias.serica").symlink_to("lib/Exact.serica")
    (project / "subalias").symlink_to("lib", target_is_directory=True)
    external = work / "External.serica"
    external.write_bytes(b"unreadable through an alias")
    (project / "Foreign.serica").symlink_to(external)
    root_alias = work / "root-link"
    root_alias.symlink_to(project, target_is_directory=True)
    for name, size in (("Limit.serica", (1 << 24) - 1), ("Large.serica", 1 << 24)):
        with (project / name).open("wb") as handle:
            handle.truncate(size)

    collect = """变量 文字=argumentText（2）。
变量 字节=array（0）。
变量 位=0。
变量 宽=textByteLength（文字）。
凡（位<宽）【
赋值 字节=arrayPush（字节，textByteAt（文字，位））。
赋值 位=位+1。
】。
"""
    module_call = "argumentText（1），字节，1，宽"
    bad_paths = (
        ("../External.serica", 41), ("/Exact.serica", 41), ("lib/../Exact.serica", 41),
        ("lib/./Exact.serica", 41), ("lib//Exact.serica", 41), ("Exact.txt", 41),
        ("lib/" + "a" * 1013 + ".serica", 41), ("甲.serica", 41),
        ("Alias.serica", 41), ("subalias/Exact.serica", 41), ("Foreign.serica", 41),
        ("lib/exact.serica", 41 if (project / "lib/exact.serica").exists() else 42),
        ("Folder.serica", 42), ("Pipe.serica", 42), ("Broken.serica", 42),
        ("Missing.serica", 42), ("Large.serica", 42),
    )
    direct_runs = (
        (b"", body + b"\n", 0, (project, "lib/Exact.serica")),
        (b"", b"\n", 0, (project, "Empty.serica")),
        (b"", b"", 41, (root_alias, "lib/Exact.serica")),
    ) + tuple((b"", b"", status, (project, relative)) for relative, status in bad_paths)
    runtime_pair(
        work, "module-direct", {"main.serica": collect + program(f"readModule（{module_call}）", MODULE_CAPS)},
        direct_runs, imports=MODULE_IMPORTS, instructions=("ldr\tx13, [x9, x11, lsl #3]", "stp\tx23, x9, [x19, #0x18]"),
        rebuild=True,
    )

    result_source = collect + (
        f"变量 结果=ifElse（argumentCount（）>2，readModuleResult（{module_call}），失败（42））。\n"
        "程序=【能力=" + MODULE_CAPS + "，输出=ifElse（是否失败（结果），错误码（结果），"
        "textByteLength（结果值（结果）））】。\n"
    )
    result_runs = (
        (b"", f"{len(body)}\n".encode(), 0, (project, "lib/Exact.serica")),
        (b"", b"0\n", 0, (project, "Empty.serica")),
        (b"", b"16777215\n", 0, (project, "Limit.serica")),
        (b"", b"41\n", 0, (root_alias, "lib/Exact.serica")),
    ) + tuple((b"", f"{status}\n".encode(), 0, (project, relative)) for relative, status in bad_paths)
    runtime_pair(work, "module-result", {"main.serica": result_source}, result_runs, imports=MODULE_IMPORTS)

    prefix = collect + f"变量 结果=readModuleResult（{module_call}）。\n"
    unwrap_runs = (
        (b"", body + b"\n", 0, (project, "lib/Exact.serica")),
        (b"", b"", 42, (project, "Broken.serica")),
        (b"", b"", 42, (project, "Missing.serica")),
        (b"", b"", 41, (project, "Alias.serica")),
    )
    runtime_pair(
        work, "module-unwrap", {"main.serica": prefix + program("结果值（结果）", MODULE_CAPS)},
        unwrap_runs, imports=MODULE_IMPORTS,
    )

    guarded = "，".join(map(str, (256, *b"lib/Exact.serica", -1)))
    ranges = (
        f"变量 字节=array（{guarded}）。\n"
        "变量 首=readInteger（）。\n变量 宽=readInteger（）。\n"
    ) + program("readModule（argumentText（1），字节，首，宽）",
                "array（『stdout』，『args』，『filesystem.read』，『stdin』）")
    range_runs = ((b"1\n16\n", body + b"\n", 0, (project,)),) + tuple(
        (f"{start}\n{width}\n".encode(), b"", 41, (project,))
        for start, width in ((0, 17), (1, 17), (-1, 16), (19, 16), (1, -1), (1, 1024), (1, 0))
    )
    runtime_pair(work, "module-ranges", {"main.serica": ranges}, range_runs, imports=MODULE_IMPORTS)

    path = "，".join(map(str, b"lib/Exact.serica"))
    path_init = f"变量 字节=array（{path}）。\n"
    regular_file = project / "Empty.serica"
    static_root = path_init + (
        f"变量 正文=readFile（『{regular_file}』）。\n"
        f"赋值 正文=readModule（『{project}』，字节，0，16）。\n"
        + program("正文", "array（『stdout』，『filesystem.read』）")
    )
    runtime_pair(
        work, "module-static-root", {"main.serica": static_root}, ((b"", body + b"\n", 0),),
        imports=MODULE_IMPORTS, pool=str(regular_file).encode() + b"\0" + str(project).encode() + b"\0",
    )

    bad_project = work / "damaged-project"
    (bad_project / "lib").mkdir(parents=True)
    (bad_project / "lib/Exact.serica").write_bytes(b"\xf4\x90\x80\x80")
    replaced = path_init + (
        "变量 结果=readModuleResult（argumentText（1），字节，0，16）。\n"
        "变量 文本=结果值（结果）。\n"
        "变量 次=0。\n"
        "凡（次<80）【\n"
        "赋值 结果=readModuleResult（argumentText（2），字节，0，16）。\n"
        "赋值 结果=readModuleResult（argumentText（1），字节，0，16）。\n"
        "赋值 次=次+1。\n"
        "】。\n"
        "赋值 结果=失败（42）。\n"
    ) + program("文本", MODULE_CAPS)
    runtime_pair(
        work, "module-replaced", {"main.serica": replaced},
        ((b"", body + b"\n", 0, (project, bad_project)),), imports=MODULE_IMPORTS,
    )
    _, hard_limit = getrlimit(RLIMIT_NOFILE)

    def limit_descriptors():
        setrlimit(RLIMIT_NOFILE, (32, hard_limit))

    for variant in ("native", "stage0"):
        ran = run([work / "module-replaced" / variant, project, bad_project], capture_output=True, timeout=10,
                  preexec_fn=limit_descriptors)
        assert (ran.returncode, ran.stdout, ran.stderr) == (0, body + b"\n", b""), ran

    rejected = work / "rejected-module"
    output = rejected / "existing"
    path_bytes = b"lib/Exact.serica"
    cases = (
        (primitive_source(path_bytes, 0, 16, capabilities=("stdout", "args")), "S-4102"),
        (primitive_source(path_bytes, 0, 16, capabilities=("stdout", "filesystem.read")), "S-4102"),
        (primitive_source(path_bytes, 0, 16, root="『』"), "S-3073"),
        (primitive_source(path_bytes, 0, 16).replace("变量 字节", "常量 字节"), "S-3073"),
    )
    for source, diagnostic in cases:
        write(rejected, {"main.serica": source})
        output.write_bytes(b"prior output")
        built = build(rejected, output)
        assert built.returncode == 1 and diagnostic in built.stderr, (source, built)
        assert output.read_bytes() == b"prior output"
    comparisons = len(direct_runs) + len(result_runs) + len(unwrap_runs) + len(range_runs) + 3
    return comparisons + verify_module_chain(work), len(cases)


def verify_byte_writer(work):
    path = work / "raw.serica"
    output = work / "raw"
    artifact = work / "writer"
    base = (
        "变量 字节=array（{}）。\n程序=【能力=array（『stdout』，『args』，『filesystem.write』），"
        "输出=writeByteArray（argumentText（1），字节）】。\n"
    )
    path.write_text(base.format("0，128，255，65"), encoding="utf-8")
    compile_source(path, artifact)
    verify_native_console_artifact(
        artifact, required_imports=("_mmap", "_munmap", "_creat", "_close"),
        required_instructions=("strb\t",),
    )
    expect_run(artifact, b"4\n", args=(str(output),))
    assert output.read_bytes() == bytes((0, 128, 255, 65))
    for invalid in ("-1", "256"):
        path.write_text(base.format(f"10，{invalid}"), encoding="utf-8")
        compile_source(path, artifact)
        expect_run(artifact, b"", status=1, args=(str(output),))
        assert output.read_bytes() == bytes((0, 128, 255, 65))
    for source, code in (
        (base.format("1").replace("，『filesystem.write』", ""), "S-4102"),
        (base.format("『字』"), "S-3074"),
    ):
        path.write_text(source, encoding="utf-8")
        checked = run([SERICA, "check", path], capture_output=True, text=True, timeout=10)
        assert checked.returncode == 1 and code in checked.stderr, checked


def verify_native_writer(work):
    writer = work / "writer-native"
    built, reference = writer / "native", writer / "stage0"
    target = writer / "raw"
    output_caps = "array（『stdout』，『args』，『stdin』，『filesystem.write』，『array.write』）"
    source = (
        "变量 字节=array（0，128，255，65）。\n"
        "变量 尾=readInteger（）。\n"
        "赋值 字节=arrayPush（字节，尾）。\n"
        + program("writeByteArray（argumentText（1），字节）", output_caps)
    )
    imports = ("_mmap", "_munmap", "_creat", "_close")
    runtime_pair(work, writer.name, {"main.serica": source}, (), imports=imports,
                 instructions=("strb\tw10, [x22, x9]", "cmp\tx10, #0xff"), rebuild=True)
    for runner in (built, reference):
        target.write_bytes(b"prior longer file")
        expect_run(runner, b"5\n", args=(target,), stdin=b"10\n")
        expected = bytes((0, 128, 255, 65, 10))
        assert target.read_bytes() == expected
        for invalid in (b"-1\n", b"256\n"):
            expect_run(runner, b"", status=1, args=(target,), stdin=invalid)
            assert target.read_bytes() == expected
        expect_run(runner, b"", status=21, args=(writer / "nonexistent-dir" / "raw",), stdin=b"1\n")

    large = work / "writer-large"
    target = large / "destination"
    source = (
        "变量 结果=array（0）。\n"
        "变量 位=1。\n"
        "凡（位<40001）【赋值 结果=arrayPush（结果，位%256）。赋值 位=位+1。】。\n"
        + program(f"writeByteArray（『{target}』，结果）", "array（『stdout』，『filesystem.write』，『array.write』）")
    )
    runtime_pair(work, large.name, {"main.serica": source}, (), reads=False, imports=imports,
                 pool=str(target).encode() + b"\0")
    expected = bytes(i % 256 for i in range(40001))
    _, file_hard_limit = getrlimit(RLIMIT_FSIZE)

    def limit_file():
        signal.signal(signal.SIGXFSZ, signal.SIG_IGN)
        setrlimit(RLIMIT_FSIZE, (128, file_hard_limit))

    for runner in (large / "native", large / "stage0"):
        expect_run(runner, b"40001\n")
        assert target.read_bytes() == expected
        executed = run([runner], capture_output=True, timeout=10, preexec_fn=limit_file)
        assert (executed.returncode, executed.stdout, executed.stderr) == (22, b"", b""), executed
        assert target.read_bytes() == expected[:128]

    # Exercise the close-error branch of the emitted writer with a scratch image and a real EBADF.
    image = bytearray((large / "native").read_bytes())
    disassembly = run(["otool", "-tV", large / "native"], capture_output=True, check=True, text=True).stdout
    lines = disassembly.splitlines()
    close_sites = [
        int(line.split()[0], 16) - (1 << 32)
        for index, line in enumerate(lines[:-4])
        if line.endswith("mov\tw0, w24") and "adrp\tx16" in lines[index + 1]
        and lines[index + 3].endswith("blr\tx16") and "cbz\tw0" in lines[index + 4]
    ]
    assert len(close_sites) == 1, close_sites
    struct.pack_into("<I", image, close_sites[0], 0x12800000)  # mov w0, #-1
    injected = large / "bad-close"
    injected.write_bytes(image)
    injected.chmod(0o755)
    run(["codesign", "-s", "-", "--force", "--timestamp=none", injected], capture_output=True, check=True)
    expect_run(injected, b"", status=23)
    assert target.read_bytes() == expected

    pipeline = work / "pipeline"
    payload_name = "payload.serica"
    path_bytes = "，".join(map(str, payload_name.encode()))
    copy_source = (
        "变量 文=readFile（argumentText（1））。\n"
        f"变量 路=array（{path_bytes}）。\n"
        f"赋值 文=readModule（argumentText（2），路，0，{len(payload_name)}）。\n"
        "变量 字节=array（textByteAt（文，0））。\n"
        "变量 位=1。\n"
        "凡（位<textByteLength（文））【赋值 字节=arrayPush（字节，textByteAt（文，位））。赋值 位=位+1。】。\n"
        + program("writeByteArray（argumentText（3），字节）",
                  "array（『stdout』，『args』，『filesystem.read』，『filesystem.write』，『array.write』）")
    )
    runtime_pair(work, pipeline.name, {"main.serica": copy_source}, (), imports=(*MODULE_IMPORTS, "_creat"))
    path, payload, copied = pipeline / "sample.serica", pipeline / payload_name, pipeline / "copy"
    path.write_text(program("readInteger（）+5", "array（『stdout』，『stdin』）"), encoding="utf-8")
    analyzed = run([TOOLS / "analyze", path.name, "native-code", pipeline, payload],
                   input=b"-1\n", timeout=10, capture_output=True, check=True)
    payload_bytes = payload.read_bytes()
    assert int(analyzed.stdout) == len(payload_bytes) and not analyzed.stderr
    for runner in (pipeline / "native", pipeline / "stage0"):
        expect_run(runner, f"{len(payload_bytes)}\n".encode(), args=(payload, pipeline, copied))
        assert copied.read_bytes() == payload_bytes
    downstream = pipeline / "emitted-image"
    assembled = run([TOOLS / "emit-console", copied, downstream, "runtime"],
                    capture_output=True, check=True, timeout=10)
    assert int(assembled.stdout) == downstream.stat().st_size and not assembled.stderr
    downstream.chmod(0o755)
    run(["codesign", "-s", "-", "--force", "--timestamp=none", downstream], capture_output=True, check=True)
    expect_run(downstream, b"9\n", stdin=b"4\n")

    rejected = work / "writer-denied"
    existing = rejected / "existing"
    raw_call = "writeByteArray（argumentText（1），字节）"
    cases = (
        ("变量 字节=array（1）。\n" + program(raw_call, "array（『stdout』，『args』）"), "S-4102"),
        ("变量 字节=array（1）。\n" + program(raw_call, "array（『stdout』，『filesystem.write』）"), "S-4102"),
        ("常量 字节=array（1）。\n" + program(raw_call, output_caps), "S-3074"),
        ("变量 字节=array（『字』）。\n" + program(raw_call, output_caps), "S-3074"),
        ("变量 字节=array（1）。\n" + program("writeByteArray（『』，字节）", output_caps), "S-3071"),
    )
    for source, code in cases:
        write(rejected, {"main.serica": source})
        existing.write_bytes(b"keep")
        result = build(rejected, existing)
        assert result.returncode == 1 and code in result.stderr, (source, result)
        assert existing.read_bytes() == b"keep"
    payload = rejected / "payload"
    malformed = (
        "W1000d65f03c0",  # The writer requires an integer array runtime.
        "W8000d65f03c0",  # Reserved feature bit.
        "W202100000011",  # Writer relocation in a comparison-only image.
        "W10g0d65f03c0",  # Non-hexadecimal feature.
    )
    for content in malformed:
        payload.write_text(content, encoding="ascii")
        result = run([TOOLS / "emit-console", payload, existing, "runtime"], capture_output=True, timeout=10)
        assert (result.returncode, result.stdout, result.stderr) == (0, b"-3080\n", b""), result
        assert existing.read_bytes() == b"keep"
    return 7, len(cases), len(malformed)


def verify_short_read_limit(work):
    project = work / "read-partial"
    source = (
        "变量 文=readFile（argumentText（1））。\n"
        + program("textByteLength（文）", "array（『stdout』，『args』，『filesystem.read』）")
    )
    runtime_pair(work, project.name, {"main.serica": source}, (),
                 imports=("_open", "_close", "_mmap", "_munmap"))
    pipe = project / "pipe"
    os.mkfifo(pipe)
    fixture = project / "input"
    for size, expected in (((1 << 24) - 1, (0, b"16777215\n", b"")), (1 << 24, (1, b"", b""))):
        with fixture.open("wb") as data:
            data.truncate(size)
        for variant in ("native", "stage0"):
            with fixture.open("rb") as data:
                with Popen([project / variant, pipe], stdout=PIPE, stderr=PIPE) as reader:
                    with Popen(["/usr/bin/tee", pipe], stdin=data, stdout=DEVNULL, stderr=PIPE) as writer:
                        try:
                            out, err = reader.communicate(timeout=20)
                            _, writer_err = writer.communicate(timeout=10)
                            assert (reader.returncode, out, err) == expected, (variant, size, reader.returncode, out, err)
                            assert (writer.returncode, writer_err) == (0, b""), (variant, size, writer_err)
                        finally:
                            for child in (reader, writer):
                                if child.poll() is None:
                                    child.kill()
    return 2


def verify_native_text_comparison(work):
    mode = "é𠮷"
    argv = program(f"ifElse（equals（argumentText（1），『{mode}』），9，4）", "array（『stdout』，『args』）")
    argv_runs = (
        (b"", b"9\n", 0, (mode,)), (b"", b"4\n", 0, ("é𠮷",)),
        (b"", b"4\n", 0, ("",)), (b"", b"4\n", 0, (mode + "a",)), (b"", b"", 2),
    )
    runtime_pair(work, "cmp-mode", {"main.serica": argv}, argv_runs, reads=False, rebuild=True,
                 pool=mode.encode() + b"\0", instructions=("ldrb\tw7, [x2], #0x1", "subs\tx0, x7, x8"))

    tests = ("甲<乙", "甲<=乙", "甲>乙", "甲>=乙", "equals（甲，乙）", "notEquals（甲，乙）")
    combined = "+".join(f"ifElse（{test}，{1 << bit}，0）" for bit, test in enumerate(tests))
    function = (
        f"函数 顺序（甲，乙）【变量 长=textByteLength（甲）+textByteLength（乙）。返回 {combined}。】。\n"
        "变量 左=readText（）。\n变量 右=readText（）。\n"
        + program("顺序（左，右）", "array（『stdout』，『stdin』）")
    )
    prefix = "行简𠮷".encode() * 14000
    owned_runs = (
        (b"\n\n", b"26\n", 0), (b"\x00\n\n", b"44\n", 0),
        ("𠮷\n𠮷a\n".encode(), b"35\n", 0), ("𠮷\n𠮸\n".encode(), b"35\n", 0),
        (prefix + b"x\n" + prefix + b"x\n", b"26\n", 0),
        (prefix + b"y\n" + prefix + b"x\n", b"44\n", 0),
        (b"ok\n\xf0\x80\n", b"", 1),
    )
    runtime_pair(work, "cmp-owned-borrowed", {"main.serica": function}, owned_runs, imports=("_mmap", "_munmap"))

    mixed = "变量 文=readText（）。\n" + program("notEquals（文，argumentText（1））",
                                                   "array（『stdout』，『stdin』，『args』）")
    mixed_runs = (("𠮷\n".encode(), b"0\n", 0, ("𠮷",)), (b"\x00\n", b"1\n", 0, ("",)))
    runtime_pair(work, "cmp-mixed", {"main.serica": mixed}, mixed_runs, imports=("_mmap", "_munmap"))
    return len(argv_runs) + len(owned_runs) + len(mixed_runs)


def verify_native_text_arrays(work):
    caps = "array（『stdout』，『stdin』，『array.write』）"
    memory = ("_mmap", "_munmap")
    snapshot = (
        "变量 源=readText（）。\n"
        "变量 队=array（『旧』，『』）。\n"
        "赋值 队=arrayPush（队，源）。\n"
        "赋值 源=readText（）。\n"
        "赋值 队=arrayPush（队，源）。\n"
        "赋值 队=arraySet（队，0，arrayGet（队，3））。\n"
        "赋值 队=arraySet（队，2，arrayGet（队，2））。\n"
        "变量 位置=readInteger（）。\n"
        + program("arrayGet（队，位置）", caps)
    )
    first = "行𠮷\0".encode()
    second = "é".encode()
    stdin_base = first + b"\n" + second + b"\n"
    snapshot_runs = tuple((stdin_base + f"{index}\n".encode(), text + b"\n", 0)
                          for index, text in enumerate((second, b"", first, second)))
    snapshot_runs += tuple((stdin_base + f"{index}\n".encode(), b"", 1) for index in (-1, 4))
    runtime_pair(work, "array-text-snapshot", {"main.serica": snapshot}, snapshot_runs, imports=memory,
                 pool="旧\0\0".encode(), rebuild=True, instructions=("ldp\tx19, x20, [x0]", "ldr\tx0, [x20], #0x8"))

    grown = (
        "函数 计字（文，偏）【变量 字节=textByteLength（文）。返回 字节+偏。】。\n"
        "函数 转发（文，偏）=计字（文，偏）。\n"
        "变量 队=array（『』）。\n"
        "变量 源=readText（）。\n"
        "变量 轮=0。\n"
        "凡（轮<1025）【赋值 队=arrayPush（队，源）。赋值 轮=轮+1。】。\n"
        "赋值 源=readText（）。\n"
        "赋值 队=arraySet（队，1024，源）。\n"
        "变量 位=readInteger（）。\n"
        + program("转发（arrayGet（队，位），arrayLength（队））", caps)
    )
    growth_runs = tuple(
        ("𠮷甲\n尾\n".encode() + f"{index}\n".encode(), f"{expected}\n".encode(), 0)
        for index, expected in ((0, 1026), (1023, 1033), (1024, 1029), (1025, 1033))
    )
    runtime_pair(work, "array-text-growth", {"main.serica": grown}, growth_runs, imports=memory, pool=b"\0")

    # The copy must remain usable after replacing the source element and reassigning a different variable.
    project = work / "array-text-copy"
    copy = (
        "变量 队=array（『行𠮷』）。\n"
        "变量 文=arrayGet（队，0）。\n"
        "赋值 队=arraySet（队，0，『新』）。\n"
        "变量 另一=arrayGet（队，0）。\n"
        "赋值 另一=readText（）。\n"
        + program("ifElse（equals（文，『行𠮷』）且 equals（另一，『尾』），1，0）", caps)
    )
    write(project, {"main.serica": copy})
    result = build(project, project / "native")
    assert result.returncode == 0, (result.stdout, result.stderr)
    verify_native_console_artifact(project / "native", require_runtime_input=True, required_imports=memory)
    expect_run(project / "native", b"1\n", stdin="尾\n".encode())
    expect_run(project / "native", b"0\n", stdin="别\n".encode())

    local = work / "array-text-local"
    source = (
        "函数 计（次数）【变量 量=0。凡（次数>0）【变量 队=array（『𠮷』，『甲』）。"
        "赋值 量=量+textByteLength（arrayGet（队，次数%2））。"
        "若（次数==1）【返回 量。】。赋值 次数=次数-1。】。返回 量。】。\n"
        + program("计（readInteger（））+计（readInteger（））", "array（『stdout』，『stdin』）")
    )
    write(local, {"main.serica": source})
    result = build(local, local / "native")
    assert result.returncode == 0, (result.stdout, result.stderr)
    verify_native_console_artifact(local / "native", require_runtime_input=True, required_imports=memory,
                                   required_instructions=("ldr\tx0, [x20], #0x8", "ret"))
    expect_run(local / "native", b"3503\n", stdin=b"1000\n1\n")
    expect_run(local / "native", b"0\n", stdin=b"0\n0\n")

    frame = work / "large-text-frame"
    # The array descriptor and its literal use five additional slots.
    variables = "".join(f"变量 局{i}={i}。\n" for i in range(4089))
    source = variables + (
        "变量 队=array（『𠮷』）。\n"
        + program("局4088+textByteLength（arrayGet（队，readInteger（）））", "array（『stdout』，『stdin』）")
    )
    write(frame, {"main.serica": source})
    result = build(frame, frame / "native")
    assert result.returncode == 0, (result.stdout, result.stderr)
    verify_native_console_artifact(frame / "native", require_runtime_input=True, required_imports=memory,
                                   required_instructions=("sub\tsp, sp, #0x8, lsl #12",))
    expect_run(frame / "native", b"4092\n", stdin=b"0\n")
    expect_run(frame / "native", b"", status=1, stdin=b"-1\n")

    rejected = work / "array-text-denied"
    denied = "变量 队=array（『行』）。\n"
    output = rejected / "existing"
    cases = (
        ("变量 队=array（『行』，1）。\n" + program("arrayLength（队）"), "S-3031"),
        (denied + "赋值 队=arrayPush（队，『字』）。\n" + program("arrayLength（队）"), "S-4102"),
        (denied + "变量 另=array（『』）。\n赋值 另=arrayPush（队，『字』）。\n"
         + program("arrayLength（另）", "array（『stdout』，『array.write』）"), "S-3033"),
        ("变量 额外=0。\n" + source, "S-3040"),
    )
    for case, issue in cases:
        write(rejected, {"main.serica": case})
        output.write_bytes(b"old image")
        result = build(rejected, output)
        assert result.returncode == 1 and issue in result.stderr, (case, result)
        assert output.read_bytes() == b"old image"
    return len(snapshot_runs) + len(growth_runs), len(cases)


def verify_self_emission(work):
    project = work / "self-emission"
    project.mkdir()
    compiler = ROOT / "compiler"
    analyzer = project / "analyzer"
    emitter = project / "emitter"
    for source, destination in (("parse-expression.serica", analyzer), ("emit-console.serica", emitter)):
        result = run([BUILD, compiler / source, "-o", destination], capture_output=True, timeout=45)
        assert result.returncode == 0, result
    analyzer_imports = (*MODULE_IMPORTS, "_creat", "_read")
    emitter_imports = ("_open", "_read", "_close", "_creat", "_mmap", "_munmap")
    verify_native_console_artifact(analyzer, required_imports=analyzer_imports)
    verify_native_console_artifact(emitter, required_imports=emitter_imports)

    anchor = (TOOLS / "analyze", TOOLS / "emit-console")

    def output(argv, stdin=None):
        executed = run(argv, input=stdin, capture_output=True, check=True, timeout=45)
        assert not executed.stderr, executed
        return executed.stdout

    native_tools = ROOT / "build/serica-native-tools"
    native_tools.mkdir(parents=True, exist_ok=True)
    candidates = (native_tools / "analyze", native_tools / "emit-console")
    for unit, destination in (("parse-expression.serica", candidates[0]),
                              ("emit-console.serica", candidates[1])):
        args = [BUILD, compiler / unit, "-o", destination]
        result = run(args, capture_output=True, timeout=45)
        assert result.returncode == 0, (args, result)
    verify_native_console_artifact(
        candidates[0], required_imports=analyzer_imports,
    )
    verify_native_console_artifact(
        candidates[1], required_imports=emitter_imports,
    )
    images = []
    for unit in ("parse-expression", "emit-console"):
        results = []
        for index, (read, emit) in enumerate((anchor, candidates)):
            payload = project / f"{unit}-{index}.payload"
            mach = project / f"{unit}-{index}.mach"
            value = output([read, f"{unit}.serica", "native-code", compiler, payload], b"-1\n")
            assert int(value) == payload.stat().st_size
            value = output([emit, payload, mach, "runtime"])
            assert int(value) == mach.stat().st_size
            results.append((payload.read_bytes(), mach.read_bytes()))
        assert results[0] == results[1], unit
        mach.chmod(0o755)
        run(["codesign", "-s", "-", "--force", "--timestamp=none", mach], capture_output=True, check=True)
        expected_imports = analyzer_imports if unit == "parse-expression" else emitter_imports
        verify_native_console_artifact(mach, required_imports=expected_imports)
        images.append(mach)

    queries = (
        ("module-graph", -1), ("module-graph", -2), ("module-bind", -4),
        ("module-const", -1), ("module-type", -1), ("module-effect", -1), ("module-abi", -1),
    )
    for mode, query in queries:
        argv = ("parse-expression.serica", mode, compiler)
        stdin = f"{query}\n".encode()
        reference = output([anchor[0], *argv], stdin)
        assert int(reference) > 0
        assert output([analyzer, *argv], stdin) == reference
        assert output([images[0], *argv], stdin) == reference

    prefix = "# 中文𠮷\n"
    broken = project / "broken.serica"
    broken.write_text(prefix + "导入 『Missing.serica』。\n" + program("0"), encoding="utf-8")
    expected = f"{-((1 << 37) + (len(prefix.encode()) << 13) + 4002)}\n".encode()
    for read in (anchor[0], analyzer, images[0]):
        assert output([read, broken.name, "module-graph", project], b"-1\n") == expected

    # The second pair receives a fresh source and produces a runnable program.
    source = project / "sample.serica"
    (project / "names.serica").write_text("常量 尾=『𠮷』。\n", encoding="utf-8")
    source.write_text("导入 『names.serica』。\n变量 队=array（『行』，尾）。\n"
                      + program("arrayGet（队，readInteger（））", "array（『stdout』，『stdin』）"), encoding="utf-8")
    payload = project / "sample.payload"
    new_image = project / "sample"
    assert int(output([images[0], source.name, "native-code", project, payload], b"-1\n")) == payload.stat().st_size
    assert int(output([images[1], payload, new_image, "runtime"])) == new_image.stat().st_size
    new_image.chmod(0o755)
    run(["codesign", "-s", "-", "--force", "--timestamp=none", new_image], capture_output=True, check=True)
    verify_layout(new_image, "行\0𠮷\0".encode(), runtime=True, reads=True, imports=("_mmap", "_munmap"))
    expect_run(new_image, "𠮷\n".encode(), stdin=b"1\n")
    return len(queries) + 1


def main():
    with TemporaryDirectory(prefix="serica-native-") as directory:
        work = Path(directory).resolve()
        verify_native_console_artifact(
            TOOLS / "emit-console", required_imports=("_open", "_read", "_close", "_creat", "_mmap", "_munmap"),
        )
        cases = verify_programs(work)
        verify_modules(work)
        large = verify_multipage(work)
        rejected = verify_rejections(work)
        runtime = verify_runtime(work)
        array_rejected = verify_arrays(work)
        text_runs, utf8_cases = verify_runtime_text(work)
        borrowed_runs, rejected_borrows, rejected_payloads = verify_borrowed_text(work)
        file_runs, rejected_files, rejected_file_payloads = verify_arguments_and_files(work)
        module_runs, rejected_modules = verify_native_module_reads(work)
        verify_byte_writer(work)
        writer_runs, rejected_writes, rejected_write_payloads = verify_native_writer(work)
        short_reads = verify_short_read_limit(work)
        compare_runs = verify_native_text_comparison(work)
        array_text_runs, array_text_rejections = verify_native_text_arrays(work)
        self_queries = verify_self_emission(work)
        print(f"P1b 行简原生发射通过：{cases + 2} 个签名运行产物、8 个 stage0 参照、"
              f"{large} 字节多页输出、{rejected} 个拒绝场景、字节写入及原位重建")
        print(f"P1b 行简运行时发射通过：{runtime} 个签名运行产物及 stage0 对照，"
              "覆盖整数运算、短路、循环、递归、跨模块八寄存器调用及多页指令段")
        print(f"P1b 行简数组发射通过：5 个 stage0 对照产物、1 个局部所有权产物、"
              f"{array_rejected} 个编译拒绝场景，覆盖几何扩容、越界写入顺序及同路径重建")
        print(f"P1b 行简 UTF-8 输入发射通过：7 个 stage0 对照产物、{text_runs} 次对照执行，"
              f"{utf8_cases} 个独立编码校验、1 个稳定临时描述符产物及 3 个编译拒绝")
        print(f"P1b 行简常量池与借用 ABI 通过：6 个签名产物、{borrowed_runs} 次 stage0 对照、"
              f"{rejected_borrows} 个源码拒绝及 {rejected_payloads} 个载荷拒绝")
        print(f"P1b 行简命令行与文件读取通过：6 个签名产物、{file_runs} 次 stage0 对照、"
              f"{rejected_files} 个源码拒绝及 {rejected_file_payloads} 个载荷拒绝，"
              "包含原始文件词法探针与低描述符限额复跑")
        print(f"P1b 行简项目限界模块读取通过：7 个签名产物、{module_runs} 次 stage0 对照、"
              f"{rejected_modules} 个源码拒绝，覆盖实际依赖链与导入位置")
        print(f"P1b 行简字节写入与文本比较通过：7 个签名产物、{writer_runs + short_reads + compare_runs} 次 stage0 对照、"
              f"{rejected_writes} 个源码拒绝及 {rejected_write_payloads} 个载荷拒绝，"
              "覆盖短读短写、关闭错误和载荷重放")
        print(f"P1b 行简文本数组与编译器自编复验通过：5 个数组产物、{array_text_runs} 次 stage0 对照、"
              f"{array_text_rejections} 个源码拒绝、两套原始源码载荷与 Mach-O 逐字节一致、"
              f"{self_queries} 个分析器查询及可运行的新程序")


if __name__ == "__main__":
    main()
