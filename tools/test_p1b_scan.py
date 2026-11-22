# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Accept ARM64 byte reads and the Serica UTF-8 scanner on real files."""

from pathlib import Path
from subprocess import run
from tempfile import TemporaryDirectory

from native_artifact import verify_native_console_artifact

ROOT = Path(__file__).resolve().parent.parent
SERICA = ROOT / "build/stage0/serica"
BOUNDED_LOAD = ("cmp\tx0, x2", "b.hs", "ldrb\tw0, [x1]")


def compile_source(source, artifact):
    # Bootstrap compiler sources take longer than the small behavioral fixtures.
    timeout = 30 if Path(source).parent == ROOT / "compiler" else 10
    for args in (("check", source), ("build", source, "-o", artifact)):
        result = run([SERICA, *args], capture_output=True, text=True, timeout=timeout)
        if result.returncode:
            raise RuntimeError(result.stdout + result.stderr)


def expect_run(artifact, stdout, status=0, args=(), stdin=b""):
    result = run([artifact, *args], input=stdin, capture_output=True, timeout=10)
    actual = result.returncode, result.stdout, result.stderr
    if actual != (status, stdout, b""):
        raise RuntimeError(f"{artifact.name} {args}: {actual!r}")


def program(expression, capabilities=""):
    return f"程序=【能力=array（『stdout』{capabilities}），输出={expression}】。\n"


def main():
    with TemporaryDirectory(prefix="serica-scan-") as work:
        root = Path(work)
        source = root / "bytes.serica"
        source.write_text(
            "常量 值=textByteLength（『é𠀀』）*1000+textByteAt（『é𠀀』，1）。\n"
            + program("值"), encoding="utf-8"
        )
        constant = root / "constant"
        compile_source(source, constant)
        expect_run(constant, b"6169\n")

        source.write_text(
            "变量 源=readText（）。\n变量 下标=readInteger（）。\n"
            + program("textByteLength（源）*1000+textByteAt（源，下标）", "，『stdin』"),
            encoding="utf-8",
        )
        stored = root / "stored"
        compile_source(source, stored)
        verify_native_console_artifact(
            stored, require_runtime_input=True,
            required_imports=("_mmap", "_munmap"), required_instructions=BOUNDED_LOAD,
        )
        expect_run(stored, b"6231\n", stdin="é甲Z\n2\n".encode())
        expect_run(stored, b"1000\n", stdin=b"\0\n0\n")
        for raw in (b"A\n1\n", b"A\n-1\n", b"A\n9223372036854775807\n", b"\n0\n"):
            expect_run(stored, b"", status=1, stdin=raw)

        source.write_text(
            "常量 表=array（『é』）。\n"
            + program("textByteAt（arrayGet（表，0），readInteger（））", "，『stdin』"),
            encoding="utf-8",
        )
        literal = root / "literal"
        compile_source(source, literal)
        expect_run(literal, b"169\n", stdin=b"1\n")

        source.write_text(
            program(
                "textByteAt（argumentText（1），textByteAt（argumentText（2），0）-48）",
                "，『args』",
            ),
            encoding="utf-8",
        )
        arguments = root / "arguments"
        compile_source(source, arguments)
        expect_run(arguments, b"169\n", args=("é", "1"))

        source.write_text(
            "变量 表=array（readText（））。\n变量 值=成功（『甲』）。\n"
            + program("textByteAt（arrayGet（表，0），1）+textByteLength（结果值（值））", "，『stdin』"),
            encoding="utf-8",
        )
        stable_values = root / "stable-values"
        compile_source(source, stable_values)
        expect_run(stable_values, b"172\n", stdin="é\n".encode())

        source.write_text(
            "变量 源=readFile（argumentText（1））。\n"
            + program("textByteAt（源，textByteLength（源）-1）", "，『args』，『filesystem.read』"),
            encoding="utf-8",
        )
        last_byte = root / "last-byte"
        compile_source(source, last_byte)
        maximum = root / "maximum.serica"
        maximum.write_bytes(b"a" * (16 * 1024 * 1024 - 2) + b"Z")
        expect_run(last_byte, b"90\n", args=(maximum,))

        decoder = ROOT / "compiler/utf8.serica"
        (root / decoder.name).write_bytes(decoder.read_bytes())
        source.write_text(
            "导入 『utf8.serica』。\n变量 源=readText（）。\n"
            + program("解码标量（源，readInteger（））", "，『stdin』"),
            encoding="utf-8",
        )
        decoded = root / "decoded"
        compile_source(source, decoded)
        for offset, scalar, width in ((0, "A", 1), (1, "é", 2), (3, "甲", 3), (6, "𠀀", 4)):
            expect_run(
                decoded, f"{ord(scalar) * 8 + width}\n".encode(),
                stdin=f"Aé甲𠀀\n{offset}\n".encode(),
            )
        expect_run(decoded, b"-1\n", stdin="Aé甲𠀀\n2\n".encode())
        expect_run(decoded, b"", status=1, stdin="Aé甲𠀀\n10\n".encode())

        malformed = (
            "textByteLength（）", "textByteLength（1）", "textByteAt（『a』）",
            "textByteAt（『a』，『0』）", "textByteAt（『a』，-1）",
            "textByteAt（『甲』，3）", "textByteAt（『』，0）",
            "textByteAt（readText（），0）",
        )
        for expression in malformed:
            source.write_text(program(expression, "，『stdin』"), encoding="utf-8")
            rejected = root / "rejected"
            for args in (("check", source), ("build", source, "-o", rejected)):
                result = run([SERICA, *args], capture_output=True, text=True, timeout=10)
                if result.returncode == 0 or "S-3072" not in result.stdout + result.stderr or rejected.exists():
                    raise RuntimeError(f"{expression}: {result.stdout}{result.stderr}")

        scanner = root / "scanner"
        compile_source(ROOT / "compiler/scan-probe.serica", scanner)
        verify_native_console_artifact(
            scanner, require_runtime_input=True,
            required_imports=("_mmap", "_munmap", "_open", "_close"),
            required_instructions=(*BOUNDED_LOAD, "bl\t", "ret"),
        )
        fixtures = (
            ("变量 名称=42。".encode(), b"20102\n", 0),
            (("甲𠀀" * 2048 + "=42。\r\n").encode(), b"10102\n", 0),
            (b"", b"0\n", 0),
            (("甲" * 94 + "é").encode(), b"", 31),
            (("甲" * 94 + "😀").encode(), b"", 31),
            (b"A\xf0\x80\x80\x80", b"", 1),
        )
        for data, output, status in fixtures:
            sample = root / "source.serica"
            sample.write_bytes(data)
            expect_run(scanner, output, status=status, args=(sample,))
        print("P1b 字节扫描子门通过：8 个 ARM64 产物、16 个编译诊断、6 个文件扫描场景")


if __name__ == "__main__":
    main()
