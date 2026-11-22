<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0119 行简程序与模块语法树

## 解析入口

`compiler/parse-expression.serica` 使用同一份词元游标和节点表解析表达式、程序和模块。省略第二个参数时解析单个表达式。`program` 模式解析导入、声明、语句和末尾的 `程序` 或 `窗口` 根绑定。`module` 模式解析导入、常量与函数，允许空文件。模块路径解析、名称绑定、类型和效果检查交由后续语义阶段。

程序驱动按显式动作栈处理声明与控制流。`compiler/program-grammar.serica` 保存语句类别，并按 UTF-8 原文字节识别根标识符和重复形参。遇到表达式时，驱动启动 D-0118 的归约过程，保留外层分隔符并接回所得子树。

## 节点与诊断

四槽布局沿用 D-0118。对下表节点，首槽按起点、长度、类别打包，第二槽保存首子节点，第三槽保存子节点个数，第四槽链接同列表的下一节点。源文件节点覆盖全文件，语句范围包含终止句号，块与参数列表包含对应定界符。

| 类别 | 子节点顺序 |
|---|---|
| 100 程序，101 模块 | 源码顺序的顶层项 |
| 102 导入 | 路径字面量 |
| 103 常量，105 变量，106 赋值，112 根绑定 | 名称、表达式 |
| 104 函数 | 名称、参数列表、可选能力、表达式或块 |
| 107 若 | 条件、若块、可选否块 |
| 108 凡 | 条件、循环体 |
| 109 跳出，110 继续 | 无 |
| 111 返回，115 能力 | 表达式 |
| 113 块，114 参数列表 | 源码顺序的语句或参数 |

标准输入查询 `-1` 返回根编号，非负整数读取节点槽。语法失败返回 `-(字节偏移 * 16 + 错误类别)`。新增类别 4 为模块中不允许的结构、5 为根名称错误、6 为名称非法、12 为块函数语句非法、13 为程序嵌套变量非法、14 为形参重名。循环控制归属和词法作用域的校验属于语义阶段。

## 验收

macOS arm64 的 `make test-p1b-program` 检查 4 棵完整树、75 个节点和 15 个与 stage0 对齐的诊断，并解析 `compiler/` 下当时的全部 7 份行简源码。用 stage0 单独构建程序测试夹具，验证原生运行输出为 `2`。解析器产物通过 Mach-O、dyld、系统导入白名单和签名检查。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse test-p1b-program doctor license seed-audit
./tools/check-independence source
build/punct-scan compiler/program-grammar.serica compiler/parse-expression.serica
```

上述命令全部退出 0。活动 Go 种子为 9915/12000 行，C-1 为 6498/9493（68.5%）。P1b 继续推进，下一项为从节点表解析名称和作用域。
