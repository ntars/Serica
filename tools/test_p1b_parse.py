# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Check the executable Serica expression parser and its arena contract."""

import re
from pathlib import Path
from random import Random
from subprocess import run
from tempfile import TemporaryDirectory

from native_artifact import verify_native_console_artifact
from test_p1b_scan import ROOT, SERICA, compile_source, expect_run, program


MODULE_READ_IMPORTS = ("_mmap", "_munmap", "_open", "_close", "_openat", "_fcntl", "_fstat", "___error")
PARSER_IMPORTS = (*MODULE_READ_IMPORTS, "_creat")


def query(parser, source, index, mode=None):
    argv = [parser, source] if mode is None else [parser, source, mode]
    result = run(argv, input=f"{index}\n".encode(), capture_output=True, timeout=8)
    if result.returncode or result.stderr:
        raise RuntimeError((result.returncode, result.stdout, result.stderr))
    return int(result.stdout)


def read_tree(parser, source, data, root, seen, mode=None):
    if root <= 0 or (root - 1) % 4 or root in seen:
        raise RuntimeError(f"非法或重复的节点编号 {root}")
    seen.add(root)
    packed, left, right, sibling = (query(parser, source, root + word, mode) for word in range(4))
    kind, offset, width = packed & 255, packed >> 32, (packed >> 8) & 0xFFFFFF
    if not (kind == 101 and not data and width == offset == 0) and not 0 < width <= len(data) - offset:
        raise RuntimeError((kind, offset, width))
    children = []
    if kind in (4, 5) or 100 <= kind <= 115:
        current = left
        if not 0 <= right <= len(data):
            raise RuntimeError(f"无效列表长度 {right}")
        for _ in range(right):
            child, current = read_tree(parser, source, data, current, seen, mode)
            children.append(child)
        if current:
            raise RuntimeError("列表长度与同级链接不一致")
    elif kind in (6, 80):
        children.append(read_tree(parser, source, data, left, seen, mode)[0])
        if right:
            raise RuntimeError(f"一元节点有右子节点 {right}")
    elif kind >= 20:
        children.extend(read_tree(parser, source, data, node, seen, mode)[0] for node in (left, right))
    elif kind == 7:
        if not offset < left < offset + width or not data[left:offset + width].isdigit():
            raise RuntimeError("负整数字面量的数字起点无效")
    return (kind, data[offset:offset + width].decode(), tuple(children)), sibling


def node(kind, text, *children):
    return kind, text, children


def number(text):
    return node(1, text)


def reference(text):
    return node(3, text)


def check_generated_cases(parser, source, wrapped):
    rng = Random(118)

    def expression(depth):
        if depth == 0 or rng.randrange(4) == 0:
            return str(rng.randrange(-5, 6))
        return "（" + expression(depth - 1) + rng.choice("+-*") + expression(depth - 1) + "）"

    for index in range(120):
        text = expression(4)
        if index % 2:
            text += rng.choice(("）", " +", "，9"))
        source.write_text(text, encoding="utf-8")
        wrapped.write_text(program(text), encoding="utf-8")
        seed = run([SERICA, "check", wrapped], capture_output=True, timeout=8)
        if (query(parser, source, -1) > 0) != (seed.returncode == 0):
            raise RuntimeError((text, seed.stdout, seed.stderr))


