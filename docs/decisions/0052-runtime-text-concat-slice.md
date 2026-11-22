# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0052：运行时拼接结果切片

## 决策

`substring（文本，起点，终点）` 的运行时文本源可为已初始化文本变量，或单层 `textConcat（变量，静态字符串表达式）` 结果。拼接结果先放入独立的有界 scratch 缓冲区，再按 Unicode 标量值计算字节边界并切片；整数索引可为静态值或运行时表达式。拼接输入/结果最多 63 字节，超过时在任何 stdout 前失败；切片终点超出拼接结果也在输出前失败。

此子门只增加一层运行时文本组合。`textConcat` 操作数仍须为文本变量或静态字符串表达式，不支持任意嵌套动态字符串、动态增长或堆分配。UTF-8 输入须先通过现有标量校验，切片不做归一化，也不合并字素簇。

## 验收

- `make test-p1b-runtime` 构建并执行 `examples/p1b-runtime-text-concat-slice.serica` 与 `examples/p1b-runtime-text-concat-slice-length.serica` 两个 ARM64 Mach-O；覆盖 ASCII/中文跨动态与静态边界、CRLF、精确 63 字节结果及嵌套码位长度，并验证源范围不足、拼接超限、非法 UTF-8 三类失败均为非零退出且无 stdout/stderr。
- 运行时门通过 92 个成功文本输入、52 个编译诊断及 67 个运行时拒绝样例。`make seed-audit` 通过，活动种子 4993/5000 行并逐行归因。
- 该子门不代表 P1b 阶段门通过；动态容器、一般 IO、结构化错误传播和行简编译器仍未完成。
