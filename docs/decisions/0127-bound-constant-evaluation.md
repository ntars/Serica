<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0127 绑定树的静态求值

`module-const` 在模块绑定、能力和类型形态检查后，按依赖完成顺序求常量初值，并尝试求控制台输出和窗口字段。引用读取绑定声明的已存值，求值使用显式栈。当前覆盖有符号 64 位整数、布尔条件、UTF-8 文本和同类标量数组，包括整数算术、位运算、文本比较、码位切片与长度、字节访问和数组索引。

算术在执行前检查边界，溢出、除零、移位量非法和越界读取沿用源码文件与字节偏移诊断。且、故和整数选择按分支短路。文本选择检查两个文本分支，以复跑现有 stage0 的静态行为。直接数组的长度按元素数求值，仍检查元素是否属于静态表达式。获取元素时求数组实际值。普通字符串按转义解码，原始字符串和 Unicode 标量保留原编码。

`-(2 << 60) - 2*n - k` 查询常量名称、初值节点或已求值总入口字段 `n`。当 `k=0` 时返回类型形态，当 `k=1` 时返回值。整数使用有符号原值，布尔使用 0 或 1，文本与数组返回值池偏移。通过 `-(3 << 60) - p` 读取值池槽 `p`。记录依次保存长度、元素类型与数据槽。文本长度取 UTF-8 字节数，元素类型槽固定为 0。文本数组中的元素槽保存对应文本记录偏移。

类型形态 0 标识待定或没有登记值的节点。用户函数调用、运行时输入及效果操作留待调用展开和原生发射；效果验证继续使用 D-0125 的声明规则。

stage0 原生整数发射向 `evaluateArrayLength` 传入完整调用，修复有运行时操作数时无法使用静态数组长度的问题。回归使用 stdin 的整数和三项常量数组，输入 `9` 得到 `12`。

## 验收

`make test-p1b-module-const` 在 macOS arm64 通过 10 个窗口文本、9 个边界值、5 个实际运行的参照产物、19 个与 stage0 对齐的位置诊断，以及跨文件常量查询和编译器自身闭包。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make -j4 stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse test-p1b-program test-p1b-bind test-p1b-import-paths test-p1b-module-read test-p1b-module-graph test-p1b-module-bind test-p1b-module-effect test-p1b-module-type test-p1b-module-const doctor license seed-audit
env GO=/Users/ntars/.local/go1.27.4/bin/go make -j4 test-p1b-parse test-p1b-program test-p1b-bind test-p1b-import-paths test-p1b-module-read test-p1b-module-graph test-p1b-module-bind test-p1b-module-effect test-p1b-module-type test-p1b-module-const
./tools/check-independence source
build/punct-scan compiler/parse-expression.serica compiler/const-ops.serica
```

四条命令均退出 0。全量回归通过 16 个测试目标。补齐直接数组长度与元素取值的差分场景后，重新执行十个前端目标，确认新增的第 9 个边界值和第 19 个位置诊断。编译器闭包为 10 文件、12 条依赖边、98 条跨文件绑定。活动 Go 种子为 10282/12000 行，C-1 为 6586/9833（67.0%）。P1b 继续推进。