def main():
    with TemporaryDirectory(prefix="serica-parse-") as temporary:
        work = Path(temporary)
        parser = work / "parser"
        compile_source(ROOT / "compiler/parse-expression.serica", parser)
        verify_native_console_artifact(
            parser, require_runtime_input=True,
            required_imports=PARSER_IMPORTS,
            required_instructions=("bl\t", "ret", "ldrb\tw0, [x1]"),
        )
        source, wrapped = work / "input.serica", work / "wrapped.serica"
        left_chain = number("1")
        for _ in range(95):
            left_chain = node(60, "+", left_chain, number("1"))
        positives = (
            ("1+2*3", node(60, "+", number("1"), node(70, "*", number("2"), number("3"))), b"7\n"),
            ("（1+2）*3", node(70, "*", node(60, "+", number("1"), number("2")), number("3")), b"9\n"),
            ("10-3-2", node(61, "-", node(61, "-", number("10"), number("3")), number("2")), b"5\n"),
            ("20/3%2", node(72, "%", node(71, "/", number("20"), number("3")), number("2")), b"0\n"),
            (
                "1 < = 2 且 8>>1>0 故 0==1",
                node(20, "故", node(30, "且", node(42, "< =", number("1"), number("2")),
                     node(41, ">", node(51, ">>", number("8"), number("1")), number("0"))),
                     node(44, "==", number("0"), number("1"))), b"1\n",
            ),
            ("1<<2+1>=8", node(43, ">=", node(50, "<<", number("1"),
             node(60, "+", number("2"), number("1"))), number("8")), b"1\n"),
            ("单 - -（2）", node(61, "-", reference("单"), node(80, "-", number("2"))), None),
            ("-9223372036854775808", node(7, "-9223372036854775808"), b"-9223372036854775808\n"),
            (
                "函（1，# 说明\n2*3，𠮷（））",
                node(4, "函（1，# 说明\n2*3，𠮷（））", number("1"),
                     node(70, "*", number("2"), number("3")), node(4, "𠮷（）")), None,
            ),
            (
                "【甲=函（- 2，3），乙=「abc」】",
                node(5, "【甲=函（- 2，3），乙=「abc」】",
                     node(6, "甲", node(4, "函（- 2，3）", node(7, "- 2"), number("3"))),
                     node(6, "乙", node(2, "「abc」"))), None,
            ),
            ("【】", node(5, "【】"), None),
            ("『甲\\n\\』』", node(2, "『甲\\n\\』』"), None),
            ("（" * 512 + "3" + "）" * 512, number("3"), None),
            ("1+" * 95 + "1", left_chain, b"96\n"),
        )
        node_count = 0
        for expression, expected, value in positives:
            data = expression.encode()
            source.write_bytes(data)
            root = query(parser, source, -1)
            seen = set()
            parsed, sibling = read_tree(parser, source, data, root, seen)
            if parsed != expected or sibling:
                raise RuntimeError((expression, parsed, expected, sibling))
            node_count += len(seen)
            if value is not None:
                wrapped.write_text(program(expression), encoding="utf-8")
                artifact = work / "seed-value"
                compile_source(wrapped, artifact)
                expect_run(artifact, value)

        # Coordinates are checked against stage0's expression in a program wrapper.
        failures = (
            ("1+", "", 1, "S-2001"),
            ("函（1，）", "）", 1, "S-2001"),
            ("【甲=1，】", "】", 1, "S-2001"),
            ("【甲 1】", "1", 1, "S-2001"),
            ("（1+2】", "】", 1, "S-2001"),
            ("𠮷（1\n 2）", "2", 1, "S-2001"),
            ("9223372036854775808", "922", 2, "S-3004"),
            ("-9223372036854775809", "-", 2, "S-3004"),
            ("『x\\w』", "『", 3, "S-1024"),
            ("7+😀", "😀", 8, "S-1001"),
            ("7+Ａ", "Ａ", 9, "S-1003"),
            ("函（1,2）", ",", 10, "S-1010"),
            ("1+「未闭合", "「", 11, "S-1023"),
        )
        wrapper = program("EXPRESSION")
        for expression, marker, tag, error in failures:
            source.write_text(expression, encoding="utf-8")
            at = expression.index(marker) if marker else len(expression)
            byte = len(expression[:at].encode())
            if query(parser, source, -1) != -(byte * 16 + tag):
                raise RuntimeError((expression, byte, tag))
            before = wrapper.partition("EXPRESSION")[0] + expression[:at]
            line, col = before.count("\n") + 1, len(before.rsplit("\n", 1)[-1]) + 1
            wrapped.write_text(wrapper.replace("EXPRESSION", expression), encoding="utf-8")
            seed = run([SERICA, "check", wrapped], capture_output=True, text=True, timeout=8)
            coordinates = rf":{line}:{col}（显示第 \d+ 格）：错误 {error}："
            if seed.returncode != 1 or not re.search(coordinates, seed.stderr):
                raise RuntimeError((expression, coordinates, seed.stdout, seed.stderr))

        carried = "变量 槽=array（7）。变量 深=0。变量 次=0。\n" \
            "凡（次<2）【若（深>0）【赋值 槽=arraySet（槽，深-1，12）。】。" \
            "赋值 深=1。赋值 次=次+1。】。\n" + program("arrayGet（槽，0）", "，『array.write』")
        source.write_text(carried, encoding="utf-8")
        loop = work / "loop"
        compile_source(source, loop)
        expect_run(loop, b"12\n")
        source.write_text(carried.replace("深-1", "深-2"), encoding="utf-8")
        compile_source(source, loop)
        expect_run(loop, b"", status=1)
        check_generated_cases(parser, source, wrapped)
        print(
            f"P1b 表达式树通过：{len(positives)} 棵树、{node_count} 个节点、"
            f"{len(failures)} 个诊断、120 个生成表达式及循环下标回归"
        )


if __name__ == "__main__":
    main()
