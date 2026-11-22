# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Enforce the Go seed's frozen subset without parsing comments or literals."""

import os
import pathlib
import re
import shutil
import subprocess
import sys


ROOT = pathlib.Path(__file__).resolve().parent.parent
SEED = ROOT / "bootstrap"
ATTRIBUTION = ROOT / "docs" / "build" / "seed-attribution.md"
MAX_SEED_LINES = 12000


def scan_source(source: str):
    """Return comment/string-masked code and tokens with source line numbers."""
    masked = list(source)
    tokens = []
    i = 0
    line = 1
    while i < len(source):
        ch = source[i]
        if ch == "\n":
            line += 1
            i += 1
            continue
        if ch.isspace():
            i += 1
            continue
        if source.startswith("//", i):
            end = source.find("\n", i)
            end = len(source) if end < 0 else end
            for pos in range(i, end):
                masked[pos] = " "
            i = end
            continue
        if source.startswith("/*", i):
            end = source.find("*/", i + 2)
            end = len(source) if end < 0 else end + 2
            for pos in range(i, end):
                if source[pos] == "\n":
                    line += 1
                else:
                    masked[pos] = " "
            i = end
            continue
        if ch in ('"', "'", "`"):
            start, start_line, quote = i, line, ch
            i += 1
            value = []
            while i < len(source):
                current = source[i]
                if current == "\n":
                    line += 1
                if current == quote:
                    i += 1
                    break
                if quote != "`" and current == "\\" and i + 1 < len(source):
                    value.append(source[i + 1])
                    i += 2
                    continue
                value.append(current)
                i += 1
            for pos in range(start, i):
                if source[pos] != "\n":
                    masked[pos] = " "
            tokens.append(("literal", "".join(value), start_line))
            continue
        if ch == "_" or ch.isalpha():
            start = i
            i += 1
            while i < len(source) and (source[i] == "_" or source[i].isalnum()):
                i += 1
            tokens.append(("ident", source[start:i], line))
            continue
        if source.startswith("<-", i):
            tokens.append(("symbol", "<-", line))
            i += 2
            continue
        tokens.append(("symbol", ch, line))
        i += 1
    return "".join(masked), tokens


def import_paths(tokens):
    paths = []
    i = 0
    while i < len(tokens):
        if tokens[i][1] != "import":
            i += 1
            continue
        i += 1
        if i < len(tokens) and tokens[i][1] == "(":
            i += 1
            while i < len(tokens) and tokens[i][1] != ")":
                kind, value, line = tokens[i]
                if kind == "literal":
                    paths.append((value, line))
                i += 1
            i += 1
        elif i < len(tokens) and tokens[i][0] == "literal":
            paths.append((tokens[i][1], tokens[i][2]))
            i += 1
    return paths


def global_var_initializers(tokens):
    errors = []
    braces = 0
    i = 0
    while i < len(tokens):
        kind, value, line = tokens[i]
        if braces == 0 and kind == "ident" and value == "var":
            j = i + 1
            if j < len(tokens) and tokens[j][1] == "(":
                j += 1
                depth = 1
                while j < len(tokens) and depth:
                    if tokens[j][1] == "(":
                        depth += 1
                    elif tokens[j][1] == ")":
                        depth -= 1
                    elif depth and tokens[j][1] == "=":
                        errors.append((line, "包级变量初始化"))
                        break
                    j += 1
            else:
                while j < len(tokens) and tokens[j][2] == line:
                    if tokens[j][1] == "=":
                        errors.append((line, "包级变量初始化"))
                        break
                    if tokens[j][1] in (";", "}"):
                        break
                    j += 1
        if value == "{":
            braces += 1
        elif value == "}":
            braces = max(0, braces - 1)
        i += 1
    return errors


