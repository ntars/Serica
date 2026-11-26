#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者
"""Validate the machine-readable decision index and its Make targets."""

from __future__ import annotations

import json
import pathlib
import re
import sys


ROOT = pathlib.Path(__file__).resolve().parent.parent
INDEX = ROOT / "DECISIONS.json"
MAKEFILE = ROOT / "Makefile"
SEED_TOOL = ROOT / "tools" / "check_seed_subset.py"
MAX_SEED_PATTERN = re.compile(r"^MAX_SEED_LINES\s*=\s*(\d+)", re.MULTILINE)
BUDGET_IDS = {"D-0087", "D-0101", "D-0173", "D-0192", "D-0208", "D-0210", "D-0229"}
REQUIRED = ("id", "title", "decision", "check", "status", "source", "makeTarget")
STATUSES = {"settled", "pending", "superseded"}
ID_PATTERN = re.compile(r"D-(\d{4})(?:-[A-Z0-9-]+)?")
SOURCE_NUMBER = re.compile(r"(\d{4})")
MAKE_TARGET = re.compile(r"\bmake\s+((?:[-A-Za-z0-9_.]+\s*)+)")
TARGET_PATTERN = re.compile(r"^([A-Za-z][A-Za-z0-9-]*):", re.MULTILINE)


def make_targets() -> set[str]:
    return set(TARGET_PATTERN.findall(MAKEFILE.read_text(encoding="utf-8")))


def first_target(check: str) -> str:
    match = MAKE_TARGET.search(check)
    if match is None:
        return "none"
    for token in match.group(1).split():
        if not token.startswith("-") and "=" not in token:
            return token
    return "none"


def main() -> int:
    try:
        data = json.loads(INDEX.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        print(f"决策索引读取失败：{error}", file=sys.stderr)
        return 1
    decisions = data.get("decisions") if isinstance(data, dict) else None
    if not isinstance(decisions, list) or not decisions:
        print("决策索引缺少非空 decisions 数组", file=sys.stderr)
        return 1

    targets = make_targets()
    failures = []
    ids = set()
    executable = 0
    manual = 0
    for position, item in enumerate(decisions, start=1):
        if not isinstance(item, dict):
            failures.append(f"第 {position} 条不是对象")
            continue
        missing = [field for field in REQUIRED if not isinstance(item.get(field), str) or not item[field].strip()]
        if missing:
            failures.append(f"第 {position} 条缺少字段：{', '.join(missing)}")
            continue
        decision_id = item["id"]
        match = ID_PATTERN.fullmatch(decision_id)
        if match is None:
            failures.append(f"{decision_id}：编号格式无效")
        elif decision_id in ids:
            failures.append(f"{decision_id}：编号重复")
        else:
            ids.add(decision_id)
        if item["status"] not in STATUSES:
            failures.append(f"{decision_id}：未知状态 {item['status']}")
        source = ROOT / item["source"]
        if not source.is_file():
            failures.append(f"{decision_id}：来源文件不存在 {item['source']}")
        elif match is not None:
            source_number = SOURCE_NUMBER.search(source.name)
            if source_number is not None and source_number.group(1) != match.group(1):
                failures.append(f"{decision_id}：来源文件名编号不匹配 {item['source']}")
        target = item["makeTarget"]
        if target == "none":
            manual += 1
        elif target not in targets:
            failures.append(f"{decision_id}：makeTarget 不存在 {target}")
        else:
            executable += 1
        expected = first_target(item["check"])
        if target != expected:
            failures.append(f"{decision_id}：makeTarget={target} 与 check 首个目标 {expected} 不一致")

    seed_source = SEED_TOOL.read_text(encoding="utf-8")
    match = MAX_SEED_PATTERN.search(seed_source)
    current_limit = int(match.group(1)) if match else None
    if current_limit is None:
        failures.append("种子预算检查失败：找不到 MAX_SEED_LINES")
    elif not any(item.get("id") in BUDGET_IDS and str(current_limit) in item.get("decision", "") for item in decisions):
        failures.append(f"种子预算改为 {current_limit} 行后没有对应的预算决策记录")

    if failures:
        print("决策索引检查：失败", file=sys.stderr)
        print("\n".join(failures), file=sys.stderr)
        return 1
    print(f"决策索引检查：通过（{len(decisions)} 条，{executable} 条有 Make 目标，{manual} 条手工验收）")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
