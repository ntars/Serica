<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0130 运行时函数签名与 ARM64 布局

`module-abi` 沿用模块绑定及静态求值。它在绑定调用图中识别递归组和已识别的运行时块。只调用递归函数的无环包装函数仍按静态调用规则处理。已降为表达式的递归块使用表达式调用约定。

签名从调用点和函数体取得类型证据，再用已确定的形参类型复算绑定树。整数形参占一个寄存器。借用 UTF-8 文本占连续的指针与字节长度两个寄存器。已知整数局部占一个槽，独立文本局部占指针、长度和容量三个槽。检查包括八个传参寄存器、502 个函数状态槽、已知实参类型、借用形参写入及拥有型文本仅直接作为程序输出的边界。

## 查询

`-(4 << 60) - (n << 6) - k` 查询函数名称节点 `n` 的第 `k` 个字段。各字段依次为函数形式、形参数、传参寄存器数、返回约定、局部槽数、16 字节对齐后的状态帧字节数、返回寄存器数、拥有型返回局部的字节偏移。状态帧字节数不含保存帧指针、链接寄存器和状态基址的区域。形式为 1 的函数使用表达式调用约定，形式为 2 的函数使用块体约定，形式为 0 表示无运行时签名。局部槽数为 -1 表示仍待推断。拥有型返回偏移为 -1 表示无该字段。

从第 8 个字段起，每个形参依次登记形态和起始寄存器编号。形态 1 为整数，形态 2 为借用文本。查询 `-(5 << 60) - n` 返回原语法树调用节点 `n` 绑定的运行时函数名称节点。查询 `-(6 << 60) - n` 返回局部声明名称节点的 `(字节偏移 << 3) + 形态`。0 表示查询不适用或尚无布局。

| 返回约定 | 寄存器 |
|---:|---|
| 0 | 待定 |
| 1 | x0 整数 |
| 2 | x0 结果标签、x1 整数载荷或错误码 |
| 3 | x0 结果标签、成功时 x1 为 0、x2 为静态文本指针、x3 为字节长度。失败时 x1 为错误码 |
| 4 | x0 为 0，x1 为转交的文本指针，x2 为字节长度，x3 为映射容量 |

本门产物为可查询的签名及调用和局部布局。运行时容器、效果函数、其他通用返回值与行简原生发射在后续任务验收。

## 验收

macOS arm64 的 `make test-p1b-module-abi` 验证六个实际运行的 stage0 原生参照产物、四种返回约定、八个位置诊断、混合参数使用 x0–x7、81 个运行时示例及编译器自身闭包。反汇编核对参数从指定寄存器保存到对应栈槽、静态文本结果的指针和长度寄存器，以及拥有型返回的三个槽。另查验递归整数结果的失败退出码。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make -j4 stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse test-p1b-program test-p1b-bind test-p1b-import-paths test-p1b-module-read test-p1b-module-graph test-p1b-module-bind test-p1b-module-effect test-p1b-module-type test-p1b-module-const test-p1b-module-abi doctor license seed-audit
env GO=/Users/ntars/.local/go1.27.4/bin/go make test-p1b-module-abi
./tools/check-independence source
build/punct-scan compiler/parse-expression.serica
```

四条命令均退出 0。全量 17 个回归目标通过，纳入 81 个已有示例后复跑 ABI 子门。编译器闭包为 10 文件、12 条依赖边和 104 条跨文件绑定。活动种子为 10303/12000 行，C-1 为 6599/9854（67.0%）。P1b 继续推进。
