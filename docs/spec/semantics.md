# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 行简语义规范（M4 约束稿）

本文定义行简程序在运行前必须满足的语义约束。它是 [语法规范](grammar.md) 的下游约束，不重复词法与产生式；[模块与导入](modules.md) 定义文件、导入和名称空间；[诊断格式](diagnostics.md) 定义错误呈现。本文描述目标语义，而非宣称当前树已经通过 P1b–P6b。每条规则都给出可机器验证的一致性用例；`S-` 编号是稳定诊断标识。

## 1. 值与类型

| 类型 | 字面/构造 | 可观察行为 | 对应用例 |
|---|---|---|---|
| 整数 | `-42`、`0`、`7`、算术与比较结果 | 有符号 64 位；输出为十进制文本加换行 | `integer-literal`、`arith2-negative-literal`、`comparison-eq` |
| 文本 | `『…』`、`「…」`、文本内建结果 | UTF-8；码位计数用于 `textLength`，字节计数用于 `textByteLength` | `text-literal`、`text-length-codepoints`、`text-byte-length` |
| 数组 | `array（…）`、追加/更新结果 | 下标从 0 起，越界或非整数下标拒绝 | `array-index`、`reject-bad-array-index`、`reject-array-get-text` |
| 结果 | `成功（载荷）`、`失败（错误码）` | 成功载荷可是整数或受支持文本；错误码为 1–255 的整数 | `result-payload`、`result-failure-code`、`reject-bad-result-code` |
| 布尔 | 比较、`且`、`故` | 用整数 1/0 表示；`且`/`故` 短路 | `logic-and-both`、`logic-or-short`、`reject-text-length-int` |

类型必须静态一致：`ifElse` 两分支类型相同，函数实参类型与形参注释一致，数组元素类型一致。文本返回函数只能进入已开放的调用边界，不能静默降级为整数或任意容器值。违反时给出 `S-3023`、`S-3044` 或更具体的类型诊断。[D-0205](../decisions/0205-effectful-text-local-direct-return.md)

## 2. 作用域与名称

1. 文件依赖闭包共享平面名称空间；同名常量或跨文件同名函数是错误，见 [模块规范](modules.md#加载和名称解析)。
2. 顶层变量在同一文件中名称唯一；函数块内局部变量名称必须唯一，不能与常量或已有局部变量冲突。重复声明为 `S-3037`，见 `reject-duplicate-local`。
3. 赋值目标必须在此前声明且类型兼容；未声明目标为 `S-3038`，见 `reject-assign-unknown`。
4. 块作用域变量不得逃逸；只有顶层变量和已开放文本返回边界的产物可以成为程序输出或后续表达式操作数。
5. `跳出`、`继续` 只能绑定最近的 `凡` 循环；在循环外出现为 `S-3067`，见 `break-outside-loop`。

## 3. 求值与控制流

算术遵循通常优先级：括号优先于乘法类运算，乘法类优先于加法类，移位优先于比较，比较优先于逻辑。短路逻辑使用全角关键字 `且`、`故`，两操作数必须是布尔条件；关键字两侧需有空格以避开全角标识符边界，见 `logic-and-both`、`logic-or-false`。

`若` 的条件必须是布尔类型；`否` 分支可选。`凡` 在条件为真时重复执行块，直到条件为假或命中 `跳出`；`继续` 进入下一次条件判定。循环体可以更新顶层变量，结果对循环外可见，见 `loop-sum`、`loop-break`、`loop-continue`。

除已开放的运行时文本边界外，函数实参按源码顺序求值；调用方能力不足时诊断不再继续发射运行时副作用。错误路径必须在写出任何 stdout 之前以非零退出结束。

## 4. 效果与能力

效果通过能力名声明，并沿显式调用链逐层传播；根程序必须声明实际使用的能力。能力边界如下：

| 能力 | 允许的操作 | 缺失诊断 |
|---|---|---|
| `stdout` | `writeText`、`writeError`、程序结果输出 | `S-4102` |
| `stdin` | `readInteger`、`readByte`、`readText` | `S-4102` |
| `args` | `argumentCount`、`argumentText`、`readEnvironment` 的动态名称 | `S-4102` |
| `filesystem.read` | `readFile`、`exists`、`isDirectory`、`directoryNames` 静态路径 | `S-4102` |
| `filesystem.write` | 静态路径覆盖写入与目录变更 | `S-4102` |
| `array.write` | `arrayPush`、`arraySet` | `S-4102` |
| `map.write` | `映射置`、`映射删` | `S-4102` |
| `process` | `runCommand` | `S-4102` |

`arrayPush`、`arraySet` 是表达式，必须把结果赋回原数组变量，否则给出 `S-3033`；能力缺失示例见 `reject-read-int-no-stdin`、`reject-write-text-no-stdout`、`reject-args-no-capability`。[D-0206](../decisions/0206-stdout-write-boundary.md) [D-0228](../decisions/0228-process-command-status.md)

## 5. 常量与纯表达式

`常量` 在模块加载阶段求值；只能引用同文件更早声明的常量和纯函数。路径字面量、数组字面量、整数与文本运算可折叠；读取进程输入、环境、文件系统或当前时间的调用不是常量表达式。常量折叠失败必须给出诊断，不得在生成产物中保留未决副作用。对应测试为 `make test-p1b-module-const` 与 `func-const`。

## 6. 运行时边界

1. `readInteger`、`readText` 的输入为 UTF-8；非法输入在写出任何 stdout 前失败。见 `io-read-int-twice`。
2. `argumentText（下标）` 的下标必须为非负整数且位于 argv 范围；返回文本为 UTF-8。见 `io-args-count`。
3. `directoryNames` 只列当前目录一层，不排序、不递归，排除 `.` 与 `..`，见 [模块规范](modules.md) 中目录枚举段。
4. `runCommand` 通过 `/bin/sh -c` 执行，继承标准流；正常退出映射为 0–255，信号终止映射为 `128 + signal`。[D-0228](../decisions/0228-process-command-status.md)
5. 文本结果载荷与容器元素的所有权以 `make test-p1b-runtime` 的复制、替换和拒绝样例为准。受限文本数组结果值使用四槽描述符，元素按指针加长度保存；结果变量复制必须深复制元素，函数返回则转移所有权。未开放的文本返回嵌套、逃逸、容器参数和容器通用载荷仍属于 P1b 未完成项，不得视为已冻结语义。[D-0236](../decisions/0236-text-array-result-payload.md)

## 7. 一致性验证

本规范的机器入口是 `make conformance`，语料见 `conformance/cases.json`。每个正例断言精确 stdout 与退出码，每个反例断言稳定 `S-` 编号和非零退出。规则只增不减；新增或改变规则必须同时更新本文、语料、`DECISIONS.json` 和路线图。当前语料为 101 例，达到 M4 的数量门槛，但数量不等于规范冻结：语法、语义与实现仍需人工审查签字。
