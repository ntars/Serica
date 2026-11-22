<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0079：五层嵌套运行时文本拼接

- 日期：2026-11-21
- 状态：已验证；P1b 阶段门仍未通过
- 范围：`textConcat` 每条表达式路径的深度及原生 ARM64 代码生成

## 决定

将运行时 `textConcat` 的最大嵌套深度从四层提高到五层。每层仍使用两个独立的 256 字节操作数区，并按 512 字节层距分配 scratch；静态/运行时栈帧计算会按实际表达式深度扩展。六层表达式在 `check` 与 `build` 中以 S-3044 拒绝。

保留各层中间结果最多 63 字节、动态文本变量写回最多 16 MiB−1 字节、根输出最多 63 字节的边界。UTF-8 校验、别名安全和运行时失败无部分输出语义不变。

## 验收

- [x] `examples/p1b-runtime-text-concat-five-levels.serica` 对输入“行简”生成 ARM64 Mach-O 并输出“行简甲乙丙丁戊”。临时原生产物覆盖五层变量写回、`equals` 比较与排序比较；对应条件均进入真分支。
- [x] 六层样例的 `check` 和 `build` 均返回 S-3044。既有 63 字节中间结果、根输出和 16 MiB−1 字节变量写回限制继续通过。
- [x] `make stage0 && make test-p1b-runtime` 通过：10 个嵌套拼接产物、6 个文本相等比较、3 个文本排序比较、139 个成功文本输入、96 个编译诊断和 96 个运行时拒绝样例。
- [x] `make test-p0b-scope test-p1b-static test-p1b-modules doctor license`、`make seed-audit`、`./tools/check-independence source`、5 个新增/相关示例的 punct-scan、`python3 -m json.tool DECISIONS.json` 和 `git diff --check` 全通过。种子保持 6500/6500 行；C-1 为 4239/6019（70.4%）。

## 阶段影响

五层动态拼接可用于根输出、变量写回和运行时文本相等/排序条件。超过五层的拼接、一般文本表达式、结构化错误传播、哈希表、一般文件/进程 IO 和行简编译器仍未完成；不得据此将 P1b 或项目标记完成。
