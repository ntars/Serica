#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
"""Check source NFC, halfwidth code punctuation, and disabled characters."""

import sys
import unicodedata


def scan(src: str) -> tuple[int, int]:
    half = disabled = 0
    quote = ""
    comment = False
    for char in src:
        if comment:
            if char == "\n":
                comment = False
            continue
        if quote:
            if char == quote:
                quote = ""
            continue
        if char == "#":
            comment = True
        elif char == "『":
            quote = "』"
        elif char == "「":
            quote = "」"
        else:
            half += char in ",;():"
            code = ord(char)
            disabled += (
                code == 0x3000
                or 0x200B <= code <= 0x200D
                or code == 0xFEFF
                or 0xFF10 <= code <= 0xFF19
                or 0xFF21 <= code <= 0xFF3A
                or 0xFF41 <= code <= 0xFF5A
            )
    return half, disabled


def main() -> int:
    if len(sys.argv) < 2:
        print("用法：punct-scan <file.serica>", file=sys.stderr)
        return 2
    failed = False
    for path in sys.argv[1:]:
        try:
            data = open(path, "rb").read()
        except OSError as error:
            print(error, file=sys.stderr)
            failed = True
            continue
        try:
            source = data.decode("utf-8")
            valid = unicodedata.normalize("NFC", source) == source
        except UnicodeDecodeError:
            source = data.decode("utf-8", errors="replace")
            valid = False
        half, disabled = scan(source)
        if not valid:
            disabled += 1
        print(f"{path}：NFC={str(valid).lower()} 半角标点={half} 禁用字符={disabled}")
        failed |= not valid or half != 0 or disabled != 0
    return int(failed)


if __name__ == "__main__":
    raise SystemExit(main())
