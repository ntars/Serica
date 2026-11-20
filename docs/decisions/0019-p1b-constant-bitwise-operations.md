# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# D-0019：P1b 常量位运算子集

## 决策

在整数常量表达式中支持 `bitAnd（左值，右值）`、`bitOr（左值，右值）` 与 `bitXor（左值，右值）`，并允许将结果用于 `substring` 索引。D-0021 将操作数扩展为有符号 64 位整数；三个函数按其 64 位二进制补码位模式执行与、或、异或，结果按有符号整数解释。位函数可以嵌套并与整数算术组合。

## 边界

stage0 在编译期求值。函数暂只允许出现在整数常量表达式中；D-0020 后也可作为顶层整数常量声明。仍不支持一般可变整数字段或运行时值。函数名遵循标准库 API 使用英文名称的既定规则；参数仍使用全角括号与逗号。

## 验收

`examples/p1b-bitwise.serica` 嵌套三种位函数计算索引，产物文本为“语言表位运算通过”，并通过 Mach-O/dyld 校验。`examples/diagnostics/p1b-bitwise-arity.serica` 返回 `S-3021` 非零诊断；两个源文件均通过 `punct-scan`。
