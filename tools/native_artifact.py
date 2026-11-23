# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Run the macOS arm64 native-artifact acceptance checks."""

import pathlib
import re
import subprocess


def verify_native_artifact(path: pathlib.Path) -> None:
    checks = [
        (["file", str(path)], "Mach-O 64-bit executable arm64"),
        (["dyld_info", "-validate_only", str(path)], None),
        (["otool", "-L", str(path)], "/usr/lib/libSystem.B.dylib"),
        (["nm", "-u", str(path)], "_objc_msgSend"),
        (["codesign", "--verify", "--strict", str(path)], None),
    ]
    for command, expected in checks:
        result = subprocess.run(command, text=True, capture_output=True, check=False)
        output = result.stdout + result.stderr
        if result.returncode != 0 or expected is not None and expected not in output:
            raise RuntimeError(f"原生产物检查失败：{' '.join(command)}\n{output}")


def verify_native_console_artifact(
    path: pathlib.Path,
    require_integer: bool = True,
    require_variable_storage: bool = False,
    required_instructions: tuple[str, ...] = (),
    require_runtime_input: bool = False,
    required_imports: tuple[str, ...] = (),
) -> None:
    """Verify a native CLI artifact has only the expected system runtime calls."""
    checks = [
        (["file", str(path)], "Mach-O 64-bit executable arm64"),
        (["dyld_info", "-validate_only", str(path)], None),
        (["otool", "-L", str(path)], "/usr/lib/libSystem.B.dylib"),
        (["codesign", "--verify", "--strict", str(path)], None),
    ]
    if require_integer:
        checks.append((["otool", "-tV", str(path)], "udiv"))
    for command, expected in checks:
        result = subprocess.run(command, text=True, capture_output=True, check=False)
        output = result.stdout + result.stderr
        if result.returncode != 0 or expected is not None and expected not in output:
            raise RuntimeError(f"原生控制台产物检查失败：{' '.join(command)}\n{output}")
    if require_variable_storage or required_instructions:
        disassembly = subprocess.run(
            ["otool", "-tV", str(path)], text=True, capture_output=True, check=False
        )
        if disassembly.returncode != 0:
            raise RuntimeError(f"无法反汇编含变量的原生产物\n{disassembly.stdout}{disassembly.stderr}")
        if require_variable_storage:
            for operation in ("str", "ldr"):
                if not re.search(rf"\b{operation}\s+x0, \[x19(?:, #[^\]]+)?\]", disassembly.stdout):
                    raise RuntimeError(
                        f"变量产物缺少 {operation} x0 的栈槽指令\n"
                        f"{disassembly.stdout}{disassembly.stderr}"
                    )
        for instruction in required_instructions:
            if instruction not in disassembly.stdout:
                raise RuntimeError(
                    f"变量产物缺少栈槽指令 {instruction!r}\n{disassembly.stdout}{disassembly.stderr}"
                )
    symbols = subprocess.run(["nm", "-u", str(path)], text=True, capture_output=True, check=False)
    if symbols.returncode != 0:
        raise RuntimeError(f"无法读取原生控制台产物导入符号\n{symbols.stdout}{symbols.stderr}")
    imports = {line.split()[-1] for line in symbols.stdout.splitlines() if line.split()}
    expected_imports = {"_exit", "_write"}
    if require_runtime_input:
        expected_imports.add("_read")
    expected_imports.update(required_imports)
    if imports != expected_imports:
        raise RuntimeError(
            f"原生控制台产物导入符号不符：期望 {sorted(expected_imports)!r}，"
            f"实际 {sorted(imports)!r}\n{symbols.stdout}{symbols.stderr}"
        )
