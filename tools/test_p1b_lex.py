# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Compare the Serica lexer with the seed on real UTF-8 source bytes."""

import re
from pathlib import Path
from subprocess import run
from tempfile import TemporaryDirectory

from native_artifact import verify_native_console_artifact
from test_p1b_scan import ROOT, SERICA, compile_source, expect_run, program

KINDS = ("", "标识符", "关键字", "数字", "字符串", "原始字符串", "符号")


def seed_lex(source):
    return run([SERICA, "lex", source], capture_output=True, timeout=10)


def token_at(scanner, source, index):
    result = run([scanner, source], input=f"{index}\n".encode(), capture_output=True, timeout=10)
    if result.returncode or result.stderr:
        raise RuntimeError((result.returncode, result.stdout, result.stderr))
    value = int(result.stdout)
    return value & 15, value >> 28, (value >> 4) & 0xFFFFFF


def check_stream(scanner, source, data, expected_spans=None):
    source.write_bytes(data)
    seed = seed_lex(source)
    if seed.returncode:
        raise RuntimeError(seed.stderr)
    previous_end = 0
    spans = []
    output = bytearray()
    for index in range(len(data) + 1):
        kind, offset, width = token_at(scanner, source, index)
        if kind == 7:
            if (offset, width) != (len(data), 0):
                raise RuntimeError((offset, width, len(data)))
            if token_at(scanner, source, index + 1) != (0, 0, 0):
                raise RuntimeError("词元请求超出 EOF 后仍返回词元")
            break
        if not (0 < kind < 7 and 0 < width and previous_end <= offset <= len(data) - width):
            raise RuntimeError((index, kind, offset, width, previous_end))
        previous_end = offset + width
        spans.append((kind, offset, width))
        output.extend(KINDS[kind].encode() + b"\t" + data[offset:previous_end] + b"\n")
    else:
        raise RuntimeError("缺少 EOF 词元")
    if output != seed.stdout:
        raise RuntimeError((output, seed.stdout))
    if expected_spans is not None and spans != expected_spans:
        raise RuntimeError((spans, expected_spans))
    return len(spans)


def main():
    with TemporaryDirectory(prefix="serica-lex-") as temporary:
        work = Path(temporary)
        scanner = work / "lexer"
        entry = ROOT / "compiler/lex-probe.serica"
        compile_source(entry, scanner)
        verify_native_console_artifact(
            scanner,
            require_runtime_input=True,
            required_imports=("_mmap", "_munmap", "_open", "_close"),
            required_instructions=("bl\t", "ret", "ldrb\tw0, [x1]"),
        )
        segments = run(["otool", "-l", scanner], capture_output=True, check=True).stdout
        text_segment = re.search(rb"segname __TEXT\s+vmaddr \S+\s+vmsize (0x[\da-f]+)", segments)
        if text_segment is None or int(text_segment[1], 16) <= 16 * 1024:
            raise RuntimeError("词法器未实际跨越旧的单页文本段限制")
        first = scanner.read_bytes()
        compile_source(entry, scanner)
        if scanner.read_bytes() != first:
            raise RuntimeError("同路径重建的词法器字节不同")

        source = work / "source.serica"
        streams = [
            (b"", []),
            ("#同名\r\n同名 𠮷2\t0x".encode(), [(1, 9, 6), (1, 16, 5), (3, 22, 1), (1, 23, 1)]),
            (
                "若 则 否 凡 每 毕 且 故 跳出 继续 返回 常量 导入 函数 变量 赋值 变量2 函数名"
                " _id_20 A0 readText （）【】<<>>+-*/%=\\，。\u0085\u2028了".encode(), None,
            ),
            ('『a\\』b』 「x\\\ny」 『#\u3000\\n』 「=\r\n」#注释\u200b\n常量 字=42。'.encode(), None),
            (("#" + "注" * 7000 + "\n甲=12。𠀀_1 𫠠 〇〻").encode(), None),
            (b"# only a comment", []),
        ]
        count = 0
        for data, spans in streams:
            count += check_stream(scanner, source, data, spans)
        self_tokens = check_stream(scanner, source, (ROOT / "compiler/lexer.serica").read_bytes())

        errors = [
            ("变量 甲。\n# 注释\n 甲😀", "S-1001", 8, "😀"),
            ("甲\n\u3000", "S-1003", 9, "\u3000"),
            ("甲\u200d", "S-1003", 9, "\u200d"),
            ("甲Ａ", "S-1003", 9, "Ａ"),
            ("甲９", "S-1003", 9, "９"),
            ("甲\ufeff", "S-1003", 9, "\ufeff"),
            ("甲\n  ,", "S-1010", 10, ","),
            ("甲（）(", "S-1010", 10, "("),
            ("甲『未闭合", "S-1023", 11, "『"),
            ("甲『结尾转义\\", "S-1023", 11, "『"),
            ("甲「原始\n字", "S-1023", 11, "「"),
        ]
        for text, code, kind, marker in errors:
            before = text.partition(marker)[0]
            source.write_text(text, encoding="utf-8")
            seed = seed_lex(source)
            row, col = before.count("\n") + 1, len(before.rsplit("\n", 1)[-1]) + 1
            expected = f"错误 {code}：第 {row} 行第 {col} 列".encode()
            if seed.returncode != 1 or expected not in seed.stderr or seed.stdout:
                raise RuntimeError((text, seed.stdout, seed.stderr))
            if token_at(scanner, source, 0) != (kind, len(before.encode()), 0):
                raise RuntimeError((text, kind, before))
        nfc_cases = [
            ("甲=『é』。\n", None),
            ("甲=『e\u0301』。\n", (5, "U+0301")),
            ("# 注释 e\u0301\n甲=『甲』。\n", (7, "U+0301")),
            ("甲e\u0301=『甲』。\n", (3, "U+0301")),
        ]
        for text, failure in nfc_cases:
            source.write_text(text, encoding="utf-8")
            seed = seed_lex(source)
            if failure is None:
                if seed.returncode:
                    raise RuntimeError((text, seed.stdout, seed.stderr))
                continue
            column, actual = failure
            expected = f"错误 S-1004：第 1 行第 {column} 列".encode()
            if (
                seed.returncode != 1
                or expected not in seed.stderr
                or actual.encode() not in seed.stderr
                or seed.stdout
            ):
                raise RuntimeError((text, seed.stdout, seed.stderr))
        source.write_bytes(b"\xc0\x8a")
        expect_run(scanner, b"", status=1, args=(source,), stdin=b"0\n")

        source.write_text(program("『" + "a" * (1 << 20) + "』"), encoding="utf-8")
        oversized = work / "oversized"
        result = run([SERICA, "build", source, "-o", oversized], capture_output=True, timeout=10)
        if result.returncode != 1 or "文本段超过 1 MiB".encode() not in result.stderr or oversized.exists():
            raise RuntimeError((result.returncode, result.stderr))
        print(
            f"P1b 词元门通过：{len(streams)} 个正例、{count} 个词元、"
            f"词法器自身 {self_tokens} 个词元、{len(errors)} 个诊断位置、"
            f"{len(nfc_cases)} 个 NFC 边界、多页 Mach-O 同路径重建"
        )


if __name__ == "__main__":
    main()
