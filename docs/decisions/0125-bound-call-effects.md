<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0125 绑定调用的能力声明与实际需求

`module-effect` 复用模块图、源码快照和 D-0124 绑定。每条调用记录包含所属函数、调用节点和源文件。函数声明使用文本字面量数组。对直接绑定记录的控制台入口，能力可直接声明，也可通过闭包内的常量引用取得该数组及其文本元素。`窗口` 保持原生窗口的入口规则。

函数调用者必须显式声明内建操作要求的能力及其所调用函数声明的能力。程序入口的需求使用调用图上的不动点传播，从入口可达的内建操作累加。未调用函数与被调用函数多声明的能力均不会自动加入入口需求。传播保留引入能力的内建节点，因此缺失的入口能力可以定位到依赖模块中的实际用处。

| 能力 | 位 |
|---|---:|
| `stdout` | 1 |
| `stdin` | 2 |
| `filesystem.read` | 4 |
| `filesystem.write` | 8 |
| `array.write` | 16 |
| `map.write` | 32 |
| `args` | 64 |

查询 `(7 << 60) + (n << 1) + k` 返回函数名称节点 `n` 的能力掩码。`n=0` 查询程序入口，`k=0` 查询声明，`k=1` 查询实际需求。诊断沿用文件编号和 UTF-8 字节偏移编码。

通用编译期表达式生成能力数组及入口记录、纯展开调用的效果实参限制、运行时函数 ABI 和类型检查由后续语义与原生发射处理。

## 验收

macOS arm64 的 `make test-p1b-module-effect` 验证全部七种能力、四个正例图、13 条与 stage0 对齐的位置诊断和编译器自身闭包。运行两个 stage0 参照产物，其中全能力样例验证了实际文件读写与输出。跨模块正例覆盖被调用函数声明的能力多于实际使用，以及未被入口调用的函数。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse test-p1b-program test-p1b-bind test-p1b-import-paths test-p1b-module-read test-p1b-module-graph test-p1b-module-bind test-p1b-module-effect doctor license seed-audit
./tools/check-independence source
build/punct-scan compiler/parse-expression.serica compiler/effects.serica
```

上述命令均退出 0。活动 Go 种子为 10282/12000 行，C-1 为 6586/9833（67.0%）。P1b 继续推进。
