<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0066 用户函数的 array.write 效果

状态：子门通过（P1b 效果传递子门；P1b 阶段门仍未通过）

## 决定

- 用户函数可用 `能力=array（『array.write』）` 声明动态数组变更效果。
- 函数体中的 `arrayPush` 和 `arraySet` 要求当前函数以及调用链中每一层函数都声明 `array.write`。它与 `stdin` 独立；读取输入仍需单独声明 `stdin`。
- 程序根也必须在 `能力` 数组声明 `array.write`，只要展开后的输出、变量初始化/赋值、条件或循环表达式中使用 `arrayPush`/`arraySet`。未调用的函数声明不要求根能力。
- 缺少函数签名或根能力使用现有 `S-4106` / `S-4102`；未知能力仍为 `S-4103`。运行时数组目标、别名和类型约束不变。
- 函数继续以有界 AST 展开执行，不增加调用 ABI、递归、其他可变状态、文件 IO 或网络 IO。

本决定不更改目标平台、符号白名单、许可、Go 子集、6500 行种子预算、独立性或 P3b 删除要求。

## 验收

- [x] 嵌套声明 `array.write` 的数组函数生成 ARM64 原生产物，真实执行 `arraySet` 后 `arrayPush` 并输出 `42`；静态模块导入的效果函数也执行并输出 `42`。
- [x] 函数任一层漏签名，以及根缺少 `array.write`，均由 `check` 和 `build` 稳定拒绝；直接数组变更也受根能力限制。90 个编译诊断覆盖既有 IO/能力反例及新增数组效果边界。
- [x] `stdin` 效果边界、数组运行时边界和既有 P0b、静态、模块、运行时、种子、doctor、许可与独立性门通过。当前运行时门执行 74 个整数 Mach-O 与 1 个嵌套 array.write 函数产物，116 个成功文本输入、90 个编译诊断、77 个运行时拒绝；模块门执行导入的 array.write 函数产物。

复跑：`make stage0`、`make test-p0b-scope`、`make test-p1b-static`、`make test-p1b-modules`、`make test-p1b-runtime`、`make doctor`、`make license`、`./tools/check-independence source`、`git diff --check`。`make seed-audit` 通过，当前 6270/6500 行，C-1 为 4001/5795（69.0%）。

## 阶段影响

仅关闭数组写入的能力声明缺口。哈希表、泛型容器、结构化错误传播、文件/进程 IO、运行时函数 ABI 和可编译行简编译器仍未实现，P1b 阶段门仍未通过。
