<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0071 四层嵌套运行时文本拼接

**状态：** 子门通过（P1b 运行时文本表达式子门；P1b 阶段门仍未通过）。

## 决定

- 将运行时 `textConcat` 每条表达式路径的最大深度从三层扩展为四层；第五层在 `check` 和 `build` 中以 `S-3044` 拒绝。
- 每层拼接使用两个独立的 256 字节操作数缓冲区，按 512 字节层距分配。根输入/输出 scratch 保留在独立的基础区，避免四层表达式越过 4 KiB 栈帧边界。
- 保持每个嵌套中间值最多 63 字节、最外层变量写回最多 16 MiB−1 字节、根直接输出最多 63 字节，以及 UTF-8 校验和别名安全。

## 验收

- [x] 四层原生 ARM64 Mach-O 对输入 `行简` 输出 `行简甲乙丙丁`；根直接输出和动态文本变量写回路径均通过。
- [x] 第五层表达式在 `check` 和 `build` 均以 `S-3044` 拒绝；先前的一至三层用例、左右嵌套、`textLength`、63 字节中间值边界和无部分 stdout 失败继续通过。
- [x] `make test-p1b-runtime` 通过：76 个整数产物、6 个嵌套文本拼接产物、122 个成功文本输入、94 个编译诊断和 78 个运行时拒绝样例。
- [x] P0b、静态、模块、doctor、许可、Go 子集/归因、源码独立性、标点扫描、决策 JSON 和 `git diff --check` 全通过；活动种子 6418/6500 行，C-1 为 4159/5939（70.0%）。

复跑：`make stage0`、`make test-p0b-scope test-p1b-static test-p1b-modules doctor license`、`make test-p1b-runtime`、`make seed-audit`、`./tools/check-independence source`、`build/punct-scan examples/p1b-runtime-text-concat-nested.serica`、`python3 -m json.tool DECISIONS.json`、`git diff --check`。

## 阶段影响

四层嵌套使用紧凑、互不重叠的 scratch 区，可用于根输出和动态变量写回。更深表达式、任意动态文本表达式、哈希表、结构化错误传播、一般 IO 和行简编译器仍未完成，P1b 阶段门未通过。
