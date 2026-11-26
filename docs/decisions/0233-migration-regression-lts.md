# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0233：迁移工具、回归语料与 LTS 策略的自动部分

## 背景

手册 S6.1–S6.4 要求弃用流程、至少一类自动迁移、LTS 窗口和把历史样例纳入 CI。仓库已有兼容策略与大量测试子门，但缺少可执行的迁移命令、独立的回归语料门和明确的 MAJOR 支持窗口。

## 决策

- 在 `docs/spec/compatibility.md` 登记 `判等（…）` 为历史拼写，替代写法为 `equals（…）`，移除版本为 1.0.0；当前只登记与迁移，不提前移除。
- 新增 `tools/migrate`，提供 `legacy-equality` 一条规则与 `--check`/`--write` 两种互斥模式。迁移只替换调用位置的标识符，不改写字符串、注释或普通标识符。
- 新增 `tools/regression-corpus.json` 与 `tools/check_regression.py`，用 10 例已有 `conformance/` 语料作为只增不减基线；`make regression` 在 CI 和 `make verify` 中执行，正例实际构建运行，反例检查 S-code 与位置。
- 在 `RELEASE.md` 写明预览包不进入 LTS；1.0 后每个 MAJOR 提供 12 个月安全修复窗口，MINOR 只新增兼容能力，破坏性变更需按兼容策略提前登记。

## 不做的事

- 不宣称编译器 warning 已实现；当前检查由独立迁移工具承担。
- 不宣称 1.0.0 或 LTS 已开始；这里只定义策略。
- 不宣称 S6.5 年度审计完成；该项仍需人工结论与行数记录。
- 不推进 P1b 阶段门或语言完整性；这些仍是未通过状态。
