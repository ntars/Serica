# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Verify deterministic static module loading and boundary diagnostics."""

import os
import pathlib
import subprocess
import sys
import tempfile

from native_artifact import verify_native_artifact, verify_native_console_artifact


ROOT = pathlib.Path(__file__).resolve().parent.parent
SERICA = ROOT / "build" / "stage0" / "serica"


def run(*args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [str(SERICA), *args], cwd=ROOT, text=True, capture_output=True, check=False
    )


def write_source(path: pathlib.Path, imports: list[str]) -> None:
    path.write_text(
        "".join(f"导入 『{item}』。\n" for item in imports)
        + "窗口=【标题=『模块测试』，文本=甲+乙】。\n",
        encoding="utf-8",
    )


def main() -> int:
    positive = run("check", "examples/p1b-modules.serica")
    if positive.returncode != 0 or "源程序结构：通过" not in positive.stdout:
        sys.stderr.write(positive.stdout + positive.stderr)
        print("模块闭包正例未通过", file=sys.stderr)
        return 1

    with tempfile.TemporaryDirectory(prefix="serica-modules-") as temporary:
        temp = pathlib.Path(temporary)
        artifact = temp / "modules"
        built = run("build", "examples/p1b-modules.serica", "-o", str(artifact))
        if built.returncode != 0 or "行简模块通过行简，共享依赖去重".encode() not in artifact.read_bytes():
            sys.stderr.write(built.stdout + built.stderr)
            print("模块闭包产物缺少预期文本", file=sys.stderr)
            return 1
        try:
            verify_native_artifact(artifact)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1

        fixtures = {
            "examples/diagnostics/p1b-module-absolute-path.serica": "S-4001",
            "examples/diagnostics/p1b-module-conflict.serica": "S-4005",
            "examples/diagnostics/p1b-module-cycle.serica": "S-4003",
            "examples/diagnostics/p1b-module-dependency-conflict.serica": "S-4005",
            "examples/diagnostics/p1b-module-invalid-extension.serica": "S-4001",
            "examples/diagnostics/p1b-module-invalid-path.serica": "S-4001",
            "examples/diagnostics/p1b-module-missing.serica": "S-4002",
            "examples/diagnostics/p1b-module-shadow.serica": "S-4005",
            "examples/diagnostics/p1b-module-window.serica": "S-4004",
        }
        for source, expected_code in fixtures.items():
            checked = run("check", source)
            output = checked.stdout + checked.stderr
            if checked.returncode == 0 or f"错误 {expected_code}" not in output:
                sys.stderr.write(output)
                print(f"模块诊断不符：{source}，期望 {expected_code}", file=sys.stderr)
                return 1

        located = run("check", "examples/diagnostics/p1b-module-window.serica")
        if "modules/has-window.serica:" not in located.stderr:
            sys.stderr.write(located.stdout + located.stderr)
            print("模块语义诊断没有定位到导入文件", file=sys.stderr)
            return 1

        project = temp / "project"
        modules = project / "modules"
        modules.mkdir(parents=True)
        (modules / "a.serica").write_text("常量 甲=『甲』。\n", encoding="utf-8")
        (modules / "b.serica").write_text("常量 乙=『乙』。\n", encoding="utf-8")
        first = project / "first.serica"
        second = project / "second.serica"
        write_source(first, ["modules/a.serica", "modules/b.serica"])
        write_source(second, ["modules/b.serica", "modules/a.serica"])
        shared_artifact = temp / "ordered"
        first_build = run("build", str(first), "-o", str(shared_artifact))
        if first_build.returncode != 0:
            sys.stderr.write(first_build.stdout + first_build.stderr)
            print("模块正序构建失败", file=sys.stderr)
            return 1
        first_bytes = shared_artifact.read_bytes()
        second_build = run("build", str(second), "-o", str(shared_artifact))
        if second_build.returncode != 0 or shared_artifact.read_bytes() != first_bytes:
            sys.stderr.write(second_build.stdout + second_build.stderr)
            print("导入顺序改变了构建产物", file=sys.stderr)
            return 1

        function_project = temp / "function-project"
        function_modules = function_project / "modules"
        function_modules.mkdir(parents=True)
        (function_modules / "math.serica").write_text(
            "函数 倍增（数值）=数值*2。\n"
            "函数 三倍（数值）=倍增（数值+0）+数值。\n"
            "函数 模块修正（数值）【变量 当前=数值*2。若（当前>50）【赋值 当前=当前-1。返回 当前。】。返回 当前。】。\n"
            "函数 读取（）能力=array（『stdin』）=readInteger（）。\n"
            "函数 追加（数组）能力=array（『array.write』）=arrayPush（数组，42）。\n",
            encoding="utf-8",
        )
        (function_modules / "offset.serica").write_text(
            "函数 加一（数值）=数值+1。\n", encoding="utf-8"
        )
        function_roots = [
            function_project / "main.serica",
            function_project / "reordered.serica",
        ]
        for root_path, imports in zip(
            function_roots,
            [
                ["modules/math.serica", "modules/offset.serica"],
                ["modules/offset.serica", "modules/math.serica"],
            ],
        ):
            root_path.write_text(
                "".join(f"导入 『{item}』。\n" for item in imports)
                + "程序=【能力=array（『stdout』），输出=三倍（14）】。\n",
                encoding="utf-8",
            )
        function_artifact = temp / "module-function"
        imported_function_build = run(
            "build", str(function_roots[0]), "-o", str(function_artifact)
        )
        if imported_function_build.returncode != 0:
            sys.stderr.write(imported_function_build.stdout + imported_function_build.stderr)
            print("模块闭包中的纯函数无法构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(function_artifact)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        executed = subprocess.run(
            [str(function_artifact)], cwd=ROOT, text=True, capture_output=True, check=False
        )
        if executed.returncode != 0 or executed.stdout != "42\n" or executed.stderr:
            sys.stderr.write(executed.stdout + executed.stderr)
            print("模块闭包中的纯函数原生结果不符", file=sys.stderr)
            return 1
        function_bytes = function_artifact.read_bytes()
        reordered_build = run(
            "build", str(function_roots[1]), "-o", str(function_artifact)
        )
        if (
            reordered_build.returncode != 0
            or function_artifact.read_bytes() != function_bytes
        ):
            sys.stderr.write(reordered_build.stdout + reordered_build.stderr)
            print("纯函数模块导入顺序改变了构建产物", file=sys.stderr)
            return 1

        block_function_root = function_project / "block.serica"
        block_function_root.write_text(
            "导入 『modules/math.serica』。\n"
            "程序=【能力=array（『stdout』），输出=模块修正（26）】。\n",
            encoding="utf-8",
        )
        block_function_artifact = temp / "module-block-function"
        block_function_build = run(
            "build", str(block_function_root), "-o", str(block_function_artifact)
        )
        if block_function_build.returncode != 0:
            sys.stderr.write(block_function_build.stdout + block_function_build.stderr)
            print("模块闭包中的块体函数无法构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(block_function_artifact)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        block_function_run = subprocess.run(
            [str(block_function_artifact)], cwd=ROOT, text=True, capture_output=True, check=False
        )
        if (
            block_function_run.returncode != 0
            or block_function_run.stdout != "51\n"
            or block_function_run.stderr
        ):
            sys.stderr.write(block_function_run.stdout + block_function_run.stderr)
            print("模块闭包中的块体函数结果不符", file=sys.stderr)
            return 1

        effect_root = function_project / "effect.serica"
        effect_root.write_text(
            "导入 『modules/math.serica』。\n"
            "程序=【能力=array（『stdout』，『stdin』），输出=读取（）】。\n",
            encoding="utf-8",
        )
        effect_artifact = temp / "module-effect-function"
        effect_build = run("build", str(effect_root), "-o", str(effect_artifact))
        if effect_build.returncode != 0:
            sys.stderr.write(effect_build.stdout + effect_build.stderr)
            print("模块闭包中的 stdin 效果函数无法构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(effect_artifact, require_runtime_input=True)
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        effect_run = subprocess.run(
            [str(effect_artifact)],
            cwd=ROOT,
            input="42\n",
            text=True,
            capture_output=True,
            check=False,
        )
        if effect_run.returncode != 0 or effect_run.stdout != "42\n" or effect_run.stderr:
            sys.stderr.write(effect_run.stdout + effect_run.stderr)
            print("模块闭包中的 stdin 效果函数结果不符", file=sys.stderr)
            return 1

        array_effect_root = function_project / "array-effect.serica"
        array_effect_root.write_text(
            "导入 『modules/math.serica』。\n"
            "变量 数组=array（7）。赋值 数组=追加（数组）。\n"
            "程序=【能力=array（『stdout』，『array.write』），"
            "输出=arrayGet（数组，arrayLength（数组）-1）】。\n",
            encoding="utf-8",
        )
        array_effect_artifact = temp / "module-array-effect-function"
        array_effect_build = run(
            "build", str(array_effect_root), "-o", str(array_effect_artifact)
        )
        if array_effect_build.returncode != 0:
            sys.stderr.write(array_effect_build.stdout + array_effect_build.stderr)
            print("模块闭包中的 array.write 效果函数无法构建", file=sys.stderr)
            return 1
        try:
            verify_native_console_artifact(
                array_effect_artifact, required_imports=("_mmap", "_munmap")
            )
        except RuntimeError as failure:
            print(failure, file=sys.stderr)
            return 1
        array_effect_run = subprocess.run(
            [str(array_effect_artifact)],
            cwd=ROOT,
            text=True,
            capture_output=True,
            check=False,
        )
        if (
            array_effect_run.returncode != 0
            or array_effect_run.stdout != "42\n"
            or array_effect_run.stderr
        ):
            sys.stderr.write(array_effect_run.stdout + array_effect_run.stderr)
            print("模块闭包中的 array.write 效果函数结果不符", file=sys.stderr)
            return 1

        duplicate_function_project = temp / "duplicate-function-project"
        duplicate_function_modules = duplicate_function_project / "modules"
        duplicate_function_modules.mkdir(parents=True)
        duplicate_function_body = "函数 计算（数值）=数值+1。\n"
        (duplicate_function_modules / "a.serica").write_text(
            duplicate_function_body, encoding="utf-8"
        )
        (duplicate_function_modules / "b.serica").write_text(
            duplicate_function_body, encoding="utf-8"
        )
        duplicate_function_root = duplicate_function_project / "main.serica"
        duplicate_function_root.write_text(
            "导入 『modules/a.serica』。\n"
            "导入 『modules/b.serica』。\n"
            "程序=【能力=array（『stdout』），输出=计算（1）】。\n",
            encoding="utf-8",
        )
        duplicate_function_check = run("check", str(duplicate_function_root))
        duplicate_function_diagnostic = (
            duplicate_function_check.stdout + duplicate_function_check.stderr
        )
        if (
            duplicate_function_check.returncode == 0
            or "错误 S-4006：" not in duplicate_function_diagnostic
        ):
            sys.stderr.write(duplicate_function_diagnostic)
            print("跨模块重复函数名未返回 S-4006", file=sys.stderr)
            return 1

        outside = temp / "outside.serica"
        outside.write_text("常量 外部=『不允许』。\n", encoding="utf-8")
        link_project = temp / "link-project"
        link_project.mkdir()
        if hasattr(os, "symlink"):
            (link_project / "linked.serica").symlink_to(outside)
            link_root = link_project / "main.serica"
            write_source(link_root, ["linked.serica"])
            checked = run("check", str(link_root))
            if checked.returncode == 0 or "错误 S-4001" not in checked.stderr:
                sys.stderr.write(checked.stdout + checked.stderr)
                print("模块符号链接边界未被拒绝", file=sys.stderr)
                return 1

            alias_project = temp / "alias-project"
            alias_project.symlink_to(link_project, target_is_directory=True)
            alias_root = alias_project / "main.serica"
            checked = run("check", str(alias_root))
            if checked.returncode == 0 or "错误 S-4001" not in checked.stderr:
                sys.stderr.write(checked.stdout + checked.stderr)
                print("项目根目录符号链接未被拒绝", file=sys.stderr)
                return 1

        invalid_project = temp / "invalid-project"
        invalid_project.mkdir()
        (invalid_project / "invalid.serica").write_bytes(b"\xff\xfe")
        invalid_root = invalid_project / "main.serica"
        write_source(invalid_root, ["invalid.serica"])
        checked = run("check", str(invalid_root))
        if checked.returncode == 0 or "错误 S-4002" not in checked.stderr:
            sys.stderr.write(checked.stdout + checked.stderr)
            print("无效 UTF-8 模块边界未被拒绝", file=sys.stderr)
            return 1

        if hasattr(os, "mkfifo"):
            fifo_project = temp / "fifo-project"
            fifo_project.mkdir()
            os.mkfifo(fifo_project / "pipe.serica")
            fifo_root = fifo_project / "main.serica"
            write_source(fifo_root, ["pipe.serica"])
            checked = run("check", str(fifo_root))
            if checked.returncode == 0 or "错误 S-4002" not in checked.stderr:
                sys.stderr.write(checked.stdout + checked.stderr)
                print("非普通模块文件未被拒绝", file=sys.stderr)
                return 1

    print(
        f"P1b 模块子门通过：1 个常量闭包产物、1 个纯函数原生产物、1 个块体函数原生产物、"
        "1 个 stdin 效果函数原生产物、"
        f"{len(fixtures)} 个路径/语义诊断、跨模块重复函数诊断、"
        "stdin/array.write 效果函数导入、函数导入顺序稳定及路径/文件类型边界"
    )
    print("此结果不证明运行时、动态容器、IO 或错误传播门通过")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
