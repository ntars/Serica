# SPDX-License-Identifier: MIT
# Copyright (c) 2026 行简（Serica）贡献者

# 种子逐行归因

范围：活动 Go 种子包含 `bootstrap/*.go` 与 `bootstrap/tools/*.go`；冻结选型原型 `bootstrap/archive/prototype-3.8/` 不属于活动种子。统计命令：`find bootstrap -path 'bootstrap/archive' -prune -o -name '*.go' -type f -print0 | xargs -0 wc -l`。当前合计 1237 行，低于 5000 行上限。

| 文件与行 | 能力 | 对应需求 |
|---|---|---|
| `bootstrap/main.go:1-29` | CLI 命令路由、版本与用法 | P0b 种子入口；`serica lex/check/build` |
| `bootstrap/main.go:31-58` | 词法输出与 token 统计 | 冻结词法 smoke；`serica lex [--stats]` |
| `bootstrap/main.go:60-67` | 源码检查与四段式诊断调用 | P0b `serica check` |
| `bootstrap/main.go:69-86` | 构建参数、窗口字段提取与目标文件写出 | P0b 源码到可执行产物 |
| `bootstrap/main.go:88-111` | UTF-8 源码读取、共享 lex/parse/semantic 流程、窗口字段提取 | P0b `check`/`build` 共用前端 |
| `bootstrap/main.go:113-129` | token 源码读取、UTF-8 拒绝和位置化错误输出 | 手册 G.7；诊断与输入边界 |
| `bootstrap/main.go:131-148` | token 类别计数 | `serica lex --stats` |
| `bootstrap/lexer.go:1-22` | token、源码位置、问题结构及 UTF-8 验证 | P0b 前端基础 |
| `bootstrap/lexer.go:24-96` | 注释、字符串、标识符、数字、符号切分及错误 | 手册 1.3；P0b 源码词法 |
| `bootstrap/lexer.go:98-138` | 中日韩标识符、禁用字符、标点修复和关键字识别 | 手册 1.3、冻结符号/词类 |
| `bootstrap/lexer.go:140-160` | 码位列、显示列和词法诊断位置 | 手册 1.3 诊断要求 |
| `bootstrap/parser.go:1-25` | P0b 程序、字面量、记录字段与解析器状态 | D-0014 的受限源子集 |
| `bootstrap/parser.go:27-51` | 唯一 `窗口` 根绑定、等号、终止符和 EOF 检查 | P0b 根语法 |
| `bootstrap/parser.go:53-72` | 普通/原始字符串字面量和记录值入口；拒绝表达式、整数和调用 | P0b 源子集；冻结 P1b 语义 |
| `bootstrap/parser.go:74-101` | 全角分隔符包围的记录与字段解析 | P0b 窗口记录 |
| `bootstrap/parser.go:103-132` | 普通串转义与原始串解码 | 手册 1.3 字符串规则 |
| `bootstrap/parser.go:134-176` | kind/symbol 期望、EOF 与 token 位置诊断 | 手册 G.7；S-2xxx 语法错误 |
| `bootstrap/semantic.go:1-29` | 记录类型、标题/文本字段、唯一字段及字符串字面量检查 | D-0014；P0b 窗口数据契约 |
| `bootstrap/macho.go:1-49` | 平台限制、Mach-O 写盘和系统 ad-hoc 签名 | P0b macOS arm64 产物路径 |
| `bootstrap/macho.go:51-112` | Mach-O 段、节、加载命令、符号与 UUID 写出 | 自有 Mach-O 写出器；不生成 C 或调用链接器 |
| `bootstrap/macho.go:114-137` | arm64 ADR/ADRP 与寄存器加载指令 | P0b 窗口代码生成 |
| `bootstrap/window_macho.go:1-27` | 导入符号、标签和地址修正数据 | P0b 原生窗口后端 |
| `bootstrap/window_macho.go:29-115` | 映像布局、系统依赖、绑定与符号表 | P0b Mach-O 产物与平台白名单 |
| `bootstrap/window_macho.go:117-225` | Objective-C 消息、AppKit 窗口、可编辑 NSTextView 与关闭代理 | P0b 最小运行时与交互验收 |
| `bootstrap/window_macho.go:227-305` | 类/selector 调用、常量布局、NSRect ABI 和地址记录 | Apple arm64 参数传递 |
| `bootstrap/window_macho.go:307-399` | 地址回填、dyld bind、Mach-O 表格和指令写出 | P0b 目标文件后端 |
| `bootstrap/tools/punct_scan.go:1-86` | NFC（通过系统 Python 标准库）、代码区标点和禁用字符检查 | 手册 1.3；`make doctor` |
| `bootstrap/tools/check_independence.go:1-102` | 源码依赖与许可证边界检查 | 手册 K.2；`tools/check-independence source` |

当前活动种子只包含 P0b 词法、窗口字面量记录解析/检查、CLI、Mac Mach-O 与必要的本地审计工具。P1b 编译期常量/算术/数组/模块实现已不在活动 Go 种子中；相关设计和样例仅供未来行简实现使用。行数低于上限不代表自举约束已经完成，`bootstrap/` 仍须在 P3b 删除。
