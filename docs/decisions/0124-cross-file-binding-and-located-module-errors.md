<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# D-0124 跨文件绑定与模块读取位置诊断

`module-bind` 沿用 `module-graph` 的参数和图查询，增加对各文件正文快照的词法绑定。参数、局部值和循环控制沿用 D-0120 的规则。调用可查找全部闭包的函数，检查实参数量。常量初值只查找先完成文件的常量和同文件的更早声明。值与函数使用各自的名称空间，比较标识符时保留原始 UTF-8 字节。

| 查询 | 结果 |
|---|---|
| `-4` | 跨文件绑定数 |
| `(5 << 60) + n` | 节点 `n` 对应的声明名称节点、循环节点或负号表示的内建编号 |
| `(6 << 60) + n` | 节点 `n` 对应目标的文件编号 |

内建及无绑定节点的文件编号为 `0`。绑定错误沿用 D-0123 的文件和字节位置编码。单文件 `bind-program`、`bind-module` 保留原待链接查询。

## 受限读取结果

`readModuleResult` 沿用 D-0122 的参数、能力、目录约束和读取上限，返回拥有 UTF-8 映射的文本结果。路径失效和读取失败分别产生错误码 `41` 与 `42`，清理本次读取持有的描述符和映射。重新赋值释放先前成功结果的映射。文本结果复制复用按长度扩容的路径，能把超过初始映射容量的正文安全复制到文本变量。

`module-graph` 与 `module-bind` 使用该结果，把两类失败转换为发起导入位置的 `S-4001` 和 `S-4002`。入口本身失败关联入口编号 `1`、偏移 `0`。`read-import`、`read-module-import` 和 `readModule` 保留原退出状态。

## 验收

macOS arm64 的 `make test-p1b-module-bind` 验证 11 个绑定查询、6 个绑定诊断、4 个读取位置诊断，以及大文本成功、无效 UTF-8 失败、再次成功的结果恢复过程。编译器自身 7 文件闭包解析了 71 个跨文件引用。

```sh
env GO=/Users/ntars/.local/go1.27.4/bin/go make stage0 test-p0b-scope test-p1b-static test-p1b-modules test-p1b-runtime test-p1b-scan test-p1b-lex test-p1b-parse test-p1b-program test-p1b-bind test-p1b-import-paths test-p1b-module-read test-p1b-module-graph test-p1b-module-bind doctor license seed-audit
./tools/check-independence source
build/punct-scan compiler/parse-expression.serica compiler/binding-names.serica
```

上述命令均退出 0。活动 Go 种子 10282/12000 行，C-1 为 6586/9833（67.0%），逐行归因完整。P1b 继续推进。
