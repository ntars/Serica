<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0070 三层嵌套运行时文本拼接

**状态：** 子门通过（P1b 运行时文本表达式子门；P1b 阶段门仍未通过）。

## 决定

- 在 D-0069 的两层语义上，将运行时 `textConcat` 每条表达式路径的最大嵌套深度扩展为三层。第四层在 `check` 和 `build` 中以 `S-3044` 拒绝。
- 代码生成按嵌套深度在栈帧 scratch 区分配独立区域，每层预留 1024 字节。嵌套中间结果仍最多 63 字节；最外层动态变量写回可至 16 MiB−1 字节，根输出最多 63 字节。
- 保持 UTF-8 校验、别名安全、已登记系统调用、6500 行种子上限和 P3b 删除种子的要求。

## 验收

- [x] 三层原生 ARM64 产物对输入 `行简` 输出 `行简尾！终`；D-0069 的左右两侧嵌套、嵌套 `textLength` 和 63 字节中间结果边界继续通过。
- [x] 四层表达式在 `check` 与 `build` 均以 `S-3044` 拒绝。
- [x] `make test-p1b-runtime` 通过：76 个整数产物、4 个嵌套拼接产物、120 个成功文本输入、94 个编译诊断和 78 个运行时拒绝样例。
- [x] P0b、静态、模块、运行时、种子、doctor、许可、源码独立性、标点扫描和 `git diff --check` 全部通过；活动种子 6413/6500 行，C-1 为 4155/5935（70.0%）。

复跑：`make stage0`、`make test-p1b-runtime`、`make test-p0b-scope test-p1b-static test-p1b-modules doctor license`、`make seed-audit`、`./tools/check-independence source`、`build/punct-scan examples/p1b-runtime-text-concat-nested.serica`、`git diff --check`。

## 阶段影响

运行时 `textConcat` 可组合至三层，受有界栈帧 scratch、63 字节中间值和 16 MiB 动态文本上限约束。任意动态文本 AST、哈希表、结构化错误传播、一般 IO、运行时函数 ABI 和行简编译器仍未完成，P1b 阶段门未通过。
