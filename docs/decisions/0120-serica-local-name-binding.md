<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0120 行简单文件名称绑定

## 实现

`compiler/parse-expression.serica` 增加 `bind-program` 和 `bind-module` 两种入口，共用 D-0119 语法树。名称绑定先登记全文件的常量和函数，再用显式栈按顺序遍历语句与表达式。函数名可向前查找。同文件常量初值只能引用更早声明的常量。变量初值先绑定，随后变量进入当前作用域。退出块时恢复进入前的可见范围。块函数参数和局部变量支持赋值；具体参数类型的可写规则留给类型检查。

函数调用绑定到函数名节点，并检查同文件声明的实参个数。值引用和赋值目标使用独立的值名称范围。循环控制绑定到最近的包围循环。字面量和记录字段名称不参与值查找。名称比较使用精确的 UTF-8 字节，可区分 `吉` 和 `𠮷`。

## 查询

查询从标准输入读取一个整数。两种绑定模式共用以下规则。

| 查询 | 返回值 |
|---|---|
| `-1` | 语法树根编号 |
| `-2` | 待模块链接的引用条数 |
| 语法树节点编号 | 绑定的声明名称节点编号、循环节点编号、内建编号或待链接标记 |
| 其他查询 | `0` |

声明本身、记录字段和字面量的绑定槽为 `0`。内建函数编号 `-1` 至 `-32` 按 `compiler/binding-names.serica` 中的表定义。存在导入的文件用 `-33` 表示待链接值引用、`-34` 表示待链接调用。该状态只保留待查找的节点，后续模块链接须验证依赖闭包和所有剩余引用。没有导入的文件立即诊断未知名称。语法和绑定错误返回 `-(UTF-8 字节偏移 * 8192 + S 错误码的数字部分)`。原语法查询编码沿用 D-0119。

## 验收

macOS arm64 上的 `make test-p1b-bind` 检查 26 个节点绑定、19 个与 stage0 对齐的诊断、两条待链接引用，以及 8 份编译器源码。测试为正例使用真实 Mach-O 解析器，并由 stage0 构建与执行两个参照程序。测试还遍历 stage0 接受的 95 个现有示例，93 个完成文件内绑定，另外 2 个模块示例各有两条待链接引用。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse test-p1b-program test-p1b-bind doctor license seed-audit
./tools/check-independence source
build/punct-scan compiler/binding-names.serica compiler/program-grammar.serica compiler/parse-expression.serica
```

上述命令均退出 0。Go 种子为 9915/12000 行，C-1 为 6498/9493（68.5%）。后续接入文件依赖图、模块导出与跨文件名称检查，再将绑定结果交给类型、效果检查和原生发射。P1b 继续推进。