def main() -> int:
    go = os.environ.get("GO", "go")
    if shutil.which(go) is None and go == "go":
        local_go = pathlib.Path.home() / ".local" / "go1.27.4" / "bin" / "go"
        if local_go.is_file() and os.access(local_go, os.X_OK):
            go = str(local_go)
    if shutil.which(go) is None and not pathlib.Path(go).is_file():
        print("Go 种子子集检查：失败（找不到 Go；请设置 GO 或配置 PATH）", file=sys.stderr)
        return 1
    env = os.environ.copy()
    env.update({"GO111MODULE": "off", "GOPROXY": "off", "GOSUMDB": "off"})
    failures = []
    files = sorted(
        path
        for path in SEED.rglob("*.go")
        if "archive" not in path.relative_to(SEED).parts
    )
    for path in files:
        source = path.read_text(encoding="utf-8")
        code, tokens = scan_source(source)
        rel = path.relative_to(ROOT)
        rules = (
            (r"\binterface\s*\{", "interface"),
            (r"\bgo\s+", "goroutine"),
            (r"\bchan\b|<-", "channel"),
            (r"\breflect\s*\.", "反射"),
            (r"\bfunc\s+init\s*\(", "init 函数"),
            (r"\b(?:defer|goto|select)\b", "非白名单控制结构"),
            (r"\btype\s+\w+\s*\[\s*\w+\s+\w+", "泛型类型参数"),
            (r"\bfunc(?:\s+\([^)]*\))?\s+\w+\s*\[\s*\w+\s+\w+", "泛型函数参数"),
        )
        for pattern, label in rules:
            for match in re.finditer(pattern, code):
                line = code.count("\n", 0, match.start()) + 1
                failures.append(f"{rel}:{line}: 禁用特性：{label}")
        failures.extend(
            f"{rel}:{line}: 禁用特性：{label}"
            for line, label in global_var_initializers(tokens)
        )
        for package, line in import_paths(tokens):
            if package == "C":
                failures.append(f"{rel}:{line}: 种子中禁止 cgo")
                continue
            if package in ("reflect", "unsafe"):
                failures.append(f"{rel}:{line}: 禁止导入 {package}")
                continue
            result = subprocess.run(
                [go, "list", "-f", "{{.Standard}}", package],
                cwd=ROOT,
                env=env,
                text=True,
                capture_output=True,
                check=False,
            )
            if result.returncode != 0 or result.stdout.strip() != "true":
                failures.append(f"{rel}:{line}: 非标准库导入：{package}")

    attribution = ATTRIBUTION.read_text(encoding="utf-8")
    coverage = {path.relative_to(ROOT): set() for path in files}
    ast_coverage = {path.relative_to(ROOT): set() for path in files}
    for row in attribution.splitlines():
        match = re.search(r"`(bootstrap/[^`]+\.go):(\d+)-(\d+)`", row)
        if match is None:
            continue
        rel = pathlib.Path(match.group(1))
        start, end = int(match.group(2)), int(match.group(3))
        if rel not in coverage:
            failures.append(f"{ATTRIBUTION.relative_to(ROOT)}: 未知或归档文件范围：{rel}")
            continue
        source = (ROOT / rel).read_text(encoding="utf-8")
        lines = source.splitlines()
        if start < 1 or end < start or end > len(lines):
            failures.append(f"{ATTRIBUTION.relative_to(ROOT)}: {rel}:{start}-{end} 超出源码行数 {len(lines)}")
            continue
        coverage[rel].update(range(start, end + 1))
        if "【AST】" in row:
            ast_coverage[rel].update(range(start, end + 1))

    total_lines = 0
    ast_code_lines = 0
    ast_basis_lines = 0
    for path in files:
        rel = path.relative_to(ROOT)
        lines = path.read_text(encoding="utf-8").splitlines()
        total_lines += len(lines)
        if len(lines) > MAX_SEED_LINES:
            failures.append(f"{rel}: 单文件 {len(lines)} 行，超过种子上限 {MAX_SEED_LINES}")
        for number, line in enumerate(lines, start=1):
            if line.strip() and number not in coverage[rel]:
                failures.append(f"{ATTRIBUTION.relative_to(ROOT)}: 未归因 {rel}:{number}")
    for path in SEED.glob("*.go"):
        rel = path.relative_to(ROOT)
        lines = path.read_text(encoding="utf-8").splitlines()
        for number, line in enumerate(lines, start=1):
            code_line = bool(line.strip()) and not line.lstrip().startswith("//")
            if code_line:
                ast_basis_lines += 1
                if number in ast_coverage.get(rel, set()):
                    ast_code_lines += 1
    if total_lines > MAX_SEED_LINES:
        failures.append(f"活动种子共 {total_lines} 行，超过 {MAX_SEED_LINES} 行上限")

    ast_ratio = ast_code_lines / ast_basis_lines if ast_basis_lines else 0
    language_review = ROOT / "docs" / "decisions" / "0009-seed-language-is-go.md"
    language_triggered = total_lines > 3000 and ast_ratio > 0.40
    if language_triggered:
        review = language_review.read_text(encoding="utf-8")
        if "C-1 复核（保留 Go 临时桥）" not in review:
            failures.append("Go 种子触发 C-1 AST 占比条件，但 D-0009 未记录所有者授权后的语言复核")

    if failures:
        print("Go 种子子集检查：失败", file=sys.stderr)
        print("\n".join(failures), file=sys.stderr)
        return 1
    print(f"Go 种子子集检查：通过（{len(files)} 个活动源码文件，{total_lines} 行，归因完整）")
    print(f"C-1 AST 归因：{ast_code_lines}/{ast_basis_lines} 行（{ast_ratio:.1%}）；种子超过 3000 行且占比超过 40% 时须记录语言复核")
    print("C-1 条件已触发；D-0009/D-0059 复核记录通过" if language_triggered else "C-1 条件未触发")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
