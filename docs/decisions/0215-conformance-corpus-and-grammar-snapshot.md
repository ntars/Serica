# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0215 一致性套件与语法实现快照落地

手动写出的语法规范若只凭构想，会与 parser 的实际接受形态分叉；而散落在 `tools/test_p1b_*.py` 中的门只回答“某个子门是否通过”，不能回答“这门语言当前支持什么”。本决策新增 `conformance/cases.json` 与 `conformance/sources/`，为每个特征保留正例和反例：正例由 stage0 构建成 arm64 产物后按 stdin 执行并逐字节比较 stdout 与退出码，反例必须由 `serica check` 报告指定 S-code 且非零退出。`make conformance` 是统一入口，并纳入 `make verify`。初始语料为 41 例（32 正例、9 反例），覆盖整数运算与比较、逻辑、条件选择、UTF-8 文本、切片/拼接/比较、数组、函数、变量、结果载荷、运行时输入、精确 stdout 写入，以及除零、类型错配、未知名、记录结构、元数、循环外跳出、能力缺失、NFC 与文本返回逃逸等诊断。

`docs/spec/grammar.md` 同步按 `bootstrap/parser.go` 的真实产生式修为“实现约束快照”：表达式按逻辑或、逻辑与、比较、移位、加法、乘法、一元、原子的层级展开；记录使用 `【字段名=值，…】`；语句包含 `删映射键`。该快照明确注明尚不构成 M4 规范冻结，后续仍需独立的人写规范与逐项对照。
