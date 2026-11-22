<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0126 绑定树的类型形态检查

`module-type` 消费模块图、名称绑定和能力检查结果。每条声明的类型来自初值或函数返回，跨文件引用沿绑定表读取同一类型槽。按模块完成顺序扫描，直到类型表稳定，再验证可达函数和顶层语句中的类型冲突。能力传播同时维护从入口及顶层表达式出发的函数可达性，因此未使用的纯函数保留类型信息，体内诊断留到调用后报告。

函数表达式直接返回形参时，按各次调用的实参返回类型。块函数在返回表达式类型都有证据且形态可汇合时，记录其返回形态。递归调用可从共同分支形态传播已知类型，遇到仍需实参推断的表达式则保留待定状态。推断循环使用节点表和原生控制流。

查询 `-(1 << 60) - n` 返回节点 `n` 的类型形态。传入函数名称节点时查询返回形态。既有模块、绑定和能力查询可在这个模式使用。

| 类型编号 | 形态 |
|---:|---|
| 0 | 待定或查询不适用 |
| 1–3 | 整数、文本、布尔 |
| 4–6 | 整数数组、文本数组、布尔数组 |
| 7–8 | 整数映射、记录 |
| 9–11 | 载荷待定的结果、整数结果、文本结果 |

验证覆盖已知类型的标量运算、内建实参、数组元素、条件分支、再赋值和控制台输出。常量值折叠、静态与运行时存储的区分、复杂能力数组、按实参展开的通用类型约束、块函数返回的完整路径检查及原生发射继续留在 P1b 清单。

## 验收

`make test-p1b-module-type` 在 macOS arm64 通过 26 个类型查询、20 个与 stage0 对齐的文件和位置诊断，以及编译器本身的 9 文件闭包。三个参照 Mach-O 实际运行，覆盖跨文件声明、同一纯函数的整数与文本调用、24 跳前向调用及递归。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse test-p1b-program test-p1b-bind test-p1b-import-paths test-p1b-module-read test-p1b-module-graph test-p1b-module-bind test-p1b-module-effect test-p1b-module-type doctor license seed-audit
./tools/check-independence source
build/punct-scan compiler/parse-expression.serica compiler/types.serica
```

三条命令均退出 0。15 个测试目标通过，编译器闭包为 9 文件、11 条依赖边和 94 条跨文件绑定。活动 Go 种子为 10282/12000 行，C-1 为 6586/9833（67.0%）。P1b 继续推进。
