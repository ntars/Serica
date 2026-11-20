<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright (c) 2026 行简（Serica）贡献者 -->

# 变更记录

本文件记录行简（Serica）的阶段里程碑与行为变更。结构遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，语言版本遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

**当前没有任何已发布的语言版本。** 下列条目中的 `P0b`、`P1b` 是开发阶段标记，不是版本号，不得用来推断兼容性或成熟度。语言版本从首个 `v0.1.0` 起算。

## [未发布]

### 新增

- P1b-06 顶层不可变常量声明与引用：`常量 中文名=表达式。`，声明须位于 `窗口` 根绑定之前，字符串与整数按表达式推断类型；仅允许引用此前声明的常量。重复名、前向引用、未定义引用与混合类型加法分别给出诊断。证据：`examples/p1b-constants.serica` 构建产物含「语言表可解析」。
- P1b-05 常量位运算子集：二元 `bitAnd`、`bitOr`、`bitXor`，允许嵌套与左结合 `+` 组合，结果可供切片索引使用。证据：`examples/p1b-bitwise.serica`；参数错误返回 `S-3021`。
- P1b-04 有符号 64 位整数常量四则运算：`+`、`-`、`*`、`/`、一元负号与全角括号分组，溢出与非整数操作数给出诊断，除零返回 `S-3024`。证据：`examples/p1b-integer-arithmetic.serica`。
- P1b-03 字符串相等与常量分支：`equals` 比较码位序列，`ifElse` 依据比较结果选择常量字符串，不做隐式 Unicode 规范化。证据：`examples/p1b-string-equality.serica`。
- P1b-02 字符串切片：`substring` 使用 Unicode 码位零起始、左闭右开区间；参数个数、类型与越界分别返回 `S-3011`、`S-3012`、`S-3013`。证据：`examples/p1b-string-slice.serica`。
- P1b-01 字符串字面量拼接：表达式 AST 支持左结合 ASCII `+`，仅接受字符串与字符串拼接。证据：`examples/p1b-string-concat.serica`。

### 变更

- 文档：README 按项目首页规范重写，补齐贡献指南、行为准则、安全政策、变更记录与 Issue／PR 模板。
- 文档：`LICENSING.md` 增补六条常见问题，回答「用它写程序要不要开源」「能否 fork 做闭源发行版」「兼容标签能否自用」。

### 已知偏差

- `docs/roadmap/p0b-p1b.md` 的 P1b-04 仍写「减法、乘除尚未实现」，与种子实现不符；`docs/build/module-tier-map.md` 记录的种子行数（1612）低于实测值（1749）。以代码与示例为准，待同步。

## [P0b] - 2026-11-20（阶段标记，非版本号）

### 新增

- stage0 种子编译器：位置化词法器、受限记录值 AST 与手写递归下降解析器、窗口字段语义检查、`lex`／`check`／`build` 三个子命令。证据：`bootstrap/` 当前 1749 行，上限 5000。
- 原生产物路径：直接写出 macOS arm64 Mach-O，不生成 C，不调用 LLVM、汇编器或链接器；仅使用系统 `codesign` 做 ad-hoc 签名。证据：`file build/p0b-window` 输出 `Mach-O 64-bit executable arm64`。
- 最小窗口与中文渲染：产物打开原生窗口显示「行简 P0b」与「点击这里编辑中文」，`NSTextView` 可接受键盘输入。窗口与文本使用已登记的 AppKit／Foundation／libobjc 通道，档位 `[B]` 过渡。
- 可复现构建：相同源码写入相同路径，重复签名构建字节一致；SHA-256 `40513babb5cc02d69e49ebdbb6e73ab0c571d5802d3d44137fee290d7d9f878f`。
- 检查工具：`build/punct-scan` 断言 NFC、半角标点与禁用字符；`tools/check-independence source` 核对源码级独立；`make doctor` 自检环境与依赖缓存位置。

### 说明

- 窗口关闭回调、系统中文输入法与除本机外的 macOS 版本均未核验，登记为后续项，不记为已支持。
- 未做第二台物理机器复验，可复现性结论限于单机。
