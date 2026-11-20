# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 零知识检查命令

每项命令须在工作树根运行；失败输出是证据，禁止改成只给建议的 warning。

| 要验证的事实 | 命令 | 期望结果 |
|---|---|---|
| Go 种子精确版本、系统 SDK、CJK 字体、punct-scan、依赖目录和缓存位置 | `make doctor` | `环境自检：通过`；任一依赖目录或 Go/Python 缓存落在仓库内即非零退出 |
| 源码级独立（K.2） | `./tools/check-independence source` | `源码级：通过`；clean checkout 也必须运行 |
| 代码区 NFC、半角标点与禁用字符 | `build/punct-scan <源码>` | `NFC=true 半角标点=0 禁用字符=0`；任何一项不满足即失败 |
| 词法器覆盖冻结符号与词类 | `build/stage0/serica lex examples/lexical-smoke.serica` | 成功输出标识符、关键字、普通/原始字符串及符号 token；该输入只作词法 smoke，不要求通过语法解析 |
| 窗口示例的记录值语法和字段 | `build/stage0/serica check examples/p0b-window.serica` | `源程序结构：通过`；源码 NFC 由 `punct-scan` 独立检查 |
| P0b 示例的原生窗口生成 | `build/stage0/serica build examples/p0b-window.serica -o build/p0b-window` | macOS arm64 Mach-O；`dyld_info -validate_only`、`otool -L`、导入符号和签名检查通过；人工看到中文窗口并用键盘改字 |
| P0b 同路径重复构建 | 先将 `build/p0b-window` 复制到仓库外，再重复运行上一条命令到相同路径并执行 `cmp` | 两次签名后字节一致；2026-11-20 SHA-256 `40513babb5cc02d69e49ebdbb6e73ab0c571d5802d3d44137fee290d7d9f878f` |
| P1b 字符串字面量拼接 | `build/stage0/serica check examples/p1b-string-concat.serica` 与 `build/stage0/serica build examples/p1b-string-concat.serica -o build/p1b-window` | 语义检查、Mach-O 校验通过；产物中含 `点击这里编辑中文` |
| P1b 字符串切片与索引常量加法 | `build/stage0/serica check examples/p1b-string-slice.serica` 与 `build/stage0/serica build examples/p1b-string-slice.serica -o build/p1b-slice-window` | `1+1`、`2+3` 求值成切片索引；语义检查、Mach-O 校验通过；产物中含 `语言表可运行` |
| P1b 字符串切片参数诊断 | 分别检查 `examples/diagnostics/p1b-substring-arity.serica`、`p1b-substring-type.serica`、`p1b-substring-bounds.serica` | 非零退出且诊断码依次为 `S-3011`、`S-3012`、`S-3013` |
| P1b 索引整数加法溢出 | `build/stage0/serica check examples/diagnostics/p1b-index-overflow.serica` | 返回 `S-3018` 非零诊断，不生成成功结果 |
| P1b 有符号整数算术 | `build/stage0/serica check examples/p1b-integer-arithmetic.serica` 与 `build/stage0/serica build examples/p1b-integer-arithmetic.serica -o build/p1b-integer-arithmetic` | 运算优先级、分组、符号边界与位函数求出“力，整数运算通过”；Mach-O/dyld 校验通过 |
| P1b 整数算术错误诊断 | 检查 `examples/diagnostics/p1b-multiply-overflow.serica`、`p1b-division-zero.serica`、`p1b-division-overflow.serica`、`p1b-negation-overflow.serica`、`p1b-arithmetic-type.serica` | 非零退出；诊断码依次为 `S-3018`、`S-3024`、`S-3018`、`S-3018`、`S-3023` |
| P1b 常量位运算 | `build/stage0/serica check examples/p1b-bitwise.serica` 与 `build/stage0/serica build examples/p1b-bitwise.serica -o build/p1b-bitwise-window` | 嵌套 `bitAnd`、`bitOr`、`bitXor` 计算索引；Mach-O 中含 `语言表位运算通过` |
| P1b 位运算参数诊断 | `build/stage0/serica check examples/diagnostics/p1b-bitwise-arity.serica` | 返回 `S-3021` 非零诊断 |
| P1b 常量绑定与引用 | `build/stage0/serica check examples/p1b-constants.serica` 与 `build/stage0/serica build examples/p1b-constants.serica -o build/p1b-constants-window` | 字符串和整数常量按声明顺序解析；Mach-O 中含 `语言表可解析` |
| P1b 常量绑定诊断 | 检查 `examples/diagnostics/p1b-constant-duplicate.serica`、`p1b-constant-forward-reference.serica`、`p1b-constant-undefined.serica`、`p1b-constant-mixed-types.serica` | 非零退出；诊断码依次为 `S-3007`、`S-3022`、`S-3022`、`S-3023` |
| P1b 字符串相等与常量分支 | `build/stage0/serica check examples/p1b-string-equality.serica` 与 `build/stage0/serica build examples/p1b-string-equality.serica -o build/p1b-equality-window` | 语义检查、Mach-O 校验通过；产物中含 `切片与相等通过，反例分支通过` |
| 活动 Go 种子行数 | `find bootstrap -path 'bootstrap/archive' -prune -o -name '*.go' -type f -print0 | xargs -0 wc -l` | 当前 1749 行，≤ 5000；归因表逐行覆盖编译器与检查工具，冻结原型排除 |
| P0b 窗口文件依赖白名单 | `otool -L build/p0b-window`；`nm -u build/p0b-window` | 仅有登记的 libSystem、libobjc、Foundation、AppKit 与相关 Objective-C runtime 符号 |
| 自举可复现性 | P3b 提供的 `make verify` | stage2/stage3 的 `.text`、`.rodata`、重定位表、导入符号表逐段一致；当前阶段未实现 |

禁止用全仓库 grep 统计代码区标点、字符串或注释；需区分代码与内容时，只用词法器区间。Go 子集黑名单的模式和注释／字符串排除规则随种子实现加入后，在本文件追加可复跑命令。
