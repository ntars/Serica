# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0007：自举路线与零依赖定义

## 决策

自举分 S0 种子、S1 自举、S2 淘汰三阶段：stage0 由 Go 种子生成，stage1 为行简编译器，stage2 与 stage3 归一化后逐字节一致时删除 `bootstrap/`。完全独立须同时通过源码级、链接级、构建级三层验证；A 档是唯一可发布档。

## 取舍

接受有限、可审计、5000 行封顶的 Go 种子作为明确脚手架；不把“能生成 stage1”误报为自举通过，也不把自举误报为独立或不套壳。

## 验证

阶段产物按 stage0 至 stage3 分开命名；stage2/stage3 比较 `.text`、`.rodata`、重定位、导入符号四段。源码、链接、构建命令按附录 K 执行。依据：手册第二节、4.0、4.6、7.1 T-4。
