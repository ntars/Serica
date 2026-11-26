# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 行简（Serica）贡献者
"""Exercise the currently implemented compile-time P1b subset."""

import pathlib
import subprocess
import sys
import tempfile

from native_artifact import verify_native_artifact


ROOT = pathlib.Path(__file__).resolve().parent.parent
SERICA = ROOT / "build" / "stage0" / "serica"
ACCEPTED = {
    "examples/p1b-string-concat.serica": "点击这里编辑中文",
    "examples/p1b-string-slice.serica": "语言表可运行",
    "examples/p1b-string-equality.serica": "切片与相等通过，反例分支通过",
    "examples/p1b-string-order.serica": "123450089",
    "examples/p1b-string-length.serica": "静态码位长度为三",
    "examples/p1b-integer-arithmetic.serica": "力，整数运算通过",
    "examples/p1b-integer-shift-remainder.serica": "67",
    "examples/p1b-bitwise.serica": "语言表位运算通过",
    "examples/p1b-constants.serica": "语言表可解析",
    "examples/p1b-arrays.serica": "行简言表，数组通过",
}
REJECTED = {
    "examples/diagnostics/p1b-constant-duplicate.serica": "S-3007",
    "examples/diagnostics/p1b-constant-forward-reference.serica": "S-3022",
    "examples/diagnostics/p1b-constant-mixed-types.serica": "S-3023",
    "examples/diagnostics/p1b-division-zero.serica": "S-3024",
    "examples/diagnostics/p1b-substring-bounds.serica": "S-3013",
    "examples/diagnostics/p1b-array-empty.serica": "S-3030",
    "examples/diagnostics/p1b-array-mixed.serica": "S-3031",
    "examples/diagnostics/p1b-array-nested.serica": "S-3031",
    "examples/diagnostics/p1b-array-bounds.serica": "S-3032",
    "examples/diagnostics/p1b-array-negative-index.serica": "S-3032",
    "examples/diagnostics/p1b-array-non-array.serica": "S-3033",
    "examples/diagnostics/p1b-array-index-type.serica": "S-3034",
    "examples/diagnostics/p1b-array-get-arity.serica": "S-3035",
    "examples/diagnostics/p1b-array-length-arity.serica": "S-3035",
    "examples/diagnostics/p1b-array-length-type.serica": "S-3036",
    "examples/diagnostics/p1b-array-int-text.serica": "S-3012",
    "examples/diagnostics/p1b-array-string-index.serica": "S-3023",
}


def run(*args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [str(SERICA), *args], cwd=ROOT, text=True, capture_output=True, check=False
    )


def main() -> int:
    with tempfile.TemporaryDirectory(prefix="serica-p1b-") as temp:
        for source, expected_text in ACCEPTED.items():
            checked = run("check", source)
            if checked.returncode != 0 or "源程序结构：通过" not in checked.stdout:
                sys.stderr.write(checked.stdout + checked.stderr)
                print(f"P1b 静态正例检查失败：{source}", file=sys.stderr)
                return 1
            output = pathlib.Path(temp) / pathlib.Path(source).stem
            built = run("build", source, "-o", str(output))
            if built.returncode != 0:
                sys.stderr.write(built.stdout + built.stderr)
                print(f"P1b 静态正例构建失败：{source}", file=sys.stderr)
                return 1
            if expected_text.encode("utf-8") not in output.read_bytes():
                print(f"产物缺少预期文本 {expected_text!r}：{source}", file=sys.stderr)
                return 1
            try:
                verify_native_artifact(output)
            except RuntimeError as failure:
                print(failure, file=sys.stderr)
                return 1

        # Source text must be NFC (D-0213). Use a compatibility character that
        # is already NFC so the test still proves runtime strings are not
        # normalised with NFKC.
        compatibility = "\ufb01"
        plain = "fi"
        unicode_source = pathlib.Path(temp) / "unicode-order.serica"
        unicode_source.write_text(
            f"窗口=【标题=『顺序』，文本=ifElse（『{compatibility}』>『{plain}』 且 textLength（『{compatibility}』）==1 且 textLength（『{plain}』）==2，『码位序列』，『被归一化』）】。\n",
            encoding="utf-8",
        )
        checked = run("check", str(unicode_source))
        if checked.returncode != 0:
            sys.stderr.write(checked.stdout + checked.stderr)
            print("静态字符串比较或码位长度未接受 NFC 兼容字符测试输入", file=sys.stderr)
            return 1
        unicode_output = pathlib.Path(temp) / "unicode-order"
        built = run("build", str(unicode_source), "-o", str(unicode_output))
        if built.returncode != 0:
            sys.stderr.write(built.stdout + built.stderr)
            print("无法构建静态字符串码位顺序测试", file=sys.stderr)
            return 1
        if "码位序列".encode("utf-8") not in unicode_output.read_bytes():
            print("静态字符串比较或码位长度意外地执行了 NFKC 归一化", file=sys.stderr)
            return 1
        try:
            verify_native_artifact(unicode_output)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1

    for source, expected_code in REJECTED.items():
        checked = run("check", source)
        output = checked.stdout + checked.stderr
        if checked.returncode == 0 or f"错误 {expected_code}：" not in output:
            sys.stderr.write(output)
            print(f"P1b 静态负例诊断不符：{source}，期望 {expected_code}", file=sys.stderr)
            return 1

    print(
        f"P1b 编译期子集通过：{len(ACCEPTED)} 个正例产物、1 个 NFC 正例与 NFKC 非归一化边界、"
        f"{len(REJECTED)} 个诊断反例"
    )
    print("此结果不证明运行时、动态容器、哈希表、IO 或模块门通过")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
