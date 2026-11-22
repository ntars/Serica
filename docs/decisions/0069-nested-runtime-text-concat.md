<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0069 有界嵌套运行时文本拼接

**状态：** 子门通过（P1b 运行时文本表达式子门；P1b 阶段门仍未通过）。

## 决定

- `textConcat（左，右）` 的运行时文本操作数可包含另一层 `textConcat`；按任一表达式路径最多两层。更深表达式由 `check` 与 `build` 返回 `S-3044`。
- 嵌套拼接中间结果保存在独立的 64 字节 scratch 缓冲区，运行时内容最多 63 字节。超过上限在 stdout 前以非零状态退出。根直接输出仍最多 63 字节；写回动态文本变量的最外层拼接仍可增长至 16 MiB−1 字节。
- 保持已有文本变量、静态字符串、`substring`、`arrayGet`、UTF-8 校验、别名安全和系统调用白名单约束；不开放任意动态文本表达式或递归表达式。

## 验收

- [x] 左侧和右侧各嵌套一层的输入文本拼接均生成并执行 ARM64 Mach-O；分别得到 `行简尾！` 与 `前中行简`。
- [x] `textLength（textConcat（textConcat（文本，『尾』），『！』））` 对输入 `行简` 输出码位数 `4`。
- [x] 嵌套中间结果超过 63 字节时原产物无 stdout、非零退出；三层表达式在 `check` 和 `build` 均以 `S-3044` 拒绝。
- [x] `make test-p1b-runtime` 通过：76 个整数产物、3 个两层嵌套拼接产物、119 个成功文本输入、94 个编译诊断和 78 个运行时拒绝样例。
- [x] P0b、静态、模块、运行时、种子、doctor、许可、源码独立性、标点扫描和 `git diff --check` 复跑通过；活动种子 6417/6500 行，C-1 为 4159/5939（70.0%）。

复跑：`make stage0`、`make test-p0b-scope test-p1b-static test-p1b-modules doctor license`、`make test-p1b-runtime`、`make seed-audit`、`./tools/check-independence source`、`build/punct-scan examples/p1b-runtime-text-concat-nested.serica`、`git diff --check`。

## 阶段影响

运行时拼接可组合一层动态拼接结果，覆盖左右嵌套和码位计数。一般文本 AST 求值、任意深度拼接、哈希表、结构化错误传播、一般 IO、运行时函数 ABI 与行简编译器仍缺失，P1b 阶段门未通过。
