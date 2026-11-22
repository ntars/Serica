<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0118 行简表达式解析与节点表

2026-11-21 在 macOS arm64 通过 P1b 子门验收。

## 表达式解析

`compiler/expression-grammar.serica` 封装跳过空白的词元游标、运算符表、整数范围和普通字符串转义检查。`compiler/parse-expression.serica` 使用显式运算栈、值栈和嵌套框架，按种子优先级归约语法树。同级中缀从左结合，多词元运算符沿用 stage0 的词元组合规则。调用和记录按源码顺序保存实参与字段。括号会改变归约顺序。

节点编号是整数表的偏移，0 表示空。每节点四槽，第一槽按起点移 32 位、长度移 8 位和低 8 位类别打包。

| 类别 | 左槽 | 右槽 |
|---|---|---|
| 1 整数，2 字符串，3 引用 | 0 | 0 |
| 4 调用，5 记录 | 第一个子节点 | 子节点个数 |
| 6 字段 | 字段值节点 | 0 |
| 7 负整数字面量 | 数字词元的字节起点 | 0 |
| 20、30、40–44、50–51、60–61、70–72 中缀 | 左操作数 | 右操作数 |
| 80 一元负号 | 操作数 | 0 |

第四槽链接同列表的下一个节点。字面量保留原文范围，解码后的值供后续语义阶段生成。调用与记录的范围包含分隔符，字段范围指向名称，运算节点范围指向运算符。当前驱动接受一个完整表达式。解析指令和声明结构属于下一项。

查询 `-1` 返回根编号，非负整数读取对应节点表槽。语法失败返回 `-(字节偏移 * 16 + 错误类别)`。类别 1 为语法错误，2 为整数越界，3 为普通串转义非法，8 至 11 沿用词法错误。按游标读取时先报告实际遇到的错误。输入文件的 UTF-8 错误继续由文件读取边界报告。

## 种子修正

stage0 先检查循环条件，再用循环体所改变量的动态类型检查循环体。初值为零的下标栈深度可在后续迭代合法使用 `深-1`。常量下标仍受静态检查，动态越界由生成代码拒绝。

## 验收

macOS arm64 的 make test-p1b-parse 通过 14 棵表达式树、252 个节点、13 个与 stage0 匹配的诊断位置，以及循环下标成功和失败回归。覆盖 512 层分组和 96 项左结合加法。活动 Go 种子 9915/12000 行，C-1 为 6498/9493（68.5%）。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse doctor license seed-audit
./tools/check-independence source
build/punct-scan compiler/expression-grammar.serica compiler/parse-expression.serica
```

测试逐槽检查树形、源码跨度、链表终止和负号元数据，使用 stage0 对照诊断并运行共同支持的整数与布尔正例。生成的解析器通过原生 ARM64、dyld、符号白名单及签名检查。

macOS arm64 整组回归 `make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse doctor license seed-audit` 退出 0。另复跑新增的 120 个确定性生成表达式，接受与拒绝结果均与 stage0 相同。
